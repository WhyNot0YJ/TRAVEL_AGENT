package eino

import (
	"context"
	"errors"
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
	req = domain.NormalizeTravelBrief(req)
	if req.DepartureCity == "" {
		return TravelPlanningState{}, fmt.Errorf("departure_city is required")
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
	if len(req.Interests) == 0 {
		return TravelPlanningState{}, fmt.Errorf("interests is required")
	}
	if req.Travelers <= 0 {
		return TravelPlanningState{}, fmt.Errorf("travelers is required")
	}
	routeMode := req.TransportMode
	if routeMode == domain.DefaultTransportMode {
		routeMode = "步行 + 打车"
	}
	interests := req.Interests
	if len(req.MustVisit) > 0 {
		interests = mergeUniqueStrings(interests, req.MustVisit)
	}

	state := TravelPlanningState{
		Request:               req,
		NormalizedDestination: strings.TrimSpace(req.DestinationCity),
		NormalizedDays:        req.Days,
		NormalizedBudget:      req.Budget,
		Interests:             interests,
		TransportMode:         routeMode,
		Pace:                  req.Pace,
		Warnings:              []string{},
		Trace:                 []TraceEvent{},
	}
	return appendTrace(ctx, state, "ParseTravelRequestNode", "request normalized", started, true), nil
}

func searchPOIsToolNode(tool POITool) func(context.Context, TravelPlanningState) (TravelPlanningState, error) {
	return func(ctx context.Context, state TravelPlanningState) (TravelPlanningState, error) {
		started := time.Now()
		pois, err := tool.Run(ctx, POIToolInput{
			City:      state.NormalizedDestination,
			Interests: state.Interests,
		})
		if err != nil {
			if isFallbackError(err) && len(pois) > 0 {
				state.Warnings = append(state.Warnings, err.Error())
				state.POIs = pois
				return appendTrace(ctx, state, "SearchPOIsToolNode", fmt.Sprintf("loaded %d fallback pois", len(pois)), started, true), nil
			}
			state = appendTrace(ctx, state, "SearchPOIsToolNode", err.Error(), started, false)
			return state, err
		}
		state.POIs = pois
		return appendTrace(ctx, state, "SearchPOIsToolNode", fmt.Sprintf("loaded %d pois", len(pois)), started, true), nil
	}
}

func getWeatherToolNode(tool WeatherTool) func(context.Context, TravelPlanningState) (TravelPlanningState, error) {
	return func(ctx context.Context, state TravelPlanningState) (TravelPlanningState, error) {
		started := time.Now()
		weather, err := tool.Run(ctx, WeatherToolInput{
			City: state.NormalizedDestination,
			Days: state.NormalizedDays,
		})
		if err != nil {
			if isFallbackError(err) && len(weather) > 0 {
				state.Warnings = append(state.Warnings, err.Error())
				state.Weather = weather
				return appendTrace(ctx, state, "GetWeatherToolNode", fmt.Sprintf("loaded %d fallback weather records", len(weather)), started, true), nil
			}
			state = appendTrace(ctx, state, "GetWeatherToolNode", err.Error(), started, false)
			return state, err
		}
		state.Weather = weather
		for _, item := range weather {
			if item.Condition == "rainy" {
				state.Warnings = append(state.Warnings, fmt.Sprintf("day %d weather is rainy; keep indoor backup options", item.Day))
			}
		}
		return appendTrace(ctx, state, "GetWeatherToolNode", fmt.Sprintf("loaded %d weather records", len(weather)), started, true), nil
	}
}

func computeRouteToolNode(tool RouteTool) func(context.Context, TravelPlanningState) (TravelPlanningState, error) {
	return func(ctx context.Context, state TravelPlanningState) (TravelPlanningState, error) {
		started := time.Now()
		routes, err := tool.Run(ctx, RouteToolInput{
			POIs: state.POIs,
			Mode: state.TransportMode,
		})
		if err != nil {
			if isFallbackError(err) && len(routes) > 0 {
				state.Warnings = append(state.Warnings, err.Error())
				state.Routes = routes
				return appendTrace(ctx, state, "ComputeRouteToolNode", fmt.Sprintf("computed %d fallback route segments", len(routes)), started, true), nil
			}
			state = appendTrace(ctx, state, "ComputeRouteToolNode", err.Error(), started, false)
			return state, err
		}
		state.Routes = routes
		return appendTrace(ctx, state, "ComputeRouteToolNode", fmt.Sprintf("computed %d route segments", len(routes)), started, true), nil
	}
}

func estimateBudgetToolNode(tool BudgetTool) func(context.Context, TravelPlanningState) (TravelPlanningState, error) {
	return func(ctx context.Context, state TravelPlanningState) (TravelPlanningState, error) {
		started := time.Now()
		budget, err := tool.Run(ctx, BudgetToolInput{
			Request: state.Request,
			Days:    state.NormalizedDays,
			POIs:    state.POIs,
			Routes:  state.Routes,
		})
		if err != nil {
			state = appendTrace(ctx, state, "EstimateBudgetToolNode", err.Error(), started, false)
			return state, err
		}
		state.Budget = domain.TravelBudget{
			Transport: budget.Transport,
			Food:      budget.Food,
			Hotel:     budget.Hotel,
			Ticket:    budget.Ticket,
			Total:     budget.Total,
		}
		return appendTrace(ctx, state, "EstimateBudgetToolNode", "budget estimated", started, true), nil
	}
}

func isFallbackError(err error) bool {
	var fallbackErr *ToolFallbackError
	return errors.As(err, &fallbackErr)
}

func optimizeItineraryNode(ctx context.Context, state TravelPlanningState) (TravelPlanningState, error) {
	started := time.Now()
	if err := ctx.Err(); err != nil {
		return state, err
	}
	if len(state.POIs) == 0 {
		return state, fmt.Errorf("cannot optimize itinerary without pois")
	}
	state.POIs = applyPOIPreferences(state.POIs, state.Request)

	itemsPerDay := 2
	if domain.IsRelaxedPace(state.Pace) || domain.IsLowWalkingTolerance(state.Request.WalkingTolerance) {
		itemsPerDay = 2
	} else if domain.IsIntensivePace(state.Pace) && !domain.IsLowWalkingTolerance(state.Request.WalkingTolerance) {
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
				Reason:          itineraryReason(state, poi),
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
	return appendTrace(ctx, state, "OptimizeItineraryNode", fmt.Sprintf("built %d itinerary days", len(days)), started, true), nil
}

func validateRouteFeasibilityNode(ctx context.Context, state TravelPlanningState) (TravelPlanningState, error) {
	started := time.Now()
	if err := ctx.Err(); err != nil {
		return state, err
	}
	result := validateRouteFeasibility(state)
	state.RouteValidation = result
	state.Warnings = append(state.Warnings, result.Warnings...)
	return appendTrace(ctx, state, "ValidateRouteFeasibilityNode", fmt.Sprintf("route feasibility score=%d checks=%d", result.Score, len(result.Checks)), started, true), nil
}

func validateRouteFeasibility(state TravelPlanningState) RouteValidationResult {
	result := RouteValidationResult{Score: 100}
	addCheck := func(name string, passed bool, message string, penalty int) {
		result.Checks = append(result.Checks, RouteValidationCheck{Name: name, Passed: passed, Message: message})
		if !passed {
			result.Score = maxInt(result.Score-penalty, 0)
			warning := fmt.Sprintf("route feasibility: check=%s score=%d message=%s", name, result.Score, message)
			result.Warnings = append(result.Warnings, warning)
		}
	}

	minItems, maxItems := paceItemRange(state.Pace)
	paceOK := true
	for _, day := range state.Itinerary {
		count := len(day.Items)
		if count < minItems || count > maxItems {
			paceOK = false
			break
		}
	}
	addCheck("daily_pace", paceOK, fmt.Sprintf("daily item count should match %s pace (%d-%d items)", fallbackValue(state.Pace, "balanced"), minItems, maxItems), 12)

	longRoute := false
	longest := 0
	for _, route := range state.Routes {
		if route.DurationMinutes > longest {
			longest = route.DurationMinutes
		}
		if route.DurationMinutes > 90 {
			longRoute = true
		}
	}
	addCheck("route_duration", !longRoute, fmt.Sprintf("longest adjacent route is %d minutes", longest), 15)

	missingCoords := false
	for _, poi := range state.POIs {
		if strings.TrimSpace(poi.Location) == "" {
			missingCoords = true
			break
		}
	}
	addCheck("poi_coordinates", !missingCoords, "some POIs do not have coordinates; route duration may use mock fallback", 10)

	rainConflict := false
	for _, weather := range state.Weather {
		if !conditionContainsRain(weather.Condition) && weather.Condition != "rainy" {
			continue
		}
		if !dayHasIndoorOption(state.Itinerary, weather.Day) {
			rainConflict = true
			break
		}
	}
	addCheck("weather_backup", !rainConflict, "rainy days should include at least one indoor-friendly option", 10)

	sum := state.Budget.Transport + state.Budget.Food + state.Budget.Hotel + state.Budget.Ticket
	budgetOK := state.Budget.Total > 0 && sum >= state.Budget.Total*0.85 && sum <= state.Budget.Total*1.15
	addCheck("budget_breakdown", budgetOK, fmt.Sprintf("budget components total %.2f vs total %.2f", sum, state.Budget.Total), 10)

	duplicate := false
	for _, day := range state.Itinerary {
		seen := map[string]struct{}{}
		for _, item := range day.Items {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				duplicate = true
				break
			}
			seen[name] = struct{}{}
		}
	}
	addCheck("duplicate_poi", !duplicate, "same-day itinerary should not repeat the same POI", 12)

	if len(state.Routes) == 0 && len(state.POIs) > 1 {
		addCheck("route_data_available", false, "route data is unavailable for adjacent POIs", 10)
	} else {
		addCheck("route_data_available", true, "route data is available or not needed", 0)
	}

	return result
}

func generateTravelPlanNode(generator TravelPlanGenerator) func(context.Context, TravelPlanningState) (*domain.TravelPlan, error) {
	return func(ctx context.Context, state TravelPlanningState) (*domain.TravelPlan, error) {
		started := time.Now()
		plan, err := generator.Generate(ctx, state)
		if err != nil {
			return nil, err
		}
		plan.Warnings = mergeWarnings(state.Warnings, plan.Warnings)
		_ = appendTrace(ctx, state, "GenerateTravelPlanNode", "travel plan generated", started, true)
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

func applyPOIPreferences(pois []MockPOI, req domain.TravelRequest) []MockPOI {
	if len(pois) == 0 {
		return pois
	}
	filtered := make([]MockPOI, 0, len(pois))
	for _, poi := range pois {
		if containsAnyPreference(poi.Name+" "+poi.Category+" "+poi.Address, req.Avoid) {
			continue
		}
		filtered = append(filtered, poi)
	}
	if len(filtered) == 0 {
		filtered = append(filtered, pois...)
	}
	for i := len(req.MustVisit) - 1; i >= 0; i-- {
		name := strings.TrimSpace(req.MustVisit[i])
		if name == "" {
			continue
		}
		for idx, poi := range filtered {
			if strings.Contains(poi.Name, name) || strings.Contains(name, poi.Name) {
				filtered = append([]MockPOI{poi}, append(filtered[:idx], filtered[idx+1:]...)...)
				break
			}
		}
	}
	return filtered
}

func containsAnyPreference(text string, prefs []string) bool {
	for _, pref := range prefs {
		pref = strings.TrimSpace(pref)
		if pref != "" && strings.Contains(text, pref) {
			return true
		}
	}
	return false
}

func itineraryReason(state TravelPlanningState, poi MockPOI) string {
	parts := []string{fmt.Sprintf("matches %s preference in %s", interestsText(state.Interests), state.NormalizedDestination)}
	if containsAnyPreference(poi.Name, state.Request.MustVisit) {
		parts = append(parts, "marked as must-visit by the user")
	}
	if domain.IsLowWalkingTolerance(state.Request.WalkingTolerance) {
		parts = append(parts, "keeps walking intensity low")
	}
	if state.Request.Travelers > 0 {
		parts = append(parts, fmt.Sprintf("planned for %d travelers", state.Request.Travelers))
	}
	return strings.Join(parts, "; ")
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

func paceItemRange(pace string) (int, int) {
	switch domain.NormalizePace(pace) {
	case "轻松":
		return 1, 2
	case "紧凑":
		return 3, 4
	default:
		return 2, 3
	}
}

func dayHasIndoorOption(days []domain.TravelDay, dayNumber int) bool {
	for _, day := range days {
		if day.Day != dayNumber {
			continue
		}
		for _, item := range day.Items {
			text := strings.ToLower(item.Type + " " + item.Name + " " + item.Reason)
			for _, token := range []string{"museum", "culture", "temple", "文化", "博物馆", "寺", "美食", "food", "indoor", "室内"} {
				if strings.Contains(text, strings.ToLower(token)) {
					return true
				}
			}
		}
	}
	return false
}

func mergeWarnings(primary, secondary []string) []string {
	out := make([]string, 0, len(primary)+len(secondary))
	seen := map[string]struct{}{}
	for _, warning := range append(primary, secondary...) {
		if strings.TrimSpace(warning) == "" {
			continue
		}
		if _, ok := seen[warning]; ok {
			continue
		}
		seen[warning] = struct{}{}
		out = append(out, warning)
	}
	return out
}
