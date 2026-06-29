package eino

import (
	"context"
	"fmt"
	"math"
	"strings"

	"travel-agent/internal/domain"
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
			Cost:                     domain.UnavailableCost("per_person", "mock.poi"),
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
			Cost:            domain.UnavailableCost("per_trip", "mock.route"),
		})
	}
	return routes, nil
}

// MockBudgetTool estimates a deterministic budget without external pricing.
type MockBudgetTool struct{}

// Run summarizes only known real costs and marks unavailable budget lines.
func (t MockBudgetTool) Run(ctx context.Context, input BudgetToolInput) (domain.TravelBudget, error) {
	if err := ctx.Err(); err != nil {
		return domain.TravelBudget{}, err
	}
	if input.Request.Budget <= 0 {
		return domain.TravelBudget{}, fmt.Errorf("budget tool request budget must be positive")
	}
	if input.Days <= 0 {
		return domain.TravelBudget{}, fmt.Errorf("budget tool days must be positive")
	}

	travelers := input.Request.Travelers
	if travelers <= 0 {
		travelers = 1
	}
	poiCosts := map[string]domain.CostInfo{}
	for _, poi := range input.POIs {
		poiCosts[poi.Name] = poi.Cost
	}

	var food, ticket, transport float64
	foodKnown, ticketKnown, transportKnown := false, false, false
	for _, day := range input.Itinerary {
		for _, item := range day.Items {
			cost := item.Cost
			if cost.Status == "" {
				cost = poiCosts[item.Name]
			}
			if cost.Status != domain.CostAvailable || cost.Amount == nil || !cost.Included {
				continue
			}
			amount := budgetAmount(cost, travelers)
			switch {
			case isFoodCategory(item.Type):
				food += amount
				foodKnown = true
			case isTicketCategory(item.Type):
				ticket += amount
				ticketKnown = true
			}
		}
	}

	for _, route := range input.Routes {
		if route.Cost.Status != domain.CostAvailable || route.Cost.Amount == nil || !route.Cost.Included {
			continue
		}
		transport += budgetAmount(route.Cost, travelers)
		transportKnown = true
	}

	food = roundMoney(food)
	ticket = roundMoney(ticket)
	transport = roundMoney(transport)
	hotel := 0.0
	knownTotal := roundMoney(food + ticket + transport + hotel)

	items := []domain.BudgetLine{
		budgetLine("food", "餐饮", food, foodKnown, "amap.poi.biz_ext.cost"),
		budgetLine("transport", "市内交通", transport, transportKnown, "amap.route.cost"),
		budgetLine("hotel", "住宿", hotel, false, ""),
		budgetLine("ticket", "门票", ticket, ticketKnown, "amap.poi.biz_ext.cost"),
	}
	missing := missingBudgetKeys(items)
	if needsIntercityTransport(input.Request) {
		items = append(items, domain.BudgetLine{
			Key:      "intercity_transport",
			Label:    "往返大交通",
			Amount:   nil,
			Currency: "CNY",
			Status:   domain.CostUnavailable,
			Display:  "暂无信息",
			Included: false,
		})
		missing = append(missing, "intercity_transport")
	}

	return domain.TravelBudget{
		Transport:  transport,
		Food:       food,
		Hotel:      hotel,
		Ticket:     ticket,
		Total:      knownTotal,
		KnownTotal: knownTotal,
		Complete:   len(missing) == 0,
		Currency:   "CNY",
		Items:      items,
		Missing:    missing,
	}, nil
}

func budgetAmount(cost domain.CostInfo, travelers int) float64 {
	if cost.Amount == nil {
		return 0
	}
	amount := *cost.Amount
	if cost.Unit == "per_person" && travelers > 0 {
		amount *= float64(travelers)
	}
	return amount
}

func budgetLine(key, label string, amount float64, known bool, source string) domain.BudgetLine {
	if !known {
		return domain.BudgetLine{
			Key:      key,
			Label:    label,
			Amount:   nil,
			Currency: "CNY",
			Status:   domain.CostUnavailable,
			Display:  "暂无信息",
			Included: false,
		}
	}
	value := amount
	return domain.BudgetLine{
		Key:      key,
		Label:    label,
		Amount:   &value,
		Currency: "CNY",
		Status:   domain.CostAvailable,
		Source:   source,
		Included: true,
	}
}

func missingBudgetKeys(items []domain.BudgetLine) []string {
	missing := []string{}
	for _, item := range items {
		if item.Status == domain.CostUnavailable {
			missing = append(missing, item.Key)
		}
	}
	return missing
}

func isFoodCategory(category string) bool {
	text := strings.ToLower(category)
	for _, token := range []string{"food", "餐饮", "中餐", "火锅", "料理", "餐厅", "肯德基", "咖啡", "茶"} {
		if strings.Contains(text, strings.ToLower(token)) {
			return true
		}
	}
	return false
}

func isTicketCategory(category string) bool {
	text := strings.ToLower(category)
	for _, token := range []string{"sightseeing", "landmark", "culture", "nature", "景点", "风景", "自然", "文化", "公园", "湿地", "博物馆", "寺"} {
		if strings.Contains(text, strings.ToLower(token)) {
			return true
		}
	}
	return false
}

func needsIntercityTransport(req domain.TravelRequest) bool {
	if req.DepartureCity == "" || req.DestinationCity == "" || req.DepartureCity == req.DestinationCity {
		return false
	}
	text := strings.Join(append([]string{req.TransportMode}, req.BudgetIncludes...), " ")
	for _, token := range []string{"高铁", "火车", "飞机", "往返", "大交通", "跨城"} {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
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
