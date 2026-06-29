# Agent 流程

## 当前阶段

项目使用 Go `TravelPlanner` 接口，并在 `internal/agent/eino` 中提供 Eino 实现。
前端是 React H5 对话界面。后端提供异步旅行规划任务、SSE 进度、基于聊天的需求收集 API、确定性测试模式，以及可选的真实 LLM / 真实外部工具。

## 聊天优先流程

```text
用户消息
  -> POST /api/v1/travel/chat/stream
  -> TravelInfoExtractor
     -> test_mode: 本地规则抽取
     -> real mode: LLM 抽取，失败时回退到规则
  -> ChatResponse(is_complete, missing, full Travel Brief)
  -> 聊天中的 Travel Brief 确认卡
  -> POST /api/v1/travel/plans
  -> Eino TravelPlanning Graph
  -> GET /api/v1/travel/plans/:task_id/stream
  -> TravelPlan
```

前端不再渲染固定的外层“生成行程”表单按钮。当 `is_complete=true` 时，助手消息会渲染 Travel Brief 确认卡和 `生成行程` 操作。点击该操作后，前端会用已收集的结构化需求创建规划任务。缺少出行人数等必填项时，前端不展示可用的生成动作，后端也会拒绝创建任务。

## 运行选项

运行行为由 `agent.PlannerOptions` 承载：

* `TestMode=true`：需求收集、工具调用和最终生成都使用本地确定性规则、mock tools 和 deterministic fallback generation。
* `TestMode=false`：需求收集和最终生成优先使用已配置的 LLM；工具按 `TRAVEL_AGENT_TOOL_MODE` 选择。
* `AgentMode=quick`：路由到 `TRAVEL_AGENT_LLM_MODEL_QUICK` 中配置的模型，默认 `deepseek-v4-flash`，延迟更低、成本更低。
* `AgentMode=expert`：路由到 `TRAVEL_AGENT_LLM_MODEL_EXPERT` 中配置的模型，默认 `deepseek-v4-pro`，推理能力更强、成本更高。`agent_mode` 会参与 `request_hash`，因此 quick/expert 的结果不会共用缓存。

DeepSeek strict tool calling 仍会发送 `thinking.type=disabled`，因为 DeepSeek thinking mode 不支持强制 `tool_choice`。当配置的模型名包含 `reasoner` 时，客户端会自动回退到 `response_format=json_schema`，并从 `message.content` 接收 JSON，因为 reasoner 类模型不产生 tool calls。

## Eino 规划图

```text
TravelRequest
  -> ParseTravelRequestNode
  -> SearchPOIsToolNode
  -> GetWeatherToolNode
  -> ComputeRouteToolNode
  -> OptimizeItineraryNode
  -> EstimateBudgetToolNode
  -> ValidateRouteFeasibilityNode
  -> GenerateTravelPlanNode
  -> ValidatePlanNode
  -> TravelPlan
```

预算节点在初版 itinerary 之后执行，只汇总已进入草稿行程且真实可得的 POI/路线费用；缺失的住宿、跨城大交通等费用标记为“暂无信息”，不计入 `known_total`。

## LLM 生成

`GenerateTravelPlanNode` 使用 `travel-plan-v1`。

启用并正确配置 LLM 后，OpenAI-compatible client 会请求 provider 原生结构化输出：

* DeepSeek：strict function tool `submit_travel_plan`。
* 其他兼容 provider：支持时使用 JSON schema response format。
* DeepSeek reasoner 类模型：自动降级到 `response_format=json_schema`，不使用 tool calls。

如果启用了测试模式、LLM 未启用、API key 缺失、provider 返回无效输出、重试耗尽或业务校验失败，该节点会回退到确定性生成，并记录 `LLM fallback` warning。

`travel-plan-v1` 约束 LLM 只能使用上下文中 `status=available` 且 `included=true` 的真实金额。`status=unavailable` 的费用必须保留为“暂无信息”，不得由模型猜测、补全或按比例拆分。

### 流式 LLM 输出

当 `TRAVEL_AGENT_LLM_STREAM_ENABLED=true`（默认）时，需求抽取、最终计划生成和助手回复都会以 `stream=true` 调用 DeepSeek chat completions。结构化调用仍使用 strict tool schemas；服务端会累积流式 tool arguments，并在 `[DONE]` 后解析最终 JSON。

