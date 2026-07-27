package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client    *redis.Client
	luaSHA    string
}

// NewRedisStore provisions an operational remote cache accessor and pre-loads the Lua script.
func NewRedisStore(ctx context.Context, addr string, password string, db int) (*RedisStore, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Millisecond,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
		PoolSize:     64,
	})

	// Pre-load the script into Redis memory to use optimized EVALSHA calls rather than sending raw string text over every request
	sha, err := rdb.ScriptLoad(ctx, SlidingWindowLuaScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load rate-limiter lua script: %w", err)
	}
	fmt.Println("Lua Script SHA:", sha)
	return &RedisStore{
		client: rdb,
		luaSHA: sha,
	}, nil
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *RedisStore) TakeAtomic(ctx context.Context, tenantID string, limit int64, window time.Duration, tokens int64) (*LimitResult, error) {
	key := fmt.Sprintf("limits:tenant:%s", tenantID)
	windowSeconds := int64(window.Seconds())

	// Execute via optimized SHA reference
	res, err := s.client.EvalSha(ctx, s.luaSHA, []string{key}, windowSeconds, limit, tokens).Result()
	if err != nil {
		return nil, fmt.Errorf("redis transaction execution failed: %w", err)
	}

	// Parse the array response returned by the Lua script execution layer
	slice, ok := res.([]interface{})
	if !ok || len(slice) < 2 {
		return nil, fmt.Errorf("unexpected lua execution payload structure")
	}

	allowedInt := slice[0].(int64)
	remaining := slice[1].(int64)

	return &LimitResult{
		Allowed:   allowedInt == 1,
		Remaining: remaining,
		ResetTTL:  window,
	}, nil
}