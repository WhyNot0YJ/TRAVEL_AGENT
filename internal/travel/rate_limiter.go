package travel

import (
	"context"
	"strconv"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type MemoryRateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	counters map[string]rateBucket
}

type rateBucket struct {
	Count     int
	ExpiresAt time.Time
}

func NewMemoryRateLimiter(limit int) *MemoryRateLimiter {
	if limit <= 0 {
		limit = 60
	}
	return &MemoryRateLimiter{limit: limit, window: time.Minute, counters: map[string]rateBucket{}}
}

func (l *MemoryRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket := l.counters[key]
	if bucket.ExpiresAt.Before(now) {
		bucket = rateBucket{ExpiresAt: now.Add(l.window)}
	}
	bucket.Count++
	l.counters[key] = bucket
	return bucket.Count <= l.limit, nil
}

type RedisRateLimiter struct {
	client *goredis.Client
	limit  int
	window time.Duration
}

func NewRedisRateLimiter(client *goredis.Client, limit int) *RedisRateLimiter {
	if limit <= 0 {
		limit = 60
	}
	return &RedisRateLimiter{client: client, limit: limit, window: time.Minute}
}

func (l *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := "travel:rate:" + key
	count, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		_ = l.client.Expire(ctx, redisKey, l.window).Err()
	}
	return count <= int64(l.limit), nil
}

func clientRateKey(ip string) string {
	if ip == "" {
		return "unknown"
	}
	return strconv.Quote(ip)
}
