package agent

import (
	"context"
	"fmt"
	"math"

	"travel-agent/internal/domain"
)

// MockPlanner generates deterministic, structured plans without calling LLMs or external APIs.
type MockPlanner struct {
	spots map[string][]string
}

// NewMockPlanner returns a planner suitable for local harness development.
func NewMockPlanner() *MockPlanner {
	return &MockPlanner{spots: citySpots()}
}

// Plan creates a budget-aware itinerary with the exact number of requested days.
func (p *MockPlanner) Plan(ctx context.Context, req domain.TravelRequest) (*domain.TravelPlan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req.Days <= 0 {
		return nil, fmt.Errorf("days must be positive")
	}
	if req.Budget <= 0 {
		return nil, fmt.Errorf("budget must be positive")
	}

	spots := p.spots[req.DestinationCity]
	if len(spots) == 0 {
		spots = p.spots["unknown"]
	}

	days := make([]domain.TravelDay, 0, req.Days)
	itemBudget := math.Max(req.Budget*0.62/float64(req.Days*2), 20)
	for day := 1; day <= req.Days; day++ {
		first := spots[(day-1)*2%len(spots)]
		second := spots[((day-1)*2+1)%len(spots)]
		theme := dailyTheme(req, day)
		days = append(days, domain.TravelDay{
			Day:   day,
			Theme: theme,
			Items: []domain.TravelItem{
				{
					Time:            "09:30",
					Type:            "sightseeing",
					Name:            first,
					Address:         fmt.Sprintf("%s\u6838\u5fc3\u6e38\u89c8\u533a", req.DestinationCity),
					Reason:          fmt.Sprintf("\u5951\u5408%s\u884c\u7a0b\u4e3b\u9898\uff0c\u9002\u5408\u4f5c\u4e3a\u5f53\u5929\u91cd\u70b9\u4f53\u9a8c\u3002", theme),
					EstimatedCost:   roundMoney(itemBudget * 0.55),
					DurationMinutes: 150,
				},
				{
					Time:            "14:30",
					Type:            "local_experience",
					Name:            second,
					Address:         fmt.Sprintf("%s\u7279\u8272\u8857\u533a", req.DestinationCity),
					Reason:          fmt.Sprintf("\u7ed3\u5408%s\u504f\u597d\uff0c\u8865\u5145\u57ce\u5e02\u6587\u5316\u548c\u7f8e\u98df\u4f53\u9a8c\u3002", interestsText(req.Interests)),
					EstimatedCost:   roundMoney(itemBudget * 0.45),
					DurationMinutes: 120,
				},
			},
		})
	}

	budget := buildBudget(req)
	return &domain.TravelPlan{
		Title:   fmt.Sprintf("%s%d\u65e5\u65c5\u884c\u89c4\u5212", req.DestinationCity, req.Days),
		Summary: fmt.Sprintf("\u4ece%s\u51fa\u53d1\uff0c\u56f4\u7ed5%s\u5728%s\u5b89\u6392%d\u5929%s\u8282\u594f\u8def\u7ebf\uff0c\u9884\u7b97\u63a7\u5236\u5728%.0f\u5143\u4ee5\u5185\u3002", req.DepartureCity, interestsText(req.Interests), req.DestinationCity, req.Days, paceText(req.Pace), req.Budget),
		Days:    days,
		Budget:  budget,
		Warnings: []string{
			"MockPlanner \u4ec5\u7528\u4e8e\u8bc4\u4f30\u6846\u67b6\u8054\u8c03\uff0c\u672a\u6821\u9a8c\u771f\u5b9e\u8425\u4e1a\u65f6\u95f4\u3001\u4ea4\u901a\u8ddd\u79bb\u548c\u5b9e\u65f6\u4ef7\u683c\u3002",
		},
	}, nil
}

func buildBudget(req domain.TravelRequest) domain.TravelBudget {
	total := roundMoney(req.Budget * 0.9)
	transport := roundMoney(total * 0.25)
	food := roundMoney(total * 0.25)
	hotel := roundMoney(total * 0.30)
	ticket := roundMoney(total - transport - food - hotel)
	return domain.TravelBudget{
		Transport: transport,
		Food:      food,
		Hotel:     hotel,
		Ticket:    ticket,
		Total:     total,
	}
}

func citySpots() map[string][]string {
	return map[string][]string{
		"\u676d\u5dde": {
			"\u897f\u6e56",
			"\u7075\u9690\u5bfa",
			"\u6cb3\u574a\u8857",
			"\u9f99\u4e95\u6751",
			"\u4eac\u676d\u5927\u8fd0\u6cb3",
		},
		"\u4e0a\u6d77": {
			"\u5916\u6ee9",
			"\u8c6b\u56ed",
			"\u6b66\u5eb7\u8def",
			"\u9646\u5bb6\u5634",
			"\u5357\u4eac\u8def\u6b65\u884c\u8857",
		},
		"\u5317\u4eac": {
			"\u6545\u5bab",
			"\u5929\u5b89\u95e8",
			"\u9890\u548c\u56ed",
			"\u56fd\u5bb6\u535a\u7269\u9986",
			"\u4ec0\u5239\u6d77",
		},
		"\u5357\u4eac": {
			"\u4e2d\u5c71\u9675",
			"\u592b\u5b50\u5e99",
			"\u79e6\u6dee\u6cb3",
			"\u5357\u4eac\u535a\u7269\u9662",
			"\u8001\u95e8\u4e1c",
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
		"\u672a\u77e5\u57ce\u5e02": genericSpots(),
		"unknown":                  genericSpots(),
	}
}

func genericSpots() []string {
	return []string{
		"\u57ce\u5e02\u535a\u7269\u9986",
		"\u7279\u8272\u8857\u533a",
		"\u672c\u5730\u7f8e\u98df\u8857",
		"\u57ce\u5e02\u516c\u56ed",
		"\u5730\u6807\u5e7f\u573a",
	}
}

func dailyTheme(req domain.TravelRequest, day int) string {
	if len(req.Interests) == 0 {
		return fmt.Sprintf("\u7b2c%d\u5929\u57ce\u5e02\u63a2\u7d22", day)
	}
	return fmt.Sprintf("\u7b2c%d\u5929%s\u4f53\u9a8c", day, req.Interests[(day-1)%len(req.Interests)])
}

func interestsText(interests []string) string {
	if len(interests) == 0 {
		return "\u57ce\u5e02\u63a2\u7d22"
	}
	if len(interests) == 1 {
		return interests[0]
	}
	return interests[0] + "\u4e0e" + interests[1]
}

func paceText(pace string) string {
	switch pace {
	case "relaxed":
		return "\u8f7b\u677e"
	case "balanced":
		return "\u5747\u8861"
	case "intensive":
		return "\u7d27\u51d1"
	default:
		return "\u5747\u8861"
	}
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
