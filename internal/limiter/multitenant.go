package limiter

import (
	"sync"
	"golang.org/x/time/rate"
)

// TenantLimiter manages independent token buckets for isolated tenant spaces.
type TenantLimiter struct {
	limiters sync.Map
	r        rate.Limit
	b        int
}

// NewTenantLimiter instantiates a configuration profile.
// r = fill rate (tokens per second), b = burst capacity.
func NewTenantLimiter(r rate.Limit, b int) *TenantLimiter {
	return &TenantLimiter{
		r: r,
		b: b,
	}
}

// GetLimiter safely fetches or generates a token bucket for a specific tenant.
func (tl *TenantLimiter) GetLimiter(tenantID string) *rate.Limiter {
	limiter, exists := tl.limiters.Load(tenantID)
	if exists {
		return limiter.(*rate.Limiter)
	}

	// Double-checked locking pattern via LoadOrStore to avoid allocation race conditions
	newLimiter := rate.NewLimiter(tl.r, tl.b)
	actual, _ := tl.limiters.LoadOrStore(tenantID, newLimiter)
	return actual.(*rate.Limiter)
}