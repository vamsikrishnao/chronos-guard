# Chronos-Guard

Chronos-Guard is an enterprise-grade, high-performance platform defense layer designed to evaluate AI agent execution loops and enforce runtime guardrails in high-concurrency microservice ecosystems. Operating as an inline gRPC interceptor proxy, the system provides real-time multi-tenant context isolation, precise token tracking, state-signature loop detection, and deterministic circuit-breaking capabilities to prevent runaway agent execution paths and safeguard core platform infrastructure.

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
├── .github/
│   └── workflows/
│       └── ci.yml                  # Automated compilation & test pipeline
├── cmd/
│   ├── client_mock/
│   │   └── main.go                 # Local gRPC testing client
│   └── server/
│       └── main.go                 # Control plane bootstrap & gRPC server
├── deploy/
│   └── kubernetes/
│       └── chronos-guard-sidecar.yaml # Production sidecar deployment manifest
├── internal/
│   ├── limiter/
│   │   ├── lua.go                  # Sliding window & loop detection Lua script
│   │   ├── mock_store.go           # Thread-safe in-memory chaos testing store
│   │   ├── multitenant.go          # Multi-tenant rate limiter with TTL memory eviction
│   │   ├── redis.go                # Production Redis cluster integration
│   │   ├── store_test.go
│   │   └── store.go
│   ├── server/
│   │   └── guard.go                # GuardService gRPC server implementation
│   └── telemetry/                  # Metrics, structured logging, and interceptors
│       ├── interceptor.go
│       ├── logger.go
│       ├── metrics.go
│       └── resilience.go
├── proto/
│   └── chronos/
│       └── v1/
│           ├── guard.proto         # Protocol Buffer contract for GuardService
│           ├── guard.pb.go
│           └── guard_grpc.pb.go
├── sdks/                           # Polyglot SDK Binding Directory
│   ├── java/                       # Java / Spring Boot AOP bindings
│   ├── python/                     # Python SDK (pip install chronos-guard-sdk)
│   └── ruby/                       # RubyGem & ActiveJob middleware
├── deployment.yaml
├── docker-compose.yml
├── Dockerfile                      # Hardened, non-root multi-stage Docker build
├── go.mod
├── Makefile
└── README.md
```



## 🎯 Enterprise Use Cases & Production Scenarios



### 1. The Autonomous LLM Agent Infinite Loop Safeguard

- **The Problem:** An autonomous customer support agent gets trapped in an infinite tools-execution loop (e.g., Agent calls Tool A → Tool A output confuses LLM → Agent calls Tool A again), consuming massive API budgets and hanging indefinitely.
- **How Chronos-Guard Solves It:** Before the runtime framework executes the next step in the loop, it dispatches a `CheckBudgetRequest` containing the current `state_signature` (a hash of the agent's history and memory parameters). Chronos-Guard detects identical state signature transitions repeating in rapid succession within the active window and returns `ACTION_BLOCK`, forcing the agent framework to abort the runaway execution loop cleanly.



### 2. Multi-Tenant API Cost Allocation & Circuit Breaking

- **The Problem:** In a B2B SaaS platform, a single enterprise tenant spins up thousands of concurrent document-processing agents. Their sudden consumption spikes saturate the platform's upstream LLM enterprise quotas, causing immediate throttling for every other customer.
- **How Chronos-Guard Solves It:** Every microservice running an AI workload sends a `CheckBudgetRequest` sharded by `tenant_id`. Chronos-Guard checks the cumulative `tokens_spent` against atomic sliding-window memory quotas in Redis. If Tenant A approaches their tier limit, it returns `ACTION_THROTTLE` to smoothly inject micro-delays into their specific workers. If they breach the hard ceiling, it trips the circuit breaker with `ACTION_BLOCK`, isolating the blast radius entirely to Tenant A.



### 3. Distributed AI Pipeline Observability & Audit Trails

- **The Problem:** Auditing AI spending across a polyglot microservice ecosystem (Python for LLM orchestration, Go for routing, Java for legacy banking logic) is fragmented, making it impossible to map operational costs to specific application flows.
- **How Chronos-Guard Solves It:** By requiring all heterogeneous services to check in with the centralized sidecar before processing an inference boundary, the `CheckBudget` request acts as a uniform telemetry checkpoint. It correlates `run_id` and `tokens_spent` with Prometheus metrics and structured JSON logs written directly to standard output, giving platform engineering teams a single pane of glass for real-time cost accounting.



## 🏗️ Architecture Flow: Inline Guardrail Transit

```text
[ Application Worker ]                [ Local Sidecar Namespace ]         [ Distributed Cache ]
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



