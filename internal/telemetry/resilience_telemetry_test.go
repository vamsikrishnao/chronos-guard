package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/vamsikrishnao/chronos-guard/internal/limiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TestResilienceInterceptor_TelemetryValidation verifies that metrics and logs capture chaos events accurately.
func TestResilienceInterceptor_TelemetryValidation(t *testing.T) {
	// 1. Initialize metrics registry idempotently
	InitializeMetrics()
	
	mockStore := limiter.NewMockRateLimitStore()
	interceptor := ResilienceInterceptor(mockStore, 100, time.Minute)
	info := &grpc.UnaryServerInfo{FullMethod: "/proto.ChronosGuard/VerifyBudget"}

	// -------------------------------------------------------------------------
	// Case A: Verify Context Deadline Exceedance Metrics
	// -------------------------------------------------------------------------
	mockStore.InjectDelay(50 * time.Millisecond) // Exceeds the 15ms interceptor timeout
	ctxLatency := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-telemetry-latency"))

	// Trigger interceptor loop to force a fail-open path
	_, err := interceptor(ctxLatency, "req", info, mockUnaryHandler)
	if err != nil {
		t.Fatalf("fail-open path broke during latency test loop: %v", err)
	}

	// Verify Prometheus accurately recorded the specific context deadline reason code
	latencyCount := testutil.ToFloat64(Metrics.FailOpenEvents.WithLabelValues("tenant-telemetry-latency", "context_deadline_exceeded"))
	if latencyCount != 1 {
		t.Errorf("expected 'context_deadline_exceeded' metric count to be 1, got %f", latencyCount)
	}

	// Clean up delay state
	mockStore.InjectDelay(0)

	// -------------------------------------------------------------------------
	// Case B: Verify Network Partition Fail-Open Metrics
	// -------------------------------------------------------------------------
	mockStore.InjectNetworkPartition(true)
	ctxNetwork := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-telemetry-network"))

	_, err = interceptor(ctxNetwork, "req", info, mockUnaryHandler)
	if err != nil {
		t.Fatalf("fail-open path broke during network partition test loop: %v", err)
	}

	// Verify Prometheus accurately recorded the generic network error tracking code
	networkCount := testutil.ToFloat64(Metrics.FailOpenEvents.WithLabelValues("tenant-telemetry-network", "timeout_or_network_error"))
	if networkCount != 1 {
		t.Errorf("expected 'timeout_or_network_error' metric count to be 1, got %f", networkCount)
	}
}