package eino

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultLLMProvider = "deepseek"
	defaultLLMModel    = "deepseek-v4-flash"
	defaultLLMBaseURL  = "https://api.deepseek.com/beta"
	defaultLLMTimeout  = 30 * time.Second
	defaultLLMRetries  = 1

	defaultToolMode           = "mock"
	defaultAMapBaseURL        = "https://restapi.amap.com/v3"
	defaultExternalAPITimeout = 10 * time.Second
)

// LLMConfig stores environment-driven model configuration.
type LLMConfig struct {
	Enabled    bool
	Provider   string
	APIKey     string
	BaseURL    string
	Model      string
	Timeout    time.Duration
	MaxRetries int
}

// ToolConfig stores environment-driven external tool configuration.
type ToolConfig struct {
	Mode               string
	AMapAPIKey         string
	AMapBaseURL        string
	WeatherAPIKey      string
	WeatherBaseURL     string
	ExternalAPITimeout time.Duration
}

func loadLLMConfigFromEnv() LLMConfig {
	provider := envOrDefault("TRAVEL_AGENT_LLM_PROVIDER", defaultLLMProvider)
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = defaultLLMProvider
	}

	cfg := LLMConfig{
		Enabled:    parseBool(os.Getenv("TRAVEL_AGENT_LLM_ENABLED")),
		Provider:   provider,
		APIKey:     strings.TrimSpace(os.Getenv("TRAVEL_AGENT_LLM_API_KEY")),
		BaseURL:    strings.TrimSpace(os.Getenv("TRAVEL_AGENT_LLM_BASE_URL")),
		Model:      envOrDefault("TRAVEL_AGENT_LLM_MODEL", defaultLLMModel),
		Timeout:    defaultLLMTimeout,
		MaxRetries: defaultLLMRetries,
	}
	if cfg.APIKey == "" && cfg.Provider == "deepseek" {
		cfg.APIKey = strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL(cfg.Provider)
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultLLMTimeout
	}
	if rawTimeout := strings.TrimSpace(os.Getenv("TRAVEL_AGENT_LLM_TIMEOUT")); rawTimeout != "" {
		if parsed, err := time.ParseDuration(rawTimeout); err == nil && parsed > 0 {
			cfg.Timeout = parsed
		} else if seconds, err := strconv.Atoi(rawTimeout); err == nil && seconds > 0 {
			cfg.Timeout = time.Duration(seconds) * time.Second
		}
	}
	if rawRetries := strings.TrimSpace(os.Getenv("TRAVEL_AGENT_LLM_MAX_RETRIES")); rawRetries != "" {
		if parsed, err := strconv.Atoi(rawRetries); err == nil && parsed >= 0 {
			cfg.MaxRetries = parsed
		}
	}
	return cfg
}

func loadToolConfigFromEnv() ToolConfig {
	cfg := ToolConfig{
		Mode:               strings.ToLower(envOrDefault("TRAVEL_AGENT_TOOL_MODE", defaultToolMode)),
		AMapAPIKey:         strings.TrimSpace(os.Getenv("TRAVEL_AGENT_AMAP_API_KEY")),
		AMapBaseURL:        envOrDefault("TRAVEL_AGENT_AMAP_BASE_URL", defaultAMapBaseURL),
		WeatherAPIKey:      strings.TrimSpace(os.Getenv("TRAVEL_AGENT_WEATHER_API_KEY")),
		WeatherBaseURL:     strings.TrimSpace(os.Getenv("TRAVEL_AGENT_WEATHER_BASE_URL")),
		ExternalAPITimeout: defaultExternalAPITimeout,
	}
	if cfg.Mode != "real" {
		cfg.Mode = defaultToolMode
	}
	if cfg.WeatherAPIKey == "" {
		cfg.WeatherAPIKey = cfg.AMapAPIKey
	}
	if cfg.WeatherBaseURL == "" {
		cfg.WeatherBaseURL = cfg.AMapBaseURL
	}
	if rawTimeout := strings.TrimSpace(os.Getenv("TRAVEL_AGENT_EXTERNAL_API_TIMEOUT")); rawTimeout != "" {
		if parsed, err := time.ParseDuration(rawTimeout); err == nil && parsed > 0 {
			cfg.ExternalAPITimeout = parsed
		} else if seconds, err := strconv.Atoi(rawTimeout); err == nil && seconds > 0 {
			cfg.ExternalAPITimeout = time.Duration(seconds) * time.Second
		}
	}
	return cfg
}

func defaultBaseURL(provider string) string {
	switch provider {
	case "deepseek":
		return defaultLLMBaseURL
	default:
		return strings.TrimSuffix(os.Getenv("OPENAI_BASE_URL"), "/")
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on", "enabled":
		return true
	default:
		return false
	}
}
