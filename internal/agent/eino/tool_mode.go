package eino

import (
	"context"
	"fmt"
	"strings"
)

type toolSet struct {
	POI     POITool
	Weather WeatherTool
	Route   RouteTool
	Budget  BudgetTool
}

type ToolFallbackError struct {
	Tool         string
	Provider     string
	Stage        string
	Category     string
	Reason       string
	MockFallback bool
}

func (e *ToolFallbackError) Error() string {
	if e == nil {
		return "tool fallback: <nil>"
	}
	provider := fallbackValue(e.Provider, "amap")
	stage := fallbackValue(e.Stage, "request")
	category := fallbackValue(e.Category, classifyToolFailure(e.Reason))
	return fmt.Sprintf("tool fallback: tool=%s provider=%s stage=%s category=%s mock_fallback=%t reason=%s",
		fallbackValue(e.Tool, "unknown"),
		provider,
		stage,
		category,
		e.MockFallback,
		fallbackValue(e.Reason, "unknown"),
	)
}

func defaultToolSet() toolSet {
	cfg := loadToolConfigFromEnv()
	mock := mockToolSet()
	if cfg.Mode != "real" {
		return mock
	}
	if cfg.AMapAPIKey == "" {
		return toolSet{
			POI:     fallbackPOITool{primary: nil, fallback: mock.POI, toolName: "poi", stage: "configuration", reason: "TRAVEL_AGENT_AMAP_API_KEY is not configured"},
			Weather: fallbackWeatherTool{primary: nil, fallback: mock.Weather, toolName: "weather", stage: "configuration", reason: "TRAVEL_AGENT_AMAP_API_KEY is not configured"},
			Route:   fallbackRouteTool{primary: nil, fallback: mock.Route, toolName: "route", stage: "configuration", reason: "TRAVEL_AGENT_AMAP_API_KEY is not configured"},
			Budget:  mock.Budget,
		}
	}

	amapClient := newAMapClient(cfg.AMapBaseURL, cfg.AMapAPIKey, cfg.ExternalAPITimeout)
	weatherClient := amapClient
	if cfg.WeatherBaseURL != cfg.AMapBaseURL || cfg.WeatherAPIKey != cfg.AMapAPIKey {
		weatherClient = newAMapClient(cfg.WeatherBaseURL, cfg.WeatherAPIKey, cfg.ExternalAPITimeout)
	}
	return toolSet{
		POI:     fallbackPOITool{primary: RealPOITool{client: amapClient}, fallback: mock.POI, toolName: "poi"},
		Weather: fallbackWeatherTool{primary: RealWeatherTool{client: weatherClient, geocoder: amapClient}, fallback: mock.Weather, toolName: "weather"},
		Route:   fallbackRouteTool{primary: RealRouteTool{client: amapClient}, fallback: mock.Route, toolName: "route"},
		Budget:  mock.Budget,
	}
}

func mockToolSet() toolSet {
	return toolSet{
		POI:     MockPOITool{},
		Weather: MockWeatherTool{},
		Route:   MockRouteTool{},
		Budget:  MockBudgetTool{},
	}
}

type fallbackPOITool struct {
	primary  POITool
	fallback POITool
	toolName string
	stage    string
	reason   string
}

func (t fallbackPOITool) Run(ctx context.Context, input POIToolInput) ([]MockPOI, error) {
	if t.primary != nil {
		out, err := t.primary.Run(ctx, input)
		if err == nil {
			return out, nil
		}
		t.reason = err.Error()
	}
	out, err := t.fallback.Run(ctx, input)
	if err != nil {
		return nil, err
	}
	return out, newToolFallbackError(t.toolName, t.stage, t.reason)
}

type fallbackWeatherTool struct {
	primary  WeatherTool
	fallback WeatherTool
	toolName string
	stage    string
	reason   string
}

func (t fallbackWeatherTool) Run(ctx context.Context, input WeatherToolInput) ([]MockWeather, error) {
	if t.primary != nil {
		out, err := t.primary.Run(ctx, input)
		if err == nil {
			return out, nil
		}
		t.reason = err.Error()
	}
	out, err := t.fallback.Run(ctx, input)
	if err != nil {
		return nil, err
	}
	return out, newToolFallbackError(t.toolName, t.stage, t.reason)
}

type fallbackRouteTool struct {
	primary  RouteTool
	fallback RouteTool
	toolName string
	stage    string
	reason   string
}

func (t fallbackRouteTool) Run(ctx context.Context, input RouteToolInput) ([]MockRoute, error) {
	if t.primary != nil {
		out, err := t.primary.Run(ctx, input)
		if err == nil {
			return out, nil
		}
		t.reason = err.Error()
	}
	out, err := t.fallback.Run(ctx, input)
	if err != nil {
		return nil, err
	}
	return out, newToolFallbackError(t.toolName, t.stage, t.reason)
}

func newToolFallbackError(tool, stage, reason string) *ToolFallbackError {
	if stage == "" {
		stage = "request"
	}
	return &ToolFallbackError{
		Tool:         tool,
		Provider:     "amap",
		Stage:        stage,
		Category:     classifyToolFailure(reason),
		Reason:       reason,
		MockFallback: true,
	}
}

func classifyToolFailure(reason string) string {
	lower := strings.ToLower(strings.TrimSpace(reason))
	switch {
	case lower == "":
		return "unknown"
	case strings.Contains(lower, "not configured"), strings.Contains(lower, "api key is empty"), strings.Contains(lower, "base url is empty"):
		return "configuration"
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "deadline exceeded"):
		return "timeout"
	case strings.Contains(lower, "status "), strings.Contains(lower, "api error"):
		return "provider_error"
	case strings.Contains(lower, "decode"), strings.Contains(lower, "invalid character"), strings.Contains(lower, "json"):
		return "invalid_json"
	case strings.Contains(lower, "empty"), strings.Contains(lower, "missing"), strings.Contains(lower, "required"), strings.Contains(lower, "no usable"):
		return "missing_field"
	default:
		return "request_error"
	}
}

func fallbackValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
