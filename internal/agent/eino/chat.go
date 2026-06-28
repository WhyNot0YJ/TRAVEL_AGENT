package eino

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"travel-agent/internal/agent"
	"travel-agent/internal/domain"
)

const chatInfoPromptVersion = "chat-info-v1"
const extractTravelInfoToolName = "extract_travel_info"

const (
	defaultDateRange        = domain.DefaultDateRange
	defaultTransportMode    = domain.DefaultTransportMode
	defaultPace             = domain.DefaultPace
	defaultWalkingTolerance = domain.DefaultWalkingTolerance
	defaultHotelArea        = domain.DefaultHotelArea
	defaultTravelerType     = domain.DefaultTravelerType
	defaultBudgetType       = domain.DefaultBudgetType
)

// chatInfoExtractor uses an OpenAI-compatible LLM to extract structured travel
// information from free-text user messages. Test mode always uses the local
// fallback extractor so the app remains deterministic.
type chatInfoExtractor struct {
	client         *openAICompatibleClient
	fallback       agent.TravelInfoExtractor
	maxRetries     int
	disabledReason string
}

func NewTravelInfoExtractor() agent.TravelInfoExtractor {
	cfg := loadLLMConfigFromEnv()
	fallback := simpleFallbackExtractor{}
	if !cfg.Enabled {
		return &chatInfoExtractor{fallback: fallback, disabledReason: "disabled"}
	}
	if cfg.APIKey == "" {
		return &chatInfoExtractor{fallback: fallback, disabledReason: "missing_api_key"}
	}
	if cfg.BaseURL == "" || cfg.Model == "" {
		return &chatInfoExtractor{fallback: fallback, disabledReason: "provider_error"}
	}
	return &chatInfoExtractor{
		client:     newOpenAICompatibleClient(cfg),
		fallback:   fallback,
		maxRetries: cfg.MaxRetries,
	}
}

func (e *chatInfoExtractor) Extract(ctx context.Context, message string, current domain.TravelRequest) (*agent.TravelInfoResult, error) {
	if e.fallback == nil {
		e.fallback = simpleFallbackExtractor{}
	}
	if agent.PlannerOptionsFromContext(ctx).TestMode {
		return e.fallback.Extract(ctx, message, current)
	}

	if e.disabledReason == "" && e.client != nil {
		attempts := e.maxRetries + 1
		if attempts <= 0 {
			attempts = 1
		}
		var lastResult *agent.TravelInfoResult
		for attempt := 0; attempt < attempts; attempt++ {
			result, err := e.callLLM(ctx, message, current)
			if err == nil {
				lastResult = result
				break
			}
		}
		if lastResult != nil {
			// Optional second pass: stream a natural-language reply via the
			// LLMDeltaReporter so the user sees a typewriter effect. The
			// structured fields stay as the strict tool call returned them.
			if reporter := agent.LLMDeltaReporterFromContext(ctx); reporter != nil && e.client.config.StreamEnabled {
				if streamed := e.client.streamChatReply(ctx, lastResult, message, reporter); streamed != "" {
					lastResult.Reply = streamed
				}
			}
			return lastResult, nil
		}
	}

	return e.fallback.Extract(ctx, message, current)
}

func (e *chatInfoExtractor) callLLM(ctx context.Context, message string, current domain.TravelRequest) (*agent.TravelInfoResult, error) {
	if e.client.config.StreamEnabled {
		return e.callLLMStreaming(ctx, message, current)
	}
	return e.callLLMBuffered(ctx, message, current)
}

func (e *chatInfoExtractor) callLLMStreaming(ctx context.Context, message string, current domain.TravelRequest) (*agent.TravelInfoResult, error) {
	options := agent.PlannerOptionsFromContext(ctx)
	messages, err := buildChatInfoMessages(message, current, options.AgentMode)
	if err != nil {
		return nil, err
	}
	payload := buildChatCompletionRequest(e.client.config, messages, chatInfoJSONSchema(), extractTravelInfoToolName, "Extract structured travel information from user message.", options.AgentMode)

	result, err := e.client.chatCompletionStream(ctx, payload, nil, nil)
	if err != nil {
		return nil, err
	}
	rawJSON, err := extractStreamToolPayload(e.client.config.Provider, payload.Model, result, extractTravelInfoToolName)
	if err != nil {
		return nil, err
	}
	parsed, err := parseChatInfoResult(rawJSON)
	if err != nil {
		return nil, err
	}
	usage := result.Usage
	log.Printf("[%s] value purpose=extract_travel_info_stream is_complete=%t missing=%d prompt_tokens=%d completion_tokens=%d total_tokens=%d duration_ms=%d", llmAPILogLabel(e.client.config), parsed.IsComplete, len(parsed.Missing), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, result.Duration.Milliseconds())
	return parsed, nil
}

