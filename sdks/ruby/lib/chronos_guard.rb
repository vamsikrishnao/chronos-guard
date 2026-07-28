require 'grpc'
require_relative 'proto/guard_services_pb'

module ChronosGuard
  class Client
    def initialize(target = '127.0.0.1:50051')
      @stub = Chronos::V1::GuardService::Stub.new(target, :this_channel_is_insecure)
    end

    def check_budget(tenant_id:, run_id:, tokens_spent:, state_signature:)
      request = Chronos::V1::CheckBudgetRequest.new(
        tenant_id: tenant_id,
        run_id: run_id,
        tokens_spent: tokens_spent,
        state_signature: state_signature
      )

      begin
        # Transit evaluation thread matching microsecond constraints
        @stub.check_budget(request, deadline: Time.now + 2)
      rescue GRPC::BadStatus => e
        # Resiliency Invariant: Fail open securely
        Rails.logger.error("Chronos-Guard proxy unreachable (#{e.message}). Falling open.") if defined?(Rails)
        Chronos::V1::CheckBudgetResponse.new(action: :ACTION_ALLOW, reason: "Fail-open active.")
      end
    end
  end
end