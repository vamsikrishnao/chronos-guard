# Chronos-Guard

Chronos-Guard is a high-performance, low-latency gRPC sidecar service engineered to provide multi-tenant rate limiting, budget evaluation, and structural telemetry isolation for upstream application workflows.

This is an intelligent platform defense layer designed to maximize system availability and optimize infrastructure costs. It acts as a highly efficient, high-speed traffic cop sitting directly in front of our services, managing exactly how and when incoming requests consume our system resources.

## How It Helps
In any large-scale, multi-tenant environment, a single customer or internal system can suddenly go rogue—generating massive spikes in automated traffic (a "noisy neighbor"). Without defense, these traffic spikes can overwhelm core data systems, causing cascades of downtime that impact all customers.

### Business Value & Strategic Impact
*   **Noisy-Neighbor Protection & SLA Preservation:** Chronos-Guard instantly detects, isolates, and throttles these anomalies in real time before they ever reach our critical infrastructure. It ensures that no single user can exhaust collective platform capacity. Prevents unpredictable or rogue traffic spikes from a single tenant from causing cascading performance degradation across the rest of the ecosystem. This guarantees core product availability and protects contractual customer SLAs.
*   **Infrastructure Cost Optimization - Lower compute costs:** By proactively filtering and dropping unauthorized or runaway traffic at the edge, Chronos-Guard prevents artificial infrastructure scaling. This yields highly predictable cloud compute margins and eliminates unnecessary database load.
*   **Decoupled Architecture with improved engineering velocity:** Standardizes platform defense and governance in a centralized sidecar, freeing up application engineering teams to focus exclusively on core feature delivery rather than reinventing rate-limiting frameworks.

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