func (e *chatInfoExtractor) callLLMBuffered(ctx context.Context, message string, current domain.TravelRequest) (*agent.TravelInfoResult, error) {
	options := agent.PlannerOptionsFromContext(ctx)
	messages, err := buildChatInfoMessages(message, current, options.AgentMode)
	if err != nil {
		return nil, err
	}
	payload := buildChatCompletionRequest(e.client.config, messages, chatInfoJSONSchema(), extractTravelInfoToolName, "Extract structured travel information from user message.", options.AgentMode)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal chat completion request: %w", err)
	}
	endpoint := chatCompletionsEndpoint(e.client.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build chat completion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+e.client.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	started := time.Now()
	log.Printf("[%s] call purpose=extract_travel_info provider=%s model=%s endpoint=%s stream=false", llmAPILogLabel(e.client.config), e.client.config.Provider, payload.Model, endpoint)
	resp, err := e.client.httpClient.Do(req)
	if err != nil {
		log.Printf("[%s] return purpose=extract_travel_info error=%v duration_ms=%d", llmAPILogLabel(e.client.config), err, time.Since(started).Milliseconds())
		return nil, fmt.Errorf("call LLM provider: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<19))
	if err != nil {
		log.Printf("[%s] return purpose=extract_travel_info status=%d error=%v duration_ms=%d", llmAPILogLabel(e.client.config), resp.StatusCode, err, time.Since(started).Milliseconds())
		return nil, fmt.Errorf("read LLM response: %w", err)
	}
	log.Printf("[%s] return purpose=extract_travel_info status=%d bytes=%d duration_ms=%d body=%s", llmAPILogLabel(e.client.config), resp.StatusCode, len(respBody), time.Since(started).Milliseconds(), string(respBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("LLM provider returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	rawJSON, usage, err := extractChatInfoPayload(e.client.config, respBody)
	if err != nil {
		return nil, err
	}
	result, err := parseChatInfoResult(rawJSON)
	if err != nil {
		return nil, err
	}
	log.Printf("[%s] value purpose=extract_travel_info is_complete=%t missing=%d prompt_tokens=%d completion_tokens=%d total_tokens=%d", llmAPILogLabel(e.client.config), result.IsComplete, len(result.Missing), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	return result, nil
}

func extractChatInfoPayload(cfg LLMConfig, data []byte) (string, LLMTokenUsage, error) {
	var resp chatCompletionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", LLMTokenUsage{}, fmt.Errorf("decode LLM response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", llmTokenUsage(resp.Usage), fmt.Errorf("LLM response has no choices")
	}

	message := resp.Choices[0].Message
	if strings.EqualFold(cfg.Provider, "deepseek") {
		for _, toolCall := range message.ToolCalls {
			if toolCall.Type == "function" && toolCall.Function.Name == extractTravelInfoToolName {
				if strings.TrimSpace(toolCall.Function.Arguments) == "" {
					return "", llmTokenUsage(resp.Usage), fmt.Errorf("extract_travel_info tool call has empty arguments")
				}
				return toolCall.Function.Arguments, llmTokenUsage(resp.Usage), nil
			}
		}
		return "", llmTokenUsage(resp.Usage), fmt.Errorf("LLM response did not call %s", extractTravelInfoToolName)
	}
	if strings.TrimSpace(message.Content) == "" {
		return "", llmTokenUsage(resp.Usage), fmt.Errorf("LLM response content is empty")
	}
	return message.Content, llmTokenUsage(resp.Usage), nil
}

func parseChatInfoResult(raw string) (*agent.TravelInfoResult, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("chat info result is empty")
	}
	var payload struct {
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
		Reply            string   `json:"reply"`
		Missing          []string `json:"missing"`
		IsComplete       bool     `json:"is_complete"`
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode chat info result: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("chat info result contains trailing data")
	}
	if strings.TrimSpace(payload.Reply) == "" {
		return nil, fmt.Errorf("chat info result reply is empty")
	}
	req := normalizeTravelBrief(domain.TravelRequest{
		DepartureCity:    payload.DepartureCity,
		DestinationCity:  payload.DestinationCity,
		Days:             payload.Days,
		Budget:           payload.Budget,
		Interests:        payload.Interests,
		Travelers:        payload.Travelers,
		DateRange:        payload.DateRange,
		TransportMode:    payload.TransportMode,
		Pace:             payload.Pace,
		WalkingTolerance: payload.WalkingTolerance,
		HotelArea:        payload.HotelArea,
		MustVisit:        payload.MustVisit,
		Avoid:            payload.Avoid,
		TravelerType:     payload.TravelerType,
		BudgetType:       payload.BudgetType,
		BudgetIncludes:   payload.BudgetIncludes,
	})
	missing := missingFieldsFromRequest(req)
	return &agent.TravelInfoResult{
		DepartureCity:    req.DepartureCity,
		DestinationCity:  req.DestinationCity,
		Days:             req.Days,
		Budget:           req.Budget,
		Interests:        req.Interests,
		Travelers:        req.Travelers,
		DateRange:        req.DateRange,
		TransportMode:    req.TransportMode,
		Pace:             req.Pace,
		WalkingTolerance: req.WalkingTolerance,
		HotelArea:        req.HotelArea,
		MustVisit:        req.MustVisit,
		Avoid:            req.Avoid,
		TravelerType:     req.TravelerType,
		BudgetType:       req.BudgetType,
		BudgetIncludes:   req.BudgetIncludes,
		Reply:            payload.Reply,
		Missing:          missing,
		IsComplete:       len(missing) == 0,
	}, nil
}

func buildChatInfoMessages(message string, current domain.TravelRequest, agentMode string) ([]chatMessage, error) {
	contextPayload := struct {
		Message          string   `json:"message"`
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
	}{
		Message:          message,
		DepartureCity:    current.DepartureCity,
		DestinationCity:  current.DestinationCity,
		Days:             current.Days,
		Budget:           current.Budget,
		Interests:        current.Interests,
		Travelers:        current.Travelers,
		DateRange:        current.DateRange,
		TransportMode:    current.TransportMode,
		Pace:             current.Pace,
		WalkingTolerance: current.WalkingTolerance,
		HotelArea:        current.HotelArea,
		MustVisit:        current.MustVisit,
		Avoid:            current.Avoid,
		TravelerType:     current.TravelerType,
		BudgetType:       current.BudgetType,
		BudgetIncludes:   current.BudgetIncludes,
	}
	data, err := json.Marshal(contextPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal chat context: %w", err)
	}

	system := strings.Join([]string{
		"You are a Chinese travel requirement collection assistant.",
		"Extract departure_city, destination_city, days, budget, interests, travelers, date_range, transport_mode, pace, walking_tolerance, hotel_area, must_visit, avoid, traveler_type, budget_type and budget_includes from the user message.",
		"Merge newly extracted fields with the existing confirmed fields. New explicit values override old values.",
		"Required fields are departure_city, destination_city, days, budget, interests and travelers.",
		"Optional fields must use product defaults when unknown: date_range=任意, transport_mode=任意, pace=适中, walking_tolerance=任意, hotel_area=任意, must_visit=[], avoid=[], traveler_type=无要求, budget_type=总预算, budget_includes=[住宿,餐饮,门票,市内交通].",
		"Prefer product-facing Chinese values. Acceptable legacy values may appear in existing context and should be normalized: transport_mode any/train_taxi/train_walk/subway_walk/flight_taxi/walk_taxi, pace relaxed/balanced/intensive, walking_tolerance any/low/medium/high, budget_type total/per_person.",
		"Reply to the user in concise Chinese. Confirm collected information and ask only for missing required fields.",
		"If every required field is present, tell the user the information is ready for itinerary generation.",
		plannerModeInstruction(agentMode),
		"Prompt version: " + chatInfoPromptVersion + ".",
	}, " ")
	user := fmt.Sprintf("Conversation context:\n%s", string(data))
	return []chatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}, nil
}

func chatInfoJSONSchema() map[string]any {
	return objectSchema(
		map[string]any{
			"departure_city":    stringSchema(),
			"destination_city":  stringSchema(),
			"days":              integerSchema(),
			"budget":            numberSchema(),
			"interests":         arraySchema(stringSchema()),
			"travelers":         integerSchema(),
			"date_range":        stringSchema(),
			"transport_mode":    stringSchema(),
			"pace":              stringSchema(),
			"walking_tolerance": stringSchema(),
			"hotel_area":        stringSchema(),
			"must_visit":        arraySchema(stringSchema()),
			"avoid":             arraySchema(stringSchema()),
			"traveler_type":     stringSchema(),
			"budget_type":       stringSchema(),
			"budget_includes":   arraySchema(stringSchema()),
			"reply":             stringSchema(),
			"missing":           arraySchema(stringSchema()),
			"is_complete":       map[string]any{"type": "boolean"},
		},
		"departure_city", "destination_city", "days", "budget", "interests",
		"travelers", "date_range", "transport_mode", "pace", "walking_tolerance",
		"hotel_area", "must_visit", "avoid", "traveler_type", "budget_type",
		"budget_includes", "reply", "missing", "is_complete",
	)
}

type simpleFallbackExtractor struct{}

func (simpleFallbackExtractor) Extract(ctx context.Context, message string, current domain.TravelRequest) (*agent.TravelInfoResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	extracted := mergeTravelRequest(current, extractTravelInfoFromMessage(message))
	extracted = normalizeTravelBrief(extracted)
	missing := missingFieldsFromRequest(extracted)
	return &agent.TravelInfoResult{
		DepartureCity:    extracted.DepartureCity,
		DestinationCity:  extracted.DestinationCity,
		Days:             extracted.Days,
		Budget:           extracted.Budget,
		Interests:        extracted.Interests,
		Travelers:        extracted.Travelers,
		DateRange:        extracted.DateRange,
		TransportMode:    extracted.TransportMode,
		Pace:             extracted.Pace,
		WalkingTolerance: extracted.WalkingTolerance,
		HotelArea:        extracted.HotelArea,
		MustVisit:        extracted.MustVisit,
		Avoid:            extracted.Avoid,
		TravelerType:     extracted.TravelerType,
		BudgetType:       extracted.BudgetType,
		BudgetIncludes:   extracted.BudgetIncludes,
		Reply:            buildFallbackChatReply(extracted, missing),
		Missing:          missing,
		IsComplete:       len(missing) == 0,
	}, nil
}

var (
	arabicDayPattern         = regexp.MustCompile(`(?i)(\d{1,2})\s*(?:天|日|days?)`)
	chineseDayPattern        = regexp.MustCompile(`([一二两三四五六七八九十]{1,3})\s*(?:天|日)`)
	arabicTravelersPattern   = regexp.MustCompile(`(?i)(\d{1,2})\s*(?:人|位|个人|adult|adults|traveler|travelers)`)
	chineseTravelersPattern  = regexp.MustCompile(`([一二两三四五六七八九十]{1,3})\s*(?:人|位|个人)`)
	budgetWithKeywordPattern = regexp.MustCompile(`(?i)(?:预算|人均|费用|花费|控制在|大概|约|左右)\s*(?:人民币|rmb|¥|￥)?\s*(\d+(?:\.\d+)?)\s*(k|K|千|万|元|块)?`)
	budgetWithUnitPattern    = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(k|K|千|万|元|块)`)
	dateRangePattern         = regexp.MustCompile(`(?i)(\d{1,2}\s*月\s*\d{1,2}\s*(?:日|号)?(?:\s*[-~到至]\s*\d{1,2}\s*月?\s*\d{0,2}\s*(?:日|号)?)?|\d{4}-\d{1,2}-\d{1,2}|本周末|周末|下周末|下周|明天|后天|暑假|国庆|春节|五一|端午|中秋)`)
	mustVisitPattern         = regexp.MustCompile(`(?:必去|一定要去|想去|必须安排)\s*([^，。；;]+)`)
	avoidPattern             = regexp.MustCompile(`(?:避开|不要|不想去|别安排|不安排|不喜欢)\s*([^，。；;]+)`)
	hotelAreaPattern         = regexp.MustCompile(`(?:酒店|住宿|住)(?:区域|位置|附近|在)?\s*([^，。；;]{2,16})`)
	departurePatterns        = []*regexp.Regexp{
		regexp.MustCompile(`(?:从|出发地(?:是|为)?|起点(?:是|为)?)\s*([A-Za-z\p{Han}]{2,20}?)(?:出发|到|去|前往|[，。；;]|$)`),
		regexp.MustCompile(`([A-Za-z\p{Han}]{2,20}?)\s*出发`),
	}
	destinationPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?:目的地(?:是|为)?|去|到|前往)\s*([A-Za-z\p{Han}]{2,20}?)(?:玩|游玩|旅游|旅行|\d|[一二两三四五六七八九十]|天|日|[，。；;]|$)`),
	}
	cityCandidatePattern = regexp.MustCompile(`[A-Za-z\p{Han}]{2,20}`)
	splitMessagePattern  = regexp.MustCompile(`[，。；;、,\n\r]+`)
)

func extractTravelInfoFromMessage(message string) domain.TravelRequest {
	normalized := strings.TrimSpace(message)
	req := domain.TravelRequest{
		DepartureCity:    extractDepartureCity(normalized),
		Days:             extractDays(normalized),
		Budget:           extractBudget(normalized),
		Interests:        extractInterests(normalized),
		Travelers:        extractTravelers(normalized),
		DateRange:        extractDateRange(normalized),
		TransportMode:    extractTransportMode(normalized),
		Pace:             extractPace(normalized),
		WalkingTolerance: extractWalkingTolerance(normalized),
		HotelArea:        extractHotelArea(normalized),
		MustVisit:        extractListByPattern(normalized, mustVisitPattern),
		Avoid:            extractListByPattern(normalized, avoidPattern),
		TravelerType:     extractTravelerType(normalized),
		BudgetType:       extractBudgetType(normalized),
		BudgetIncludes:   extractBudgetIncludes(normalized),
	}
	req.DestinationCity = extractDestinationCity(normalized, req.DepartureCity)
	return req
}

func mergeTravelRequest(current, extracted domain.TravelRequest) domain.TravelRequest {
	merged := current
	if extracted.DepartureCity != "" {
		merged.DepartureCity = extracted.DepartureCity
	}
	if extracted.DestinationCity != "" {
		merged.DestinationCity = extracted.DestinationCity
	}
	if extracted.Days > 0 {
		merged.Days = extracted.Days
	}
	if extracted.Budget > 0 {
		merged.Budget = extracted.Budget
	}
	if len(extracted.Interests) > 0 {
		merged.Interests = mergeUniqueStrings(merged.Interests, extracted.Interests)
	}
	if extracted.Travelers > 0 {
		merged.Travelers = extracted.Travelers
	}
	if extracted.DateRange != "" {
		merged.DateRange = extracted.DateRange
	}
	if extracted.TransportMode != "" {
		merged.TransportMode = extracted.TransportMode
	}
	if extracted.Pace != "" {
		merged.Pace = extracted.Pace
	}
	if extracted.WalkingTolerance != "" {
		merged.WalkingTolerance = extracted.WalkingTolerance
	}
	if extracted.HotelArea != "" {
		merged.HotelArea = extracted.HotelArea
	}
	if len(extracted.MustVisit) > 0 {
		merged.MustVisit = mergeUniqueStrings(merged.MustVisit, extracted.MustVisit)
	}
	if len(extracted.Avoid) > 0 {
		merged.Avoid = mergeUniqueStrings(merged.Avoid, extracted.Avoid)
	}
	if extracted.TravelerType != "" {
		merged.TravelerType = extracted.TravelerType
	}
	if extracted.BudgetType != "" {
		merged.BudgetType = extracted.BudgetType
	}
	if len(extracted.BudgetIncludes) > 0 {
		merged.BudgetIncludes = mergeUniqueStrings(nil, extracted.BudgetIncludes)
	}
	return merged
}

func extractDepartureCity(message string) string {
	for _, pattern := range departurePatterns {
		if match := pattern.FindStringSubmatch(message); len(match) > 1 {
			if city := cleanCityCandidate(match[1]); isUsableCityCandidate(city) {
				return city
			}
		}
	}
	return ""
}

func extractDestinationCity(message, departureCity string) string {
	for _, pattern := range destinationPatterns {
		if match := pattern.FindStringSubmatch(message); len(match) > 1 {
			if city := cleanCityCandidate(match[1]); isUsableCityCandidate(city) && city != departureCity {
				return city
			}
		}
	}
	for _, segment := range splitMessagePattern.Split(message, -1) {
		if !arabicDayPattern.MatchString(segment) && !chineseDayPattern.MatchString(segment) {
			continue
		}
		if strings.Contains(segment, "出发") {
			continue
		}
		if match := cityCandidatePattern.FindString(segment); match != "" {
			if city := cleanCityCandidate(match); isUsableCityCandidate(city) && city != departureCity {
				return city
			}
		}
	}
	return ""
}

func extractDays(message string) int {
	if match := arabicDayPattern.FindStringSubmatch(message); len(match) > 1 {
		if days, err := strconv.Atoi(match[1]); err == nil {
			return days
		}
	}
	if match := chineseDayPattern.FindStringSubmatch(message); len(match) > 1 {
		return parseSmallChineseNumber(match[1])
	}
	return 0
}

func extractTravelers(message string) int {
	if match := arabicTravelersPattern.FindStringSubmatch(message); len(match) > 1 {
		if travelers, err := strconv.Atoi(match[1]); err == nil {
			return travelers
		}
	}
	if match := chineseTravelersPattern.FindStringSubmatch(message); len(match) > 1 {
		return parseSmallChineseNumber(match[1])
	}
	if strings.Contains(message, "情侣") || strings.Contains(message, "两个人") || strings.Contains(message, "俩人") {
		return 2
	}
	if strings.Contains(message, "独自") || strings.Contains(message, "一个人") || strings.Contains(message, "单人") {
		return 1
	}
	return 0
}

func extractDateRange(message string) string {
	if match := dateRangePattern.FindString(message); match != "" {
		return strings.TrimSpace(match)
	}
	return ""
}

func extractBudget(message string) float64 {
	if budget := extractBudgetWithPattern(message, budgetWithKeywordPattern); budget > 0 {
		return budget
	}
	return extractBudgetWithPattern(message, budgetWithUnitPattern)
}

func extractBudgetWithPattern(message string, pattern *regexp.Regexp) float64 {
	match := pattern.FindStringSubmatch(message)
	if len(match) <= 1 {
		return 0
	}
	amount, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0
	}
	if len(match) > 2 {
		switch strings.ToLower(match[2]) {
		case "k", "千":
			amount *= 1000
		case "万":
			amount *= 10000
		}
	}
	return amount
}

