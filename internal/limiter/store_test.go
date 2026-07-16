package limiter

import (
	"context"
	"testing"
	"time"
)

func TestMockRateLimitStore_IsolationAndThrottling(t *testing.T) {
	store := NewMockRateLimitStore()
	ctx := context.Background()

	// Tenant A consumes budget
	resA1, err := store.TakeAtomic(ctx, "tenant-a", 10, time.Minute, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resA1.Allowed || resA1.Remaining != 6 {
		t.Errorf("Tenant A unexpected allocation state: %+v", resA1)
	}

	// Tenant B should be completely isolated from Tenant A's usage
	resB, err := store.TakeAtomic(ctx, "tenant-b", 10, time.Minute, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resB.Allowed || resB.Remaining != 8 {
		t.Errorf("Tenant B affected by Tenant A's state: %+v", resB)
	}

	// Over-consume Tenant A to force throttling execution path
	resA2, err := store.TakeAtomic(ctx, "tenant-a", 10, time.Minute, 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resA2.Allowed {
		t.Error("expected Tenant A to be throttled due to budget exhaustion")
	}
}

func TestMockRateLimitStore_FailureInjection(t *testing.T) {
	store := NewMockRateLimitStore()
	ctx := context.Background()

	store.InjectFailure(true)
	_, err := store.TakeAtomic(ctx, "tenant-c", 10, time.Minute, 1)
	if err == nil {
		t.Error("expected error return path when failure state is injected")
	}
}