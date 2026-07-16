package limiter

// SlidingWindowLuaScript defines the atomic transaction executed directly inside Redis.
// Keys: KEYS[1] -> Tenant's tracking cache key (e.g., limits:tenant:tenant-id)
// ARGS: ARGV[1] -> Window size in seconds (e.g., 60)
//       ARGV[2] -> Maximum permitted bucket limit (e.g., 100)
//       ARGV[3] -> Requested token cost for this specific operation (e.g., 1)
const SlidingWindowLuaScript = `
local key = KEYS[1]
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local cost = tonumber(ARGV[3])

local now = redis.call('TIME')
local current_time = tonumber(now[1])

-- Remove outdated log points that fall outside the active sliding window
redis.call('ZREMRANGEBYSCORE', key, 0, current_time - window)

-- Evaluate current traffic volume inside the window
local current_requests = redis.call('ZCARD', key)

if current_requests + cost <= limit then
    -- Capacity is available; log the request tokens using non-colliding members
    for i = 1, cost do
        redis.call('ZADD', key, current_time, current_time .. '_' .. math.random())
    end
    -- Set TTL to ensure automated cleanup if traffic stops
    redis.call('EXPIRE', key, window)
    
    return {1, limit - (current_requests + cost)} -- [Allowed=True, RemainingTokens]
else
    -- Capacity exhausted
    return {0, limit - current_requests} -- [Allowed=False, RemainingTokens]
end
`