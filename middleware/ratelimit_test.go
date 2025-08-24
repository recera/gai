package middleware

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

func TestRateLimitMiddleware_BasicRateLimit(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "success"}, nil
		},
	}

	opts := RateLimitOpts{
		RPS:   2, // 2 requests per second
		Burst: 2,
	}
	
	provider := WithRateLimit(opts)(mock)
	
	ctx := context.Background()
	start := time.Now()
	
	// First 2 requests should go through immediately (burst)
	for i := 0; i < 2; i++ {
		_, err := provider.GenerateText(ctx, core.Request{})
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
	}
	
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("burst requests took too long: %v", elapsed)
	}
	
	// Third request should be rate limited
	start = time.Now()
	_, err := provider.GenerateText(ctx, core.Request{})
	elapsed = time.Since(start)
	
	if err != nil {
		t.Fatalf("rate limited request failed: %v", err)
	}
	
	// Should have waited approximately 500ms (1/2 RPS)
	if elapsed < 400*time.Millisecond || elapsed > 600*time.Millisecond {
		t.Errorf("rate limit delay incorrect: %v", elapsed)
	}
}

func TestRateLimitMiddleware_WaitTimeout(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "success"}, nil
		},
	}

	opts := RateLimitOpts{
		RPS:         1,
		Burst:       1,
		WaitTimeout: 50 * time.Millisecond,
	}
	
	provider := WithRateLimit(opts)(mock)
	
	ctx := context.Background()
	
	// First request should succeed
	_, err := provider.GenerateText(ctx, core.Request{})
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	
	// Second request should timeout
	start := time.Now()
	_, err = provider.GenerateText(ctx, core.Request{})
	elapsed := time.Since(start)
	
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !core.IsRateLimited(err) {
		t.Errorf("expected rate limit error, got %v", err)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

func TestRateLimitMiddleware_PerMethodLimits(t *testing.T) {
	generateCount := int32(0)
	streamCount := int32(0)
	
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			atomic.AddInt32(&generateCount, 1)
			return &core.TextResult{Text: "generate"}, nil
		},
		streamTextFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			atomic.AddInt32(&streamCount, 1)
			return &mockTextStream{}, nil
		},
	}

	opts := RateLimitOpts{
		RPS:   1, // Global fallback
		Burst: 1,
		PerMethod: map[string]*RateLimitConfig{
			"GenerateText": {RPS: 5, Burst: 5},
			"StreamText":   {RPS: 2, Burst: 2},
		},
	}
	
	provider := WithRateLimit(opts)(mock)
	
	ctx := context.Background()
	
	// GenerateText should allow 5 burst requests
	start := time.Now()
	for i := 0; i < 5; i++ {
		_, err := provider.GenerateText(ctx, core.Request{})
		if err != nil {
			t.Fatalf("generate request %d failed: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	
	if elapsed > 100*time.Millisecond {
		t.Errorf("generate burst took too long: %v", elapsed)
	}
	if atomic.LoadInt32(&generateCount) != 5 {
		t.Errorf("expected 5 generate calls, got %d", generateCount)
	}
	
	// StreamText should allow 2 burst requests
	start = time.Now()
	for i := 0; i < 2; i++ {
		_, err := provider.StreamText(ctx, core.Request{})
		if err != nil {
			t.Fatalf("stream request %d failed: %v", i, err)
		}
	}
	elapsed = time.Since(start)
	
	if elapsed > 100*time.Millisecond {
		t.Errorf("stream burst took too long: %v", elapsed)
	}
	if atomic.LoadInt32(&streamCount) != 2 {
		t.Errorf("expected 2 stream calls, got %d", streamCount)
	}
}

func TestRateLimitMiddleware_ConcurrentRequests(t *testing.T) {
	successCount := int32(0)
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			atomic.AddInt32(&successCount, 1)
			time.Sleep(10 * time.Millisecond) // Simulate work
			return &core.TextResult{Text: "success"}, nil
		},
	}

	opts := RateLimitOpts{
		RPS:   10,
		Burst: 5,
	}
	
	provider := WithRateLimit(opts)(mock)
	
	ctx := context.Background()
	var wg sync.WaitGroup
	errors := make(chan error, 20)
	
	// Launch 20 concurrent requests
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := provider.GenerateText(ctx, core.Request{})
			if err != nil {
				errors <- err
			}
		}()
	}
	
	// Wait for all requests with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("concurrent requests timed out")
	}
	
	close(errors)
	
	// Check for errors
	var errCount int
	for err := range errors {
		t.Errorf("request error: %v", err)
		errCount++
	}
	
	if errCount > 0 {
		t.Fatalf("%d requests failed", errCount)
	}
	
	// All requests should have succeeded
	if atomic.LoadInt32(&successCount) != 20 {
		t.Errorf("expected 20 successful requests, got %d", successCount)
	}
}

