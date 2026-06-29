package eino

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"travel-agent/internal/domain"
)

type RealRouteTool struct {
	client *amapClient
}

func (t RealRouteTool) Run(ctx context.Context, input RouteToolInput) ([]MockRoute, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(input.POIs) == 0 {
		return nil, fmt.Errorf("route tool requires at least one poi")
	}
	if len(input.POIs) == 1 {
		return []MockRoute{}, nil
	}

	mode := input.Mode
	if mode == "" {
		mode = "walk_taxi"
	}
	routes := make([]MockRoute, 0, len(input.POIs)-1)
	for i := 0; i < len(input.POIs)-1; i++ {
		from := input.POIs[i]
		to := input.POIs[i+1]
		if from.Location == "" || to.Location == "" {
			return nil, fmt.Errorf("poi location is required for real route")
		}
		route, err := t.querySegment(ctx, mode, from, to)
		if err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, nil
}

func (t RealRouteTool) querySegment(ctx context.Context, mode string, from, to MockPOI) (MockRoute, error) {
	path := routePath(mode)
	if path == "/direction/transit/integrated" {
		return t.queryTransitSegment(ctx, mode, from, to)
	}
	var resp amapRouteResponse
	query := map[string]string{
		"origin":      from.Location,
		"destination": to.Location,
	}
	if path == "/direction/driving" {
		query["extensions"] = "all"
	}
	if err := t.client.get(ctx, path, query, &resp); err != nil {
		return MockRoute{}, err
	}
	if len(resp.Route.Paths) == 0 {
		return MockRoute{}, fmt.Errorf("amap route response is empty")
	}
	first := resp.Route.Paths[0]
	duration, _ := strconv.Atoi(first.Duration)
	distance, _ := strconv.Atoi(first.Distance)
	if duration <= 0 && distance <= 0 {
		return MockRoute{}, fmt.Errorf("amap route response missing distance and duration")
	}
	return MockRoute{
		From:            from.Name,
		To:              to.Name,
		DurationMinutes: maxInt(duration/60, 1),
		DistanceMeters:  distance,
		Mode:            mode,
		Cost:            routeCost(mode, path, resp.Route.TaxiCost, first.Tolls),
	}, nil
}

func (t RealRouteTool) queryTransitSegment(ctx context.Context, mode string, from, to MockPOI) (MockRoute, error) {
	var resp amapRouteResponse
	if err := t.client.get(ctx, "/direction/transit/integrated", map[string]string{
		"origin":      from.Location,
		"destination": to.Location,
		"city":        fallbackValue(from.City, to.City),
		"cityd":       to.City,
	}, &resp); err != nil {
		return MockRoute{}, err
	}
	if len(resp.Route.Transits) == 0 {
		return MockRoute{}, fmt.Errorf("amap transit response is empty")
	}
	first := resp.Route.Transits[0]
	duration, _ := strconv.Atoi(first.Duration)
	distance, _ := strconv.Atoi(first.WalkingDistance)
	if duration <= 0 && distance <= 0 {
		return MockRoute{}, fmt.Errorf("amap transit response missing distance and duration")
	}
	return MockRoute{
		From:            from.Name,
		To:              to.Name,
		DurationMinutes: maxInt(duration/60, 1),
		DistanceMeters:  distance,
		Mode:            mode,
		Cost:            transitCost(first.Cost),
	}, nil
}

func routePath(mode string) string {
	lower := strings.ToLower(mode)
	switch {
	case strings.Contains(lower, "transit"), strings.Contains(mode, "公交"), strings.Contains(mode, "地铁"), strings.Contains(mode, "公共交通"):
		return "/direction/transit/integrated"
	case strings.Contains(lower, "taxi"), strings.Contains(lower, "drive"), strings.Contains(lower, "car"), strings.Contains(mode, "打车"), strings.Contains(mode, "自驾"):
		return "/direction/driving"
	case strings.Contains(lower, "bike"):
		return "/direction/bicycling"
	case strings.Contains(lower, "walk"), strings.Contains(mode, "步行"):
		return "/direction/walking"
	default:
		return "/direction/driving"
	}
}

func routeCost(mode, path string, taxiCost, tolls any) domain.CostInfo {
	if path == "/direction/walking" || path == "/direction/bicycling" {
		return domain.NotApplicableCost("per_trip", "amap.route.no_cost")
	}
	if strings.Contains(mode, "自驾") || strings.Contains(strings.ToLower(mode), "driving") {
		if amount, ok := parseAMapFloat(tolls); ok {
			return domain.AvailableCost(amount, "per_trip_reference", "amap.route.tolls", true)
		}
		return domain.UnavailableCost("per_trip_reference", "amap.route.tolls")
	}
	if amount, ok := parseAMapFloat(taxiCost); ok {
		return domain.AvailableCost(amount, "per_trip", "amap.route.taxi_cost", true)
	}
	return domain.UnavailableCost("per_trip", "amap.route.taxi_cost")
}

func transitCost(cost any) domain.CostInfo {
	if amount, ok := parseAMapFloat(cost); ok {
		return domain.AvailableCost(amount, "per_person", "amap.route.transits.cost", true)
	}
	return domain.UnavailableCost("per_person", "amap.route.transits.cost")
}

type amapRouteResponse struct {
	Status string        `json:"status"`
	Info   string        `json:"info"`
	Route  amapRouteBody `json:"route"`
}

type amapRouteBody struct {
	Paths    []amapRoutePath    `json:"paths"`
	TaxiCost any                `json:"taxi_cost"`
	Transits []amapTransitRoute `json:"transits"`
}

type amapRoutePath struct {
	Distance string `json:"distance"`
	Duration string `json:"duration"`
	Tolls    any    `json:"tolls"`
}

type amapTransitRoute struct {
	Cost            any    `json:"cost"`
	Duration        string `json:"duration"`
	WalkingDistance string `json:"walking_distance"`
}
