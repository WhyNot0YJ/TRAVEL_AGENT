package eino

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

func TestAMapClientLimitsExternalConcurrency(t *testing.T) {
	var current int32
	var maxConcurrent int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inFlight := atomic.AddInt32(&current, 1)
		for {
			observed := atomic.LoadInt32(&maxConcurrent)
			if inFlight <= observed || atomic.CompareAndSwapInt32(&maxConcurrent, observed, inFlight) {
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
		atomic.AddInt32(&current, -1)
		writeJSON(t, w, map[string]string{"status": "1", "info": "OK"})
	}))
	defer server.Close()

	client := newAMapClient(server.URL, "test-key", time.Second, newExternalAPILimiter(2, 1000))
	var wg sync.WaitGroup
	errs := make(chan error, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out map[string]string
			errs <- client.get(context.Background(), "/limited", nil, &out)
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("client.get returned error: %v", err)
		}
	}
	if got := atomic.LoadInt32(&maxConcurrent); got > 2 {
		t.Fatalf("expected at most 2 concurrent external calls, got %d", got)
	}
}

func TestLoadToolConfigExternalAPILimits(t *testing.T) {
	t.Setenv("TRAVEL_AGENT_EXTERNAL_API_CONCURRENCY", "3")
	t.Setenv("TRAVEL_AGENT_EXTERNAL_API_QPS", "4")

	cfg := loadToolConfigFromEnv()
	if cfg.ExternalAPIConcurrency != 3 {
		t.Fatalf("expected external API concurrency 3, got %d", cfg.ExternalAPIConcurrency)
	}
	if cfg.ExternalAPIQPS != 4 {
		t.Fatalf("expected external API QPS 4, got %d", cfg.ExternalAPIQPS)
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
		if strings.Contains(warning, "tool fallback:") &&
			strings.Contains(warning, "tool=poi") &&
			strings.Contains(warning, "category=configuration") &&
			strings.Contains(warning, "mock_fallback=true") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected fallback warning, got %#v", plan.Warnings)
	}
}

