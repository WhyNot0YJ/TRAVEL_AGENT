package travel

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"travel-agent/internal/domain"
)

func RequestHash(req domain.TravelRequest) (string, error) {
	normalized := struct {
		DepartureCity   string   `json:"departure_city"`
		DestinationCity string   `json:"destination_city"`
		Days            int      `json:"days"`
		Budget          float64  `json:"budget"`
		Interests       []string `json:"interests"`
		TransportMode   string   `json:"transport_mode"`
		Pace            string   `json:"pace"`
	}{
		DepartureCity:   req.DepartureCity,
		DestinationCity: req.DestinationCity,
		Days:            req.Days,
		Budget:          req.Budget,
		Interests:       append([]string{}, req.Interests...),
		TransportMode:   req.TransportMode,
		Pace:            req.Pace,
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
