# Chronos-Guard

Chronos-Guard is an enterprise-grade, high-performance platform defense layer designed to evaluate AI agent execution loops and enforce runtime guardrails in high-concurrency microservice ecosystems. Operating as an inline gRPC interceptor proxy, the system provides real-time multi-tenant context isolation, precise token tracking, and deterministic circuit-breaking capabilities to prevent runaway agent execution paths and safeguard core platform infrastructure.

---

## 💡 How It Helps

### Business Value & Strategic Impact

- **Cost Containment & Margin Protection:** Out-of-control or looping AI agents can consume millions of API tokens in minutes. Chronos-Guard injects hard execution boundaries directly at the platform layer, neutralizing infrastructure budget spikes before they impact commercial margins.
- **Cascading Failure Mitigation:** By isolating tenant execution pools, a runaway workload or unhandled recursive loop inside a single workspace cannot saturate shared platform capacity, guaranteeing tenant-level SLAs and overall system predictability.
- **Operational Risk Automation:** Transforms non-deterministic AI agent behaviors into an auditable, enterprise-safe operational tier, giving engineering and platform teams centralized control over real-time resource allocations.



### Core Platform Standards

- **Microsecond-Level Performance:** Optimized to run as a sidecar proxy sharing local loopback networking (`127.0.0.1:50051`). By bypassing traditional network serialization boundaries and multi-hop transport hops, the proxy ensures inline guardrail evaluations remain sub-millisecond.
- **Resilient Fail-Open Strategy:** Prioritizes core service availability. If downstream state engines experience degradation or absolute latency spikes, interceptors seamlessly transition to structured fail-open thresholds to ensure continuous application uptime.
- **Zero-Trust Multi-Tenancy:** Enforces deep cryptographic context tracking and token accounting across isolated tenant vectors, verifying every evaluation request against a secure cache layer before allowing downstream execution threads to progress.

---



## 📂 Project Structure

```text
chronos-guard/
├── cmd/
│   └── server/
│       └── main.go                 # Application bootstrap and gRPC server initialization
├── deploy/
│   └── kubernetes/
├── internal/
│   ├── limiter/
│   │   ├── lua.go                  # Sliding Window Lua Script for atomic transaction execution
│   │   ├── mock_store.go           # Manages independent token buckets for isolated tenant
│   │   ├── multitenant.go          # spaces.
│   │   ├── redis.go                
│   │   ├── store_test.go           
│   │   └── store.go                
│   └── telemetry/                  # Unary and streaming gRPC interceptor middleware logic and
│       ├── interceptor.go          # OpenTelemetry tracing hooks and structured JSON logging
│       ├── logger.go               # Circuit breaker state engine & fail-open boundary control
│       ├── metrics.go              
│       ├── resilience_bench_test.go
│       ├── resilience_chaos_test.go
│       ├── resilience_e2e_test.go
│       ├── resilience_telemetry_test.go
│       ├── resilience_test.go
│       └── resilience.go 
├── proto/
│   └── chronos/
│       └── v1/
│           ├── guard.proto         # Protocol Buffer contract for GuardService
│           ├── guard.pb.go         # Generated type-safe serialization structures
│           └── guard_grpc.pb.go    # Generated gRPC client/server interfaces
├── sdks/                           # Central SDK Distribution Directory
│   ├── python/                     # Packaged for PyPI (pip install chronos-guard-sdk)
│   │   ├── pyproject.toml
│   │   ├── setup.py
│   │   └── chronos_guard/
│   │       ├── __init__.py
│   │       └── decorator.py        # The @guard_budget decorator logic
│   ├── java/                       # Configured for Maven/Central (pom.xml)
│   │   ├── pom.xml
│   │   └── src/main/java/com/chronos/sdk/
│   │       ├── ChronosGuardClient.java
│   │       └── ChronosGuardAspect.java
│   └── ruby/                       # Packaged as a RubyGem (gem install chronos-guard-sdk)
│       ├── chronos-guard-sdk.gemspec
│       └── lib/
│           ├── chronos_guard.rb
│           └── chronos_guard/
│               └── rails_middleware.rb
├── deployment.yaml
├── docker-compose.yml
├── Dockerfile                      # Hardened, non-root multi-stage compilation engine
├── go.mod                          # Module dependency configuration
├── Makefile                        
└── README.md                       # Platform documentation
```