func TestRateLimitMiddleware_ContextCancellation(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "success"}, nil
		},
	}

	opts := RateLimitOpts{
		RPS:   1,
		Burst: 1,
	}
	
	provider := WithRateLimit(opts)(mock)
	
	// Use up the burst
	ctx := context.Background()
	_, err := provider.GenerateText(ctx, core.Request{})
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	
	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start a request that will need to wait
	errChan := make(chan error)
	go func() {
		_, err := provider.GenerateText(ctx, core.Request{})
		errChan <- err
	}()
	
	// Cancel context while waiting
	time.Sleep(50 * time.Millisecond)
	cancel()
	
	// Check that the request was cancelled
	select {
	case err := <-errChan:
		if err == nil {
			t.Fatal("expected context cancellation error")
		}
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("request didn't respond to context cancellation")
	}
}

func TestRateLimitMiddleware_OnRateLimited(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "success"}, nil
		},
	}

	var rateLimitedMethod string
	var rateLimitedWait time.Duration
	
	opts := RateLimitOpts{
		RPS:   1,
		Burst: 1,
		OnRateLimited: func(method string, wait time.Duration) {
			rateLimitedMethod = method
			rateLimitedWait = wait
		},
	}
	
	provider := WithRateLimit(opts)(mock)
	
	ctx := context.Background()
	
	// First request uses the burst
	_, err := provider.GenerateText(ctx, core.Request{})
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	
	// Second request should be rate limited
	_, err = provider.GenerateText(ctx, core.Request{})
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	
	// Check that callback was called
	if rateLimitedMethod != "GenerateText" {
		t.Errorf("expected method 'GenerateText', got '%s'", rateLimitedMethod)
	}
	if rateLimitedWait < 900*time.Millisecond || rateLimitedWait > 1100*time.Millisecond {
		t.Errorf("unexpected wait time: %v", rateLimitedWait)
	}
}

func TestRateLimitMiddleware_UpdateRateLimit(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "success"}, nil
		},
	}

	opts := RateLimitOpts{
		RPS:   1,
		Burst: 1,
	}
	
	provider := WithRateLimit(opts)(mock)
	middleware := provider.(*rateLimitMiddleware)
	
	ctx := context.Background()
	
	// Use initial limit
	_, err := provider.GenerateText(ctx, core.Request{})
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	
	// Update to higher limit
	middleware.UpdateRateLimit("", 10, 10)
	
	// Should now allow multiple requests quickly
	start := time.Now()
	for i := 0; i < 5; i++ {
		_, err := provider.GenerateText(ctx, core.Request{})
		if err != nil {
			t.Fatalf("request %d after update failed: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	
	// Should complete quickly with new higher limit
	if elapsed > 200*time.Millisecond {
		t.Errorf("requests after limit update took too long: %v", elapsed)
	}
}

func TestRateLimitMiddleware_AllMethods(t *testing.T) {
	mock := &mockProvider{}
	
	opts := RateLimitOpts{
		RPS:   100, // High limit to avoid blocking
		Burst: 10,
	}
	
	provider := WithRateLimit(opts)(mock)
	ctx := context.Background()
	
	// Test GenerateText
	result, err := provider.GenerateText(ctx, core.Request{})
	if err != nil {
		t.Errorf("GenerateText failed: %v", err)
	}
	if result == nil {
		t.Error("GenerateText returned nil result")
	}
	
	// Test StreamText
	stream, err := provider.StreamText(ctx, core.Request{})
	if err != nil {
		t.Errorf("StreamText failed: %v", err)
	}
	if stream == nil {
		t.Error("StreamText returned nil stream")
	}
	
	// Test GenerateObject
	objResult, err := provider.GenerateObject(ctx, core.Request{}, nil)
	if err != nil {
		t.Errorf("GenerateObject failed: %v", err)
	}
	if objResult == nil {
		t.Error("GenerateObject returned nil result")
	}
	
	// Test StreamObject
	objStream, err := provider.StreamObject(ctx, core.Request{}, nil)
	if err != nil {
		t.Errorf("StreamObject failed: %v", err)
	}
	if objStream == nil {
		t.Error("StreamObject returned nil stream")
	}
}