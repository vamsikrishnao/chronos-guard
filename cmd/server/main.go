package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os_signal"
	"syscall"
	"time"

	"github.com/vamsikrishnao/chronos-guard/internal/limiter"
	"github.com/vamsikrishnao/chronos-guard/internal/telemetry"
	"google.golang.org/grpc"
)

func main() {
	// Initialize structured JSON logging for production observability
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("initializing chronos-guard control plane...")

	// Establish a deterministic 5-second context window for the infrastructure bootstrap phase
	initCtx, cancelInit := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelInit()

	// 1. Initialize the live Redis persistence driver client
	// (Tuned with the high-concurrency pool sizes established in Milestone 2.1)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Local fallback baseline
	}

	slog.Info("connecting to distributed cache layer", "address", redisAddr)
	store, err := limiter.NewRedisStore(initCtx, redisAddr, "", 0)
	if err != nil {
		slog.Error("fatal: infrastructure bootstrap failed to load distributed store", "error", err)
		os.Exit(1)
	}

	// Verify live cache health before opening network interfaces
	if err := store.Ping(initCtx); err != nil {
		slog.Error("fatal: initial cache ping verification failed", "error", err)
		os.Exit(1)
	}
	slog.Info("distributed cache health check verified successfully")

	// 2. Bind the network port interface
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		slog.Error("fatal: failed to bind TCP port interface", "error", err)
		os.Exit(1)
	}

	// 3. Register the gRPC server architecture and attach our resilience interceptor
	// Dynamic tenant settings default to a baseline of 100 requests per minute
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			telemetry.ResilienceInterceptor(store, 100, time.Minute),
		),
	)

	// 4. Handle clean, zero-downtime orchestration shutdowns using OS signals
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("chronos-guard sidecar listening for incoming traffic", "port", 50051)
		if err := server.Serve(listener); err != nil && err != grpc.ErrServerStopped {
			slog.Error("fatal: unexpected gRPC runtime server error", "error", err)
			os.Exit(1)
		}
	}()

	// Block main execution thread until a shutdown signal clears the channel
	<-shutdownChan
	slog.Info("shutdown signal captured; executing graceful platform termination...")

	// Allow pending in-flight requests 10 seconds to complete execution before hard termination
	server.GracefulStop()
	slog.Info("chronos-guard control plane shut down cleanly. Execution complete.")
}