func extractInterests(message string) []string {
	groups := []struct {
		value    string
		keywords []string
	}{
		{value: "美食", keywords: []string{"美食", "小吃", "餐厅", "吃"}},
		{value: "自然风光", keywords: []string{"自然风光", "自然", "风景", "山水", "西湖"}},
		{value: "历史文化", keywords: []string{"历史文化", "历史", "文化", "博物馆", "古镇", "人文"}},
		{value: "购物", keywords: []string{"购物", "商场"}},
		{value: "亲子", keywords: []string{"亲子", "孩子", "儿童"}},
		{value: "夜景", keywords: []string{"夜景", "夜生活"}},
		{value: "摄影", keywords: []string{"摄影", "拍照"}},
		{value: "咖啡", keywords: []string{"咖啡"}},
		{value: "citywalk", keywords: []string{"citywalk", "城市漫步", "步行"}},
	}
	var interests []string
	for _, group := range groups {
		for _, keyword := range group.keywords {
			if strings.Contains(strings.ToLower(message), strings.ToLower(keyword)) {
				interests = append(interests, group.value)
				break
			}
		}
	}
	return interests
}

func extractWalkingTolerance(message string) string {
	switch {
	case strings.Contains(message, "少走") || strings.Contains(message, "不想走") || strings.Contains(message, "少步行") || strings.Contains(message, "走路少"):
		return "low"
	case strings.Contains(message, "步行适中") || strings.Contains(message, "适中步行") || strings.Contains(message, "能走一点"):
		return "medium"
	case strings.Contains(message, "不介意走") || strings.Contains(message, "多走路") || strings.Contains(message, "徒步"):
		return "high"
	default:
		return ""
	}
}

