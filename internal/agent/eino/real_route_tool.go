package eino

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
	path := routePath(mode)
	routes := make([]MockRoute, 0, len(input.POIs)-1)
	for i := 0; i < len(input.POIs)-1; i++ {
		from := input.POIs[i]
		to := input.POIs[i+1]
		if from.Location == "" || to.Location == "" {
			return nil, fmt.Errorf("poi location is required for real route")
		}
		route, err := t.querySegment(ctx, path, mode, from, to)
		if err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, nil
}

func (t RealRouteTool) querySegment(ctx context.Context, path, mode string, from, to MockPOI) (MockRoute, error) {
	var resp amapRouteResponse
	if err := t.client.get(ctx, path, map[string]string{
		"origin":      from.Location,
		"destination": to.Location,
	}, &resp); err != nil {
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
	}, nil
}

func routePath(mode string) string {
	lower := strings.ToLower(mode)
	switch {
	case strings.Contains(lower, "walk"), strings.Contains(mode, "步行"):
		return "/direction/walking"
	case strings.Contains(lower, "bike"):
		return "/direction/bicycling"
	default:
		return "/direction/driving"
	}
}

type amapRouteResponse struct {
	Status string        `json:"status"`
	Info   string        `json:"info"`
	Route  amapRouteBody `json:"route"`
}

type amapRouteBody struct {
	Paths []amapRoutePath `json:"paths"`
}

type amapRoutePath struct {
	Distance string `json:"distance"`
	Duration string `json:"duration"`
}
