# Chronos-Guard

Chronos-Guard is a high-performance, low-latency gRPC sidecar service engineered to provide multi-tenant rate limiting, budget evaluation, and structural telemetry isolation for upstream application workflows.

This is an intelligent platform defense layer designed to maximize system availability and optimize infrastructure costs. It acts as a highly efficient, high-speed traffic cop sitting directly in front of our services, managing exactly how and when incoming requests consume our system resources.

## How It Helps
In any large-scale, multi-tenant environment, a single customer or internal system can suddenly go rogue—generating massive spikes in automated traffic (a "noisy neighbor"). Without defense, these traffic spikes can overwhelm core data systems, causing cascades of downtime that impact all customers.

### Business Value & Strategic Impact
*   **Noisy-Neighbor Protection & SLA Preservation:** Chronos-Guard instantly detects, isolates, and throttles these anomalies in real time before they ever reach our critical infrastructure. It ensures that no single user can exhaust collective platform capacity. Prevents unpredictable or rogue traffic spikes from a single tenant from causing cascading performance degradation across the rest of the ecosystem. This guarantees core product availability and protects contractual customer SLAs.
*   **Infrastructure Cost Optimization - Lower compute costs:** By proactively filtering and dropping unauthorized or runaway traffic at the edge, Chronos-Guard prevents artificial infrastructure scaling. This yields highly predictable cloud compute margins and eliminates unnecessary database load.
*   **Decoupled Architecture with improved engineering velocity:** Standardizes platform defense and governance in a centralized sidecar, freeing up application engineering teams to focus exclusively on core feature delivery rather than reinventing rate-limiting frameworks.
*   **AI-Augmented Operational Efficiency:** Emits structured JSON payloads with uniform keys, turning standard system stdout streams into high-fidelity context blocks that integrate seamlessly with modern AI-driven Root Cause Analysis (RCA) and automated issue resolution pipelines.

### Core Platform Standards

#### 1. Decoupled Persistence Interface (`RateLimitStore`)
Maintains clean structural separation through a unified store interface. This layer abstracts all underlying storage operations, allowing developers to switch seamlessly between a high-throughput remote caching provider (`go-redis/v9`) in production environments and thread-safe mock storage drivers during isolated testing passes without changing the primary application codebase.

#### 2. Atomic Multi-Tenant Isolation (Sliding Window Lua Scripting)
To guarantee consistency across horizontally scaled sidecar nodes under intense concurrent loads, rate-limiting limits are evaluated entirely within remote cache instances via optimized Lua scripts executed via `EVALSHA`. 
* **Atomicity:** Sliding-window log pruning (`ZREMRANGEBYSCORE`) and available token tallies occur inside a single thread-safe runtime step, eliminating the race conditions common to traditional `GET`/`SET` network steps.

#### 3. Fault Tolerance & Fail-Open Resilience
System survival and customer uptime SLAs take absolute priority during localized infrastructure degradation.
* **Context Timeouts:** Every network transaction hitting the remote cache is protected by a strict `15ms` Go context deadline.
* **Fail-Open Mitigation:** If the remote storage cluster becomes unreachable or times out, the gRPC interceptor automatically bypasses the restriction, writes a structured anomaly log, increments the Prometheus fail-open vector, and lets the request pass safely down the pipeline to protect service availability.
* **Deterministic Initialization:** Employs a strict fail-fast policy during server bootstrap (`cmd/server/main.go`). The runtime halts immediately if the database ping fails or if the Lua scripts fail to preload, preventing misconfigured deployments from entering the routing mesh.

#### 4. High-Fidelity Production Telemetry & AI-Driven RCA Support
The observability layer is designed to give infrastructure teams immediate visibility into multi-tenant unit economics while structuring logs to fit automated tool contexts.
* **Granular Metrics:** Exposes thread-safe Prometheus counter and histogram arrays (idempotently registered via `sync.Once`) tracking requests (`allowed`/`throttled`) by `tenant_id`, along with precise $p99$ storage latencies.
* **Structured JSON Logging Schema (`slog`):** Emits all execution logs in structured JSON format with uniform schema groups (`telemetry_context` and `infrastructure_state`). This format encapsulates exact trace IDs, tenant IDs, calculated latencies, and active mitigation steps, allowing AI troubleshooting agents to instantly locate and resolve root causes without manual regex parsing.

## Project Structure

```text
chronos-guard/
├── cmd/
│   └── server/
│       └── main.go         # Deterministic gRPC lifecycle setup & graceful OS signal trapping
├── internal/
│   ├── limiter/
│   │   ├── lua.go          # Atomic sliding-window log evaluation scripts
│   │   ├── mock_store.go   # Thread-safe local test driver with failure injection hooks
│   │   ├── redis.go        # Low-latency production Redis engine adaptor implementation
│   │   ├── store.go        # Structural data layer interface abstraction
│   │   └── store_test.go   # Network simulation and validation test harness
│   └── telemetry/
│       ├── logger.go       # AI-optimized JSON slog telemetry schemas and formatters
│       ├── metrics.go      # Idempotent thread-safe Prometheus vector registries
│       ├── resilience.go   # Observable, context-bound fail-open gRPC interceptor middleware
│       └── resilience_test.go # Comprehensive unit test suite covering full traffic lifecycles
└── proto/                  # Protocol buffer service definitions