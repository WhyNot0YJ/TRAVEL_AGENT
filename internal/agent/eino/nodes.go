package eino

import (
	"context"
	"fmt"
	"strings"
	"time"

	"travel-agent/internal/domain"
)

func parseTravelRequestNode(ctx context.Context, req domain.TravelRequest) (TravelPlanningState, error) {
	started := time.Now()
	if err := ctx.Err(); err != nil {
		return TravelPlanningState{}, err
	}
	if req.DestinationCity == "" {
		return TravelPlanningState{}, fmt.Errorf("destination_city is required")
	}
	if req.Days <= 0 {
		return TravelPlanningState{}, fmt.Errorf("days must be positive")
	}
	if req.Budget <= 0 {
		return TravelPlanningState{}, fmt.Errorf("budget must be positive")
	}
	if req.Pace == "" {
		req.Pace = "balanced"
	}
	if req.TransportMode == "" {
		req.TransportMode = "walk_taxi"
	}
	interests := req.Interests
	if len(interests) == 0 {
		interests = []string{"\u57ce\u5e02\u63a2\u7d22"}
	}

	state := TravelPlanningState{
		Request:               req,
		NormalizedDestination: strings.TrimSpace(req.DestinationCity),
		NormalizedDays:        req.Days,
		NormalizedBudget:      req.Budget,
		Interests:             interests,
		TransportMode:         req.TransportMode,
		Pace:                  req.Pace,
		Warnings:              []string{},
		Trace:                 []TraceEvent{},
	}
	return appendTrace(state, "ParseTravelRequestNode", "request normalized", started, true), nil
}

func searchPOIsToolNode(tool MockPOITool) func(context.Context, TravelPlanningState) (TravelPlanningState, error) {
	return func(ctx context.Context, state TravelPlanningState) (TravelPlanningState, error) {
		started := time.Now()
		pois, err := tool.Run(ctx, POIToolInput{
			City:      state.NormalizedDestination,
			Interests: state.Interests,
		})
		if err != nil {
			state = appendTrace(state, "SearchPOIsToolNode", err.Error(), started, false)
			return state, err
		}
		state.POIs = pois
		return appendTrace(state, "SearchPOIsToolNode", fmt.Sprintf("loaded %d pois", len(pois)), started, true), nil
	}
}

func getWeatherToolNode(tool MockWeatherTool) func(context.Context, TravelPlanningState) (TravelPlanningState, error) {
	return func(ctx context.Context, state TravelPlanningState) (TravelPlanningState, error) {
		started := time.Now()
		weather, err := tool.Run(ctx, WeatherToolInput{
			City: state.NormalizedDestination,
			Days: state.NormalizedDays,
		})
		if err != nil {
			state = appendTrace(state, "GetWeatherToolNode", err.Error(), started, false)
			return state, err
		}
		state.Weather = weather
		for _, item := range weather {
			if item.Condition == "rainy" {
				state.Warnings = append(state.Warnings, fmt.Sprintf("day %d weather is rainy; keep indoor backup options", item.Day))
			}
		}
		return appendTrace(state, "GetWeatherToolNode", fmt.Sprintf("loaded %d weather records", len(weather)), started, true), nil
	}
}

func computeRouteToolNode(tool MockRouteTool) func(context.Context, TravelPlanningState) (TravelPlanningState, error) {
	return func(ctx context.Context, state TravelPlanningState) (TravelPlanningState, error) {
		started := time.Now()
		routes, err := tool.Run(ctx, RouteToolInput{
			POIs: state.POIs,
			Mode: state.TransportMode,
		})
		if err != nil {
			state = appendTrace(state, "ComputeRouteToolNode", err.Error(), started, false)
			return state, err
		}
		state.Routes = routes
		return appendTrace(state, "ComputeRouteToolNode", fmt.Sprintf("computed %d route segments", len(routes)), started, true), nil
	}
}

func estimateBudgetToolNode(tool MockBudgetTool) func(context.Context, TravelPlanningState) (TravelPlanningState, error) {
	return func(ctx context.Context, state TravelPlanningState) (TravelPlanningState, error) {
		started := time.Now()
		budget, err := tool.Run(ctx, BudgetToolInput{
			Request: state.Request,
			Days:    state.NormalizedDays,
			POIs:    state.POIs,
			Routes:  state.Routes,
		})
		if err != nil {
			state = appendTrace(state, "EstimateBudgetToolNode", err.Error(), started, false)
			return state, err
		}
		state.Budget = domain.TravelBudget{
			Transport: budget.Transport,
			Food:      budget.Food,
			Hotel:     budget.Hotel,
			Ticket:    budget.Ticket,
			Total:     budget.Total,
		}
		return appendTrace(state, "EstimateBudgetToolNode", "budget estimated", started, true), nil
	}
}

