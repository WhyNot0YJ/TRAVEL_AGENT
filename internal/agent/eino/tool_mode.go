package eino

import (
	"context"
	"fmt"
)

type toolSet struct {
	POI     POITool
	Weather WeatherTool
	Route   RouteTool
	Budget  BudgetTool
}

type ToolFallbackError struct {
	Tool   string
	Reason string
}

func (e *ToolFallbackError) Error() string {
	return fmt.Sprintf("%s fallback used: %s", e.Tool, e.Reason)
}

func defaultToolSet() toolSet {
	cfg := loadToolConfigFromEnv()
	mock := mockToolSet()
	if cfg.Mode != "real" {
		return mock
	}
	if cfg.AMapAPIKey == "" {
		return toolSet{
			POI:     fallbackPOITool{primary: nil, fallback: mock.POI, toolName: "poi", reason: "TRAVEL_AGENT_AMAP_API_KEY is not configured"},
			Weather: fallbackWeatherTool{primary: nil, fallback: mock.Weather, toolName: "weather", reason: "TRAVEL_AGENT_AMAP_API_KEY is not configured"},
			Route:   fallbackRouteTool{primary: nil, fallback: mock.Route, toolName: "route", reason: "TRAVEL_AGENT_AMAP_API_KEY is not configured"},
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
	return out, &ToolFallbackError{Tool: t.toolName, Reason: t.reason}
}

type fallbackWeatherTool struct {
	primary  WeatherTool
	fallback WeatherTool
	toolName string
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
	return out, &ToolFallbackError{Tool: t.toolName, Reason: t.reason}
}

type fallbackRouteTool struct {
	primary  RouteTool
	fallback RouteTool
	toolName string
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
	return out, &ToolFallbackError{Tool: t.toolName, Reason: t.reason}
}
