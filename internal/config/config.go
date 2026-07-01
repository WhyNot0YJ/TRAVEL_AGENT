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
	RedisLockTTL       time.Duration
	RateLimitPerMinute int
	SQLEnabled         bool
	SQLDSN             string
	SQLMaxOpenConns    int
	SQLMaxIdleConns    int
	SQLConnMaxLifetime time.Duration

	// Stage 21 — auth, sessions and access policy.
	AuthEnabled                  bool
	SessionCookieName            string
	SessionTTL                   time.Duration
	PasswordMinLength            int
	PublicPlanPageSize           int
	AllowAnonymousPlanGeneration bool
	CookieSecure                 bool
	CookieDomain                 string
	AllowedOrigins               []string
}

func Load() Config {
	loadDotEnv()
	return Config{
		HTTPAddr:           envOrDefault("TRAVEL_AGENT_HTTP_ADDR", ":8080"),
		Planner:            strings.ToLower(envOrDefault("TRAVEL_AGENT_PLANNER", "mock")),
		RedisAddr:          strings.TrimSpace(os.Getenv("TRAVEL_AGENT_REDIS_ADDR")),
		RedisPassword:      os.Getenv("TRAVEL_AGENT_REDIS_PASSWORD"),
		RedisDB:            envInt("TRAVEL_AGENT_REDIS_DB", 0),
		CacheTTL:           time.Duration(envInt("TRAVEL_AGENT_CACHE_TTL_SECONDS", 1800)) * time.Second,
		RedisLockTTL:       time.Duration(envInt("TRAVEL_AGENT_REDIS_LOCK_TTL_SECONDS", 15)) * time.Second,
		RateLimitPerMinute: envInt("TRAVEL_AGENT_RATE_LIMIT_PER_MINUTE", 60),
		SQLEnabled:         parseBool(os.Getenv("TRAVEL_AGENT_SQL_ENABLED")),
		SQLDSN:             strings.TrimSpace(os.Getenv("TRAVEL_AGENT_SQL_DSN")),
		SQLMaxOpenConns:    envInt("TRAVEL_AGENT_SQL_MAX_OPEN_CONNS", 10),
		SQLMaxIdleConns:    envInt("TRAVEL_AGENT_SQL_MAX_IDLE_CONNS", 5),
		SQLConnMaxLifetime: time.Duration(envInt("TRAVEL_AGENT_SQL_CONN_MAX_LIFETIME_SECONDS", 1800)) * time.Second,

		AuthEnabled:                  parseBoolDefault("TRAVEL_AGENT_AUTH_ENABLED", true),
		SessionCookieName:            envOrDefault("TRAVEL_AGENT_SESSION_COOKIE_NAME", "travel_agent_session"),
		SessionTTL:                   time.Duration(envInt("TRAVEL_AGENT_SESSION_TTL_HOURS", 168)) * time.Hour,
		PasswordMinLength:            envInt("TRAVEL_AGENT_PASSWORD_MIN_LENGTH", 8),
		PublicPlanPageSize:           envInt("TRAVEL_AGENT_PUBLIC_PLAN_PAGE_SIZE", 20),
		AllowAnonymousPlanGeneration: parseBool(os.Getenv("TRAVEL_AGENT_ALLOW_ANONYMOUS_PLAN_GENERATION")),
		CookieSecure:                 parseBool(os.Getenv("TRAVEL_AGENT_COOKIE_SECURE")),
		CookieDomain:                 strings.TrimSpace(os.Getenv("TRAVEL_AGENT_COOKIE_DOMAIN")),
		AllowedOrigins:               splitCSV(envOrDefault("TRAVEL_AGENT_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
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

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on", "enabled":
		return true
	default:
		return false
	}
}

// parseBoolDefault returns the env var if set, otherwise the supplied default.
// parseBool always returns false for an unset key, which is wrong for flags
// whose default we want to keep on (auth_enabled).
func parseBoolDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return parseBool(value)
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

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
