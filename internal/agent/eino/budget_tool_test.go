package eino

import (
	"context"
	"testing"

	"travel-agent/internal/domain"
)

func TestMockBudgetToolSummarizesOnlyAvailableCosts(t *testing.T) {
	foodCost := domain.AvailableCost(100, "per_person", "amap.poi.biz_ext.cost", true)
	ticketCost := domain.AvailableCost(20, "per_person", "amap.poi.biz_ext.cost", true)
	routeCost := domain.AvailableCost(35, "per_trip", "amap.route.taxi_cost", true)

	budget, err := (MockBudgetTool{}).Run(context.Background(), BudgetToolInput{
		Request: domain.TravelRequest{
			DepartureCity:   "上海",
			DestinationCity: "杭州",
			Days:            1,
			Budget:          1000,
			Travelers:       2,
			TransportMode:   "高铁 + 打车",
			BudgetIncludes:  []string{"住宿", "餐饮", "门票", "市内交通", "往返大交通"},
		},
		Days: 1,
		POIs: []MockPOI{
			{Name: "餐厅", Category: "餐饮服务", Cost: foodCost},
			{Name: "西湖", Category: "风景名胜", Cost: ticketCost},
			{Name: "未知店", Category: "餐饮服务", Cost: domain.UnavailableCost("per_person", "amap.poi.biz_ext.cost")},
		},
		Routes: []MockRoute{{From: "西湖", To: "餐厅", Cost: routeCost}},
		Itinerary: []domain.TravelDay{{
			Day: 1,
			Items: []domain.TravelItem{
				{Name: "餐厅", Type: "餐饮服务", Cost: foodCost},
				{Name: "西湖", Type: "风景名胜", Cost: ticketCost},
				{Name: "未知店", Type: "餐饮服务", Cost: domain.UnavailableCost("per_person", "amap.poi.biz_ext.cost")},
			},
		}},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if budget.Food != 200 || budget.Ticket != 40 || budget.Transport != 35 {
		t.Fatalf("unexpected budget components: %#v", budget)
	}
	if budget.KnownTotal != 275 || budget.Total != budget.KnownTotal {
		t.Fatalf("unexpected known total: %#v", budget)
	}
	if budget.Complete {
		t.Fatalf("expected incomplete budget: %#v", budget)
	}
	if !containsBudgetKey(budget.Missing, "hotel") || !containsBudgetKey(budget.Missing, "intercity_transport") {
		t.Fatalf("expected missing hotel and intercity transport, got %#v", budget.Missing)
	}
}

func containsBudgetKey(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
