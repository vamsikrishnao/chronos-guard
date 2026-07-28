package com.chronos.sdk;

import org.aspectj.lang.ProceedingJoinPoint;
import org.aspectj.lang.annotation.Around;
import org.aspectj.lang.annotation.Aspect;
import org.springframework.stereotype.Component;
import chronos.v1.CheckBudgetResponse;

import java.util.Map;

@Aspect
@Component
public class ChronosGuardAspect {

    private final ChronosGuardClient client;

    public ChronosGuardAspect(ChronosGuardClient client) {
        this.client = client;
    }

    @Around("@annotation(com.chronos.sdk.GuardedAgentStep) && args(contextMap,..)")
    public Object enforceGuardrails(ProceedingJoinPoint joinPoint, Map<String, Object> contextMap) throws Throwable {
        String tenantId = String.valueOf(contextMap.getOrDefault("tenant_id", "default_tenant"));
        String runId = String.valueOf(contextMap.getOrDefault("run_id", "untracked_run"));
        
        // Safe numeric parsing preventing ClassCastException if caller passes Integer, Long, Double, etc.
        long tokensSpent = 0L;
        Object tokensObj = contextMap.get("tokens_spent");
        if (tokensObj instanceof Number) {
            tokensSpent = Math.max(0L, ((Number) tokensObj).longValue());
        }

        String signature = String.valueOf(contextMap.getOrDefault("state_signature", ""));

        CheckBudgetResponse response = client.checkBudget(tenantId, runId, tokensSpent, signature);

        if (response.getAction() == CheckBudgetResponse.Action.ACTION_BLOCK) {
            throw new RuntimeException("AI agent execution blocked by platform guardrails: " + response.getReason());
        } else if (response.getAction() == CheckBudgetResponse.Action.ACTION_THROTTLE) {
            Thread.sleep(100); // Standard platform micro-delay
        }

        return joinPoint.proceed();
    }
}