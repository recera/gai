package middleware

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/recera/gai/core"
)

func TestChain_OrderOfExecution(t *testing.T) {
	var executionOrder []string
	
	// Create middleware that record their execution
	middleware1 := func(provider core.Provider) core.Provider {
		return &orderTrackingProvider{
			baseMiddleware: baseMiddleware{provider: provider},
			name:          "middleware1",
			order:         &executionOrder,
		}
	}
	
	middleware2 := func(provider core.Provider) core.Provider {
		return &orderTrackingProvider{
			baseMiddleware: baseMiddleware{provider: provider},
			name:          "middleware2",
			order:         &executionOrder,
		}
	}
	
	middleware3 := func(provider core.Provider) core.Provider {
		return &orderTrackingProvider{
			baseMiddleware: baseMiddleware{provider: provider},
			name:          "middleware3",
			order:         &executionOrder,
		}
	}
	
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			executionOrder = append(executionOrder, "provider")
			return &core.TextResult{Text: "result"}, nil
		},
	}
	
	// Chain middleware: 1 -> 2 -> 3 -> provider
	chained := Chain(middleware1, middleware2, middleware3)(mock)
	
	ctx := context.Background()
	_, err := chained.GenerateText(ctx, core.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Check execution order
	expected := []string{"middleware1", "middleware2", "middleware3", "provider"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("expected %d executions, got %d", len(expected), len(executionOrder))
	}
	
	for i, name := range expected {
		if executionOrder[i] != name {
			t.Errorf("position %d: expected %s, got %s", i, name, executionOrder[i])
		}
	}
}

type orderTrackingProvider struct {
	baseMiddleware
	name  string
	order *[]string
}

func (p *orderTrackingProvider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	*p.order = append(*p.order, p.name)
	return p.provider.GenerateText(ctx, req)
}

func TestChain_EmptyChain(t *testing.T) {
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "direct"}, nil
		},
	}
	
	// Empty chain should return provider unchanged
	chained := Chain()(mock)
	
	ctx := context.Background()
	result, err := chained.GenerateText(ctx, core.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if result.Text != "direct" {
		t.Errorf("expected 'direct', got '%s'", result.Text)
	}
	
	// Verify it's the same instance
	if chained != mock {
		t.Error("empty chain should return the same provider instance")
	}
}

func TestChain_SingleMiddleware(t *testing.T) {
	var called bool
	middleware := func(provider core.Provider) core.Provider {
		return &baseMiddleware{
			provider: &mockProvider{
				generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
					called = true
					return provider.GenerateText(ctx, req)
				},
			},
		}
	}
	
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "result"}, nil
		},
	}
	
	chained := Chain(middleware)(mock)
	
	ctx := context.Background()
	result, err := chained.GenerateText(ctx, core.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if !called {
		t.Error("middleware was not called")
	}
	if result.Text != "result" {
		t.Errorf("expected 'result', got '%s'", result.Text)
	}
}

func TestChain_RealMiddleware(t *testing.T) {
	// Test with actual middleware implementations
	attempts := int32(0)
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			count := atomic.AddInt32(&attempts, 1)
			if count == 1 {
				// First attempt fails with transient error
				return nil, core.NewError(core.ErrorInternal, "temporary failure", core.WithProvider("test"))
			}
			// Check if content was filtered
			if len(req.Messages) > 0 && len(req.Messages[0].Parts) > 0 {
				if text, ok := req.Messages[0].Parts[0].(core.Text); ok {
					return &core.TextResult{Text: text.Text}, nil
				}
			}
			return &core.TextResult{Text: "success"}, nil
		},
	}
	
	// Chain retry, rate limit, and safety middleware
	provider := Chain(
		WithRetry(RetryOpts{
			MaxAttempts: 2,
			BaseDelay:   1 * time.Millisecond,
		}),
		WithRateLimit(RateLimitOpts{
			RPS:   100,
			Burst: 10,
		}),
		WithSafety(SafetyOpts{
			RedactPatterns:    []string{`\d{3}-\d{2}-\d{4}`},
			RedactReplacement: "[SSN]",
		}),
	)(mock)
	
	ctx := context.Background()
	req := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "My SSN is 123-45-6789"},
				},
			},
		},
	}
	
	result, err := provider.GenerateText(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Check that:
	// 1. Retry worked (2 attempts)
	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts (retry), got %d", attempts)
	}
	
	// 2. Safety filtering worked
	if result.Text != "My SSN is [SSN]" {
		t.Errorf("expected SSN to be redacted, got: %s", result.Text)
	}
}

