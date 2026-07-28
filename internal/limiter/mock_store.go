package limiter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type MockRateLimitStore struct {
	mu           sync.Mutex
	balances     map[string]int64
	lastUpdated  map[string]time.Time
	sigCounts    map[string]int64
	shouldFail   bool
	failPing     bool
	injectDelay  time.Duration
	networkSplit bool
}

func NewMockRateLimitStore() *MockRateLimitStore {
	return &MockRateLimitStore{
		balances:    make(map[string]int64),
		lastUpdated: make(map[string]time.Time),
		sigCounts:   make(map[string]int64),
	}
}

func (m *MockRateLimitStore) InjectFailure(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = fail
}

func (m *MockRateLimitStore) InjectPingFailure(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failPing = fail
}

func (m *MockRateLimitStore) InjectDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.injectDelay = delay
}

func (m *MockRateLimitStore) InjectNetworkPartition(split bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.networkSplit = split
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
	return m.TakeAtomicWithSignature(ctx, tenantID, limit, window, tokens, "")
}

func (m *MockRateLimitStore) TakeAtomicWithSignature(ctx context.Context, tenantID string, limit int64, window time.Duration, tokens int64, signature string) (*LimitResult, error) {
	m.mu.Lock()
	delay := m.injectDelay
	shouldFail := m.shouldFail
	networkSplit := m.networkSplit
	m.mu.Unlock()

	// Leak-free delay timer pattern
	if delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil, ctx.Err()
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if shouldFail {
		return nil, errors.New("redis command timed out: context deadline exceeded")
	}

	if networkSplit {
		return nil, errors.New("redis: connection refused; network partition active")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	last, exists := m.lastUpdated[tenantID]

	if !exists || now.Sub(last) >= window {
		m.balances[tenantID] = limit
		m.lastUpdated[tenantID] = now
		m.sigCounts = make(map[string]int64)
	}

	loopDetected := false
	if signature != "" {
		sigKey := fmt.Sprintf("%s:%s", tenantID, signature)
		m.sigCounts[sigKey]++
		if m.sigCounts[sigKey] > 5 {
			loopDetected = true
		}
	}

	currentBalance := m.balances[tenantID]

	if currentBalance >= tokens && !loopDetected {
		m.balances[tenantID] = currentBalance - tokens
		return &LimitResult{
			Allowed:      true,
			Remaining:    m.balances[tenantID],
			ResetTTL:     window - now.Sub(m.lastUpdated[tenantID]),
			LoopDetected: false,
		}, nil
	}

	return &LimitResult{
		Allowed:      false,
		Remaining:    currentBalance,
		ResetTTL:     window - now.Sub(m.lastUpdated[tenantID]),
		LoopDetected: loopDetected,
	}, nil
}