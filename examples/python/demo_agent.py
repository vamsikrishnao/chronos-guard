#!/usr/bin/env python3
import time
import sys
import os

# Import Chronos-Guard Python SDK
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), '../../sdks/python')))
from chronos_guard import ChronosGuardClient
from chronos_guard.decorator import guard_budget, AgentBlockedException

# Initialize persistent gRPC client pointing to sidecar proxy
client = ChronosGuardClient(target_address="127.0.0.1:50051")

@guard_budget(client_instance=client)
def execute_agent_step(context: dict):
    step_num = context.get("step_num", 1)
    tokens = context.get("tokens_spent", 0)
    sig = context.get("state_signature", "")
    print(f"   [WORKER] Step {step_num}: Executing LLM Tool (Spent: {tokens} tokens, Signature: '{sig[:12]}...')")

def run_finops_demo():
    print("\n--- SCENARIO: FinOps LLM Cost Containment & Runaway Agent Safeguard ---")
    print("Tenant 'Acme-Corp' starts bulk invoice audit run (Quota Limit: 100 tokens/min)...\n")

    # 1. Normal Execution Steps (Safe Range)
    for step in range(1, 3):
        ctx = {
            "tenant_id": "Acme-Corp",
            "run_id": "invoice_run_101",
            "tokens_spent": 20,
            "state_signature": f"sig_normal_step_{step}",
            "step_num": step
        }
        execute_agent_step(ctx)

    # 2. Step 3: Approaching Limit (Soft Warning / Throttle)
    print("\n   [WARNING] Step 3: High token consumption spike detected...")
    ctx_throttle = {
        "tenant_id": "Acme-Corp",
        "run_id": "invoice_run_101",
        "tokens_spent": 55,
        "state_signature": "sig_high_consumption",
        "step_num": 3
    }
    execute_agent_step(ctx_throttle)

    # 3. Steps 4+: Infinite Tool Loop Simulation (Repeated State Signature)
    print("\n   [CRITICAL] Step 4+: Corrupted PDF encountered! Agent enters infinite OCR tool loop...")
    loop_sig = "corrupted_pdf_sig_hash_99x"

    for step in range(4, 10):
        ctx_loop = {
            "tenant_id": "Acme-Corp",
            "run_id": "invoice_run_101",
            "tokens_spent": 10,
            "state_signature": loop_sig,
            "step_num": step
        }
        try:
            execute_agent_step(ctx_loop)
        except AgentBlockedException as e:
            print(f"\n   [CIRCUIT BREAKER TRIPPED] {e}")
            print("   [FINOPS ACTION] Runaway agent halted! Saved ~$5,000 in runaway LLM API costs.\n")
            break

    # 4. Multi-Tenant Isolation Verification
    print("--- MULTI-TENANT ISOLATION CHECK ---")
    print("Tenant 'Beta-Inc' executes concurrently on isolated token bucket:")
    beta_ctx = {
        "tenant_id": "Beta-Inc",
        "run_id": "live_chat_55",
        "tokens_spent": 15,
        "state_signature": "beta_chat_sig",
        "step_num": 1
    }
    execute_agent_step(beta_ctx)
    print("   [SUCCESS] Tenant 'Beta-Inc' executed with zero disruption from Acme's circuit breaker!\n")

if __name__ == "__main__":
    try:
        run_finops_demo()
    finally:
        client.close()