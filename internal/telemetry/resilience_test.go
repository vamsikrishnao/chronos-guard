package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/vamsikrishnao/chronos-guard/internal/limiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// 1. mockUnaryHandler (Dummy handler simulating downstream business logic)
func mockUnaryHandler(ctx context.Context, req interface{}) (interface{}, error) {
	return "success", nil
}

// 2. TestResilienceInterceptor_SuccessAndFailOpen (Updated to support metrics)
func TestResilienceInterceptor_SuccessAndFailOpen(t *testing.T) {
	InitializeMetrics()
	mockStore := limiter.NewMockRateLimitStore()
	interceptor := ResilienceInterceptor(mockStore, 5, time.Minute)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-alpha"))

	// Case 1: Standard Happy Path
	resp, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, mockUnaryHandler)
	if err != nil {
		t.Fatalf("unexpected happy-path failure: %v", err)
	}
	if resp.(string) != "success" {
		t.Errorf("expected 'success' payload, got: %v", resp)
	}

	// Case 2: Store Failure Simulation (Should Fail Open)
	mockStore.InjectFailure(true)
	
	respFailOpen, errFailOpen := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, mockUnaryHandler)
	if errFailOpen != nil {
		t.Errorf("interceptor failed closed during store outage; expected fail-open behavior. Error: %v", errFailOpen)
	}
	if respFailOpen.(string) != "success" {
		t.Errorf("expected 'success' payload via fail-open fallback, got: %v", respFailOpen)
	}
}

// 3. TestResilienceInterceptor_Throttling (Updated to support metrics)
func TestResilienceInterceptor_Throttling(t *testing.T) {
	InitializeMetrics()
	mockStore := limiter.NewMockRateLimitStore()
	interceptor := ResilienceInterceptor(mockStore, 1, time.Minute)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-beta"))

	// First request consumes the lone token
	_, _ = interceptor(ctx, "req", &grpc.UnaryServerInfo{}, mockUnaryHandler)

	// Second request should be blocked
	_, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, mockUnaryHandler)
	if err == nil {
		t.Fatal("expected request to be blocked by rate limit, but it passed")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.ResourceExhausted {
		t.Errorf("expected gRPC code ResourceExhausted, got: %v", st.Code())
	}
}

// 4. TestResilienceInterceptor_WithMetrics (New test case validating explicit value increments)
func TestResilienceInterceptor_WithMetrics(t *testing.T) {
	InitializeMetrics()
	mockStore := limiter.NewMockRateLimitStore()
	interceptor := ResilienceInterceptor(mockStore, 1, time.Minute)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-metrics-verify"))

	// Trigger 'allowed' path
	_, _ = interceptor(ctx, "req", &grpc.UnaryServerInfo{}, mockUnaryHandler)
	allowedCount := testutil.ToFloat64(Metrics.RequestCounter.WithLabelValues("tenant-metrics-verify", "allowed"))
	if allowedCount != 1 {
		t.Errorf("expected 'allowed' metric counter to be 1, got %f", allowedCount)
	}

	// Trigger 'throttled' path
	_, _ = interceptor(ctx, "req", &grpc.UnaryServerInfo{}, mockUnaryHandler)
	throttledCount := testutil.ToFloat64(Metrics.RequestCounter.WithLabelValues("tenant-metrics-verify", "throttled"))
	if throttledCount != 1 {
		t.Errorf("expected 'throttled' metric counter to be 1, got %f", throttledCount)
	}

	// Trigger 'fail-open' path
	mockStore.InjectFailure(true)
	_, _ = interceptor(ctx, "req", &grpc.UnaryServerInfo{}, mockUnaryHandler)
	failOpenCount := testutil.ToFloat64(Metrics.FailOpenEvents.WithLabelValues("tenant-metrics-verify", "timeout_or_network_error"))
	if failOpenCount != 1 {
		t.Errorf("expected fail-open events counter to be 1, got %f", failOpenCount)
	}
}