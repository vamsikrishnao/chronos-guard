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

// ResilienceInterceptor wraps distributed store calls with context deadlines and fail-open resilience.
func ResilienceInterceptor(store limiter.RateLimitStore, defaultLimit int64, defaultWindow time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract tenant metadata (standard multi-tenant validation pattern)
		md, ok := metadata.FromIncomingContext(ctx)
		var tenantID string
		if ok {
			if values := md.Get("x-tenant-id"); len(values) > 0 {
				tenantID = values[0]
			}
		}

		// If no tenant context is provided, reject early via gRPC codes
		if tenantID == "" {
			return nil, status.Error(codes.InvalidArgument, "missing x-tenant-id metadata context")
		}

		// Enforce a strict 15ms deadline specifically for the external cache budget check
		limiterCtx, cancel := context.WithTimeout(ctx, 15*time.Millisecond)
		defer cancel()

		// Evaluate tenant allowance against the store interface
		res, err := store.TakeAtomic(limiterCtx, tenantID, defaultLimit, defaultWindow, 1)
		
		if err != nil {
			// FAIL-OPEN RESILIENCE PATHWAY
			// Log structural telemetry with clear alert hooks so infrastructure teams track degradation
			slog.Warn("rate limit store unreachable or timed out; failing open to preserve tenant availability",
				"tenant_id", tenantID,
				"error", err.Error(),
				"fallback_action", "fail_open_allow_traffic",
			)
			
			// Execute downstream handler to preserve availability SLA
			return handler(ctx, req)
		}

		// Standard Throttling Path: Reject request if budget is definitively exhausted
		if !res.Allowed {
			slog.Info("tenant rate limit exhausted", 
				"tenant_id", tenantID, 
				"remaining_tokens", res.Remaining,
			)
			return nil, status.Error(codes.ResourceExhausted, "tenant rate limit exceeded")
		}

		// Budget checks out cleanly, proceed down the application execution tree
		return handler(ctx, req)
	}
}