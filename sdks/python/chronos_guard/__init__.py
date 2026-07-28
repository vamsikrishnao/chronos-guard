import logging
import grpc

try:
    from . import guard_pb2
    from . import guard_pb2_grpc
except ImportError:
    import guard_pb2
    import guard_pb2_grpc

logger = logging.getLogger("chronos_guard")


class ChronosGuardClient:
    """Production-grade Python SDK client for Chronos-Guard sidecar proxy."""

    def __init__(self, target_address: str = "127.0.0.1:50051", timeout: float = 2.0):
        self.target_address = target_address
        self.timeout = timeout
        # Persistent channel connection pool reuse
        self.channel = grpc.insecure_channel(target_address)
        self.stub = guard_pb2_grpc.GuardServiceStub(self.channel)

    def check_budget(self, tenant_id: str, run_id: str, tokens_spent: int, state_signature: str):
        # 1. Parameter Sanitization
        tenant_id = str(tenant_id) if tenant_id else "default_tenant"
        run_id = str(run_id) if run_id else "untracked_run"
        try:
            tokens_spent = max(0, int(tokens_spent))
        except (ValueError, TypeError):
            tokens_spent = 0

        state_signature = str(state_signature) if state_signature else ""

        request = guard_pb2.CheckBudgetRequest(
            tenant_id=tenant_id,
            run_id=run_id,
            tokens_spent=tokens_spent,
            state_signature=state_signature,
        )

        try:
            return self.stub.CheckBudget(request, timeout=self.timeout)
        except grpc.RpcError as e:
            logger.warning(
                "Chronos-Guard proxy unreachable (%s). Adhering to fail-open safety standard.",
                e.code(),
            )
            # Guarantee fail-open fallback response matching CheckBudgetResponse schema
            return guard_pb2.CheckBudgetResponse(
                action=guard_pb2.CheckBudgetResponse.Action.ACTION_ALLOW,
                reason=f"Fail-open active: proxy communication degraded ({e.code()}).",
            )

    def close(self):
        if self.channel:
            self.channel.close()