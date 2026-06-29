# Stage 20: 业务事件级路线流式生成

## 任务目标

本阶段目标是把当前“任务状态流式 + 最终 plan 一次性返回”的体验，升级为“业务事件级流式路线生成”。

核心体验：

```text
正在理解需求
正在查找 POI
正在查询天气
正在计算路线
正在核算预算
第 1 天草稿已生成
第 2 天草稿已生成
第 3 天草稿已生成
完整路线生成完成
```

本阶段不追求 token 级或半截 JSON 级流式。路线规划是结构化产品，前端应该消费稳定的业务事件，而不是消费不完整的 LLM JSON。

## 当前上下文

当前已有 SSE 端点：

```text
GET /api/v1/travel/plans/:task_id/stream
```

当前已有事件类型：

```text
progress
node
warning
error
done
heartbeat
assistant_delta
assistant_done
```

当前问题：

* 完整 `plan` 只在 `done` 事件中一次性返回。
* `assistant_delta` 只尝试流式展示 `summary` 文本，不包含结构化 `days/items/budget`。
* 如果前端连接 SSE 时任务已经完成，后端直接返回 `done`，用户只能看到完整结果一下子出现。
* `EventBus` 不保存历史事件，不支持新连接回放已发生的业务事件。
* 前端 `PlanDetail` 只有 `plan` 存在后才展示完整路线，不支持草稿天数逐步出现。

## 设计原则

* 做业务事件级流式，不做半截 JSON 流式。
* 后端发出的每个事件都必须是完整、可解析、可渲染的 JSON。
* `done` 仍然作为最终权威结果。
* `day_delta` 等中间事件只代表草稿或阶段性结果，最终以 `done.plan` 覆盖。
* 前端必须明确区分“草稿生成中”和“最终路线”。
* 事件必须支持断线重连后的合理恢复。
* 不能让 LLM 的未完成 tool call JSON 直接进入前端 UI。

## 本阶段不做什么

* 不做 token 级结构化 JSON 解析。
* 不把 LLM 未闭合 JSON 直接发给前端。
* 不要求每个 POI、路线、预算字段都实时逐 token 展示。
* 不改变 `POST /api/v1/travel/plans` 的异步任务模型。
* 不移除现有 `done` 事件。
* 不破坏轮询 fallback。
* 不新增大型前端状态管理库。

## 需要阅读的文件

后端：

* `internal/travel/events.go`
* `internal/travel/event_bus.go`
* `internal/travel/service.go`
* `internal/travel/handler.go`
* `internal/agent/metadata.go`
* `internal/agent/eino/graph.go`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/types.go`
* `internal/agent/eino/llm.go`
* `internal/domain/travel.go`

前端：

* `web/src/api/types.ts`
* `web/src/hooks/useTravelPlanStream.ts`
* `web/src/components/PlanProgress.tsx`
* `web/src/components/PlanDetail.tsx`
* `web/src/components/AgentConversation.tsx`
* `web/src/styles.css`

文档：

* `docs/api.md`
* `docs/agent-flow.md`
* `README.md`

## 需要新增或修改的文件

预计修改：

* `internal/travel/events.go`
* `internal/travel/event_bus.go`
* `internal/travel/service.go`
* `internal/travel/handler.go`
* `internal/agent/metadata.go`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/graph.go`
* `internal/agent/eino/types.go`
* `internal/travel/*_test.go`
* `internal/agent/eino/*_test.go`
* `web/src/api/types.ts`
* `web/src/hooks/useTravelPlanStream.ts`
* `web/src/components/PlanProgress.tsx`
* `web/src/components/PlanDetail.tsx`
* `web/src/components/AgentConversation.tsx`
* `web/src/styles.css`
* `docs/api.md`
* `docs/agent-flow.md`
* `README.md`

## 新增事件类型

建议新增事件类型：

```text
brief_delta
poi_batch
weather_delta
route_delta
budget_delta
day_delta
plan_draft
```

保留已有事件：

```text
progress
node
warning
error
done
heartbeat
assistant_delta
assistant_done
```

## 事件语义

### progress

用于粗粒度任务状态：

```json
{
  "type": "progress",
  "task_id": "task_x",
  "status": "running",
  "message": "planner started"
}
```

### node

用于 Eino Graph 节点开始、成功、失败：

```json
{
  "type": "node",
  "node_name": "search_pois",
  "node_status": "succeeded",
  "message": "已找到 10 个候选地点",
  "duration_ms": 380
}
```

节点状态建议：

```text
started
succeeded
failed
skipped
fallback
```

