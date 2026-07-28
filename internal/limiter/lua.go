package limiter

// SlidingWindowLuaScript defines the atomic transaction executed directly inside Redis.
// KEYS[1] -> Tenant tracking key (limits:tenant:{tenant_id})
// KEYS[2] -> State signature loop tracking key (limits:tenant:{tenant_id}:sig:{signature})
// ARGV[1] -> Window size in seconds
// ARGV[2] -> Maximum token capacity limit
// ARGV[3] -> Requested token cost for this step
// ARGV[4] -> Unique request UUID (prevents ZSET member collisions)
// ARGV[5] -> Max allowed loop signature count (0 to disable loop checks)
const SlidingWindowLuaScript = `
local rate_key = KEYS[1]
local sig_key = KEYS[2]

local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local cost = tonumber(ARGV[3])
local req_uuid = ARGV[4]
local loop_threshold = tonumber(ARGV[5])

local now = redis.call('TIME')
local current_time = tonumber(now[1])
local current_micros = tonumber(now[2])

-- 1. Check AI Agent Loop Signature
local loop_detected = 0
if sig_key ~= "" and loop_threshold > 0 then
    local sig_count = redis.call('INCR', sig_key)
    if sig_count == 1 then
        redis.call('EXPIRE', sig_key, window)
    end
    if sig_count > loop_threshold then
        loop_detected = 1
    end
end

-- 2. Evict expired entries outside active sliding window
redis.call('ZREMRANGEBYSCORE', rate_key, 0, current_time - window)

-- 3. Calculate current accumulated token usage in window
local members = redis.call('ZRANGE', rate_key, 0, -1)
local current_used = 0
for _, m in ipairs(members) do
    local last_colon = string.find(m, ":[^:]*$")
    if last_colon then
        local token_val = tonumber(string.sub(m, last_colon + 1))
        if token_val then
            current_used = current_used + token_val
        end
    else
        current_used = current_used + 1
    end
end

-- 4. Evaluate capacity and loop invariants
if (current_used + cost <= limit) and (loop_detected == 0) then
    local member_val = current_micros .. ":" .. req_uuid .. ":" .. cost
    redis.call('ZADD', rate_key, current_time, member_val)
    redis.call('EXPIRE', rate_key, window)
    
    local remaining = limit - (current_used + cost)
    return {1, remaining, 0}
else
    local remaining = limit - current_used
    if remaining < 0 then remaining = 0 end
    return {0, remaining, loop_detected}
end
`