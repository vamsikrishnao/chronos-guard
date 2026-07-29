# 😎 Chronos-Guard Executable Demos

This directory contains standalone, executable demonstration applications showing how Chronos-Guard enforces real-time FinOps LLM cost containment, token budget tracking, and infinite loop circuit breaking across **Python**, **Java**, and **Ruby** applications.

---

## 📖 The Demo App Story:

A B2B SaaS platform ("AutoAudit AI") processes corporate expense reports using autonomous AI agents.

- **Tenant A ("Acme Corp" - Enterprise):** Triggers a bulk document analysis run (`run_id: run_audit_101`).
- **Normal Flow (Steps 1–2):** Agent processes receipts smoothly (1,200 tokens/step) → `ACTION_ALLOW`.
- **Runaway Loop Incident (Step 3):** The agent encounters a corrupted receipt PDF, gets confused, and enters an infinite re-try loop calling `OCRTool` → `LLM_Parse` repeatedly with identical input history (`state_signature`).
- **Chronos-Guard Intervention:**
  - **Step 3 (Quota Limit Warning):** Reaches 80% quota threshold → `ACTION_THROTTLE` (Injects a 100ms micro-delay).
  - **Step 4 (Infinite Loop Detected):** Same state signature repeated 5 times → `ACTION_BLOCK` (Circuit breaker trips, raises an exception, and halts execution before Acme burns $5,000 in LLM API calls).
- **Tenant B ("Beta Inc"):** Concurrently executes queries on their own isolated token pool completely unaffected.



## 🏆 What this demo showcases:

- Protected API margins, zero unexpected $10k OpenAI bill spikes, and automated budget enforcement.
- Tenant isolation (Tenant Beta unaffected) and multi-tenant SLA protection.
- Cleaner @guard_budget / @GuardedAgentStep catches AgentBlockedException with zero boilerplate code.



## 📂 Demo App Directory Structure

```text
chronos-guard/
├── examples/
│   ├── README.md               # Quickstart guide for running the demos
│   ├── run_all_demos.sh        # One-line bash script executing all 3 demos
│   ├── python/
│   │   ├── demo_agent.py
│   │   └── requirements.txt
│   ├── java/
│   │   ├── pom.xml
│   │   └── src/main/java/com/chronos/demo/AgentDemoApp.java
│   └── ruby/
│       ├── demo_agent.rb
│       └── Gemfile
```



## 🚀 Quickstart: Running the Demos



### Prerequisites

1. Start the local Chronos-Guard sidecar stack using Docker Compose:
  ```bash
  docker-compose up -d
  ```
2. Ensure the gRPC proxy is listening on port `50051`

  
    How to verify that the Chronos-Guard gRPC proxy is up, listening on port `50051`, and accepting traffic across local, Docker, and Kubernetes environments.  
  


> #### Direct gRPC Check using `grpcurl`

  `grpcurl` is the most reliable tool because it verifies both the **TCP network connection** and the **gRPC protocol/serialization layer**.
  


> #### Verify Health Server `Status`

```bash
# Query the gRPC Health Checking Protocol endpoint:
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

  
Expected Response:

```json
{
  "status": "SERVING"
}
```

  


> #### Dispatch a Diagnostic `CheckBudget` RPC

```bash
# Send a sample verification payload directly to the `GuardService`:
grpcurl -plaintext \
-d '{"tenant_id": "probe-test", "run_id": "run-001", "tokens_spent": 10}' \
localhost:50051 chronos.v1.GuardService/CheckBudget
```

  
Expected Response:

```json
{
  "action": "ACTION_ALLOW",
  "reason": "Budget verification successful"
}
```

1. Make run_all_demos.sh executable (If you are choosing script path for demo):
  ```bash
  chmod +x examples/run_all_demos.sh
  ```



## 🎯 Option A: Interactive / Single Command Wrapper

Run the master script to choose interactively or run all demos in sequence:

```bash
# Interactive menu:
./examples/run_all_demos.sh

# Direct language execution options:
./examples/run_all_demos.sh python
./examples/run_all_demos.sh java
./examples/run_all_demos.sh ruby
./examples/run_all_demos.sh all
```



## 🎯 Option B: Running Individual SDK Demos Manually



### 🐍 Python SDK Demo

```python
cd examples/python
pip install -r requirements.txt
python3 demo_agent.py
```



### ☕ Java SDK Demo

```java
cd examples/java
mvn clean compile exec:java -Dexec.mainClass="com.chronos.demo.AgentDemoApp"
```



### 💎 Ruby SDK Demo

```ruby
cd examples/ruby
bundle install
ruby demo_agent.rb
```



## 📊 Business Scenario Visualized in Output

Every SDK demo executes the FinOps Runaway Agent Safeguard Scenario:

- Steps 1–2 (Safe Consumption): Token spend within tenant budget → `ACTION_ALLOW`.
- Step 3 (Soft Threshold Warning): Quota capacity reaches 80% → `ACTION_THROTTLE` (Injects 100ms micro-delay).
- Steps 4+ (Infinite Loop Detected): Repeated state_signature signature detected 5 times → `ACTION_BLOCK` (Circuit breaker trips, raises exception, and halts agent loop before burning thousands in LLM API fees).
- Tenant Isolation: Concurrently executes a step for Tenant `Beta-Inc` to prove their isolated token pool is completely unaffected by Tenant `Acme-Corp`'s circuit breaker.

