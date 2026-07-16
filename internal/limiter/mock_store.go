package limiter

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MockRateLimitStore simulates a distributed persistence layer in memory for unit testing.
type MockRateLimitStore struct {
	mu           sync.Mutex
	balances     map[string]int64
	lastUpdated  map[string]time.Time
	shouldFail   bool
	failPing     bool
}

// NewMockRateLimitStore provisions a clean, thread-safe mock persistence driver.
func NewMockRateLimitStore() *MockRateLimitStore {
	return &MockRateLimitStore{
		balances:    make(map[string]int64),
		lastUpdated: make(map[string]time.Time),
	}
}

// InjectFailure forces the store to return network errors to test circuit-breaker topologies.
func (m *MockRateLimitStore) InjectFailure(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}

// InjectPingFailure forces the Ping endpoint to report unhealthy states.
func (m *MockRateLimitStore) InjectPingFailure(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failPing = fail
}

func (m *MockRateLimitStore) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.failPing {
		return errors.New("mock cache connection partition")
	}
	return nil
}

func (m *MockRateLimitStore) TakeAtomic(ctx context.Context, tenantID string, limit int64, window time.Duration, tokens int64) (*LimitResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Short-circuit if a network/infrastructure failure is simulated
	if m.shouldFail {
		return nil, errors.New("redis command timed out: context deadline exceeded")
	}

	// Respect context cancellations passed from upper layers
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	now := time.Now()
	last, exists := m.lastUpdated[tenantID]
	
	// Simple token replenishment simulation based on elapsed time window
	if !exists || now.Sub(last) >= window {
		m.balances[tenantID] = limit
		m.lastUpdated[tenantID] = now
	}

	currentBalance := m.balances[tenantID]

	if currentBalance >= tokens {
		m.balances[tenantID] = currentBalance - tokens
		return &LimitResult{
			Allowed:   true,
			Remaining: m.balances[tenantID],
			ResetTTL:  window - now.Sub(m.lastUpdated[tenantID]),
		}, nil
	}

	return &LimitResult{
		Allowed:   false,
		Remaining: currentBalance,
		ResetTTL:  window - now.Sub(m.lastUpdated[tenantID]),
	}, nil
}