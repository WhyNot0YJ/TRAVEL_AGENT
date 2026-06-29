package eino

import (
	"encoding/json"
	"fmt"
	"strings"
)

const travelPlanPromptVersion = "travel-plan-v1"

func buildTravelPlanMessages(state TravelPlanningState, agentMode string) ([]chatMessage, error) {
	contextPayload := struct {
		Request   any `json:"request"`
		POIs      any `json:"pois"`
		Weather   any `json:"weather"`
		Routes    any `json:"routes"`
		Budget    any `json:"budget"`
		Itinerary any `json:"itinerary"`
	}{
		Request:   state.Request,
		POIs:      state.POIs,
		Weather:   state.Weather,
		Routes:    state.Routes,
		Budget:    state.Budget,
		Itinerary: state.Itinerary,
	}
	data, err := json.Marshal(contextPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal planning context: %w", err)
	}

	system := strings.Join([]string{
		"你是旅游路线规划组件。",
		"提示词版本：" + travelPlanPromptVersion + "。",
		"请基于给定的规划上下文，生成实用、可执行的旅行行程。",
		plannerModeInstruction(agentMode),
		"价格和预算只能使用上下文中 status=available 且 included=true 的真实金额。",
		"status=unavailable 的费用必须展示为“暂无信息”，不要猜测、补全、估算或按比例分摊缺失费用。",
		"预算合计只能统计真实可得金额；如果预算不完整，必须在 summary 或 warnings 中说明哪些项目暂无信息。",
		"最终路线只能通过已配置的结构化输出通道返回。",
		"不要包含密钥、API Key、隐藏推理过程，或任何上下文没有支持的外部断言。",
	}, " ")
	user := fmt.Sprintf("规划上下文：\n%s", string(data))
	return []chatMessage{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}, nil
}

func plannerModeInstruction(_ string) string {
	return "请在保证信息准确和结构完整的前提下，生成简洁、实用的行程。"
}
