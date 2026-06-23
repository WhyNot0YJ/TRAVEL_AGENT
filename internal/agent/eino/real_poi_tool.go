package eino

import (
	"context"
	"fmt"
)

type RealPOITool struct {
	client *amapClient
}

func (t RealPOITool) Run(ctx context.Context, input POIToolInput) ([]MockPOI, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.City == "" {
		return nil, fmt.Errorf("poi city is required")
	}
	keywords := "景点"
	if len(input.Interests) > 0 {
		keywords = input.Interests[0]
	}

	var resp amapPOIResponse
	if err := t.client.get(ctx, "/place/text", map[string]string{
		"city":       input.City,
		"citylimit":  "true",
		"keywords":   keywords,
		"extensions": "base",
		"offset":     "10",
		"page":       "1",
	}, &resp); err != nil {
		return nil, err
	}
	if len(resp.POIs) == 0 {
		return nil, fmt.Errorf("amap poi response is empty")
	}

	pois := make([]MockPOI, 0, len(resp.POIs))
	for i, poi := range resp.POIs {
		if poi.Name == "" {
			continue
		}
		category := poi.Type
		if category == "" {
			category = poiCategory(i)
		}
		address := normalizeAMapText(poi.Address)
		if address == "" {
			address = fmt.Sprintf("%s POI %d", input.City, i+1)
		}
		pois = append(pois, MockPOI{
			Name:                     poi.Name,
			City:                     input.City,
			Category:                 category,
			Address:                  address,
			Location:                 poi.Location,
			SuggestedDurationMinutes: 90 + (i%3)*30,
			EstimatedCost:            float64(30 + (i%4)*20),
		})
	}
	if len(pois) == 0 {
		return nil, fmt.Errorf("amap poi response has no usable pois")
	}
	return pois, nil
}

type amapPOIResponse struct {
	Status string        `json:"status"`
	Info   string        `json:"info"`
	POIs   []amapPOIItem `json:"pois"`
}

type amapPOIItem struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Address  any    `json:"address"`
	Location string `json:"location"`
}

func normalizeAMapText(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		if len(v) == 0 {
			return ""
		}
		if text, ok := v[0].(string); ok {
			return text
		}
	}
	return ""
}