### poi_batch

POI 搜索完成后发送。

```json
{
  "type": "poi_batch",
  "node_name": "search_pois",
  "message": "已找到 10 个杭州候选地点",
  "pois": [
    {
      "name": "牛New寿喜烧(来福士店)",
      "type": "餐饮服务;外国餐厅;日本料理",
      "address": "新业路228号杭州来福士中心六楼",
      "location": "120.212943,30.249170",
      "cost": {
        "amount": 172,
        "status": "available",
        "source": "amap.poi.biz_ext.cost"
      }
    }
  ]
}
```

要求：

* 事件中的 POI 必须是完整对象。
* 如果 POI 价格缺失，`cost.status=unavailable`。
* 如果 POI 来自 fallback/mock，必须带 source 或 warning，不能伪装成真实数据。

### weather_delta

天气查询完成后发送。

```json
{
  "type": "weather_delta",
  "node_name": "get_weather",
  "message": "已获取 3 天天气",
  "weather": [
    {
      "day": 1,
      "condition": "中雨",
      "temperature": "24-30°C",
      "suggestion": "建议携带雨具"
    }
  ]
}
```

### route_delta

路线计算完成后发送。

```json
{
  "type": "route_delta",
  "node_name": "compute_route",
  "message": "已计算 5 段路线",
  "routes": [
    {
      "from": "西湖风景区",
      "to": "牛New寿喜烧(来福士店)",
      "duration_minutes": 24,
      "distance_meters": 6200,
      "mode": "打车",
      "cost": {
        "amount": 28,
        "status": "available",
        "source": "amap.route.taxi_cost"
      }
    }
  ]
}
```

### budget_delta

预算节点完成后发送。

```json
{
  "type": "budget_delta",
  "node_name": "estimate_budget",
  "message": "已核算真实可得费用",
  "budget": {
    "known_total": 1032,
    "complete": false,
    "missing": ["住宿", "往返大交通"]
  }
}
```

要求：

* 只统计真实可得金额。
* 缺失金额不计入 `known_total`。
* 页面文案使用“已知预算”，不能说“完整总预算”。

### day_delta

初版 itinerary 按天生成后发送。

```json
{
  "type": "day_delta",
  "node_name": "optimize_itinerary",
  "message": "第 1 天草稿已生成",
  "day": {
    "day": 1,
    "theme": "钱江新城美食初探 + 西湖漫步",
    "items": [
      {
        "time": "09:30",
        "type": "自然风光",
        "name": "西湖风景区",
        "address": "杭州市西湖区北山街",
        "reason": "杭州标志性自然风光",
        "duration_minutes": 90
      }
    ]
  },
  "draft": true
}
```

要求：

* `day_delta.day` 必须是完整 `TravelDay`。
* 允许后续 `done.plan.days` 覆盖草稿。
* 前端展示时必须标注“生成中”或“草稿”。

### plan_draft

可选事件，用于发送当前已形成的局部 plan 草稿。

```json
{
  "type": "plan_draft",
  "message": "路线草稿已更新",
  "plan": {
    "title": "杭州 3 日路线草稿",
    "summary": "正在完善路线...",
    "days": []
  },
  "draft": true
}
```

本事件不是必须实现。若实现，必须保证对象完整可解析。

### assistant_delta

继续用于 LLM 摘要或解释文本。

要求：

* 只展示为辅助说明，不作为结构化 plan 数据源。
* 不得把 `assistant_delta` 中的文本解析成路线数据。

### done

最终事件，仍然携带完整、校验通过的 `plan`。

```json
{
  "type": "done",
  "status": "succeeded",
  "message": "planner finished",
  "plan": {
    "title": "...",
    "days": []
  }
}
```

要求：

* `done.plan` 是最终权威结果。
* 前端收到后，应将草稿状态标记为完成，并用最终 plan 覆盖草稿。

## 后端事件结构要求

当前 `TaskEvent` 只包含：

```go
Type
RequestID
TaskID
Status
Message
Plan
NodeName
NodeStatus
DurationMs
CreatedAt
```

需要扩展为支持业务 payload：

