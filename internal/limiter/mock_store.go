package limiter

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MockRateLimitStore simulates a distributed persistence layer in memory with advanced chaos injection.
type MockRateLimitStore struct {
	mu            sync.Mutex
	balances      map[string]int64
	lastUpdated   map[string]time.Time
	shouldFail    bool
	failPing      bool
	injectDelay   time.Duration
	networkSplit  bool
}

// NewMockRateLimitStore provisions a clean, thread-safe mock persistence driver.
func NewMockRateLimitStore() *MockRateLimitStore {
	return &MockRateLimitStore{
		balances:    make(map[string]int64),
		lastUpdated: make(map[string]time.Time),
	}
}

// InjectFailure forces the store to return network timeout errors.
func (m *MockRateLimitStore) InjectFailure(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}

// InjectPingFailure forces the Ping endpoint to report unhealthy partition states.
func (m *MockRateLimitStore) InjectPingFailure(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failPing = fail
}

// InjectDelay configures a temporary execution delay to simulate tail latency spikes.
func (m *MockRateLimitStore) InjectDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.injectDelay = delay
}

// InjectNetworkPartition simulates a hard remote cache connection refusal.
func (m *MockRateLimitStore) InjectNetworkPartition(split bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.networkSplit = split
}

// Ping checks the health state of the simulated datastore engine.
func (m *MockRateLimitStore) Ping(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.failPing {
		return errors.New("mock cache connection partition")
	}
	return nil
}

// TakeAtomic evaluates rate limits using your original time-replenishment logic combined with chaos bounds.
func (m *MockRateLimitStore) TakeAtomic(ctx context.Context, tenantID string, limit int64, window time.Duration, tokens int64) (*LimitResult, error) {
	m.mu.Lock()
	delay := m.injectDelay
	shouldFail := m.shouldFail
	networkSplit := m.networkSplit
	m.mu.Unlock()

	// 1. Chaos Simulation: Tail-Latency Outlier
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// 2. Evaluate Context Cancellation Bounds Upfront
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 3. Chaos Simulation: General Failure Trigger
	if shouldFail {
		return nil, errors.New("redis command timed out: context deadline exceeded")
	}

	// 4. Chaos Simulation: Hard Network Split Trigger
	if networkSplit {
		return nil, errors.New("redis: connection refused; network partition active")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	last, exists := m.lastUpdated[tenantID]
	
	// Original token replenishment logic preserved exactly
	if !exists || now.Sub(last) >= window {
		m.balances[tenantID] = limit
		m.lastUpdated[tenantID] = now
	}

	currentBalance := m.balances[tenantID]

	// Budget execution
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