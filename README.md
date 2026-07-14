# Chronos-Guard

Chronos-Guard is a high-performance, low-latency gRPC sidecar service engineered to provide multi-tenant rate limiting, budget evaluation, and structural telemetry isolation for upstream application workflows.

## Architecture Overview

*   **Multi-Tenant Isolation:** Leverages thread-safe token bucket semantics (`golang.org/x/time/rate`) encapsulated inside a localized memory space to enforce strict noisy-neighbor protection.
*   **Structured Telemetry:** Injects context-scoped logging (`char/slog`) through a unary gRPC server interceptor, ensuring distributed trace parameters translate downstream.
*   **Resilience & Deadlines:** Honors incoming client context boundaries to automatically short-circuit processing when service execution latency spikes past thresholds.

## Project Structure

```text
chronos-guard/
├── cmd/
│   └── sidecar/
│       ├── main.go         # Core gRPC server initialization
│       └── main_test.go    # Parallel race-detection validation suite
├── internal/
│   ├── limiter/            # Multi-tenant token bucket registry
│   └── telemetry/          # gRPC unary log interceptors and context propagation
└── proto/                  # Protocol buffer service definitions