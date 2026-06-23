package eino

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRealToolsUseFakeAMapServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/place/text":
			writeJSON(t, w, amapPOIResponse{
				Status: "1",
				Info:   "OK",
				POIs: []amapPOIItem{
					{Name: "西湖", Type: "景点", Address: "杭州西湖", Location: "120.1,30.2"},
					{Name: "灵隐寺", Type: "寺庙", Address: "杭州灵隐", Location: "120.2,30.3"},
				},
			})
		case "/geocode/geo":
			writeJSON(t, w, amapGeocodeResponse{
				Status:   "1",
				Info:     "OK",
				Geocodes: []amapGeocodeItem{{Adcode: "330100"}},
			})
		case "/weather/weatherInfo":
			writeJSON(t, w, amapWeatherResponse{
				Status: "1",
				Info:   "OK",
				Forecasts: []amapWeatherForecast{{
					Casts: []amapWeatherCast{{DayWeather: "晴", NightWeather: "晴", DayTemp: "25", NightTemp: "18"}},
				}},
			})
		case "/direction/walking":
			writeJSON(t, w, amapRouteResponse{
				Status: "1",
				Info:   "OK",
				Route:  amapRouteBody{Paths: []amapRoutePath{{Distance: "1500", Duration: "900"}}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newAMapClient(server.URL, "test-key", defaultExternalAPITimeout)

	pois, err := (RealPOITool{client: client}).Run(context.Background(), POIToolInput{
		City:      "杭州",
		Interests: []string{"自然风光"},
	})
	if err != nil {
		t.Fatalf("RealPOITool returned error: %v", err)
	}
	if len(pois) != 2 || pois[0].Location == "" {
		t.Fatalf("unexpected pois: %#v", pois)
	}

	weather, err := (RealWeatherTool{client: client, geocoder: client}).Run(context.Background(), WeatherToolInput{City: "杭州", Days: 2})
	if err != nil {
		t.Fatalf("RealWeatherTool returned error: %v", err)
	}
	if len(weather) != 2 || weather[0].Condition == "" {
		t.Fatalf("unexpected weather: %#v", weather)
	}

	routes, err := (RealRouteTool{client: client}).Run(context.Background(), RouteToolInput{POIs: pois, Mode: "walk"})
	if err != nil {
		t.Fatalf("RealRouteTool returned error: %v", err)
	}
	if len(routes) != 1 || routes[0].DistanceMeters != 1500 || routes[0].DurationMinutes != 15 {
		t.Fatalf("unexpected routes: %#v", routes)
	}
}

func TestRealToolFallbackWarningIsAddedToPlan(t *testing.T) {
	t.Setenv("TRAVEL_AGENT_TOOL_MODE", "real")
	t.Setenv("TRAVEL_AGENT_AMAP_API_KEY", "")

	planner, err := NewEinoTravelPlanner()
	if err != nil {
		t.Fatalf("NewEinoTravelPlanner returned error: %v", err)
	}
	plan, err := planner.Plan(context.Background(), testRequest("杭州", 2, 2000))
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	found := false
	for _, warning := range plan.Warnings {
		if strings.Contains(warning, "fallback used") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected fallback warning, got %#v", plan.Warnings)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json: %v", err)
	}
}
