package config

import (
	"fmt"
	"os"
	"strings"

	"travel-agent/internal/agent"
	einoagent "travel-agent/internal/agent/eino"
)

type Config struct {
	HTTPAddr string
	Planner  string
}

func Load() Config {
	return Config{
		HTTPAddr: envOrDefault("TRAVEL_AGENT_HTTP_ADDR", ":8080"),
		Planner:  strings.ToLower(envOrDefault("TRAVEL_AGENT_PLANNER", "mock")),
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
