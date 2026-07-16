package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/vamsikrishnao/chronos-guard/internal/limiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Dummy gRPC handler to simulate business logic downstream
func mockUnaryHandler(ctx context.Context, req interface{}) (interface{}, error) {
	return "success", nil
}

func TestResilienceInterceptor_SuccessAndFailOpen(t *testing.T) {
	mockStore := limiter.NewMockRateLimitStore()
	interceptor := ResilienceInterceptor(mockStore, 5, time.Minute)

	// Inject target tenant metadata context
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-alpha"))

	// Case 1: Standard Happy Path (Allowed Traffic)
	resp, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, mockUnaryHandler)
	if err != nil {
		t.Fatalf("unexpected happy-path failure: %v", err)
	}
	if resp.(string) != "success" {
		t.Errorf("expected 'success' payload, got: %v", resp)
	}

	// Case 2: Store Failure / Timeout Simulation (Should Fail Open)
	mockStore.InjectFailure(true)
	
	respFailOpen, errFailOpen := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, mockUnaryHandler)
	if errFailOpen != nil {
		t.Errorf("interceptor failed closed during store outage; expected fail-open behavior. Error: %v", errFailOpen)
	}
	if respFailOpen.(string) != "success" {
		t.Errorf("expected 'success' payload via fail-open fallback, got: %v", respFailOpen)
	}
}

func TestResilienceInterceptor_Throttling(t *testing.T) {
	mockStore := limiter.NewMockRateLimitStore()
	interceptor := ResilienceInterceptor(mockStore, 1, time.Minute)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-beta"))

	// First request consumes the lone available token
	_, _ = interceptor(ctx, "req", &grpc.UnaryServerInfo{}, mockUnaryHandler)

	// Second request should be rejected deterministically with ResourceExhausted
	_, err := interceptor(ctx, "req", &grpc.UnaryServerInfo{}, mockUnaryHandler)
	if err == nil {
		t.Fatal("expected request to be blocked by rate limit, but it passed")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.ResourceExhausted {
		t.Errorf("expected gRPC code ResourceExhausted, got: %v", st.Code())
	}
}