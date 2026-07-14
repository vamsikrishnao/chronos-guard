package telemetry

import (
	"context"
	"log/slog"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type contextKey string
const LoggerKey contextKey = "slog_logger"

// InitializeGlobalLogger configures structured JSON outputs ready for log aggregators.
func InitializeGlobalLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	return logger
}

// UnaryServerInterceptor automatically injects context metadata fields into logs.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		startTime := time.Now()

		// Attempt to extract incoming metadata headers if present
		var traceID string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if ids := md.Get("x-trace-id"); len(ids) > 0 {
				traceID = ids[0]
			}
		}

		// Inject structural tracking parameters onto the baseline logger context
		logger := slog.With(
			slog.String("rpc.method", info.FullMethod),
			slog.String("trace_id", traceID),
		)

		// Pack the scoped logger into the context wrapper
		ctx = context.WithValue(ctx, LoggerKey, logger)

		// Execute the downstream business logic handler
		resp, err := handler(ctx, req)

		// Record the execution latency telemetry metrics
		logger.Info("RPC transaction execution processed",
			slog.Duration("latency_ms", time.Since(startTime)),
			slog.Bool("success", err == nil),
		)

		return resp, err
	}
}

// FromContext extracts the scoped structured logger from the context boundary safely.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}