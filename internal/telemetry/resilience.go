package telemetry

import (
	"context"
	"log/slog"
	"time"

	"github.com/vamsikrishnao/chronos-guard/internal/limiter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Pre-allocated error pointers to prevent runtime heap allocation on validation failures
var (
	errMissingTenant = status.Error(codes.InvalidArgument, "missing x-tenant-id metadata context")
	errRateExhausted = status.Error(codes.ResourceExhausted, "tenant rate limit exceeded")
)

// ResilienceInterceptor wraps distributed store calls with context deadlines and optimized low-allocation telemetry.
func ResilienceInterceptor(store limiter.RateLimitStore, defaultLimit int64, defaultWindow time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Low-allocation extraction: inspect MD without creating deep copies
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, errMissingTenant
		}

		values := md["x-tenant-id"]
		if len(values) == 0 || values[0] == "" {
			return nil, errMissingTenant
		}
		tenantID := values[0]

		// Configure strict 15ms deadline for cache check
		limiterCtx, cancel := context.WithTimeout(ctx, 15*time.Millisecond)
		defer cancel()

		startTime := time.Now()
		res, err := store.TakeAtomic(limiterCtx, tenantID, defaultLimit, defaultWindow, 1)
		duration := time.Since(startTime)

		// Record latency profile to Prometheus histogram
		if Metrics != nil && Metrics.CacheLatency != nil {
			Metrics.CacheLatency.Observe(duration.Seconds())
		}

		if err != nil {
			storeState := "cluster_unreachable"
			reason := "timeout_or_network_error"
			if limiterCtx.Err() == context.DeadlineExceeded {
				storeState = "context_deadline_exceeded"
				reason = "context_deadline_exceeded"
			}

			// Emit structured log anomaly frame
			LogAnomalyStructured(ctx, slog.LevelWarn, "rate limit store error encountered; degrading gracefully", 
				SystemStateVector{
					TenantID:   tenantID,
					Latency:    duration,
					StoreState: storeState,
					Action:     "fail_open_allow_traffic",
				}, 
				err,
			)

			if Metrics != nil && Metrics.FailOpenEvents != nil {
				Metrics.FailOpenEvents.WithLabelValues(tenantID, reason).Inc()
			}

			return handler(ctx, req)
		}

		// Standard Throttling Path
		if !res.Allowed {
			LogAnomalyStructured(ctx, slog.LevelInfo, "tenant rate limit budget exhausted", 
				SystemStateVector{
					TenantID:   tenantID,
					Latency:    duration,
					StoreState: "healthy",
					Action:     "block_resource_exhausted",
				}, 
				nil,
			)

			if Metrics != nil && Metrics.RequestCounter != nil {
				Metrics.RequestCounter.WithLabelValues(tenantID, "throttled").Inc()
			}
			
			return nil, errRateExhausted
		}

		if Metrics != nil && Metrics.RequestCounter != nil {
			Metrics.RequestCounter.WithLabelValues(tenantID, "allowed").Inc()
		}

		return handler(ctx, req)
	}
}