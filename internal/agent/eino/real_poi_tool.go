package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"travel-agent/internal/domain"
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
		"extensions": "all",
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
		metadata := poi.metadata()
		pois = append(pois, MockPOI{
			Name:                     poi.Name,
			City:                     input.City,
			Category:                 category,
			Address:                  address,
			Location:                 poi.Location,
			SuggestedDurationMinutes: 90 + (i%3)*30,
			EstimatedCost:            legacyCost(metadata.Cost),
			Cost:                     metadata.Cost,
			Metadata:                 &metadata,
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
	ID           string         `json:"id"`
	Parent       any            `json:"parent"`
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	TypeCode     string         `json:"typecode"`
	BizType      string         `json:"biz_type"`
	Address      any            `json:"address"`
	Location     string         `json:"location"`
	Tel          any            `json:"tel"`
	Postcode     any            `json:"postcode"`
	Website      any            `json:"website"`
	Email        any            `json:"email"`
	PCode        string         `json:"pcode"`
	PName        string         `json:"pname"`
	CityCode     string         `json:"citycode"`
	CityName     string         `json:"cityname"`
	ADCode       string         `json:"adcode"`
	ADName       string         `json:"adname"`
	EntrLocation string         `json:"entr_location"`
	ExitLocation string         `json:"exit_location"`
	NaviPOIID    string         `json:"navi_poiid"`
	BusinessArea any            `json:"business_area"`
	Tag          any            `json:"tag"`
	BizExt       amapPOIBizExt  `json:"biz_ext"`
	Photos       []amapPOIPhoto `json:"photos"`
}

type amapPOIBizExt struct {
	Rating any `json:"rating"`
	Cost   any `json:"cost"`
}

type amapPOIPhoto struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (p amapPOIItem) metadata() domain.POIMetadata {
	cost := domain.UnavailableCost("per_person", "amap.poi.biz_ext.cost")
	if amount, ok := parseAMapFloat(p.BizExt.Cost); ok {
		cost = domain.AvailableCost(amount, poiCostUnit(p.Type), "amap.poi.biz_ext.cost", true)
	}
	photos := make([]domain.POIPhoto, 0, len(p.Photos))
	for _, photo := range p.Photos {
		if photo.Title == "" && photo.URL == "" {
			continue
		}
		photos = append(photos, domain.POIPhoto{Title: photo.Title, URL: photo.URL})
	}
	rating, _ := parseAMapFloatPtr(p.BizExt.Rating)
	return domain.POIMetadata{
		Provider:     "amap",
		ID:           p.ID,
		Parent:       normalizeAMapText(p.Parent),
		TypeCode:     p.TypeCode,
		BizType:      p.BizType,
		Tel:          normalizeAMapText(p.Tel),
		Postcode:     normalizeAMapText(p.Postcode),
		Website:      normalizeAMapText(p.Website),
		Email:        normalizeAMapText(p.Email),
		PCode:        p.PCode,
		PName:        p.PName,
		CityCode:     p.CityCode,
		CityName:     p.CityName,
		ADCode:       p.ADCode,
		ADName:       p.ADName,
		EntrLocation: p.EntrLocation,
		ExitLocation: p.ExitLocation,
		NaviPOIID:    p.NaviPOIID,
		BusinessArea: normalizeAMapText(p.BusinessArea),
		Tag:          normalizeAMapText(p.Tag),
		Rating:       rating,
		Photos:       photos,
		Cost:         cost,
	}
}

func poiCostUnit(category string) string {
	if strings.Contains(category, "酒店") {
		return "per_night_reference"
	}
	return "per_person"
}

func legacyCost(cost domain.CostInfo) float64 {
	if cost.Amount == nil {
		return 0
	}
	return *cost.Amount
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

func parseAMapFloatPtr(value any) (*float64, bool) {
	parsed, ok := parseAMapFloat(value)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func parseAMapFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" || v == "[]" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(v, 64)
		return parsed, err == nil
	case float64:
		return v, true
	case int:
		return float64(v), true
	case json.Number:
		parsed, err := v.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}