规划生成时，同一份流式 `submit_travel_plan` tool arguments 会被增量扫描顶层 `summary` 字段。如果挂载了 `LLMDeltaReporter`，新观察到的 summary 文本会作为 `assistant_delta` 转发；流结束后，累积参数会被解析为最终 `TravelPlan`。如果没有 reporter，例如在 `cmd/harness` 中，provider 调用仍使用 `stream=true`，但 delta 只在内部累积。

`LLMDeltaReporter` 定义在 `internal/agent/metadata.go`，使用与 `PlannerEventReporter` 相同的 context-key 模式。`internal/travel` 包提供实现，把 delta 转换为 `EventAssistantDelta` 并发布到 `EventBus`。`internal/agent/eino` 只依赖该抽象，不 import `internal/travel`。

## 需求收集

`TravelInfoExtractor` 使用 `chat-info-v1`。

* `POST /api/v1/travel/chat` 返回单次 JSON 响应。
* `POST /api/v1/travel/chat/stream` 返回 SSE `assistant_delta` 分片，最后返回带同一份结构化 `ChatResponse` 的 `done`。

必填字段：

* `departure_city`
* `destination_city`
* `days`
* `budget`
* `interests`
* `travelers`

可选字段缺失时不阻塞生成，统一默认：

* `date_range=任意`
* `pace=适中`
* `transport_mode=任意`
* `walking_tolerance=任意`
* `hotel_area=任意`
* `must_visit=[]`
* `avoid=[]`
* `traveler_type=无要求`
* `budget_type=总预算`
* `budget_includes=["住宿","餐饮","门票","市内交通"]`

测试模式始终使用本地规则抽取器。真实模式会先尝试流式 LLM 抽取，不可用时回退到规则。Planner 会读取 `travelers`、`walking_tolerance`、`hotel_area`、`must_visit`、`avoid`、`traveler_type`、`budget_type` 和 `budget_includes`，用于预算、强度、POI 排序和 warning 文案。

## 工具

Tools 支持 `mock` 和 `real` 两种模式：

```text
TRAVEL_AGENT_TOOL_MODE=mock
TRAVEL_AGENT_TOOL_MODE=real
```

真实高德/天气/路线请求共享后端限流器。默认值：

* `TRAVEL_AGENT_EXTERNAL_API_CONCURRENCY=2`
* `TRAVEL_AGENT_EXTERNAL_API_QPS=2`

当 `test_mode=true` 时，即使服务端配置为真实工具，tools 也会被强制设置为 mock。

当前工具职责：

* `RealPOITool`：调用高德 `place/text`，使用 `extensions=all`，转换名称、类型、地址、坐标、评分、人均消费、照片等字段。
* `RealWeatherTool`：先调用高德 `geocode/geo` 获取 adcode，再调用 `weather/weatherInfo` 查询预报。
* `RealRouteTool`：按交通模式选择 walking / bicycling / driving / transit，并解析可得的 `taxi_cost`、`tolls` 或 `transits[].cost`。
* `MockBudgetTool`：名称保留，但职责已改为“真实可得费用汇总 + 缺失项标记”，不再按用户预算固定比例拆分。

## SSE 事件

规划任务流事件：

* `progress`
* `node`
* `warning`
* `assistant_delta`
* `assistant_done`
* `brief_delta`
* `poi_batch`
* `weather_delta`
* `route_delta`
* `budget_delta`
* `day_delta`
* `done`
* `error`
* `heartbeat`

`done.plan` 是结构化行程数据的最终可信来源。`day_delta` 是初版行程草稿，可在前端逐天展示并在 `done.plan` 到达后覆盖。`assistant_delta` 只用于展示面向用户的生成中文本，不能被解析成结构化路线。

每个 task 会在内存 EventBus 中保留最近 100 条事件并带递增 `sequence`，用于新 SSE 连接回放。该缓存是进程内能力，不替代持久化任务 store。

## 边界

* `internal/harness` 只依赖 `TravelPlanner`，不依赖 Eino 实现。
* Eino 代码只放在 `internal/agent/eino` 下。
* HTTP/SSE 代码放在 `internal/travel` 和 `internal/server` 下。
* API key 从环境变量或 `.env` 读取，不硬编码。
