package com.chronos.sdk;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import chronos.v1.GuardServiceGrpc;
import chronos.v1.Guard.CheckBudgetRequest;
import chronos.v1.Guard.CheckBudgetResponse;

import java.util.concurrent.TimeUnit;
import java.util.logging.Level;
import java.util.logging.Logger;

public class ChronosGuardClient {
    private static final Logger logger = Logger.getLogger(ChronosGuardClient.class.getName());
    private final ManagedChannel channel;
    private final GuardServiceGrpc.GuardServiceBlockingStub blockingStub;

    public ChronosGuardClient(String host, int port) {
        this.channel = ManagedChannelBuilder.forAddress(host, port)
                .usePlaintext()
                .build();
        this.blockingStub = GuardServiceGrpc.newBlockingStub(channel);
    }

    public CheckBudgetResponse checkBudget(String tenantId, String runId, long tokensSpent, String stateSignature) {
        CheckBudgetRequest request = CheckBudgetRequest.newBuilder()
                .setTenantId(tenantId)
                .setRunId(runId)
                .setTokensSpent(tokensSpent)
                .setStateSignature(stateSignature)
                .build();

        try {
            return blockingStub.withDeadlineAfter(2, TimeUnit.SECONDS).checkBudget(request);
        } catch (Exception e) {
            logger.log(Level.WARNING, "Chronos-Guard proxy path degraded. Adhering to fail-open safety standard.", e);
            // Default fail-open construct mapping back to the proto enumeration schema
            return CheckBudgetResponse.newBuilder()
                    .setAction(CheckBudgetResponse.Action.ACTION_ALLOW)
                    .setReason("Fallback executed cleanly.")
                    .build();
        }
    }

    public void shutdown() throws InterruptedException {
        channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
    }
}