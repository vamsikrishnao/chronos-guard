package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vamsikrishnao/chronos-guard/internal/limiter"
	"github.com/vamsikrishnao/chronos-guard/internal/telemetry"
	pb "github.com/vamsikrishnao/chronos-guard/proto/chronos/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedGuardServiceServer
	tenantLimiter *limiter.TenantLimiter
}

func (s *server) CheckBudget(ctx context.Context, req *pb.CheckBudgetRequest) (*pb.CheckBudgetResponse, error) {
	logger := telemetry.FromContext(ctx).With(
		slog.String("tenant_id", req.TenantId),
		slog.String("run_id", req.RunId),
	)

	// Step 1: Enforce Strict In-Memory Multi-Tenant Isolation Rate Limit
	lim := s.tenantLimiter.GetLimiter(req.TenantId)
	if !lim.Allow() {
		logger.Warn("Multi-tenant circuit breaker tripped: Rate limit exceeded")
		return &pb.CheckBudgetResponse{
			Action: pb.CheckBudgetResponse_ACTION_THROTTLE,
			Reason: "Rate limit threshold breached for tenant context.",
		}, nil
	}

	// Step 2: Simulate Evaluation / Core Data Verification Check
	// This block validates context deadlines and handles timeouts gracefully
	select {
	case <-time.After(10 * time.Millisecond): // Simulating standard database/evaluation processing
		// Execution completed normally within acceptable bounds
	case <-ctx.Done():
		logger.Error("Resilience timeout breached: Client context deadline exceeded during processing")
		return nil, status.Error(codes.DeadlineExceeded, "Chronos-Guard processing aborted due to context timeout.")
	}

	logger.Info("Evaluation step completed successfully", slog.Int64("token_delta", req.TokensSpent))

	return &pb.CheckBudgetResponse{
		Action: pb.CheckBudgetResponse_ACTION_ALLOW,
		Reason: "Chronos-Guard verification clear.",
	}, nil
}

func main() {
	// Initialize logging engine configuration
	logger := telemetry.InitializeGlobalLogger()
	port := ":50051"

	lis, err := net.Listen("tcp", port)
	if err != nil {
		logger.Error("Critical system error: Failed to bind port", slog.String("port", port), slog.Any("error", err))
		os.Exit(1)
	}

	// Instantiate multi-tenant configuration profile (e.g., 200 requests/sec limit, 50 burst capacity)
	tl := limiter.NewTenantLimiter(200, 50)

	// Chain the structural logging and context lifecycle interceptors into the server lifecycle
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(telemetry.UnaryServerInterceptor()),
	)
	
	pb.RegisterGuardServiceServer(grpcServer, &server{tenantLimiter: tl})

	go func() {
		logger.Info("Chronos-Guard Core Engine running", slog.String("port", port))
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			logger.Error("Runtime Server Failure", slog.Any("error", err))
		}
	}()

	// Graceful termination sequence for clean platform container lifecycle transitions
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("Graceful shutdown sequence initiated...")
	grpcServer.GracefulStop()
	logger.Info("Chronos-Guard completely stopped.")
}