func extractHotelArea(message string) string {
	if match := hotelAreaPattern.FindStringSubmatch(message); len(match) > 1 {
		area := strings.TrimSpace(strings.Trim(match[1], " ，。；;、,"))
		if area != "" {
			return area
		}
	}
	return ""
}

func extractListByPattern(message string, pattern *regexp.Regexp) []string {
	match := pattern.FindStringSubmatch(message)
	if len(match) <= 1 {
		return nil
	}
	raw := strings.TrimSpace(match[1])
	raw = strings.Trim(raw, " ，。；;、,")
	if raw == "" {
		return nil
	}
	parts := regexp.MustCompile(`[、,，和及/]+`).Split(raw, -1)
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, " ，。；;、,"))
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func extractTravelerType(message string) string {
	switch {
	case strings.Contains(message, "老人") || strings.Contains(message, "父母") || strings.Contains(message, "长辈"):
		return "老人"
	case strings.Contains(message, "孩子") || strings.Contains(message, "儿童") || strings.Contains(message, "亲子"):
		return "亲子"
	case strings.Contains(message, "情侣"):
		return "情侣"
	case strings.Contains(message, "朋友") || strings.Contains(message, "同学"):
		return "朋友"
	case strings.Contains(message, "商务") || strings.Contains(message, "出差"):
		return "商务"
	default:
		return ""
	}
}

