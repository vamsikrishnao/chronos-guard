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

var (
	errMissingTenant = status.Error(codes.InvalidArgument, "missing x-tenant-id metadata or tenant_id field in payload")
	errRateExhausted = status.Error(codes.ResourceExhausted, "tenant rate limit exceeded")
)

type tenantGetter interface {
	GetTenantId() string
}

// ResilienceInterceptor wraps gRPC execution with context deadlines, metric tracking, and fail-open resilience.
func ResilienceInterceptor(store limiter.RateLimitStore, defaultLimit int64, defaultWindow time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var tenantID string

		// 1. Try metadata extraction (x-tenant-id header)
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get("x-tenant-id"); len(values) > 0 && values[0] != "" {
				tenantID = values[0]
			}
		}

		// 2. Fall back to request payload extraction if missing in metadata
		if tenantID == "" {
			if tg, ok := req.(tenantGetter); ok {
				tenantID = tg.GetTenantId()
			}
		}

		if tenantID == "" {
			return nil, errMissingTenant
		}

		// Configure strict 15ms deadline for backend cache evaluation
		limiterCtx, cancel := context.WithTimeout(ctx, 15*time.Millisecond)
		defer cancel()

		startTime := time.Now()
		res, err := store.TakeAtomic(limiterCtx, tenantID, defaultLimit, defaultWindow, 1)
		duration := time.Since(startTime)

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