func optimizeItineraryNode(ctx context.Context, state TravelPlanningState) (TravelPlanningState, error) {
	started := time.Now()
	if err := ctx.Err(); err != nil {
		return state, err
	}
	if len(state.POIs) == 0 {
		return state, fmt.Errorf("cannot optimize itinerary without pois")
	}

	itemsPerDay := 2
	if state.Pace == "intensive" {
		itemsPerDay = 3
	}
	days := make([]domain.TravelDay, 0, state.NormalizedDays)
	for day := 1; day <= state.NormalizedDays; day++ {
		items := make([]domain.TravelItem, 0, itemsPerDay)
		for slot := 0; slot < itemsPerDay; slot++ {
			poi := state.POIs[((day-1)*itemsPerDay+slot)%len(state.POIs)]
			items = append(items, domain.TravelItem{
				Time:            itemTime(slot),
				Type:            poi.Category,
				Name:            poi.Name,
				Address:         poi.Address,
				Reason:          fmt.Sprintf("matches %s preference in %s", interestsText(state.Interests), state.NormalizedDestination),
				EstimatedCost:   poi.EstimatedCost,
				DurationMinutes: poi.SuggestedDurationMinutes,
			})
		}
		days = append(days, domain.TravelDay{
			Day:   day,
			Theme: fmt.Sprintf("day %d %s route in %s", day, interestsText(state.Interests), state.NormalizedDestination),
			Items: items,
		})
	}
	state.Itinerary = days
	return appendTrace(state, "OptimizeItineraryNode", fmt.Sprintf("built %d itinerary days", len(days)), started, true), nil
}

func generateTravelPlanNode(generator TravelPlanGenerator) func(context.Context, TravelPlanningState) (*domain.TravelPlan, error) {
	return func(ctx context.Context, state TravelPlanningState) (*domain.TravelPlan, error) {
		started := time.Now()
		plan, err := generator.Generate(ctx, state)
		if err != nil {
			return nil, err
		}
		_ = appendTrace(state, "GenerateTravelPlanNode", "travel plan generated", started, true)
		return plan, nil
	}
}

func generateDeterministicTravelPlan(ctx context.Context, state TravelPlanningState) (*domain.TravelPlan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(state.Itinerary) == 0 {
		return nil, fmt.Errorf("cannot generate plan without itinerary")
	}
	warnings := append([]string{}, state.Warnings...)
	warnings = append(warnings, "EinoTravelPlanner uses deterministic mock tools; no real LLM or external API was called.")
	return &domain.TravelPlan{
		Title:    fmt.Sprintf("%s%d-day Eino travel plan", state.NormalizedDestination, state.NormalizedDays),
		Summary:  fmt.Sprintf("Eino workflow planned %d days in %s from %s with budget %.0f.", state.NormalizedDays, state.NormalizedDestination, state.Request.DepartureCity, state.NormalizedBudget),
		Days:     state.Itinerary,
		Budget:   state.Budget,
		Warnings: warnings,
	}, nil
}

func validatePlanNode(ctx context.Context, plan *domain.TravelPlan) (*domain.TravelPlan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, fmt.Errorf("plan is nil")
	}
	if strings.TrimSpace(plan.Title) == "" {
		return nil, fmt.Errorf("plan title is empty")
	}
	if strings.TrimSpace(plan.Summary) == "" {
		return nil, fmt.Errorf("plan summary is empty")
	}
	if len(plan.Days) == 0 {
		return nil, fmt.Errorf("plan days is empty")
	}
	if plan.Budget.Total < 0 {
		return nil, fmt.Errorf("plan budget total is negative")
	}
	for i, day := range plan.Days {
		expected := i + 1
		if day.Day != expected {
			return nil, fmt.Errorf("day number mismatch at index %d: got %d", i, day.Day)
		}
		if len(day.Items) == 0 {
			return nil, fmt.Errorf("day %d has no items", day.Day)
		}
		for idx, item := range day.Items {
			if item.Name == "" || item.Type == "" || item.Reason == "" {
				return nil, fmt.Errorf("day %d item %d misses required fields", day.Day, idx)
			}
			if item.EstimatedCost < 0 || item.DurationMinutes < 0 {
				return nil, fmt.Errorf("day %d item %d has illegal numeric fields", day.Day, idx)
			}
		}
	}
	return plan, nil
}

func itemTime(slot int) string {
	switch slot {
	case 0:
		return "09:30"
	case 1:
		return "14:30"
	default:
		return "18:30"
	}
}

func interestsText(interests []string) string {
	if len(interests) == 0 {
		return "city exploration"
	}
	if len(interests) == 1 {
		return interests[0]
	}
	return interests[0] + " and " + interests[1]
}