func extractBudgetType(message string) string {
	if strings.Contains(message, "人均") || strings.Contains(strings.ToLower(message), "per person") {
		return "per_person"
	}
	if strings.Contains(message, "总预算") || strings.Contains(message, "总共") || strings.Contains(message, "总计") {
		return "total"
	}
	return ""
}

func extractBudgetIncludes(message string) []string {
	lower := strings.ToLower(message)
	includes := []string{}
	if strings.Contains(message, "含住宿") || strings.Contains(message, "包含住宿") || strings.Contains(message, "包括住宿") {
		includes = append(includes, "住宿")
	}
	if strings.Contains(message, "含餐") || strings.Contains(message, "包含餐饮") || strings.Contains(message, "包括餐饮") {
		includes = append(includes, "餐饮")
	}
	if strings.Contains(message, "含门票") || strings.Contains(message, "包含门票") || strings.Contains(message, "包括门票") {
		includes = append(includes, "门票")
	}
	if strings.Contains(message, "含交通") || strings.Contains(message, "市内交通") || strings.Contains(message, "打车") {
		includes = append(includes, "市内交通")
	}
	if strings.Contains(message, "大交通") || strings.Contains(message, "往返交通") || strings.Contains(lower, "round trip") {
		if strings.Contains(message, "不含") || strings.Contains(message, "不包含") {
			return defaultBudgetIncludesCopy()
		}
		includes = append(includes, "往返大交通")
	}
	return mergeUniqueStrings(nil, includes)
}

