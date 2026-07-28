module ChronosGuard
  class RailsMiddleware
    def initialize(client_instance = ChronosGuard::Client.new)
      @client = client_instance
    end

    # ActiveJob Server interceptor loop processing blocks seamlessly
    def call(worker, job, queue)
      # Dynamic payload hook inspection matching active execution arrays
      args = job.arguments.first || {}
      
      return yield unless args.is_a?(Hash) && args.key?(:tenant_id)

      response = @client.check_budget(
        tenant_id: args[:tenant_id].to_s,
        run_id: args[:run_id].to_s,
        tokens_spent: args.fetch(:tokens_spent, 0).to_i,
        state_signature: args.fetch(:state_signature, "").to_s
      )

      case response.action
      when :ACTION_BLOCK
        raise RuntimeError, "Chronos-Guard Circuit Breaker Tripped: #{response.reason}"
      when :ACTION_THROTTLE
        sleep(0.1) # Inject deliberate micro-delay 
        yield
      else
        yield
      end
    end
  end
end