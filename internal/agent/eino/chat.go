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
		DepartureCity   string   `json:"departure_city"`
		DestinationCity string   `json:"destination_city"`
		Days            int      `json:"days"`
		Budget          float64  `json:"budget"`
		Interests       []string `json:"interests"`
		TransportMode   string   `json:"transport_mode"`
		Pace            string   `json:"pace"`
		Reply           string   `json:"reply"`
		Missing         []string `json:"missing"`
		IsComplete      bool     `json:"is_complete"`
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
	return &agent.TravelInfoResult{
		DepartureCity:   payload.DepartureCity,
		DestinationCity: payload.DestinationCity,
		Days:            payload.Days,
		Budget:          payload.Budget,
		Interests:       payload.Interests,
		TransportMode:   payload.TransportMode,
		Pace:            payload.Pace,
		Reply:           payload.Reply,
		Missing:         payload.Missing,
		IsComplete:      payload.IsComplete,
	}, nil
}

func buildChatInfoMessages(message string, current domain.TravelRequest, agentMode string) ([]chatMessage, error) {
	contextPayload := struct {
		Message         string   `json:"message"`
		DepartureCity   string   `json:"departure_city"`
		DestinationCity string   `json:"destination_city"`
		Days            int      `json:"days"`
		Budget          float64  `json:"budget"`
		Interests       []string `json:"interests"`
		TransportMode   string   `json:"transport_mode"`
		Pace            string   `json:"pace"`
	}{
		Message:         message,
		DepartureCity:   current.DepartureCity,
		DestinationCity: current.DestinationCity,
		Days:            current.Days,
		Budget:          current.Budget,
		Interests:       current.Interests,
		TransportMode:   current.TransportMode,
		Pace:            current.Pace,
	}
	data, err := json.Marshal(contextPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal chat context: %w", err)
	}

	system := strings.Join([]string{
		"You are a Chinese travel requirement collection assistant.",
		"Extract departure_city, destination_city, days, budget, interests, transport_mode and pace from the user message.",
		"Merge newly extracted fields with the existing confirmed fields. New explicit values override old values.",
		"Required fields are departure_city, destination_city, days, budget and interests.",
		"transport_mode values: train_taxi, train_walk, subway_walk, flight_taxi, walk_taxi.",
		"pace values: relaxed, balanced, intensive.",
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
			"departure_city":   stringSchema(),
			"destination_city": stringSchema(),
			"days":             integerSchema(),
			"budget":           numberSchema(),
			"interests":        arraySchema(stringSchema()),
			"transport_mode":   stringSchema(),
			"pace":             stringSchema(),
			"reply":            stringSchema(),
			"missing":          arraySchema(stringSchema()),
			"is_complete":      map[string]any{"type": "boolean"},
		},
		"departure_city", "destination_city", "days", "budget", "interests",
		"transport_mode", "pace", "reply", "missing", "is_complete",
	)
}

type simpleFallbackExtractor struct{}

func (simpleFallbackExtractor) Extract(ctx context.Context, message string, current domain.TravelRequest) (*agent.TravelInfoResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	extracted := mergeTravelRequest(current, extractTravelInfoFromMessage(message))
	missing := missingFieldsFromRequest(extracted)
	return &agent.TravelInfoResult{
		DepartureCity:   extracted.DepartureCity,
		DestinationCity: extracted.DestinationCity,
		Days:            extracted.Days,
		Budget:          extracted.Budget,
		Interests:       extracted.Interests,
		TransportMode:   extracted.TransportMode,
		Pace:            extracted.Pace,
		Reply:           buildFallbackChatReply(extracted, missing),
		Missing:         missing,
		IsComplete:      len(missing) == 0,
	}, nil
}

var (
	arabicDayPattern         = regexp.MustCompile(`(?i)(\d{1,2})\s*(?:天|日|days?)`)
	chineseDayPattern        = regexp.MustCompile(`([一二两三四五六七八九十]{1,3})\s*(?:天|日)`)
	budgetWithKeywordPattern = regexp.MustCompile(`(?i)(?:预算|人均|费用|花费|控制在|大概|约|左右)\s*(?:人民币|rmb|¥|￥)?\s*(\d+(?:\.\d+)?)\s*(k|K|千|万|元|块)?`)
	budgetWithUnitPattern    = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(k|K|千|万|元|块)`)
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
		DepartureCity: extractDepartureCity(normalized),
		Days:          extractDays(normalized),
		Budget:        extractBudget(normalized),
		Interests:     extractInterests(normalized),
		TransportMode: extractTransportMode(normalized),
		Pace:          extractPace(normalized),
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
	if extracted.TransportMode != "" {
		merged.TransportMode = extracted.TransportMode
	}
	if extracted.Pace != "" {
		merged.Pace = extracted.Pace
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

func extractTransportMode(message string) string {
	switch {
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
	if len(req.Interests) > 0 {
		parts = append(parts, "偏好"+strings.Join(req.Interests, "、"))
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
	return missing
}