func extractTransportMode(message string) string {
	switch {
	case strings.Contains(message, "交通无要求") || strings.Contains(message, "交通任意"):
		return "any"
	case strings.Contains(message, "飞机") || strings.Contains(message, "航班"):
		return "flight_taxi"
	case strings.Contains(message, "地铁"):
		return "subway_walk"
	case strings.Contains(message, "高铁") || strings.Contains(message, "火车") || strings.Contains(message, "动车"):
		if strings.Contains(message, "步行") && !strings.Contains(message, "少走") {
			return "train_walk"
		}
		return "train_taxi"
	case strings.Contains(message, "步行") && (strings.Contains(message, "打车") || strings.Contains(message, "出租")):
		return "walk_taxi"
	default:
		return ""
	}
}

func extractPace(message string) string {
	switch {
	case strings.Contains(message, "轻松") || strings.Contains(message, "悠闲") || strings.Contains(message, "慢") || strings.Contains(message, "不赶"):
		return "relaxed"
	case strings.Contains(message, "紧凑") || strings.Contains(message, "多安排") || strings.Contains(message, "充实") || strings.Contains(message, "特种兵"):
		return "intensive"
	case strings.Contains(message, "平衡") || strings.Contains(message, "适中"):
		return "balanced"
	default:
		return ""
	}
}

