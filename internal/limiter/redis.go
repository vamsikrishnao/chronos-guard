package limiter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	luaSHA string
}

func NewRedisStore(ctx context.Context, addr string, password string, db int) (*RedisStore, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  100 * time.Millisecond,
		WriteTimeout: 100 * time.Millisecond,
		PoolSize:     64,
	})

	sha, err := rdb.ScriptLoad(ctx, SlidingWindowLuaScript).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to load rate-limiter lua script: %w", err)
	}

	return &RedisStore{
		client: rdb,
		luaSHA: sha,
	}, nil
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

func (s *RedisStore) TakeAtomic(ctx context.Context, tenantID string, limit int64, window time.Duration, tokens int64) (*LimitResult, error) {
	return s.TakeAtomicWithSignature(ctx, tenantID, limit, window, tokens, "")
}

func (s *RedisStore) TakeAtomicWithSignature(ctx context.Context, tenantID string, limit int64, window time.Duration, tokens int64, signature string) (*LimitResult, error) {
	rateKey := fmt.Sprintf("limits:tenant:%s", tenantID)
	sigKey := ""
	if signature != "" {
		sigKey = fmt.Sprintf("limits:tenant:%s:sig:%s", tenantID, signature)
	}

	windowSeconds := int64(window.Seconds())
	if windowSeconds <= 0 {
		windowSeconds = 60
	}

	reqUUID := generateUUID()
	loopThreshold := int64(5)

	res, err := s.client.EvalSha(ctx, s.luaSHA, []string{rateKey, sigKey}, windowSeconds, limit, tokens, reqUUID, loopThreshold).Result()
	if err != nil {
		return nil, fmt.Errorf("redis transaction execution failed: %w", err)
	}

	slice, ok := res.([]interface{})
	if !ok || len(slice) < 3 {
		return nil, fmt.Errorf("unexpected lua execution payload structure")
	}

	allowedInt, ok1 := toInt64(slice[0])
	remaining, ok2 := toInt64(slice[1])
	loopDetectedInt, ok3 := toInt64(slice[2])

	if !ok1 || !ok2 || !ok3 {
		return nil, fmt.Errorf("failed to parse lua numeric response safely")
	}

	return &LimitResult{
		Allowed:      allowedInt == 1,
		Remaining:    remaining,
		ResetTTL:     window,
		LoopDetected: loopDetectedInt == 1,
	}, nil
}

func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case float64:
		return int64(val), true
	default:
		return 0, false
	}
}

func generateUUID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}