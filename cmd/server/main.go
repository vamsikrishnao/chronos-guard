package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vamsikrishnao/chronos-guard/internal/limiter"
	"github.com/vamsikrishnao/chronos-guard/internal/server"
	"github.com/vamsikrishnao/chronos-guard/internal/telemetry"
	pb "github.com/vamsikrishnao/chronos-guard/proto/chronos/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Initialize Prometheus metrics registry
	telemetry.InitializeMetrics()

	slog.Info("initializing chronos-guard control plane...")

	initCtx, cancelInit := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelInit()

	// 1. Initialize Redis Store
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")

	slog.Info("connecting to distributed cache layer", "address", redisAddr)
	store, err := limiter.NewRedisStore(initCtx, redisAddr, redisPassword, 0)
	if err != nil {
		slog.Error("fatal: failed to initialize distributed store", "error", err)
		os.Exit(1)
	}

	if err := store.Ping(initCtx); err != nil {
		slog.Error("fatal: cache health check ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("distributed cache health check verified successfully")

	// 2. Start Prometheus HTTP Metrics Server on :9090
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		slog.Info("starting prometheus metrics server", "port", 9090)
		metricsServer := &http.Server{
			Addr:         ":9090",
			Handler:      mux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		}
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("prometheus metrics server error", "error", err)
		}
	}()

	// 3. Bind TCP Port for gRPC
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		slog.Error("fatal: failed to bind TCP port interface", "error", err)
		os.Exit(1)
	}

	// 4. Initialize gRPC Server with Interceptor
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(
			telemetry.ResilienceInterceptor(store, 100, time.Minute),
		),
	)

	// 5. Register GuardService Implementation
	guardServer := server.NewGuardServer(store, 100, time.Minute)
	pb.RegisterGuardServiceServer(grpcServer, guardServer)

	// 6. Register gRPC Health Check Server
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("chronos.v1.GuardService", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// 7. Handle Shutdown OS Signals
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("chronos-guard sidecar listening for incoming traffic", "port", 50051)
		if err := grpcServer.Serve(listener); err != nil && err != grpc.ErrServerStopped {
			slog.Error("fatal: unexpected gRPC runtime server error", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownChan
	slog.Info("shutdown signal captured; executing graceful platform termination...")

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	grpcServer.GracefulStop()
	slog.Info("chronos-guard control plane shut down cleanly.")
}