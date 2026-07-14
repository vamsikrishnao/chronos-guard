package main

import (
	"context"
	"log"
	"time"

	// CRITICAL: Change 'your-github-username' to match your actual handle
	pb "github.com/vamsikrishnao/chronos-guard/proto/chronos/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Dial the local sidecar proxy over standard gRPC
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Integration Error: Connection failed: %v", err)
	}
	defer conn.Close()

	client := pb.NewGuardServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	// Dispatch an active transaction step payload for budget analysis
	resp, err := client.CheckBudget(ctx, &pb.CheckBudgetRequest{
		TenantId:        "tenant_org_fresh_88x",
		RunId:           "agent_workflow_run_01J",
		TokensSpent:     4200,
		StateSignature: "8f9b1c4c28cb38d5f260853678922e04",
	})
	if err != nil {
		log.Fatalf("RPC Execution Failure: %v", err)
	}

	log.Printf("[MOCK CLIENT] Received Decision: %s | Diagnostic Message: %s", resp.GetAction(), resp.GetReason())
}