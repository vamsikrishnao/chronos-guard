package telemetry

import (
	"context"
	"log/slog"
	"os"
	"time"

	"google.golang.org/grpc/metadata"
)

// SystemStateVector captures structural telemetry fields for AI parsing engines.
type SystemStateVector struct {
	TenantID   string        `json:"tenant_id"`
	Latency    time.Duration `json:"latency_duration"`
	StoreState string        `json:"store_state"`
	Action     string        `json:"fallback_action"`
}

// InitializeProductionLogger configures the global slog instance to output deterministic JSON.
func InitializeProductionLogger() {
	// Use standard JSON format writing directly to stdout
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}

// LogAnomalyStructured records log blocks specifically formatted for automated log parsing patterns.
func LogAnomalyStructured(ctx context.Context, level slog.Level, message string, vector SystemStateVector, err error) {
	// Extract incoming metadata for trace/request isolation if present
	var traceID string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get("x-request-id"); len(values) > 0 {
			traceID = values[0]
		}
	}

	// Dynamic error string generation
	errStr := "none"
	if err != nil {
		errStr = err.Error()
	}

	// Emit structured JSON with predictable key schemas
	slog.Log(ctx, level, message,
		slog.Group("telemetry_context",
			slog.String("trace_id", traceID),
			slog.String("tenant_id", vector.TenantID),
			slog.Float64("latency_ms", float64(vector.Latency.Microseconds())/1000.0),
		),
		slog.Group("infrastructure_state",
			slog.String("store_status", vector.StoreState),
			slog.String("mitigation_strategy", vector.Action),
			slog.String("error_payload", errStr),
		),
	)
}