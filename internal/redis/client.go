package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
	"travel-agent/internal/config"
)

func NewClient(ctx context.Context, cfg config.Config) (*goredis.Client, error) {
	if cfg.RedisAddr == "" {
		return nil, fmt.Errorf("TRAVEL_AGENT_REDIS_ADDR is not configured")
	}
	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}
