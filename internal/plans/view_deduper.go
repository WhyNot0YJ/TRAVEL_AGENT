package plans

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const defaultPublicViewDedupTTL = time.Hour

type PublicViewDeduper interface {
	Allow(ctx context.Context, publicPlanID, viewerKey string, now time.Time) (bool, error)
}

type MemoryPublicViewDeduper struct {
	mu   sync.Mutex
	ttl  time.Duration
	seen map[string]time.Time
}

func NewMemoryPublicViewDeduper(ttl time.Duration) *MemoryPublicViewDeduper {
	if ttl <= 0 {
		ttl = defaultPublicViewDedupTTL
	}
	return &MemoryPublicViewDeduper{ttl: ttl, seen: map[string]time.Time{}}
}

func (d *MemoryPublicViewDeduper) Allow(ctx context.Context, publicPlanID, viewerKey string, now time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if d == nil || publicPlanID == "" || viewerKey == "" {
		return true, nil
	}
	key := publicPlanID + ":" + viewerHash(viewerKey)
	d.mu.Lock()
	defer d.mu.Unlock()
	if expiresAt, ok := d.seen[key]; ok && now.Before(expiresAt) {
		return false, nil
	}
	if len(d.seen) > 4096 {
		d.dropExpired(now)
	}
	d.seen[key] = now.Add(d.ttl)
	return true, nil
}

func (d *MemoryPublicViewDeduper) dropExpired(now time.Time) {
	for key, expiresAt := range d.seen {
		if !now.Before(expiresAt) {
			delete(d.seen, key)
		}
	}
}

type RedisPublicViewDeduper struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewRedisPublicViewDeduper(client *goredis.Client, ttl time.Duration) *RedisPublicViewDeduper {
	if ttl <= 0 {
		ttl = defaultPublicViewDedupTTL
	}
	return &RedisPublicViewDeduper{client: client, ttl: ttl}
}

func (d *RedisPublicViewDeduper) Allow(ctx context.Context, publicPlanID, viewerKey string, _ time.Time) (bool, error) {
	if d == nil || d.client == nil || publicPlanID == "" || viewerKey == "" {
		return true, nil
	}
	return d.client.SetNX(ctx, publicViewDedupKey(publicPlanID, viewerKey), "1", d.ttl).Result()
}

func publicViewDedupKey(publicPlanID, viewerKey string) string {
	return "travel:public_plan:view:" + publicPlanID + ":" + viewerHash(viewerKey)
}

func viewerHash(viewerKey string) string {
	sum := sha256.Sum256([]byte(viewerKey))
	return hex.EncodeToString(sum[:16])
}
