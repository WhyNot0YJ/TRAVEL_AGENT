package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"travel-agent/internal/agent"
	einoagent "travel-agent/internal/agent/eino"
)

type Config struct {
	HTTPAddr           string
	Planner            string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	CacheTTL           time.Duration
	RateLimitPerMinute int
}

func Load() Config {
	return Config{
		HTTPAddr:           envOrDefault("TRAVEL_AGENT_HTTP_ADDR", ":8080"),
		Planner:            strings.ToLower(envOrDefault("TRAVEL_AGENT_PLANNER", "mock")),
		RedisAddr:          strings.TrimSpace(os.Getenv("TRAVEL_AGENT_REDIS_ADDR")),
		RedisPassword:      os.Getenv("TRAVEL_AGENT_REDIS_PASSWORD"),
		RedisDB:            envInt("TRAVEL_AGENT_REDIS_DB", 0),
		CacheTTL:           time.Duration(envInt("TRAVEL_AGENT_CACHE_TTL_SECONDS", 1800)) * time.Second,
		RateLimitPerMinute: envInt("TRAVEL_AGENT_RATE_LIMIT_PER_MINUTE", 60),
	}
}

func BuildPlanner(plannerType string) (agent.TravelPlanner, error) {
	switch strings.ToLower(strings.TrimSpace(plannerType)) {
	case "", "mock":
		return agent.NewMockPlanner(), nil
	case "eino":
		return einoagent.NewEinoTravelPlanner()
	default:
		return nil, fmt.Errorf("unsupported planner type: %s", plannerType)
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
