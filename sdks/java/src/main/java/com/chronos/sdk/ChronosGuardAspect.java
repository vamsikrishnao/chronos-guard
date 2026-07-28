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
        String tenantId = (String) contextMap.getOrDefault("tenant_id", "default_tenant");
        String runId = (String) contextMap.getOrDefault("run_id", "untracked_run");
        long tokensSpent = (long) contextMap.getOrDefault("tokens_spent", 0L);
        String signature = (String) contextMap.getOrDefault("state_signature", "");

        CheckBudgetResponse response = client.checkBudget(tenantId, runId, tokensSpent, signature);

        if (response.getAction() == CheckBudgetResponse.Action.ACTION_BLOCK) {
            throw new RuntimeException("AI agent execution blocked by platform layer. Reason: " + response.getReason());
        } else if (response.getAction() == CheckBudgetResponse.Action.ACTION_THROTTLE) {
            // Apply platform delay penalty standard
            Thread.sleep(100);
        }

        return joinPoint.proceed();
    }
}