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
│   ├── limiter/                    # Sliding Window Lua Script for atomic transaction execution
│   │                               # Manages independent token buckets for isolated tenant
│   │                               # spaces.
│   └── telemetry/                  # Unary and streaming gRPC interceptor middleware logic and
│                                   # OpenTelemetry tracing hooks and structured JSON logging
│                                   # Circuit breaker state engine and fail-open boundary control
├── proto/
│   └── chronos/
│       └── v1/
│           ├── guard.proto         # Protocol Buffer contract for GuardService
│           ├── guard.pb.go         # Generated type-safe serialization structures
│           └── guard_grpc.pb.go    # Generated gRPC client/server interfaces
├── Dockerfile                      # Hardened, non-root multi-stage compilation engine
├── go.mod                          # Module dependency configuration
└── README.md                       # Platform documentation
```



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

### 🐹 Go Integration

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

## 🌐 Multi-Language Production Integration

To utilize Chronos-Guard in polyglot environments, developers first generate native client stubs from the central `guard.proto` contract using the language-specific `protoc` compiler plugins.

### ☕ Java / Spring Boot Integration

Java applications utilize the high-performance `grpc-netty-shaded` dependency to connect over the local network namespace.

```java
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import chronos.v1.GuardServiceGrpc;
import chronos.v1.Chronos.CheckBudgetRequest;
import chronos.v1.Chronos.CheckBudgetResponse;

import java.util.concurrent.TimeUnit;

public class ChronosGuardClient {
    private final GuardServiceGrpc.GuardServiceBlockingStub blockingStub;

    public ChronosGuardClient() {
        // Wire to the high-performance loopback proxy target
        ManagedChannel channel = ManagedChannelBuilder.forAddress("127.0.0.1", 50051)
                .usePlaintext()
                .build();
        this.blockingStub = GuardServiceGrpc.newBlockingStub(channel);
    }

    public boolean isBudgetValid(String tenantId, String runId, long tokensSpent, String signature) {
        CheckBudgetRequest request = CheckBudgetRequest.newBuilder()
                .setTenantId(tenantId)
                .setRunId(runId)
                .setTokensSpent(tokensSpent)
                .setStateSignature(signature)
                .build();

        try {
            // Send inline message vector with explicit transit timeout
            CheckBudgetResponse response = blockingStub.withDeadlineAfter(2, TimeUnit.SECONDS)
                    .checkBudget(request);
            
            if (response.getAction() == CheckBudgetResponse.Action.ACTION_BLOCK) {
                System.err.println("Execution blocked. Reason: " + response.getReason());
                return false;
            }
            
            if (response.getAction() == CheckBudgetResponse.Action.ACTION_THROTTLE) {
                System.out.println("Execution throttled. Reason: " + response.getReason());
                Thread.sleep(100); // Inject delay block
            }
            
            return true;
        } catch (Exception e) {
            // Fail-open infrastructure invariant
            System.err.println("Chronos-Guard exception captured: " + e.getMessage() + ". Defaulting to ALLOW.");
            return true;
        }
    }
}
```

### 🐍 Python Integration

Python applications (e.g., FastAPI or Celery tasks managing AI agents) utilize the native asynchronous gRPC library.

```python
import grpc
import time
import chronos_pb2
import chronos_pb2_grpc

def check_agent_budget(tenant_id: str, run_id: str, tokens_spent: int, state_signature: str) -> bool:
    # Bind to local sidecar proxy namespace
    with grpc.insecure_channel('127.0.0.1:50051') as channel:
        stub = chronos_pb2_grpc.GuardServiceStub(channel)
        
        request = chronos_pb2.CheckBudgetRequest(
            tenant_id=tenant_id,
            run_id=run_id,
            tokens_spent=tokens_spent,
            state_signature=state_signature
        )
        
        try:
            # Enforce microsecond inspection timeout bounds
            response = stub.CheckBudget(request, timeout=2.0)
            
            if response.action == chronos_pb2.CheckBudgetResponse.ACTION_BLOCK:
                print(f"Execution blocked. Reason: {response.reason}")
                return False
                
            if response.action == chronos_pb2.CheckBudgetResponse.ACTION_THROTTLE:
                print(f"Execution throttled. Reason: {response.reason}")
                time.sleep(0.1) # Inject deliberate latency delay
                
            return True
            
        except grpc.RpcError as e:
            # Resiliency fallback standard: Fail open
            print(f"Chronos-Guard transit degraded ({e.code()}). Falling open.")
            return True
```

### 💎 Ruby on Rails Integration

For modern enterprise Rails apps serving asynchronous background jobs or running middleware layers, the standard `grpc` gem provides synchronous, low-overhead loopback execution.

```ruby
require 'grpc'
require 'proto/guard_services_pb'

class ChronosGuardClient
  def self.check_budget?(tenant_id, run_id, tokens_spent, state_signature)
    # Establish local loopback communication endpoint
    stub = Chronos::V1::GuardService::Stub.new('127.0.0.1:50051', :this_channel_is_insecure)
    
    request = Chronos::V1::CheckBudgetRequest.new(
      tenant_id: tenant_id,
      run_id: run_id,
      tokens_spent: tokens_spent,
      state_signature: state_signature
    )
    
    begin
      # Execute type-safe evaluation transit
      response = stub.check_budget(request, deadline: Time.now + 2)
      
      if response.action == :ACTION_BLOCK
        Rails.logger.warn("Chronos-Guard blocked execution. Reason: #{response.reason}")
        return false
      end
      
      if response.action == :ACTION_THROTTLE
        Rails.logger.warn("Chronos-Guard throttled execution. Reason: #{response.reason}")
        sleep(0.1) # Deliberate slowdown
      end
      
      true
    rescue GRPC::BadStatus => e
      # Secure fail-open design variant
      Rails.logger.error("Chronos-Guard sidecar connection failure: #{e.message}. System failing open.")
      true
    end
  end
end
```

### 🌐 Polyglot Support (Java, Python, Ruby)

Chronos-Guard exposes a pure, language-agnostic gRPC interface. Any microservice ecosystem that supports Protocol Buffers can interface directly with the sidecar over `127.0.0.1:50051`.

To generate code stubs for your specific application stack, execute `protoc` targeting your runtime language:

```bash
# Generate Go Stubs
protoc --go_out=. --go-grpc_out=. proto/chronos/v1/guard.proto

# Generate Java Stubs
protoc --java_out=. --grpc-java_out=. proto/chronos/v1/guard.proto

# Generate Python Stubs
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. proto/chronos/v1/guard.proto

# Generate Ruby Stubs
grpc_tools_ruby_protoc -I. --ruby_out=. --grpc_out=. proto/chronos/v1/guard.proto
```