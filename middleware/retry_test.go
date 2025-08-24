package middleware

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

// mockProvider implements core.Provider for testing.
type mockProvider struct {
	generateTextFunc   func(context.Context, core.Request) (*core.TextResult, error)
	streamTextFunc     func(context.Context, core.Request) (core.TextStream, error)
	generateObjectFunc func(context.Context, core.Request, any) (*core.ObjectResult[any], error)
	streamObjectFunc   func(context.Context, core.Request, any) (core.ObjectStream[any], error)
	callCount          int32
}

func (m *mockProvider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	atomic.AddInt32(&m.callCount, 1)
	if m.generateTextFunc != nil {
		return m.generateTextFunc(ctx, req)
	}
	return &core.TextResult{Text: "test"}, nil
}

func (m *mockProvider) StreamText(ctx context.Context, req core.Request) (core.TextStream, error) {
	atomic.AddInt32(&m.callCount, 1)
	if m.streamTextFunc != nil {
		return m.streamTextFunc(ctx, req)
	}
	return &mockTextStream{}, nil
}

func (m *mockProvider) GenerateObject(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
	atomic.AddInt32(&m.callCount, 1)
	if m.generateObjectFunc != nil {
		return m.generateObjectFunc(ctx, req, schema)
	}
	return &core.ObjectResult[any]{Value: "test"}, nil
}

func (m *mockProvider) StreamObject(ctx context.Context, req core.Request, schema any) (core.ObjectStream[any], error) {
	atomic.AddInt32(&m.callCount, 1)
	if m.streamObjectFunc != nil {
		return m.streamObjectFunc(ctx, req, schema)
	}
	return &mockObjectStream{}, nil
}

func (m *mockProvider) getCallCount() int {
	return int(atomic.LoadInt32(&m.callCount))
}

// mockTextStream implements core.TextStream for testing.
type mockTextStream struct {
	events chan core.Event
}

func (m *mockTextStream) Events() <-chan core.Event {
	if m.events == nil {
		m.events = make(chan core.Event)
		close(m.events)
	}
	return m.events
}

func (m *mockTextStream) Close() error {
	return nil
}

// mockObjectStream implements core.ObjectStream for testing.
type mockObjectStream struct {
	mockTextStream
}

func (m *mockObjectStream) Final() (*any, error) {
	result := any("test")
	return &result, nil
}

func TestRetryMiddleware_SuccessOnFirstAttempt(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "success"}, nil
		},
	}

	opts := RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
	}
	
	provider := WithRetry(opts)(mock)
	
	ctx := context.Background()
	result, err := provider.GenerateText(ctx, core.Request{})
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "success" {
		t.Errorf("expected 'success', got '%s'", result.Text)
	}
	if mock.getCallCount() != 1 {
		t.Errorf("expected 1 call, got %d", mock.getCallCount())
	}
}

func TestRetryMiddleware_RetryOnTransientError(t *testing.T) {
	attempts := 0
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			attempts++
			if attempts < 3 {
				return nil, core.NewError(core.ErrorProviderUnavailable, "transient error", core.WithProvider("test"))
			}
			return &core.TextResult{Text: "success after retries"}, nil
		},
	}

	opts := RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		Jitter:      false, // Disable jitter for predictable timing
	}
	
	provider := WithRetry(opts)(mock)
	
	ctx := context.Background()
	start := time.Now()
	result, err := provider.GenerateText(ctx, core.Request{})
	elapsed := time.Since(start)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "success after retries" {
		t.Errorf("expected 'success after retries', got '%s'", result.Text)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	// Should have waited ~10ms + ~20ms = ~30ms (exponential backoff)
	if elapsed < 25*time.Millisecond {
		t.Errorf("retry delays too short: %v", elapsed)
	}
}

func TestRetryMiddleware_NoRetryOnBadRequest(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return nil, core.NewError(core.ErrorInvalidRequest, "bad request", core.WithProvider("test"))
		},
	}

	opts := RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
	}
	
	provider := WithRetry(opts)(mock)
	
	ctx := context.Background()
	_, err := provider.GenerateText(ctx, core.Request{})
	
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !core.IsBadRequest(err) {
		t.Errorf("expected bad request error, got %v", err)
	}
	if mock.getCallCount() != 1 {
		t.Errorf("expected 1 call (no retries), got %d", mock.getCallCount())
	}
}

func TestRetryMiddleware_RateLimitWithRetryAfter(t *testing.T) {
	retryAfterSeconds := 1
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return nil, core.NewError(core.ErrorRateLimited, "rate limited", 
				core.WithProvider("test"),
				core.WithRetryAfter(time.Duration(retryAfterSeconds)*time.Second))
		},
	}

	opts := RetryOpts{
		MaxAttempts: 1,
		BaseDelay:   100 * time.Millisecond,
	}
	
	provider := WithRetry(opts)(mock)
	
	ctx := context.Background()
	start := time.Now()
	_, _ = provider.GenerateText(ctx, core.Request{})
	elapsed := time.Since(start)
	
	// Should wait for the retry-after duration
	if elapsed < time.Duration(retryAfterSeconds)*time.Second {
		t.Errorf("didn't wait for retry-after: elapsed %v", elapsed)
	}
	if mock.getCallCount() != 2 {
		t.Errorf("expected 2 calls, got %d", mock.getCallCount())
	}
}