## 🚀 Production Integration & SDK Quickstart

Chronos-Guard provides native SDK clients and middleware adapters for Go, Python, Java, and Ruby. The SDKs automatically handle local loopback gRPC connection pooling (`127.0.0.1:50051`), transaction deadline timeouts, and enforce a strict **fail-open safety standard** if the proxy layer becomes unavailable.

---



### 📊 Runtime Decision Matrix

When evaluating `CheckBudget`, your application framework must handle the response actions as follows:

- `ACTION_ALLOW`: The agent loop is operating safely within bounds. Proceed with the execution step immediately.
- `ACTION_THROTTLE`: Early loop signatures or budget warnings detected. The SDK automatically injects a deliberate $100\text{ms}$ delay to backoff execution gracefully.
- `ACTION_BLOCK`: Hard budget ceiling breached or infinite loop confirmed. **Abort the execution step immediately**, roll back uncommitted transactions, and alert your application runtime.



### a. Go-Based Application Integration



#### Steps to Integrate

1. Generate Go client stubs from the central `guard.proto` interface:
  ```bash
  protoc --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
        proto/chronos/v1/guard.proto
  ```
2. Import the generated protobuf package into your Go worker service:
  ```bash
  import pb "github.com/vamsikrishnao/chronos-guard/proto/chronos/v1"
  ```



#### How to Use

```go
package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/vamsikrishnao/chronos-guard/proto/chronos/v1"
)

func verifyAgentStep(tenantID, runID string, tokens int64, signature string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "127.0.0.1:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Printf("Chronos-Guard proxy path degraded; failing open: %v", err)
		return true // Fail-open safety standard
	}
	defer conn.Close()

	client := pb.NewGuardServiceClient(conn)

	resp, err := client.CheckBudget(ctx, &pb.CheckBudgetRequest{
		TenantId:       tenantID,
		RunId:          runID,
		TokensSpent:    tokens,
		StateSignature: signature,
	})
	if err != nil {
		log.Printf("Guardrail verification failed; failing open: %v", err)
		return true
	}

	if resp.GetAction() == pb.CheckBudgetResponse_ACTION_BLOCK {
		log.Printf("Execution halted by guardrail proxy. Reason: %s", resp.GetReason())
		return false
	}

	if resp.GetAction() == pb.CheckBudgetResponse_ACTION_THROTTLE {
		log.Printf("Warning: Throttling active. Reason: %s", resp.GetReason())
		time.Sleep(100 * time.Millisecond) // Backoff delay
	}

	return true
}
```



### b. Java SDK Integration



#### Steps to Integrate

1. Add the SDK dependency stub to your project's `pom.xml`:
  ```xml
  <dependency>
      <groupId>com.vamsikrishnao.chronos</groupId>
      <artifactId>chronos-guard-sdk</artifactId>
      <version>1.0.0</version>
  </dependency>
  ```



#### How to Use

```java
import com.chronos.sdk.ChronosGuardClient;
import com.chronos.sdk.GuardedAgentStep;
import org.springframework.stereotype.Service;

import java.util.Map;

@Service
public class AgentWorkflowService {

    // Automatically intercepted by ChronosGuardAspect
    @GuardedAgentStep
    public void processAgentPipeline(Map<String, Object> contextMap) {
        // Evaluates tenant_id, run_id, tokens_spent, and state_signature dynamically.
        // Throws RuntimeException immediately if ACTION_BLOCK is returned.
        System.out.println("Processing safe AI agent step execution...");
    }
}
```



### c. Python SDK Integration



#### Steps to Integrate