func cleanCityCandidate(value string) string {
	city := strings.TrimSpace(value)
	city = strings.Trim(city, " ，。；;、,")
	replacements := []string{"目的地是", "目的地为", "目的地", "出发地是", "出发地为", "从", "去", "到", "前往"}
	for _, replacement := range replacements {
		city = strings.TrimPrefix(city, replacement)
	}
	for _, suffix := range []string{"出发", "游玩", "旅游", "旅行", "玩", "市"} {
		city = strings.TrimSuffix(city, suffix)
	}
	return strings.TrimSpace(strings.Trim(city, " ，。；;、,"))
}

func isUsableCityCandidate(city string) bool {
	if city == "" {
		return false
	}
	blocked := map[string]struct{}{
		"预算":   {},
		"喜欢":   {},
		"偏好":   {},
		"轻松":   {},
		"自然风光": {},
	}
	_, blockedCity := blocked[city]
	return !blockedCity
}

func parseSmallChineseNumber(value string) int {
	digits := map[rune]int{
		'一': 1,
		'二': 2,
		'两': 2,
		'三': 3,
		'四': 4,
		'五': 5,
		'六': 6,
		'七': 7,
		'八': 8,
		'九': 9,
	}
	if value == "十" {
		return 10
	}
	runes := []rune(value)
	if len(runes) == 1 {
		return digits[runes[0]]
	}
	if strings.Contains(value, "十") {
		parts := strings.Split(value, "十")
		tens := 1
		if parts[0] != "" {
			tens = digits[[]rune(parts[0])[0]]
		}
		ones := 0
		if len(parts) > 1 && parts[1] != "" {
			ones = digits[[]rune(parts[1])[0]]
		}
		return tens*10 + ones
	}
	return 0
}

