package travel

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"travel-agent/internal/domain"
)

func RequestHash(req domain.TravelRequest) (string, error) {
	return RequestHashWithOptions(req, false)
}

func RequestHashWithOptions(req domain.TravelRequest, testMode bool, agentModeValues ...string) (string, error) {
	agentMode := normalizeAgentMode("")
	if len(agentModeValues) > 0 {
		agentMode = normalizeAgentMode(agentModeValues[0])
	}
	normalized := struct {
		DepartureCity   string   `json:"departure_city"`
		DestinationCity string   `json:"destination_city"`
		Days            int      `json:"days"`
		Budget          float64  `json:"budget"`
		Interests       []string `json:"interests"`
		TransportMode   string   `json:"transport_mode"`
		Pace            string   `json:"pace"`
		TestMode        bool     `json:"test_mode"`
		AgentMode       string   `json:"agent_mode"`
	}{
		DepartureCity:   req.DepartureCity,
		DestinationCity: req.DestinationCity,
		Days:            req.Days,
		Budget:          req.Budget,
		Interests:       append([]string{}, req.Interests...),
		TransportMode:   req.TransportMode,
		Pace:            req.Pace,
		TestMode:        testMode,
		AgentMode:       agentMode,
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
