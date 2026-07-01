package travel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type RequestLock interface {
	Acquire(ctx context.Context, requestHash string) (release func(context.Context) error, acquired bool, err error)
}

type MemoryRequestLock struct {
	mu    sync.Mutex
	locks map[string]time.Time
	ttl   time.Duration
	now   func() time.Time
}

func NewMemoryRequestLock(ttl time.Duration) *MemoryRequestLock {
	if ttl <= 0 {
		ttl = 15 * time.Second
	}
	return &MemoryRequestLock{locks: map[string]time.Time{}, ttl: ttl, now: time.Now}
}

func (l *MemoryRequestLock) Acquire(ctx context.Context, requestHash string) (func(context.Context) error, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if requestHash == "" {
		return noopRelease, true, nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if expiresAt, ok := l.locks[requestHash]; ok && expiresAt.After(now) {
		return noopRelease, false, nil
	}
	l.locks[requestHash] = now.Add(l.ttl)
	return func(context.Context) error {
		l.mu.Lock()
		defer l.mu.Unlock()
		delete(l.locks, requestHash)
		return nil
	}, true, nil
}

type RedisRequestLock struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewRedisRequestLock(client *goredis.Client, ttl time.Duration) *RedisRequestLock {
	if ttl <= 0 {
		ttl = 15 * time.Second
	}
	return &RedisRequestLock{client: client, ttl: ttl}
}

func (l *RedisRequestLock) Acquire(ctx context.Context, requestHash string) (func(context.Context) error, bool, error) {
	if l == nil || l.client == nil || requestHash == "" {
		return noopRelease, true, nil
	}
	token := randomLockToken()
	key := requestLockKey(requestHash)
	acquired, err := l.client.SetNX(ctx, key, token, l.ttl).Result()
	if err != nil || !acquired {
		return noopRelease, acquired, err
	}
	return func(ctx context.Context) error {
		return l.client.Eval(ctx, redisUnlockScript, []string{key}, token).Err()
	}, true, nil
}

const redisUnlockScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`

func requestLockKey(hash string) string {
	return "travel:lock:request_hash:" + hash
}

func randomLockToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func noopRelease(context.Context) error {
	return nil
}
