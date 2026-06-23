package eino

import (
	"context"
	"fmt"
	"math"
)

// MockPOITool returns stable POIs without calling external map APIs.
type MockPOITool struct{}

// Run executes the mock POI lookup.
func (t MockPOITool) Run(ctx context.Context, input POIToolInput) ([]MockPOI, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.City == "" {
		return nil, fmt.Errorf("poi tool city is required")
	}
	names := cityPOINames(input.City)
	pois := make([]MockPOI, 0, len(names))
	for i, name := range names {
		pois = append(pois, MockPOI{
			Name:                     name,
			City:                     input.City,
			Category:                 poiCategory(i),
			Address:                  fmt.Sprintf("%s POI %d", input.City, i+1),
			SuggestedDurationMinutes: 90 + (i%3)*30,
			EstimatedCost:            float64(30 + (i%4)*20),
		})
	}
	return pois, nil
}

// MockWeatherTool returns deterministic weather data without external APIs.
type MockWeatherTool struct{}

// Run executes the mock weather lookup.
func (t MockWeatherTool) Run(ctx context.Context, input WeatherToolInput) ([]MockWeather, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.Days <= 0 {
		return nil, fmt.Errorf("weather tool days must be positive")
	}
	conditions := []string{"sunny", "cloudy", "rainy", "cloudy"}
	weather := make([]MockWeather, 0, input.Days)
	for day := 1; day <= input.Days; day++ {
		condition := conditions[(day+len(input.City))%len(conditions)]
		suggestion := "outdoor friendly"
		if condition == "rainy" {
			suggestion = "keep indoor backup plan"
		}
		weather = append(weather, MockWeather{
			Day:         day,
			Condition:   condition,
			Temperature: fmt.Sprintf("%d-%dC", 18+day%5, 25+day%6),
			Suggestion:  suggestion,
		})
	}
	return weather, nil
}

// MockRouteTool returns deterministic route segments without route APIs.
type MockRouteTool struct{}

// Run executes mock route computation between adjacent POIs.
func (t MockRouteTool) Run(ctx context.Context, input RouteToolInput) ([]MockRoute, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(input.POIs) == 0 {
		return nil, fmt.Errorf("route tool requires at least one poi")
	}
	mode := input.Mode
	if mode == "" {
		mode = "walk_taxi"
	}
	routes := make([]MockRoute, 0, maxInt(len(input.POIs)-1, 0))
	for i := 0; i < len(input.POIs)-1; i++ {
		routes = append(routes, MockRoute{
			From:            input.POIs[i].Name,
			To:              input.POIs[i+1].Name,
			DurationMinutes: 15 + (i%4)*5,
			DistanceMeters:  1200 + i*450,
			Mode:            mode,
		})
	}
	return routes, nil
}

// MockBudgetTool estimates a deterministic budget without external pricing.
type MockBudgetTool struct{}

// Run executes mock budget estimation and keeps total within the requested limit.
func (t MockBudgetTool) Run(ctx context.Context, input BudgetToolInput) (domainBudget, error) {
	if err := ctx.Err(); err != nil {
		return domainBudget{}, err
	}
	if input.Request.Budget <= 0 {
		return domainBudget{}, fmt.Errorf("budget tool request budget must be positive")
	}
	if input.Days <= 0 {
		return domainBudget{}, fmt.Errorf("budget tool days must be positive")
	}
	total := roundMoney(input.Request.Budget * 0.92)
	transport := roundMoney(total * 0.24)
	food := roundMoney(total * 0.26)
	hotel := roundMoney(total * 0.30)
	ticket := roundMoney(math.Max(total-transport-food-hotel, 0))
	return domainBudget{
		Transport: transport,
		Food:      food,
		Hotel:     hotel,
		Ticket:    ticket,
		Total:     total,
	}, nil
}

type domainBudget struct {
	Transport float64 `json:"transport"`
	Food      float64 `json:"food"`
	Hotel     float64 `json:"hotel"`
	Ticket    float64 `json:"ticket"`
	Total     float64 `json:"total"`
}

func cityPOINames(city string) []string {
	if names, ok := knownCityPOIs()[city]; ok {
		return names
	}
	return knownCityPOIs()["unknown"]
}

func knownCityPOIs() map[string][]string {
	return map[string][]string{
		"\u676d\u5dde": {
			"\u897f\u6e56",
			"\u7075\u9690\u5bfa",
			"\u6cb3\u574a\u8857",
			"\u9f99\u4e95\u6751",
			"\u4eac\u676d\u5927\u8fd0\u6cb3",
		},
		"\u5357\u4eac": {
			"\u4e2d\u5c71\u9675",
			"\u592b\u5b50\u5e99",
			"\u79e6\u6dee\u6cb3",
			"\u5357\u4eac\u535a\u7269\u9662",
			"\u8001\u95e8\u4e1c",
		},
		"\u5317\u4eac": {
			"\u6545\u5bab",
			"\u5929\u5b89\u95e8",
			"\u9890\u548c\u56ed",
			"\u56fd\u5bb6\u535a\u7269\u9986",
			"\u4ec0\u5239\u6d77",
		},
		"\u6210\u90fd": {
			"\u5bbd\u7a84\u5df7\u5b50",
			"\u6b66\u4faf\u7960",
			"\u9526\u91cc",
			"\u4eba\u6c11\u516c\u56ed",
			"\u6625\u7199\u8def",
		},
		"\u5e7f\u5dde": {
			"\u6c99\u9762",
			"\u6c38\u5e86\u574a",
			"\u9648\u5bb6\u7960",
			"\u5317\u4eac\u8def",
			"\u73e0\u6c5f\u591c\u6e38",
		},
		"\u897f\u5b89": {
			"\u5175\u9a6c\u4fd1",
			"\u5927\u96c1\u5854",
			"\u5927\u5510\u4e0d\u591c\u57ce",
			"\u897f\u5b89\u57ce\u5899",
			"\u9655\u897f\u5386\u53f2\u535a\u7269\u9986",
		},
		"\u82cf\u5dde": {
			"\u62d9\u653f\u56ed",
			"\u5e73\u6c5f\u8def",
			"\u82cf\u5dde\u535a\u7269\u9986",
			"\u4e03\u91cc\u5c71\u5858",
			"\u7559\u56ed",
		},
		"\u672a\u77e5\u57ce\u5e02": {
			"\u57ce\u5e02\u535a\u7269\u9986",
			"\u7279\u8272\u8857\u533a",
			"\u672c\u5730\u7f8e\u98df\u8857",
			"\u57ce\u5e02\u516c\u56ed",
			"\u5730\u6807\u5e7f\u573a",
		},
		"unknown": {
			"\u57ce\u5e02\u535a\u7269\u9986",
			"\u7279\u8272\u8857\u533a",
			"\u672c\u5730\u7f8e\u98df\u8857",
			"\u57ce\u5e02\u516c\u56ed",
			"\u5730\u6807\u5e7f\u573a",
		},
	}
}

func poiCategory(index int) string {
	categories := []string{"landmark", "culture", "food", "nature", "citywalk"}
	return categories[index%len(categories)]
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