## 🎯 Enterprise Use Cases & Production Scenarios

While Chronos-Guard exposes a single, highly optimized API (`CheckBudget`), it acts as the centralized enforcement engine for critical business logic across several distributed architecture patterns. 

Here is how organizations deploy this system to solve real-world AI scale and safety problems:

### 1. The Autonomous LLM Agent Loop Safeguard

- **The Problem:** An autonomous customer support agent gets trapped in an infinite tools-execution loop (e.g., Agent calls Tool A $\rightarrow$ Tool A output confuses LLM $\rightarrow$ Agent calls Tool A again), consuming massive API budgets and hanging indefinitely.
- **How Chronos-Guard Solves It:** Before the runtime framework executes the next step in the loop, it dispatches a `CheckBudgetRequest` containing the current `state_signature` (a hash of the agent's history and memory parameters). Chronos-Guard detects identical state transitions or token velocities spiking within the same `run_id` and fires back an `ACTION_BLOCK`, forcing the agent framework to cleanly abort the runaway execution path.



### 2. Multi-Tenant API Cost Allocation & Circuit Breaking

- **The Problem:** In a B2B SaaS platform, a single enterprise tenant spins up thousands of concurrent document-processing agents. Their sudden consumption spikes saturate the company's OpenAI/Anthropic enterprise API quotas, causing immediate throttling for every other customer on the platform.
- **How Chronos-Guard Solves It:** Every microservice running an AI workload sends a `CheckBudgetRequest` sharded by `tenant_id`. Chronos-Guard checks the cumulative `tokens_spent` against sliding-window memory quotas in Redis. If Tenant A approaches their tier limit, it returns `ACTION_THROTTLE` to smoothly inject micro-delays into their specific workers. If they breach the hard ceiling, it trips the circuit breaker with `ACTION_BLOCK`, isolating the blast radius entirely to Tenant A while keeping the rest of the platform responsive.



### 3. Distributed AI Pipeline Observability & Audit Trails

- **The Problem:** Auditing AI spending across a polyglot microservice ecosystem (e.g., Python for the LLM orchestration layer, Go for internal high-speed routing, Java for legacy core banking logic) is fragmented, making it impossible to map operational costs to specific application flows.
- **How Chronos-Guard Solves It:** By requiring all heterogeneous services to check in with the centralized sidecar before processing an inference boundary, the `CheckBudget` request acts as a uniform telemetry checkpoint. It correlates `run_id` and `tokens_spent` with OpenTelemetry distributed trace headers, writing structured JSON audit logs directly to standard output. This gives platform engineering teams a single pane of glass for real-time cost accounting across the entire enterprise stack.

---



## 🏗️ Architecture Flow: Inline Guardrail Transit

```text
[ Application Worker ]                [ Local Sidecar Namespace ]           [ Distributed Cache ]
  (Go/Python/Java/Ruby)                  (Chronos-Guard Proxy)                    (Redis)
           │                                       │                                 │
           │ ─── 1. CheckBudgetRequest ──────────> │                                 │
           │      (tenant, run_id, tokens)         │                                 │
           │                                       │ ─── 2. Evaluate state & cost ─> │
           │                                       │      (Atomic sliding-window)    │
           │                                       │                                 │
           │                                       │ <── 3. Return Metrics/State ─── │
           │                                       │                                 │
           │ <── 4. CheckBudgetResponse ────────── │                                 │
           │      (ALLOW / THROTTLE / BLOCK)       │                                 │
```

---



## 🚀 Production Integration & SDK Quickstart

Chronos-Guard provides native, zero-boilerplate SDK clients and middleware adapters for Go, Python, Java, and Ruby. The SDKs automatically handle local loopback gRPC connection pooling (`127.0.0.1:50051`), transaction deadline timeouts, and enforce a strict **fail-open safety standard** if the proxy layer becomes unavailable.

### 📊 Runtime Decision Matrix

When evaluating `CheckBudget`, your application framework must handle the response actions as follows:

- `ACTION_ALLOW`: The agent loop is operating safely within bounds. Proceed with the execution step immediately.
- `ACTION_THROTTLE`: Early loop signatures or budget warnings detected. The SDK automatically injects a deliberate $100\text{ms}$ delay to backoff execution gracefully.
- `ACTION_BLOCK`: Hard budget ceiling breached or infinite loop confirmed. **Abort the execution step immediately**, roll back uncommitted transactions, and alert your application runtime.

---



## 🚀 Production Deployment & Integration



### How to Utilise

To minimize inter-container latency, Chronos-Guard should be deployed via the Sidecar Pattern inside your pod topology. This ensures both your application worker and the guardrail proxy share the same networking namespace.

**1. Build the Hardened Production Container:**

```bash
docker build -t internal-registry.io/security/chronos-guard:latest .
```

**2. Apply to Cluster Pod Namespace:**
Mount the chronos-guard-proxy alongside your primary worker container within your Kubernetes deployment manifest, leveraging native gRPC probes (startup, liveness, and readiness) on port 50051.

### How to Use in Code

To utilize Chronos-Guard in Go environment, developers first generate native client stubs from the central `guard.proto` contract using the `protoc` compiler plugins.

```bash
# Generate Go Stubs
protoc --go_out=. --go-grpc_out=. proto/chronos/v1/guard.proto
```



###### 🐹 Go Integration

```Go
package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/vamsikrishnao/chronos-guard/proto/chronos/v1"
)

func checkAgentBudget(tenantID, runID string, tokens int64, signature string) bool {
	// Establish ultra-low latency connection over local loopback namespace
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "127.0.0.1:50051", 
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("Chronos-Guard unavailable, fallback to fail-open: %v", err)
		return true
	}
	defer conn.Close()

	client := pb.NewGuardServiceClient(conn)

	// Execute evaluation request matching the guard.proto schema
	resp, err := client.CheckBudget(context.Background(), &pb.CheckBudgetRequest{
		TenantId:        tenantID,
		RunId:           runID,
		TokensSpent:     tokens,
		StateSignature:  signature,
	})
	if err != nil {
		log.Printf("Guardrail verification failed, failing open: %v", err)
		return true 
	}

	if resp.Action == pb.CheckBudgetResponse_ACTION_BLOCK {
		log.Printf("Execution blocked. Reason: %s", resp.Reason)
		return false
	}

	if resp.Action == pb.CheckBudgetResponse_ACTION_THROTTLE {
		log.Printf("Warning: Execution throttled. Reason: %s", resp.Reason)
		time.Sleep(100 * time.Millisecond) // Inject deliberate micro-delay
	}

	return true
}
```



### 🌐 Polyglot SDK Support (Java, Python, Ruby)

🐍 Python SDK

```python
pip install chronos-guard-sdk
```

```python
# Approach A: Passing a Dictionary Context

from chronos_guard import ChronosGuardClient, guard_budget

client = ChronosGuardClient()

@guard_budget(client_instance=client)
def execute_agent_loop(agent_context: dict):
    # Core LLM tool orchestrations happen here cleanly
    print("Agent step running safely within budget limits.")

# Execution is clean and readable:
context = {
    "tenant_id": "tenant-prod-100",
    "run_id": "run-uuid-999x",
    "tokens_spent": 420,
    "state_signature": "8f3b20a1..."
}

execute_agent_loop(context)
```

```python
# Approach B: Passing a Structured Object / Data Class

from dataclasses import dataclass
from chronos_guard import ChronosGuardClient, guard_budget

client = ChronosGuardClient()

@dataclass
class AgentContext:
    tenant_id: str
    run_id: str
    tokens_spent: int
    state_signature: str

@guard_budget(client_instance=client)
def execute_agent_step(ctx: AgentContext):
    # Core execution path
    pass

# Execution:
my_context = AgentContext("tenant-99", "run-abc", 150, "e3b0c442...")
execute_agent_step(my_context)
```



#### ☕ Java / Spring Boot SDK

```xml
<!-- Add dependency coordinate stub to your pom.xml -->
<dependency>
    <groupId>com.vamsikrishnao.chronos</groupId>
    <artifactId>chronos-guard-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```

```java
// Leverage clean Aspect-Oriented Programming (AOP) to auto-intercept steps
@GuardedAgentStep
public void processAgentPipeline(Map<String, Object> contextMap) {
    // Automatically evaluates tenant_id, run_id, and signatures from the context map
    // Throws a RuntimeException immediately if ACTION_BLOCK is returned
}
```



#### 💎 Ruby on Rails SDK

```ruby
# Gemfile
gem 'chronos-guard-sdk', path: 'sdks/ruby'
```

```ruby
# Intercept asynchronous jobs cleanly using ActiveJob Server Middleware
class OpenAIAgentJob < ApplicationJob
  # Middleware automatically intercepts, processes throttles, or halts on ACTION_BLOCK
  include ChronosGuard::RailsMiddleware

  def perform(payload)
    # Execute non-deterministic execution tasks safely
  end
end
```



### How to Debug

- Trace Analysis via OpenTelemetry: Every evaluation request context generates a distinct trace vector. Inspect downstream spans within your distributed tracing dashboard (e.g., Jaeger) to isolate latency anomalies or verify interceptor overhead.
- Parsing Structured Observability Logs: The runtime engine pipes structured JSON directly to standard output. Filter for execution interventions or anomalous state changes using structured command-line log utilities:
  ```bash
  kubectl logs deployment/ai-agent-runtime -c chronos-guard-proxy | jq 'select(.level == "error" or .action == "BLOCK")'
  ```
- Direct gRPC Wire Diagnostics: You can query the proxy directly from within the running pod environment to verify network and runtime health using native gRPC probing tools:
  ```bash
  grpcurl -plaintext -d '{"tenant_id": "test-debug", "agent_id": "probe"}' localhost:50051 chronos.v1.GuardService/EvaluateLoop
  ```



## 👥 Contributor & Consumer Guidelines

To maintain the architectural integrity, sub-millisecond execution speeds, and zero-trust isolation boundaries of the Chronos-Guard ecosystem, all teams consuming or contributing to this service tier must adhere to the following framework standards:

### 📡 Protocol Buffer Contracts First

- **Contract Schema Evolution:** Any structural additions, modifications to request payloads, or extension of evaluation actions must originate strictly within the `proto/chronos/v1/guard.proto` definitions file. 
- **Zero Manual Stubs:** Never manually edit or hotfix the auto-generated Go serialization artifacts (`*.pb.go` or `*_grpc.pb.go`). All changes must be processed through the central compilation pipeline to ensure absolute type safety.
- **Deterministic Code Generation:** Regenerate stubs exclusively using the dedicated workspace engine to prevent configuration drift:
  ```bash
  protoc --go_out=. --go-grpc_out=. proto/chronos/v1/guard.proto
  ```



### ⚡ Non-Blocking State Operations

- O(1) Execution Metrics: The distributed state engine relies on atomic Redis transactional models. All sliding-window evaluators, token accounting operations, and loop metrics must maintain strict O(1) time complexity profiles.
- No Blocking Commands: The use of blocking primitives (e.g., KEYS, SMEMBERS on large sets, or long-running Lua scripts) is explicitly banned. Any state evaluations that risk blocking execution threads will fail validation during peer review.
- Connection Pooling: Always leverage the authenticated thread-safe connection pool (Pool) for external state calls. Opening or closing ad-hoc TCP connections inside the execution interceptor path is prohibited.



### 🛡️ Resilience & Fail-Safe Invariants

- Strict Fail-Open Safety: Interceptor logic must consistently protect the availability of core platform services. If a state engine or database dependency throws an unhandled error or a strict timeout occurs, the proxy must log the degradation and gracefully fall back to a structured fail-open state.
- Circuit Breaker Boundaries: The deterministic state transitions (Allow, Throttle, Block) must remain independent per tenant. Ensure no cross-tenant state leakage or global shared state counters are introduced outside of explicit global throttle quotas.



### 🧪 Regression Verification Matrix

- Absolute Compliance: All architectural modifications, telemetry adjustments, or platform refactors must preserve complete backward compatibility with the automated integration test matrix.
- Pre-Flight Test Loop: Run the complete end-to-end type-safe validation suite locally before pushing any commits to your remote feature branch:
  ```bash
  go test -v ./internal/telemetry/... -run TestChronosGuard_LiveE2E
  ```
- Coverage Requirements: Any new evaluation parameters or circuit-breaking behaviors require corresponding unit test coverage alongside explicit failure injection blocks to verify fallback states.

