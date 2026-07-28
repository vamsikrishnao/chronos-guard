package limiter

import (
	"context"
	"time"
)

// LimitResult defines the payload returned after evaluating a tenant's budget status.
type LimitResult struct {
	Allowed      bool          // True if the request falls within the allocated tenant budget
	Remaining    int64         // Remaining tokens left in the current tracking window
	ResetTTL     time.Duration // Time remaining until the current bucket window resets
	LoopDetected bool          // True if an infinite AI agent loop was detected via state signature
}

// RateLimitStore abstracts distributed persistence engines (Redis, Valkey, or Local Mocks).
type RateLimitStore interface {
	// TakeAtomic evaluates and decrements a tenant's token budget within an atomic transaction.
	TakeAtomic(ctx context.Context, tenantID string, limit int64, window time.Duration, tokens int64) (*LimitResult, error)

	// TakeAtomicWithSignature evaluates token budget AND state signature loop metrics atomically.
	TakeAtomicWithSignature(ctx context.Context, tenantID string, limit int64, window time.Duration, tokens int64, signature string) (*LimitResult, error)

	// Ping validates health status to inform sidecar circuit-breaking topologies.
	Ping(ctx context.Context) error
}