```go
type TaskEvent struct {
	Type       EventType          `json:"type"`
	RequestID  string             `json:"request_id,omitempty"`
	TaskID     string             `json:"task_id,omitempty"`
	Status     TaskStatus         `json:"status,omitempty"`
	Message    string             `json:"message,omitempty"`
	Plan       *domain.TravelPlan `json:"plan,omitempty"`
	Day        *domain.TravelDay  `json:"day,omitempty"`
	POIs       []eino.MockPOI     `json:"pois,omitempty"`
	Weather    []eino.MockWeather `json:"weather,omitempty"`
	Routes     []eino.MockRoute   `json:"routes,omitempty"`
	Budget     *domain.TravelBudget `json:"budget,omitempty"`
	NodeName   string             `json:"node_name,omitempty"`
	NodeStatus string             `json:"node_status,omitempty"`
	DurationMs int64              `json:"duration_ms,omitempty"`
	Draft      bool               `json:"draft,omitempty"`
	Sequence   int64              `json:"sequence,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
}
```

注意：

* 如果直接引用 `internal/agent/eino` 会造成包边界问题，应优先把可公开 payload 类型放到 `internal/domain`，或定义 DTO 类型，避免 `internal/travel` 直接依赖 Eino 实现细节。
* `TaskEvent` 应保持对前端友好，不暴露不必要的内部状态。

## EventBus 历史回放要求

当前 `EventBus` 不保存历史事件，只发给在线订阅者。

需要新增每个 task 的有限事件缓冲：

```text
每个 task 保留最近 N 条事件，建议 N=100
或保留直到任务 TTL 结束
```

新订阅时：

* 先发送已缓存事件。
* 再订阅后续实时事件。
* 如果任务已经完成，仍然可以回放关键过程事件，然后发送 `done`。

事件需要递增 `sequence`：

```json
{
  "sequence": 12
}
```

前端可用 `sequence` 去重。

如果后续支持 `Last-Event-ID`，可按 sequence 从指定位置恢复；本阶段可以先不实现完整断点续传，但事件结构要预留。

## Eino 节点改造要求

### parse_request

发送：

```text
node started
brief_delta 或 node succeeded
```

可展示已确认的旅行 brief。

### search_pois

发送：

```text
node started
poi_batch
node succeeded
```

POI batch 应包含地点名称、地址、类型、来源、可得价格。

### get_weather

发送：

```text
node started
weather_delta
node succeeded
```

### compute_route

发送：

```text
node started
route_delta
node succeeded
```

### optimize_itinerary

发送：

```text
node started
day_delta day=1
day_delta day=2
day_delta day=3
node succeeded
```

如果 itinerary 是一次性生成的，也应在生成后逐天发布 `day_delta`，不需要人为等待。

### estimate_budget

发送：

```text
node started
budget_delta
node succeeded
```

如果 Stage 19 同时调整 Graph 顺序，应确保 `budget_delta` 基于最终草稿 itinerary。

### generate_plan

发送：

```text
node started
assistant_delta...
node succeeded
```

LLM 阶段可以继续只流摘要文本，不要求流完整 plan JSON。

### validate_plan

发送：

```text
node started
node succeeded
done
```

## Reporter 设计

当前已有：

```go
PlannerEventReporter
LLMDeltaReporter
```

本阶段建议新增业务事件 reporter，而不是让每个节点直接依赖 `travel.Service`。

示例：

```go
type PlannerBusinessEventReporter interface {
	ReportBusinessEvent(ctx context.Context, event PlannerBusinessEvent)
}
```

业务事件结构建议放在 `internal/agent` 或 `internal/domain`，避免 Eino 直接依赖 travel 包。

节点中通过 context 获取 reporter：

```go
reporter := agent.PlannerBusinessEventReporterFromContext(ctx)
```

`internal/travel/service.go` 负责把 business event 转成 `TaskEvent` 并发布到 EventBus。

## 前端状态模型要求

`useTravelPlanStream` 需要从只维护：

```ts
events
plan
assistantText
status
```

扩展为：

```ts
interface StreamState {
  events: TaskEvent[];
  connected: boolean;
  polling: boolean;
  error: string;
  plan?: TravelPlan;
  draftPlan?: TravelPlan;
  draftDays: TravelDay[];
  pois: POI[];
  weather: Weather[];
  routes: Route[];
  budget?: TravelBudget;
  status: string;
  assistantText: string;
}
```

事件处理规则：

* `poi_batch`：更新候选地点区域。
* `weather_delta`：更新天气区域。
* `route_delta`：更新路线计算区域。
* `budget_delta`：更新预算区域。
* `day_delta`：按 `day.day` upsert 到 `draftDays`。
* `done`：设置最终 `plan`，清理或标记草稿为最终。
* `error`：停止流并展示错误。

## 前端 UI 要求

路线生成过程中应展示三个层级：

### 1. 生成进度

显示节点状态：

```text
理解需求：完成
查找地点：完成，找到 10 个
查询天气：完成
计算路线：进行中
核算预算：等待中
生成最终路线：等待中
```

### 2. 阶段数据

可展示：

* 已找到的 POI 数量和前几个名称。
* 天气摘要。
* 已知预算。
* 缺失预算项。

### 3. 草稿路线

当 `day_delta` 到达时，逐天展示：

```text
第 1 天 草稿
第 2 天 草稿
第 3 天 草稿
```

收到 `done` 后：

* 去掉草稿标记。
* 用最终 plan 替换草稿。
* 保留生成过程日志作为折叠区域或进度历史。

## API 文档要求

`docs/api.md` 必须新增所有 SSE 事件：

* `brief_delta`
* `poi_batch`
* `weather_delta`
* `route_delta`
* `budget_delta`
* `day_delta`
* `plan_draft`

每个事件必须包含：

* 事件名
* 字段说明
* 示例 JSON
* 是否草稿
* 是否最终权威结果

## Agent Flow 文档要求

`docs/agent-flow.md` 必须说明：

* Graph 节点顺序。
* 每个节点会产生哪些业务事件。
* 哪些事件是阶段性草稿。
* `done.plan` 是最终结果。
* 不做半截 JSON 流式的原因。

## 测试要求

后端测试：

* EventBus 能缓存并回放最近事件。
* EventBus 新订阅者能收到历史 `day_delta` 和最终 `done`。
* `runTask` 能把 business event 转成 SSE `TaskEvent`。
* `optimize_itinerary` 生成多天时发布多个 `day_delta`。
* `search_pois` 成功后发布 `poi_batch`。
* `estimate_budget` 成功后发布 `budget_delta`。
* 如果任务已完成，`StreamPlan` 不应只返回一个 `done`，应至少能回放缓存的关键事件。
* 如果事件缓存不存在或过期，仍能返回最终 `done`。

前端测试：

* 收到 `day_delta` 后，页面出现对应天数草稿。
* 收到多个 `day_delta` 后，按 day 排序展示。
* 收到重复 `day_delta` 时，按 day 或 sequence 更新而不是重复追加。
* 收到 `budget_delta` 时，预算区域更新。
* 收到 `done` 后，最终 plan 覆盖草稿。
* 断线转轮询时，最终 plan 仍可展示。

建议运行：

```bash
go test ./...
go vet ./...
npm run typecheck
npm run lint
```

前端改动后应使用 Playwright 检查：

* 桌面端生成过程。
* 移动端生成过程。
* 慢速任务下 day_delta 逐步展示。
* 快速任务或缓存任务下历史事件回放。
* 文本不溢出、不重叠。

## 验收标准

本阶段完成后必须满足：

* 用户点击生成后，不再只看到等待和最终一次性结果。
* 至少能看到节点进度、预算阶段结果和按天草稿。
* 完整计划仍通过 `done.plan` 返回。
* 前端能清楚区分草稿和最终路线。
* 如果 SSE 连接晚于任务执行，仍能通过事件缓存看到关键生成过程。
* 如果事件缓存缺失，系统仍能返回最终结果，不阻塞用户。
* 后端不暴露半截 LLM JSON。
* `go test ./...` 通过。
* API 和 Agent Flow 文档同步更新。

## 推荐实施顺序

1. 扩展 `TaskEvent` 类型和前端 `TaskEvent` 类型。
2. 为 EventBus 增加 per-task 有限历史缓存和 sequence。
3. 新增 business event reporter，打通 Eino 节点到 `TravelPlanService`。
4. 在 `search_pois`、`get_weather`、`compute_route`、`estimate_budget`、`optimize_itinerary` 发布业务事件。
5. 前端 `useTravelPlanStream` 支持新事件并维护草稿状态。
6. `PlanProgress` 展示节点和阶段数据。
7. `PlanDetail` 支持草稿天数展示。
8. 更新文档和测试。

## 示例完整事件序列

```text
progress: task created
progress: planner started
node: parse_request started
node: parse_request succeeded
node: search_pois started
poi_batch: 10 pois
node: search_pois succeeded
node: get_weather started
weather_delta: 3 days
node: get_weather succeeded
node: compute_route started
route_delta: 5 routes
node: compute_route succeeded
node: optimize_itinerary started
day_delta: day 1
day_delta: day 2
day_delta: day 3
node: optimize_itinerary succeeded
node: estimate_budget started
budget_delta: known_total 1032, missing hotel/intercity_transport
node: estimate_budget succeeded
node: generate_plan started
assistant_delta: ...
node: generate_plan succeeded
node: validate_plan started
node: validate_plan succeeded
done: final plan
```