func mergeUniqueStrings(existing, incoming []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(incoming))
	merged := make([]string, 0, len(existing)+len(incoming))
	for _, value := range append(existing, incoming...) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		merged = append(merged, value)
	}
	return merged
}

func buildFallbackChatReply(req domain.TravelRequest, missing []string) string {
	summary := formatTravelInfoSummary(req)
	reply := "收到。"
	if summary != "" {
		reply += "我整理到：" + summary + "。"
	}
	if len(missing) > 0 {
		reply += fmt.Sprintf("还需要确认：%s。", strings.Join(missing, "、"))
	} else {
		reply += "信息已经齐全，可以生成行程了。"
	}
	return reply
}

func formatTravelInfoSummary(req domain.TravelRequest) string {
	var parts []string
	if req.DepartureCity != "" {
		parts = append(parts, req.DepartureCity+"出发")
	}
	if req.DestinationCity != "" {
		parts = append(parts, "去"+req.DestinationCity)
	}
	if req.Days > 0 {
		parts = append(parts, fmt.Sprintf("%d天", req.Days))
	}
	if req.Budget > 0 {
		parts = append(parts, "预算"+formatBudget(req.Budget)+"元")
	}
	if req.Travelers > 0 {
		parts = append(parts, fmt.Sprintf("%d人出行", req.Travelers))
	}
	if len(req.Interests) > 0 {
		parts = append(parts, "偏好"+strings.Join(req.Interests, "、"))
	}
	if len(req.MustVisit) > 0 {
		parts = append(parts, "必去"+strings.Join(req.MustVisit, "、"))
	}
	if len(req.Avoid) > 0 {
		parts = append(parts, "避开"+strings.Join(req.Avoid, "、"))
	}
	return strings.Join(parts, "，")
}

func formatBudget(budget float64) string {
	if budget == float64(int64(budget)) {
		return strconv.FormatInt(int64(budget), 10)
	}
	return strconv.FormatFloat(budget, 'f', 2, 64)
}

func missingFieldsFromRequest(req domain.TravelRequest) []string {
	var missing []string
	if req.DepartureCity == "" {
		missing = append(missing, "出发城市")
	}
	if req.DestinationCity == "" {
		missing = append(missing, "目的地")
	}
	if req.Days <= 0 {
		missing = append(missing, "天数")
	}
	if req.Budget <= 0 {
		missing = append(missing, "预算")
	}
	if len(req.Interests) == 0 {
		missing = append(missing, "兴趣偏好")
	}
	if req.Travelers <= 0 {
		missing = append(missing, "出行人数")
	}
	return missing
}

func normalizeTravelBrief(req domain.TravelRequest) domain.TravelRequest {
	return domain.NormalizeTravelBrief(req)
}

func defaultBudgetIncludesCopy() []string {
	return domain.DefaultBudgetIncludes()
}
