package com.chronos.demo;

import com.chronos.sdk.ChronosGuardClient;
import chronos.v1.CheckBudgetResponse;

import java.util.HashMap;
import java.util.Map;

public class AgentDemoApp {

    public static void main(String[] args) {
        System.out.println("\n--- JAVA SDK DEMO: FinOps LLM Cost Containment & Safeguard ---");
        System.out.println("Connecting to Chronos-Guard Sidecar Proxy at 127.0.0.1:50051...\n");

        ChronosGuardClient client = new ChronosGuardClient("127.0.0.1", 50051);

        try {
            String tenantId = "Acme-Corp-Java";
            String runId = "java_run_202";

            // 1. Safe execution steps
            for (int step = 1; step <= 2; step++) {
                CheckBudgetResponse resp = client.checkBudget(tenantId, runId, 25, "java_sig_step_" + step);
                System.out.printf("   [WORKER] Step %d: Action -> %s | Reason: %s%n", step, resp.getAction(), resp.getReason());
            }

            // 2. High consumption step (Throttling)
            System.out.println("\n   [WARNING] Step 3: High token consumption spike...");
            CheckBudgetResponse throttleResp = client.checkBudget(tenantId, runId, 60, "java_sig_high");
            System.out.printf("   [WORKER] Step 3: Action -> %s | Reason: %s%n", throttleResp.getAction(), throttleResp.getReason());

            // 3. Infinite loop simulation (Repeated state signature)
            System.out.println("\n   [CRITICAL] Step 4+: Infinite loop detected (Repeated state signature)...");
            String loopSig = "corrupted_pdf_java_sig_99x";

            for (int step = 4; step <= 10; step++) {
                CheckBudgetResponse loopResp = client.checkBudget(tenantId, runId, 10, loopSig);
                System.out.printf("   [WORKER] Step %d: Action -> %s | Reason: %s%n", step, loopResp.getAction(), loopResp.getReason());

                if (loopResp.getAction() == CheckBudgetResponse.Action.ACTION_BLOCK) {
                    System.out.println("\n   [CIRCUIT BREAKER TRIPPED] AI Agent loop halted by Java SDK!");
                    System.out.println("   [FINOPS SUCCESS] Prevented runaway API spend.\n");
                    break;
                }
            }

        } finally {
            try {
                client.shutdown();
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }
    }
}