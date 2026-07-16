package limiter

import (
	"context"
	"time"
)

// LimitResult defines the payload returned after evaluating a tenant's budget status.
type LimitResult struct {
	Allowed   bool          // True if the request falls within the allocated tenant budget
	Remaining int64         // Remaining tokens left in the current tracking window
	ResetTTL  time.Duration // Time remaining until the current bucket windows fully resets
}

// RateLimitStore abstracts distributed persistence engines (Redis, Valkey, or Local Mocks).
type RateLimitStore interface {
	// TakeAtomic evaluates and decrements a tenant's budget within a thread-safe transaction.
	TakeAtomic(ctx context.Context, tenantID string, limit int64, window time.Duration, tokens int64) (*LimitResult, error)
	
	// Ping validates health status to inform sidecar circuit-breaking topologies.
	Ping(ctx context.Context) error
}