func TestBaseMiddleware_AllMethods(t *testing.T) {
	// Test that baseMiddleware correctly delegates all methods
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return &core.TextResult{Text: "generate"}, nil
		},
		streamTextFunc: func(ctx context.Context, req core.Request) (core.TextStream, error) {
			return &mockTextStream{}, nil
		},
		generateObjectFunc: func(ctx context.Context, req core.Request, schema any) (*core.ObjectResult[any], error) {
			return &core.ObjectResult[any]{Value: "object"}, nil
		},
		streamObjectFunc: func(ctx context.Context, req core.Request, schema any) (core.ObjectStream[any], error) {
			return &mockObjectStream{}, nil
		},
	}
	
	base := &baseMiddleware{provider: mock}
	ctx := context.Background()
	
	// Test GenerateText
	textResult, err := base.GenerateText(ctx, core.Request{})
	if err != nil {
		t.Errorf("GenerateText failed: %v", err)
	}
	if textResult.Text != "generate" {
		t.Errorf("expected 'generate', got '%s'", textResult.Text)
	}
	
	// Test StreamText
	stream, err := base.StreamText(ctx, core.Request{})
	if err != nil {
		t.Errorf("StreamText failed: %v", err)
	}
	if stream == nil {
		t.Error("StreamText returned nil")
	}
	
	// Test GenerateObject
	objResult, err := base.GenerateObject(ctx, core.Request{}, nil)
	if err != nil {
		t.Errorf("GenerateObject failed: %v", err)
	}
	if objResult.Value != "object" {
		t.Errorf("expected 'object', got '%v'", objResult.Value)
	}
	
	// Test StreamObject
	objStream, err := base.StreamObject(ctx, core.Request{}, nil)
	if err != nil {
		t.Errorf("StreamObject failed: %v", err)
	}
	if objStream == nil {
		t.Error("StreamObject returned nil")
	}
}

func TestChain_ErrorPropagation(t *testing.T) {
	expectedErr := errors.New("provider error")
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			return nil, expectedErr
		},
	}
	
	// Create a middleware that should pass errors through
	middleware := func(provider core.Provider) core.Provider {
		return provider
	}
	
	chained := Chain(middleware, middleware)(mock)
	
	ctx := context.Background()
	_, err := chained.GenerateText(ctx, core.Request{})
	
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestChain_ContextPropagation(t *testing.T) {
	var receivedCtx context.Context
	mock := &mockProvider{
		generateTextFunc: func(ctx context.Context, req core.Request) (*core.TextResult, error) {
			receivedCtx = ctx
			return &core.TextResult{Text: "result"}, nil
		},
	}
	
	// Middleware that adds a value to context
	middleware := func(provider core.Provider) core.Provider {
		return &contextMiddleware{
			baseMiddleware: baseMiddleware{provider: provider},
		}
	}
	
	chained := Chain(middleware)(mock)
	
	ctx := context.WithValue(context.Background(), testContextKey{}, "test-value")
	_, err := chained.GenerateText(ctx, core.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// Check that context was propagated
	if receivedCtx == nil {
		t.Fatal("context was not propagated")
	}
	
	value := receivedCtx.Value(testContextKey{})
	if value != "test-value" {
		t.Errorf("context value not propagated, got %v", value)
	}
}

type testContextKey struct{}

type contextMiddleware struct {
	baseMiddleware
}

func (m *contextMiddleware) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
	// Just pass through the context
	return m.provider.GenerateText(ctx, req)
}