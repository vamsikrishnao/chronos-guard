package server

import (
	"context"
	"fmt"
	"time"

	"github.com/vamsikrishnao/chronos-guard/internal/limiter"
	pb "github.com/vamsikrishnao/chronos-guard/proto/chronos/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GuardServer struct {
	pb.UnimplementedGuardServiceServer
	store         limiter.RateLimitStore
	defaultLimit  int64
	defaultWindow time.Duration
}

func NewGuardServer(store limiter.RateLimitStore, defaultLimit int64, defaultWindow time.Duration) *GuardServer {
	return &GuardServer{
		store:         store,
		defaultLimit:  defaultLimit,
		defaultWindow: defaultWindow,
	}
}

func (s *GuardServer) CheckBudget(ctx context.Context, req *pb.CheckBudgetRequest) (*pb.CheckBudgetResponse, error) {
	tenantID := req.GetTenantId()
	if tenantID == "" {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get("x-tenant-id"); len(vals) > 0 {
				tenantID = vals[0]
			}
		}
	}

	if tenantID == "" {
		return nil, status.Error(codes.InvalidArgument, "missing tenant_id in request payload or x-tenant-id metadata")
	}

	tokens := req.GetTokensSpent()
	if tokens <= 0 {
		tokens = 1
	}

	signature := req.GetStateSignature()

	// Evaluate token usage AND loop signatures
	res, err := s.store.TakeAtomicWithSignature(ctx, tenantID, s.defaultLimit, s.defaultWindow, tokens, signature)
	if err != nil {
		return &pb.CheckBudgetResponse{
			Action: pb.CheckBudgetResponse_ACTION_ALLOW,
			Reason: fmt.Sprintf("Guard store degraded (%v); failing open for platform safety", err),
		}, nil
	}

	if res.LoopDetected {
		return &pb.CheckBudgetResponse{
			Action: pb.CheckBudgetResponse_ACTION_BLOCK,
			Reason: fmt.Sprintf("AI agent infinite loop detected for state signature: %s", signature),
		}, nil
	}

	if !res.Allowed {
		return &pb.CheckBudgetResponse{
			Action: pb.CheckBudgetResponse_ACTION_BLOCK,
			Reason: fmt.Sprintf("Tenant %s rate limit exceeded. Remaining tokens: %d", tenantID, res.Remaining),
		}, nil
	}

	if res.Remaining < (s.defaultLimit / 10) {
		return &pb.CheckBudgetResponse{
			Action: pb.CheckBudgetResponse_ACTION_THROTTLE,
			Reason: fmt.Sprintf("Tenant %s approaching quota limit. Remaining tokens: %d", tenantID, res.Remaining),
		}, nil
	}

	return &pb.CheckBudgetResponse{
		Action: pb.CheckBudgetResponse_ACTION_ALLOW,
		Reason: "Budget verification successful",
	}, nil
}