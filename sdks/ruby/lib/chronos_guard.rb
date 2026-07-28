require 'grpc'

module ChronosGuard
  class Client
    def initialize(target = '127.0.0.1:50051', timeout: 2)
      @target = target
      @timeout = timeout
      # Lazily initialized reusable stub channel
      @stub = Chronos::V1::GuardService::Stub.new(target, :this_channel_is_insecure)
    end

    def check_budget(tenant_id:, run_id:, tokens_spent:, state_signature:)
      safe_tenant = tenant_id.to_s.empty? ? "default_tenant" : tenant_id.to_s
      safe_run = run_id.to_s.empty? ? "untracked_run" : run_id.to_s
      safe_tokens = [0, tokens_spent.to_i].max
      safe_sig = state_signature.to_s

      request = Chronos::V1::CheckBudgetRequest.new(
        tenant_id: safe_tenant,
        run_id: safe_run,
        tokens_spent: safe_tokens,
        state_signature: safe_sig
      )

      begin
        @stub.check_budget(request, deadline: Time.now + @timeout)
      rescue StandardError => e
        if defined?(Rails)
          Rails.logger.warn("Chronos-Guard proxy unreachable (#{e.message}). Falling open.")
        end
        Chronos::V1::CheckBudgetResponse.new(
          action: :ACTION_ALLOW,
          reason: "Fail-open active: proxy communication degraded."
        )
      end
    end
  end
end