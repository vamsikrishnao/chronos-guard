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

// ResilienceInterceptor wraps distributed store calls with context deadlines, metrics tracking, and AI-optimized telemetry.
func ResilienceInterceptor(store limiter.RateLimitStore, defaultLimit int64, defaultWindow time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract tenant metadata
		md, ok := metadata.FromIncomingContext(ctx)
		var tenantID string
		if ok {
			if values := md.Get("x-tenant-id"); len(values) > 0 {
				tenantID = values[0]
			}
		}

		if tenantID == "" {
			return nil, status.Error(codes.InvalidArgument, "missing x-tenant-id metadata context")
		}

		// Setup strict 15ms deadline for cache check
		limiterCtx, cancel := context.WithTimeout(ctx, 15*time.Millisecond)
		defer cancel()

		startTime := time.Now()
		res, err := store.TakeAtomic(limiterCtx, tenantID, defaultLimit, defaultWindow, 1)
		duration := time.Since(startTime)

		// Telemetry: Record latency profile to Prometheus histogram
		if Metrics != nil && Metrics.CacheLatency != nil {
			Metrics.CacheLatency.Observe(duration.Seconds())
		}

		if err != nil {
			// Determine specific failure category for the state vector
			storeState := "cluster_unreachable"
			reason := "timeout_or_network_error"
			if limiterCtx.Err() == context.DeadlineExceeded {
				storeState = "context_deadline_exceeded"
				reason = "context_deadline_exceeded"
			}

			// AI-OPTIMIZED STRUCTURED WARN LOG
			LogAnomalyStructured(ctx, slog.LevelWarn, "rate limit store error encountered; degrading gracefully", 
				SystemStateVector{
					TenantID:   tenantID,
					Latency:    duration,
					StoreState: storeState,
					Action:     "fail_open_allow_traffic",
				}, 
				err,
			)

			// Telemetry: Increment fail-open counter
			if Metrics != nil && Metrics.FailOpenEvents != nil {
				Metrics.FailOpenEvents.WithLabelValues(tenantID, reason).Inc()
			}

			return handler(ctx, req)
		}

		// Standard Throttling Path
		if !res.Allowed {
			// AI-OPTIMIZED STRUCTURED INFO LOG
			LogAnomalyStructured(ctx, slog.LevelInfo, "tenant rate limit budget exhausted", 
				SystemStateVector{
					TenantID:   tenantID,
					Latency:    duration,
					StoreState: "healthy",
					Action:     "block_resource_exhausted",
				}, 
				nil,
			)

			// Telemetry: Increment request counter labeled as "throttled"
			if Metrics != nil && Metrics.RequestCounter != nil {
				Metrics.RequestCounter.WithLabelValues(tenantID, "throttled").Inc()
			}
			
			return nil, status.Error(codes.ResourceExhausted, "tenant rate limit exceeded")
		}

		// Telemetry: Increment request counter labeled as "allowed"
		if Metrics != nil && Metrics.RequestCounter != nil {
			Metrics.RequestCounter.WithLabelValues(tenantID, "allowed").Inc()
		}

		return handler(ctx, req)
	}
}