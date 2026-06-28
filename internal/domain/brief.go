package domain

import "strings"

const (
	DefaultDateRange        = "任意"
	DefaultTransportMode    = "任意"
	DefaultPace             = "适中"
	DefaultWalkingTolerance = "任意"
	DefaultHotelArea        = "任意"
	DefaultTravelerType     = "无要求"
	DefaultBudgetType       = "总预算"
)

var defaultBudgetIncludes = []string{"住宿", "餐饮", "门票", "市内交通"}

func DefaultBudgetIncludes() []string {
	return append([]string{}, defaultBudgetIncludes...)
}

// NormalizeTravelBrief applies stable optional defaults and maps legacy enum
// values to product-facing Chinese labels. It intentionally does not default
// required fields such as travelers.
func NormalizeTravelBrief(req TravelRequest) TravelRequest {
	req.DepartureCity = strings.TrimSpace(req.DepartureCity)
	req.DestinationCity = strings.TrimSpace(req.DestinationCity)
	req.Interests = normalizeStringSlice(req.Interests)
	req.DateRange = defaultString(req.DateRange, DefaultDateRange)
	req.TransportMode = NormalizeTransportMode(req.TransportMode)
	req.Pace = NormalizePace(req.Pace)
	req.WalkingTolerance = NormalizeWalkingTolerance(req.WalkingTolerance)
	req.HotelArea = defaultString(req.HotelArea, DefaultHotelArea)
	req.MustVisit = normalizeStringSlice(req.MustVisit)
	req.Avoid = normalizeStringSlice(req.Avoid)
	req.TravelerType = defaultString(req.TravelerType, DefaultTravelerType)
	req.BudgetType = NormalizeBudgetType(req.BudgetType)
	req.BudgetIncludes = normalizeStringSlice(req.BudgetIncludes)
	if len(req.BudgetIncludes) == 0 {
		req.BudgetIncludes = DefaultBudgetIncludes()
	}
	return req
}

func NormalizeTransportMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "any", "任意", "不限", "无要求":
		return DefaultTransportMode
	case "train_taxi", "高铁 + 打车", "高铁+打车":
		return "高铁 + 打车"
	case "train_walk", "高铁 + 步行", "高铁+步行":
		return "高铁 + 步行"
	case "train_subway", "高铁 + 地铁", "高铁+地铁":
		return "高铁 + 地铁"
	case "subway_walk", "地铁 + 步行", "地铁+步行":
		return "地铁 + 步行"
	case "subway_taxi", "地铁 + 打车", "地铁+打车":
		return "地铁 + 打车"
	case "flight_taxi", "飞机 + 打车", "飞机+打车":
		return "飞机 + 打车"
	case "flight_subway", "飞机 + 地铁", "飞机+地铁":
		return "飞机 + 地铁"
	case "walk_taxi", "步行 + 打车", "步行+打车":
		return "步行 + 打车"
	default:
		return strings.TrimSpace(value)
	}
}

func NormalizePace(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "balanced", "balance", "适中", "均衡", "平衡":
		return DefaultPace
	case "relaxed", "轻松", "悠闲":
		return "轻松"
	case "intensive", "紧凑", "充实":
		return "紧凑"
	default:
		return strings.TrimSpace(value)
	}
}

func NormalizeWalkingTolerance(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "any", "任意", "不限", "无要求":
		return DefaultWalkingTolerance
	case "low", "低", "少走路", "少步行":
		return "低"
	case "medium", "中", "适中":
		return "中"
	case "high", "高", "不介意多走":
		return "高"
	default:
		return strings.TrimSpace(value)
	}
}

func NormalizeBudgetType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "total", "总预算":
		return DefaultBudgetType
	case "per_person", "per person", "人均", "人均预算":
		return "人均预算"
	default:
		return strings.TrimSpace(value)
	}
}

func IsRelaxedPace(value string) bool {
	return NormalizePace(value) == "轻松"
}

func IsIntensivePace(value string) bool {
	return NormalizePace(value) == "紧凑"
}

func IsLowWalkingTolerance(value string) bool {
	return NormalizeWalkingTolerance(value) == "低"
}

func IsBudgetPerPerson(value string) bool {
	return NormalizeBudgetType(value) == "人均预算"
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func normalizeStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if out == nil {
		return []string{}
	}
	return out
}
