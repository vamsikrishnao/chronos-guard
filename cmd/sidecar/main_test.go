package main

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/vamsikrishnao/chronos-guard/internal/limiter"
	"github.com/vamsikrishnao/chronos-guard/internal/telemetry"
	pb "github.com/vamsikrishnao/chronos-guard/proto/chronos/v1"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// setupTestServer boots an ephemeral in-memory gRPC server to avoid port collisions
func setupTestServer(t *testing.T, fillRate float64, burst int) (pb.GuardServiceClient, func()) {
	telemetry.InitializeGlobalLogger()
	lis, err := net.Listen("tcp", "127.0.0.1:0") // 0 auto-assigns a free local port
	if err != nil {
		t.Fatalf("Failed to bind ephemeral port: %v", err)
	}

	tl := limiter.NewTenantLimiter(rate.Limit(fillRate), burst)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(telemetry.UnaryServerInterceptor()),
	)

	pb.RegisterGuardServiceServer(grpcServer, &server{tenantLimiter: tl})

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to testing gRPC server: %v", err)
	}

	client := pb.NewGuardServiceClient(conn)

	// Return the client alongside a teardown cleanup function
	return client, func() {
		conn.Close()
		grpcServer.GracefulStop()
		lis.Close()
	}
}

// 1. CONCURRENCY TEST: Validates thread-safety and race condition immunity
func TestConcurrency_RaceConditions(t *testing.T) {
	// Set high rate limits to ensure we are testing concurrency, not rate throttling
	client, teardown := setupTestServer(t, 5000, 1000)
	defer teardown()

	var wg sync.WaitGroup
	concurrentRequests := 100
	wg.Add(concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			_, err := client.CheckBudget(ctx, &pb.CheckBudgetRequest{
				TenantId:    "tenant_concurrency_test",
				RunId:       "run_parallel_flow",
				TokensSpent: 5,
			})
			if err != nil {
				t.Errorf("Concurrent request ID %d failed unexpectedly: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
}

// 2. ISOLATION TEST: Proves that an exhausted tenant space cannot starve a healthy neighbor
func TestTenant_Isolation_And_Throttling(t *testing.T) {
	// Strict limits: 10 requests/sec fill rate, burst capacity of 5
	client, teardown := setupTestServer(t, 10, 5)
	defer teardown()

	ctx := context.Background()

	// Step A: Exhaust the token bucket for Tenant A instantly by hitting the burst limit
	for i := 0; i < 5; i++ {
		resp, err := client.CheckBudget(ctx, &pb.CheckBudgetRequest{
			TenantId:    "tenant_A",
			RunId:       "run_exhaust_burst",
			TokensSpent: 1,
		})
		if err != nil {
			t.Fatalf("Setup phase failed for Tenant A: %v", err)
		}
		if resp.Action != pb.CheckBudgetResponse_ACTION_ALLOW {
			t.Errorf("Expected initial burst requests to be ALLOWED, got %v", resp.Action)
		}
	}

	// The 6th continuous request from Tenant A must trigger the circuit-breaking THROTTLE response
	throttledResp, err := client.CheckBudget(ctx, &pb.CheckBudgetRequest{
		TenantId:    "tenant_A",
		RunId:       "run_breach_step",
		TokensSpent: 1,
	})
	if err != nil {
		t.Fatalf("Throttling verification failed: %v", err)
	}
	if throttledResp.Action != pb.CheckBudgetResponse_ACTION_THROTTLE {
		t.Errorf("Tenant Isolation Failure: Tenant A should be throttled, but received action %v", throttledResp.Action)
	}

	// Step B: Verify Tenant B remains completely unimpacted (Noisy Neighbor Protection)
	healthyResp, err := client.CheckBudget(ctx, &pb.CheckBudgetRequest{
		TenantId:    "tenant_B",
		RunId:       "run_isolated_flow",
		TokensSpent: 1,
	})
	if err != nil {
		t.Fatalf("Tenant B check failed: %v", err)
	}
	if healthyResp.Action != pb.CheckBudgetResponse_ACTION_ALLOW {
		t.Errorf("Noisy Neighbor Leakage: Tenant B was incorrectly penalized by Tenant A's state. Action: %v", healthyResp.Action)
	}
}

// 3. TELEMETRY & RESILIENCE CHECK: Asserts that context deadlines gracefully terminate execution
func TestResilience_ContextTimeout(t *testing.T) {
	client, teardown := setupTestServer(t, 100, 100)
	defer teardown()

	// Create an aggressively short timeout context (Our engine simulation takes 10ms minimum)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
	defer cancel()

	_, err := client.CheckBudget(ctx, &pb.CheckBudgetRequest{
		TenantId:    "tenant_timeout_test",
		RunId:       "run_faulty_latency",
		TokensSpent: 10,
	})

	if err == nil {
		t.Fatal("Resilience Error: Expected request to fail with a context deadline error, but it passed.")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.DeadlineExceeded {
		t.Errorf("Expected gRPC status Code 'DeadlineExceeded', received code: %v with error: %v", st.Code(), err)
	}
}