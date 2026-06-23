# Agent Flow 设计文档

## 1. 当前阶段

当前阶段是第二版 Agent Flow：

* 已接入 CloudWeGo Eino
* 已实现 EinoTravelPlanner
* 已实现 Eino Graph / Workflow
* 已支持可选 LLM TravelPlan 生成节点
* LLM 输出通过 provider-native JSON Schema 约束，不依赖 prompt-only JSON 约束
* 已实现 Mock POI Tool
* 已实现 Mock Weather Tool
* 已实现 Mock Route Tool
* 已实现 Mock Budget Tool
* 已支持 real/mock Tool mode
* 已接入高德 POI、天气、路线 API adapter，默认不启用
* 未接入 Gin

## 2. 流程图

```text
TravelRequest
  -> ParseTravelRequestNode
  -> SearchPOIsToolNode
  -> GetWeatherToolNode
  -> ComputeRouteToolNode
  -> EstimateBudgetToolNode
  -> OptimizeItineraryNode
  -> GenerateTravelPlanNode (LLM schema output or deterministic fallback)
  -> ValidatePlanNode
  -> TravelPlan
```

## 3. 节点说明

### ParseTravelRequestNode

输入：`domain.TravelRequest`

输出：`TravelPlanningState`

职责：

* 校验目的地、天数和预算
* 标准化目的地城市、天数、预算和兴趣标签
* 给 `pace` 和 `transport_mode` 设置默认值

### SearchPOIsToolNode

输入：`TravelPlanningState`

输出：`TravelPlanningState`

职责：

* 调用 POI Tool 接口
* mock mode 使用 `MockPOITool`
* real mode 使用 `RealPOITool` 调用高德 POI 搜索
* real tool 失败时 fallback 到 mock，并追加 warning

### GetWeatherToolNode

输入：`TravelPlanningState`

输出：`TravelPlanningState`

职责：

* 调用 Weather Tool 接口
* mock mode 使用 `MockWeatherTool`
* real mode 使用 `RealWeatherTool` 查询高德天气
* real tool 失败时 fallback 到 mock，并追加 warning

### ComputeRouteToolNode

输入：`TravelPlanningState`

输出：`TravelPlanningState`

职责：

* 调用 Route Tool 接口
* mock mode 使用 `MockRouteTool`
* real mode 使用 `RealRouteTool` 调用高德路径规划
* real tool 失败时 fallback 到 mock，并追加 warning

### EstimateBudgetToolNode

输入：`TravelPlanningState`

输出：`TravelPlanningState`

职责：

* 调用 `MockBudgetTool`
* 估算交通、餐饮、酒店和门票预算
* 控制总预算不明显超过用户预算

### OptimizeItineraryNode

输入：`TravelPlanningState`

输出：`TravelPlanningState`

职责：

* 将 POI 按天分配
* 每天至少生成 2 个 TravelItem
* 根据兴趣和城市生成每日主题
* `relaxed` 节奏保持轻量安排，`intensive` 节奏可安排更多 item

### GenerateTravelPlanNode

输入：`TravelPlanningState`

输出：`domain.TravelPlan`

职责：

* 生成最终结构化 TravelPlan
* 默认使用 deterministic generator
* 启用 LLM 后调用 provider-native schema output
* DeepSeek 使用 `submit_travel_plan` strict tool call，tool parameters 是 `TravelPlan` JSON Schema
* OpenAI-compatible provider 如果支持 Structured Outputs，可使用 `response_format.type=json_schema`
* Title 包含目的地城市
* Summary 包含目的地城市和天数
* Days 数量匹配请求
* Budget 使用 MockBudgetTool 的估算结果
* Warnings 记录天气、Mock Tool 限制和 LLM fallback 原因

### ValidatePlanNode

输入：`domain.TravelPlan`

输出：`domain.TravelPlan`

职责：

* 校验 title、summary、days、items 和 budget
* 校验天数匹配、预算阈值、负数、空字段和目的地关键词
* 发现严重错误时返回 error
* 保证输出能被 Evaluation Harness 继续评估

## 4. LLM 结构化输出

LLM 模式默认不启用。启用后，配置来自环境变量：

* `TRAVEL_AGENT_LLM_ENABLED`
* `TRAVEL_AGENT_LLM_PROVIDER`
* `TRAVEL_AGENT_LLM_API_KEY`
* `TRAVEL_AGENT_LLM_BASE_URL`
* `TRAVEL_AGENT_LLM_MODEL`
* `TRAVEL_AGENT_LLM_TIMEOUT`
* `TRAVEL_AGENT_LLM_MAX_RETRIES`

DeepSeek 默认配置：

* provider：`deepseek`
* base URL：`https://api.deepseek.com/beta`
* model：`deepseek-v4-flash`

DeepSeek 主路径使用 strict tool calling：

```text
Chat Completions
  -> tools[0].function.name = submit_travel_plan
  -> tools[0].function.strict = true
  -> tools[0].function.parameters = TravelPlan JSON Schema
  -> tool_choice 强制 submit_travel_plan
```

该 JSON Schema 对所有 object 设置 `additionalProperties=false`，并把 object 字段全部列入 `required`。本地解析时仍使用 unknown-field 拒绝和业务校验，避免 provider 兼容性差异污染 domain 模型。

如果 LLM 未启用、配置缺失、provider 不支持 schema 输出、tool call 缺失、JSON 无效、业务校验失败或重试耗尽，系统会 fallback 到 deterministic generator，并在 `warnings` 中记录 fallback 原因。

## 5. Mock Tools

当前 Tools 支持 `mock` 和 `real` 两种模式：

```text
TRAVEL_AGENT_TOOL_MODE=mock
TRAVEL_AGENT_TOOL_MODE=real
```

默认是 `mock`。real mode 需要配置：

* `TRAVEL_AGENT_AMAP_API_KEY`
* `TRAVEL_AGENT_AMAP_BASE_URL`
* `TRAVEL_AGENT_WEATHER_API_KEY`
* `TRAVEL_AGENT_WEATHER_BASE_URL`
* `TRAVEL_AGENT_EXTERNAL_API_TIMEOUT`

Mock Tools 均为稳定、可复现的本地实现：

* `MockPOITool`：返回常见城市 POI 和未知城市兜底 POI
* `MockWeatherTool`：根据城市和天数生成固定天气
* `MockRouteTool`：根据 POI 顺序生成模拟路线耗时和距离
* `MockBudgetTool`：生成预算拆分，并控制总额不明显超过用户预算

Mock Tools 不会调用真实外部 API，也不会读取 API Key。real tools 的原始响应只在 `internal/agent/eino` 内解析，不污染 `internal/domain`。

## 6. 后续计划

1. 接入高德 POI API
2. 接入高德路线规划 API
3. 接入天气 API
4. 增加 Eino Callback Trace
5. 增加 Tool 调用轨迹评估
6. 接入 Gin API
7. 接入 SSE 流式输出
8. 接入 Redis 任务状态和缓存
