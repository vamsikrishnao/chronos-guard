package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/vamsikrishnao/chronos-guard/internal/limiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// BenchmarkResilienceInterceptor_Parallel checks gRPC interceptor performance under high concurrency.
func BenchmarkResilienceInterceptor_Parallel(b *testing.B) {
	// Initialize metrics registry silently to mimic production baseline state
	InitializeMetrics()

	mockStore := limiter.NewMockRateLimitStore()
	// Set an expansive token budget so throttling doesn't skew our latency profiles
	interceptor := ResilienceInterceptor(mockStore, 1000000, time.Hour)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Prepare tenant context per goroutine worker
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-tenant-id", "tenant-bench-parallel"))
		req := "benchmark-payload"
		info := &grpc.UnaryServerInfo{FullMethod: "/proto.ChronosGuard/VerifyBudget"}

		for pb.Next() {
			_, _ = interceptor(ctx, req, info, mockUnaryHandler)
		}
	})
}