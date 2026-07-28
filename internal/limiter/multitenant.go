package limiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type tenantEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// TenantLimiter manages independent token buckets with automated TTL memory eviction.
type TenantLimiter struct {
	mu       sync.Mutex
	limiters map[string]*tenantEntry
	r        rate.Limit
	b        int
	ttl      time.Duration
}

// NewTenantLimiter instantiates a configuration profile with background TTL eviction.
func NewTenantLimiter(r rate.Limit, b int) *TenantLimiter {
	tl := &TenantLimiter{
		limiters: make(map[string]*tenantEntry),
		r:        r,
		b:        b,
		ttl:      30 * time.Minute,
	}

	// Background sweeper goroutine to evict inactive tenants
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			tl.evictStale()
		}
	}()

	return tl
}

// GetLimiter fetches or instantiates a rate limiter for a specific tenant ID.
func (tl *TenantLimiter) GetLimiter(tenantID string) *rate.Limiter {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	now := time.Now()
	if entry, exists := tl.limiters[tenantID]; exists {
		entry.lastSeen = now
		return entry.limiter
	}

	limiter := rate.NewLimiter(tl.r, tl.b)
	tl.limiters[tenantID] = &tenantEntry{
		limiter:  limiter,
		lastSeen: now,
	}
	return limiter
}

func (tl *TenantLimiter) evictStale() {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	now := time.Now()
	for tenantID, entry := range tl.limiters {
		if now.Sub(entry.lastSeen) > tl.ttl {
			delete(tl.limiters, tenantID)
		}
	}
}