func TestRetryMiddleware_ExhaustedAttempts(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return nil, core.NewError(core.ErrorProviderUnavailable, "always fails", core.WithProvider("test"))
		},
	}

	opts := RetryOpts{
		MaxAttempts: 2,
		BaseDelay:   1 * time.Millisecond,
	}
	
	provider := WithRetry(opts)(mock)
	
	ctx := context.Background()
	_, err := provider.GenerateText(ctx, core.Request{})
	
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if !core.IsTransient(err) {
		t.Errorf("expected transient error, got %v", err)
	}
	if mock.getCallCount() != 3 { // Initial + 2 retries
		t.Errorf("expected 3 calls, got %d", mock.getCallCount())
	}
}

func TestRetryMiddleware_ContextCancellation(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return nil, core.NewError(core.ErrorProviderUnavailable, "transient", core.WithProvider("test"))
		},
	}

	opts := RetryOpts{
		MaxAttempts: 10,
		BaseDelay:   100 * time.Millisecond,
	}
	
	provider := WithRetry(opts)(mock)
	
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	start := time.Now()
	_, err := provider.GenerateText(ctx, core.Request{})
	elapsed := time.Since(start)
	
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
	// Should fail quickly due to context timeout
	if elapsed > 100*time.Millisecond {
		t.Errorf("took too long despite context timeout: %v", elapsed)
	}
	// Should have attempted at least once but not all retries
	calls := mock.getCallCount()
	if calls < 1 || calls > 3 {
		t.Errorf("unexpected call count: %d", calls)
	}
}

func TestRetryMiddleware_CustomRetryIf(t *testing.T) {
	customErr := errors.New("custom error")
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return nil, customErr
		},
	}

	retryCount := 0
	opts := RetryOpts{
		MaxAttempts: 2,
		BaseDelay:   1 * time.Millisecond,
		RetryIf: func(err error) bool {
			retryCount++
			return errors.Is(err, customErr)
		},
	}
	
	provider := WithRetry(opts)(mock)
	
	ctx := context.Background()
	_, err := provider.GenerateText(ctx, core.Request{})
	
	if !errors.Is(err, customErr) {
		t.Errorf("expected custom error, got %v", err)
	}
	if retryCount != 3 { // Called for initial attempt + 2 retries
		t.Errorf("expected RetryIf to be called 3 times, got %d", retryCount)
	}
	if mock.getCallCount() != 3 {
		t.Errorf("expected 3 attempts, got %d", mock.getCallCount())
	}
}

func TestRetryMiddleware_ExponentialBackoff(t *testing.T) {
	attempts := 0
	delays := []time.Duration{}
	lastCallTime := time.Now()
	
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			now := time.Now()
			if attempts > 0 {
				delays = append(delays, now.Sub(lastCallTime))
			}
			lastCallTime = now
			attempts++
			
			if attempts <= 3 {
				return nil, core.NewError(core.ErrorProviderUnavailable, "transient", core.WithProvider("test"))
			}
			return &core.TextResult{Text: "success"}, nil
		},
	}

	opts := RetryOpts{
		MaxAttempts: 3,
		BaseDelay:   10 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      false,
	}
	
	provider := WithRetry(opts)(mock)
	
	ctx := context.Background()
	_, err := provider.GenerateText(ctx, core.Request{})
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Check exponential backoff: ~10ms, ~20ms, ~40ms
	if len(delays) != 3 {
		t.Fatalf("expected 3 delays, got %d", len(delays))
	}
	
	expectedDelays := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		40 * time.Millisecond,
	}
	
	for i, delay := range delays {
		// Allow 5ms tolerance for timing variations
		diff := delay - expectedDelays[i]
		if diff < -5*time.Millisecond || diff > 5*time.Millisecond {
			t.Errorf("delay %d: expected ~%v, got %v", i, expectedDelays[i], delay)
		}
	}
}

func TestRetryMiddleware_MaxDelay(t *testing.T) {
	attempts := 0
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			attempts++
			if attempts < 5 {
				return nil, core.NewError(core.ErrorProviderUnavailable, "transient", core.WithProvider("test"))
			}
			return &core.TextResult{Text: "success"}, nil
		},
	}

	opts := RetryOpts{
		MaxAttempts: 5,
		BaseDelay:   10 * time.Millisecond,
		MaxDelay:    25 * time.Millisecond, // Cap at 25ms
		Multiplier:  10.0,                  // Would exceed max without cap
		Jitter:      false,
	}
	
	provider := WithRetry(opts)(mock)
	
	ctx := context.Background()
	start := time.Now()
	_, err := provider.GenerateText(ctx, core.Request{})
	elapsed := time.Since(start)
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Total delay should be approximately: 10 + 25 + 25 + 25 = 85ms
	// (first delay is 10ms, rest are capped at 25ms)
	if elapsed < 75*time.Millisecond || elapsed > 120*time.Millisecond {
		t.Errorf("total delay out of expected range: %v", elapsed)
	}
}

func TestRetryMiddleware_StreamingRetry(t *testing.T) {
	attempts := 0
	mock := &mockProvider{
		streamTextFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			attempts++
			if attempts < 2 {
				return nil, core.NewError(core.ErrorProviderUnavailable, "connection failed", core.WithProvider("test"))
			}
			return &mockTextStream{}, nil
		},
	}

	opts := RetryOpts{
		MaxAttempts: 2,
		BaseDelay:   1 * time.Millisecond,
	}
	
	provider := WithRetry(opts)(mock)
	
	ctx := context.Background()
	stream, err := provider.StreamText(ctx, core.Request{})
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stream == nil {
		t.Fatal("expected stream, got nil")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}