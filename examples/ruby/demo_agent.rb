#!/usr/bin/env ruby
require 'grpc'
require_relative '../../sdks/ruby/lib/chronos_guard'

puts "\n--- RUBY SDK DEMO: FinOps LLM Cost Containment & Safeguard ---"
puts "Connecting to Chronos-Guard Sidecar Proxy at 127.0.0.1:50051...\n\n"

client = ChronosGuard::Client.new('127.0.0.1:50051')

tenant_id = 'Acme-Corp-Ruby'
run_id = 'ruby_run_303'

# 1. Normal execution steps
(1..2).each do |step|
  resp = client.check_budget(
    tenant_id: tenant_id,
    run_id: run_id,
    tokens_spent: 20,
    state_signature: "ruby_sig_step_#{step}"
  )
  puts "   [WORKER] Step #{step}: Action -> #{resp.action} | Reason: #{resp.reason}"
end

# 2. High consumption step
puts "\n   [WARNING] Step 3: High token consumption spike..."
throttle_resp = client.check_budget(
  tenant_id: tenant_id,
  run_id: run_id,
  tokens_spent: 60,
  state_signature: "ruby_high_sig"
)
puts "   [WORKER] Step 3: Action -> #{throttle_resp.action} | Reason: #{throttle_resp.reason}"

# 3. Infinite loop simulation
puts "\n   [CRITICAL] Step 4+: Infinite loop simulation (Repeated state signature)..."
loop_sig = 'corrupted_pdf_ruby_sig_99x'

(4..10).each do |step|
  resp = client.check_budget(
    tenant_id: tenant_id,
    run_id: run_id,
    tokens_spent: 10,
    state_signature: loop_sig
  )
  puts "   [WORKER] Step #{step}: Action -> #{resp.action} | Reason: #{resp.reason}"

  if resp.action == :ACTION_BLOCK
    puts "\n   [CIRCUIT BREAKER TRIPPED] AI Agent loop halted by Ruby SDK!"
    puts "   [FINOPS SUCCESS] Prevented runaway API spend.\n\n"
    break
  end
end