func TestRealToolFallbackWarningsAreClassified(t *testing.T) {
	tests := []struct {
		name             string
		handler          http.HandlerFunc
		run              func(*amapClient) error
		expectedCategory string
	}{
		{
			name: "http status",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, `{"status":"0","info":"upstream unavailable"}`, http.StatusBadGateway)
			},
			run: func(client *amapClient) error {
				_, err := (fallbackPOITool{primary: RealPOITool{client: client}, fallback: MockPOITool{}, toolName: "poi"}).Run(context.Background(), POIToolInput{City: "杭州"})
				return err
			},
			expectedCategory: "provider_error",
		},
		{
			name: "provider status",
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeJSON(t, w, amapPOIResponse{Status: "0", Info: "INVALID_USER_KEY"})
			},
			run: func(client *amapClient) error {
				_, err := (fallbackPOITool{primary: RealPOITool{client: client}, fallback: MockPOITool{}, toolName: "poi"}).Run(context.Background(), POIToolInput{City: "杭州"})
				return err
			},
			expectedCategory: "provider_error",
		},
		{
			name: "invalid json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":`))
			},
			run: func(client *amapClient) error {
				_, err := (fallbackPOITool{primary: RealPOITool{client: client}, fallback: MockPOITool{}, toolName: "poi"}).Run(context.Background(), POIToolInput{City: "杭州"})
				return err
			},
			expectedCategory: "invalid_json",
		},
		{
			name: "missing poi fields",
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeJSON(t, w, amapPOIResponse{Status: "1", Info: "OK", POIs: []amapPOIItem{{Address: "missing name"}}})
			},
			run: func(client *amapClient) error {
				_, err := (fallbackPOITool{primary: RealPOITool{client: client}, fallback: MockPOITool{}, toolName: "poi"}).Run(context.Background(), POIToolInput{City: "杭州"})
				return err
			},
			expectedCategory: "missing_field",
		},
		{
			name: "missing route coordinates",
			handler: func(w http.ResponseWriter, r *http.Request) {
				t.Fatalf("route API should not be called when coordinates are missing")
			},
			run: func(client *amapClient) error {
				_, err := (fallbackRouteTool{primary: RealRouteTool{client: client}, fallback: MockRouteTool{}, toolName: "route"}).Run(context.Background(), RouteToolInput{
					POIs: []MockPOI{
						{Name: "A"},
						{Name: "B", Location: "120.1,30.2"},
					},
				})
				return err
			},
			expectedCategory: "missing_field",
		},
		{
			name: "weather city coding",
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeJSON(t, w, amapGeocodeResponse{Status: "1", Info: "OK", Geocodes: []amapGeocodeItem{{}}})
			},
			run: func(client *amapClient) error {
				_, err := (fallbackWeatherTool{primary: RealWeatherTool{client: client, geocoder: client}, fallback: MockWeatherTool{}, toolName: "weather"}).Run(context.Background(), WeatherToolInput{City: "杭州", Days: 2})
				return err
			},
			expectedCategory: "missing_field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			err := tt.run(newAMapClient(server.URL, "test-key", defaultExternalAPITimeout))
			var fallbackErr *ToolFallbackError
			if !errors.As(err, &fallbackErr) {
				t.Fatalf("expected ToolFallbackError, got %T %v", err, err)
			}
			if fallbackErr.Category != tt.expectedCategory {
				t.Fatalf("category mismatch: got %q want %q from %q", fallbackErr.Category, tt.expectedCategory, fallbackErr.Error())
			}
			if !strings.Contains(fallbackErr.Error(), "provider=amap") || !strings.Contains(fallbackErr.Error(), "mock_fallback=true") {
				t.Fatalf("fallback warning is not structured enough: %q", fallbackErr.Error())
			}
		})
	}
}

func TestRealToolTimeoutFallbackIsClassified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		writeJSON(t, w, amapPOIResponse{Status: "1", Info: "OK"})
	}))
	defer server.Close()

	client := newAMapClient(server.URL, "test-key", time.Millisecond)
	_, err := (fallbackPOITool{primary: RealPOITool{client: client}, fallback: MockPOITool{}, toolName: "poi"}).Run(context.Background(), POIToolInput{City: "杭州"})
	var fallbackErr *ToolFallbackError
	if !errors.As(err, &fallbackErr) {
		t.Fatalf("expected timeout ToolFallbackError, got %T %v", err, err)
	}
	if fallbackErr.Category != "timeout" {
		t.Fatalf("expected timeout category, got %q from %q", fallbackErr.Category, fallbackErr.Error())
	}
}

func TestMockToolModeDoesNotCallExternalServer(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer server.Close()

	t.Setenv("TRAVEL_AGENT_TOOL_MODE", "mock")
	t.Setenv("TRAVEL_AGENT_AMAP_API_KEY", "test-key")
	t.Setenv("TRAVEL_AGENT_AMAP_BASE_URL", server.URL)

	tools := defaultToolSet()
	pois, err := tools.POI.Run(context.Background(), POIToolInput{City: "杭州"})
	if err != nil {
		t.Fatalf("mock poi returned error: %v", err)
	}
	if _, err := tools.Weather.Run(context.Background(), WeatherToolInput{City: "杭州", Days: 2}); err != nil {
		t.Fatalf("mock weather returned error: %v", err)
	}
	if _, err := tools.Route.Run(context.Background(), RouteToolInput{POIs: pois}); err != nil {
		t.Fatalf("mock route returned error: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Fatalf("mock mode called external server %d times", got)
	}
}

func TestRealModeMissingAPIKeyFallsBackWithConfigurationCategory(t *testing.T) {
	t.Setenv("TRAVEL_AGENT_TOOL_MODE", "real")
	t.Setenv("TRAVEL_AGENT_AMAP_API_KEY", "")

	tools := defaultToolSet()
	_, err := tools.POI.Run(context.Background(), POIToolInput{City: "杭州"})
	var fallbackErr *ToolFallbackError
	if !errors.As(err, &fallbackErr) {
		t.Fatalf("expected configuration ToolFallbackError, got %T %v", err, err)
	}
	if fallbackErr.Stage != "configuration" || fallbackErr.Category != "configuration" {
		t.Fatalf("unexpected fallback classification: %#v", fallbackErr)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json: %v", err)
	}
}
