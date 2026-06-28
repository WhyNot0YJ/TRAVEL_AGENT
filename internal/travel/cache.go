package travel

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"travel-agent/internal/domain"
)

func RequestHash(req domain.TravelRequest) (string, error) {
	return RequestHashWithOptions(req, false)
}

func RequestHashWithOptions(req domain.TravelRequest, testMode bool, agentModeValues ...string) (string, error) {
	req = domain.NormalizeTravelBrief(req)
	agentMode := normalizeAgentMode("")
	if len(agentModeValues) > 0 {
		agentMode = normalizeAgentMode(agentModeValues[0])
	}
	normalized := struct {
		DepartureCity    string   `json:"departure_city"`
		DestinationCity  string   `json:"destination_city"`
		Days             int      `json:"days"`
		Budget           float64  `json:"budget"`
		Interests        []string `json:"interests"`
		Travelers        int      `json:"travelers"`
		DateRange        string   `json:"date_range"`
		TransportMode    string   `json:"transport_mode"`
		Pace             string   `json:"pace"`
		WalkingTolerance string   `json:"walking_tolerance"`
		HotelArea        string   `json:"hotel_area"`
		MustVisit        []string `json:"must_visit"`
		Avoid            []string `json:"avoid"`
		TravelerType     string   `json:"traveler_type"`
		BudgetType       string   `json:"budget_type"`
		BudgetIncludes   []string `json:"budget_includes"`
		TestMode         bool     `json:"test_mode"`
		AgentMode        string   `json:"agent_mode"`
	}{
		DepartureCity:    req.DepartureCity,
		DestinationCity:  req.DestinationCity,
		Days:             req.Days,
		Budget:           req.Budget,
		Interests:        sortedCopy(req.Interests),
		Travelers:        req.Travelers,
		DateRange:        req.DateRange,
		TransportMode:    req.TransportMode,
		Pace:             req.Pace,
		WalkingTolerance: req.WalkingTolerance,
		HotelArea:        req.HotelArea,
		MustVisit:        sortedCopy(req.MustVisit),
		Avoid:            sortedCopy(req.Avoid),
		TravelerType:     req.TravelerType,
		BudgetType:       req.BudgetType,
		BudgetIncludes:   sortedCopy(req.BudgetIncludes),
		TestMode:         testMode,
		AgentMode:        agentMode,
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func sortedCopy(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	return out
}