1. Install the SDK package from PyPI or local source:
  ```python
  pip install chronos-guard-sdk
  ```



#### How to Use

```python
from chronos_guard import ChronosGuardClient, guard_budget

# Persistent client connection reuse
client = ChronosGuardClient(target_address="127.0.0.1:50051")

@guard_budget(client_instance=client)
def execute_agent_step(context: dict):
    # Core LLM tool orchestrations happen here
    print("Agent step running safely within platform budget limits.")

# Execution with dynamic context dictionary:
context = {
    "tenant_id": "tenant-prod-100",
    "run_id": "run-uuid-999x",
    "tokens_spent": 420,
    "state_signature": "8f3b20a1c..."
}

execute_agent_step(context)
```



### d. Ruby SDK Integration



#### Steps to Integrate

1. Include the gem in your application `Gemfile`:
  ```ruby
  gem 'chronos-guard-sdk', path: 'sdks/ruby'
  ```



#### How to Use

```ruby
# Intercept asynchronous ActiveJob execution pipelines cleanly
class OpenAIAgentJob < ApplicationJob
  include ChronosGuard::RailsMiddleware

  def perform(payload)
    # Payload automatically evaluated for tenant_id, tokens_spent, and signatures.
    # Throws RuntimeError if ACTION_BLOCK is returned.
    puts "Executing agent background step safely."
  end
end
```



## 🛠️ How to Debug

- **Prometheus Metrics Dashboard:** Scrape execution counters and cache latencies directly from port `9090`:
  ```bash
  curl [http://localhost:9090/metrics](http://localhost:9090/metrics) | grep chronos_guard  
  ```
- **Parsing Structured JSON Observability Logs:** Filter for execution interventions or anomalous state changes using `jq`:
  ```bash
  kubectl logs deployment/ai-agent-runtime -c chronos-guard-proxy | jq 'select(.level == "WARN" or .action == "BLOCK")'
  ```
- **Direct gRPC Wire Diagnostics:** Query the running sidecar directly to verify network and runtime health:
  ```bash
  grpcurl -plaintext \
    -d '{"tenant_id": "test-debug", "run_id": "probe-01", "tokens_spent": 100, "state_signature": "sig-88x"}' \
    localhost:50051 chronos.v1.GuardService/CheckBudget
  ```



## 👥 Contributor & Consumer Guidelines

All teams consuming or contributing to this service tier must adhere to the following framework standards:

- **Protocol Buffer Contracts First:** Any structural additions or modifications to request payloads must originate strictly within `proto/chronos/v1/guard.proto`. Never manually edit auto-generated `*.pb.go` stubs.
- **Strict Fail-Open Safety:** Interceptor and SDK logic must consistently protect the availability of core application workflows. If a state engine or Redis dependency experiences an outage, the proxy must fail open gracefully (`ACTION_ALLOW`).
- **Regression Verification:** All architectural modifications must pass the complete unit and integration test suite before merging:
  ```bash
  go test -v ./internal/...
  ```



## 🏛️ Architecture Benefits



### ⚡ Non-Blocking State Operations

- **O(1) Atomic Operations:** All token accounting and sliding-window evaluators execute within atomic Lua scripts directly inside Redis memory.
- **Zero Lock Contention:** Uses non-blocking ZSET score ranges rather than global Mutexes, ensuring sub-millisecond proxy response times even under heavy concurrent load.



### 🛡️ Resilience & Fail-Safe Invariants

- **Fail-Open Strategy:** If Redis becomes partitioned or context deadlines exceed 15ms, interceptors log the degradation and gracefully return `ACTION_ALLOW` to prevent core application downtime.
- **Isolated Blast Radius:** Quotas and circuit-breaker states are strictly isolated per `tenant_id`. Over-consumption by one workspace never degrades performance for other tenants.



### 🧪 Regression Verification Matrix

- **Automated CI Integration:** Every push and pull request triggers automated Go test execution, static analysis, and multi-binary compilation via GitHub Actions (`ci.yml`).
- **Chaos Injections:** Built-in chaos test suites simulate tail-latency spikes, network partitions, and Redis timeout scenarios to continuously verify fail-open safety invariants.

