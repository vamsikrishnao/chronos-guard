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

func TestResilienceInterceptor_ChaosScenarios(t *testing.T) {
	// Ensure global metric vectors are registered idempotently
	InitializeMetrics()
	
	mockStore := limiter.NewMockRateLimitStore()
	interceptor := ResilienceInterceptor(mockStore, 100, time.Minute)
	info := &grpc.UnaryServerInfo{FullMethod: "/proto.ChronosGuard/VerifyBudget"}

	// -------------------------------------------------------------------------
	// Scenario A: Tail Latency Injection (50ms Delay vs 15ms Interceptor Timeout)
	// -------------------------------------------------------------------------
	mockStore.InjectDelay(50 * time.Millisecond)
	ctxA := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-chaos-latency"))

	respA, errA := interceptor(ctxA, "req", info, mockUnaryHandler)
	if errA != nil {
		t.Fatalf("interceptor failed closed during tail-latency spike; expected fail-open allowance: %v", errA)
	}
	if respA.(string) != "success" {
		t.Errorf("expected success payload payload string, got: %v", respA)
	}

	// Assert that Prometheus registered a context deadline exceedance error
	latencyExceededMetric := testutil.ToFloat64(Metrics.FailOpenEvents.WithLabelValues("tenant-chaos-latency", "context_deadline_exceeded"))
	if latencyExceededMetric != 1 {
		t.Errorf("expected metric 'context_deadline_exceeded' tally to equal 1, got %f", latencyExceededMetric)
	}

	// Restore delay latency state to baseline
	mockStore.InjectDelay(0)

	// -------------------------------------------------------------------------
	// Scenario B: Remote Cache Hard Network Partition Simulation
	// -------------------------------------------------------------------------
	mockStore.InjectNetworkPartition(true)
	ctxB := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-chaos-partition"))

	respB, errB := interceptor(ctxB, "req", info, mockUnaryHandler)
	if errB != nil {
		t.Fatalf("interceptor failed closed during active network split; expected fail-open resilience: %v", errB)
	}
	if respB.(string) != "success" {
		t.Errorf("expected success payload via fallback channel, got: %v", respB)
	}

	// Assert that Prometheus tracked the network timeout error code counter
	networkSplitMetric := testutil.ToFloat64(Metrics.FailOpenEvents.WithLabelValues("tenant-chaos-partition", "timeout_or_network_error"))
	if networkSplitMetric != 1 {
		t.Errorf("expected metric 'timeout_or_network_error' tally to equal 1, got %f", networkSplitMetric)
	}
}