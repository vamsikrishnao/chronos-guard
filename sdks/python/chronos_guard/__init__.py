import grpc
from . import guard_pb2
from . import guard_pb2_grpc
from .decorator import guard_budget

class ChronosGuardClient:
    def __init__(self, target_address="127.0.0.1:50051"):
        self.channel = grpc.insecure_channel(target_address)
        self.stub = guard_pb2_grpc.GuardServiceStub(self.channel)

    def check_budget(self, tenant_id: str, run_id: str, tokens_spent: int, state_signature: str):
        request = guard_pb2.CheckBudgetRequest(
            tenant_id=tenant_id,
            run_id=run_id,
            tokens_spent=tokens_spent,
            state_signature=state_signature
        )
        try:
            # Enforce execution timeout limits on evaluation paths
            return self.stub.CheckBudget(request, timeout=2.0)
        except grpc.RpcError as e:
            # Fail-open invariant: Return a mock ALLOW action if proxy connectivity is degraded
            return guard_pb2.CheckBudgetResponse(
                action=guard_pb2.CheckBudgetResponse.Action.ACTION_ALLOW,
                reason=f"Chronos-Guard communications degraded ({e.code()}). System dropped to fail-open boundary safety."
            )

    def close(self):
        self.channel.close()