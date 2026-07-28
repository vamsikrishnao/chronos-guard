import time
import functools
import logging
from typing import Callable, Any, Union

logger = logging.getLogger("chronos_guard")

class AgentBlockedException(Exception):
    """Raised when execution limits are breached or loops are detected."""
    pass

def guard_budget(client_instance: Any):
    """
    A plug-and-play decorator matching Java/Ruby standards.
    Automatically extracts context fields from an object or dict passed as the first argument.
    """
    def decorator(func: Callable):
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            # Fallback to empty context if none provided
            ctx = args[0] if args else {}
            
            # Helper to extract fields dynamically whether the context is a Dict or an Object
            def get_field(key: str, default: Any = "") -> Any:
                if isinstance(ctx, dict):
                    return ctx.get(key, default)
                return getattr(ctx, key, default)

            # Extract standard proto fields dynamically
            tenant_id = str(get_field("tenant_id", "default_tenant"))
            run_id = str(get_field("run_id", "untracked_run"))
            tokens_spent = int(get_field("tokens_spent", 0))
            state_signature = str(get_field("state_signature", ""))

            # Query the sidecar proxy pipeline
            response = client_instance.check_budget(
                tenant_id=tenant_id,
                run_id=run_id,
                tokens_spent=tokens_spent,
                state_signature=state_signature
            )

            # Process deterministic action matrix (0: Unspecified, 1: Allow, 2: Throttle, 3: Block)
            if response.action == 3:  # ACTION_BLOCK
                logger.error(f"Chronos-Guard Circuit Breaker Tripped: {response.reason}")
                raise AgentBlockedException(f"AI Agent halted by platform guardrails: {response.reason}")
            
            elif response.action == 2:  # ACTION_THROTTLE
                logger.warning(f"Chronos-Guard Throttling active: {response.reason}")
                time.sleep(0.1)  # Auto-inject standard 100ms platform delay
                
            return func(*args, **kwargs)
        return wrapper
    return decorator