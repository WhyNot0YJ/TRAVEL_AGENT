# API 文档

## 2026-06 聊天优先契约

前端现在以聊天为主入口：先收集旅行需求，信息齐全后展示 Travel Brief 确认卡和“生成行程”按钮。必填项为出发地、目的地、天数、预算、兴趣偏好和出行人数；可选项缺失时使用稳定默认值。

### POST /api/v1/travel/chat/stream

用于用户信息收集阶段，返回 `text/event-stream`。测试模式使用本地规则提取；真实模式使用配置的 LLM 提取，失败时 fallback 到规则提取。

请求：

```json
{
  "message": "上海出发，杭州 3 天，2 人，预算 3000，喜欢美食和自然风光",
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["美食"],
  "travelers": 2,
  "date_range": "任意",
  "transport_mode": "任意",
  "pace": "适中",
  "walking_tolerance": "任意",
  "hotel_area": "任意",
  "must_visit": [],
  "avoid": [],
  "traveler_type": "无要求",
  "budget_type": "总预算",
  "budget_includes": ["住宿", "餐饮", "门票", "市内交通"],
  "test_mode": true,
  "agent_mode": "quick"
}
```

`agent_mode` 可选：`quick` 为快速模式，`expert` 为专家模式。`test_mode=true` 时始终走本地规则、Mock tools 和 deterministic planner；`test_mode=false` 时按服务端 LLM 与真实工具配置执行。

流式事件：

* `assistant_delta`：助手回复增量文本。`message` 字段长度**不再保证等长**——启用流式（`TRAVEL_AGENT_LLM_STREAM_ENABLED=true`，默认）时按 LLM token 增量到达，单帧通常 1~6 个汉字；前端必须**累加**而不是替换。
* `assistant_done`：助手回复文本结束。
* `done`：最终结构化 `ChatResponse`。
* `error`：处理失败。

最终 `done` payload：

```json
{
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["美食", "自然风光"],
  "travelers": 2,
  "date_range": "任意",
  "transport_mode": "高铁 + 打车",
  "pace": "轻松",
  "walking_tolerance": "任意",
  "hotel_area": "任意",
  "must_visit": [],
  "avoid": [],
  "traveler_type": "无要求",
  "budget_type": "总预算",
  "budget_includes": ["住宿", "餐饮", "门票", "市内交通"],
  "reply": "信息已经齐全，可以生成行程了。",
  "missing": [],
  "is_complete": true,
  "agent_mode": "quick"
}
```

### POST /api/v1/travel/plans

新增 `agent_mode` 字段。`request_hash` 会同时包含 `test_mode`、`agent_mode` 和完整 Travel Brief，避免不同人数、避开内容、必去地点或步行强度复用同一结果。

```json
{
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["美食", "自然风光"],
  "travelers": 2,
  "date_range": "任意",
  "transport_mode": "高铁 + 打车",
  "pace": "轻松",
  "walking_tolerance": "任意",
  "hotel_area": "任意",
  "must_visit": ["西湖"],
  "avoid": ["网红店"],
  "traveler_type": "情侣",
  "budget_type": "总预算",
  "budget_includes": ["住宿", "餐饮", "门票", "市内交通"],
  "test_mode": false,
  "agent_mode": "expert"
}
```

`POST /api/v1/travel/chat` 仍保留为兼容接口，但前端主链路使用 `POST /api/v1/travel/chat/stream`。`GET /api/v1/travel/plans/:task_id` 继续作为 SSE 断线后的任务查询兜底。

### GET /api/v1/travel/plans/:task_id/stream

规划任务 SSE 继续返回 `progress`、`node`、`warning`、`error`、`done`、`heartbeat`，并新增：

* `assistant_delta`：生成过程中的助手文本增量。启用流式时，规划链路在结构化 plan 落库前会通过 LLM streaming 推送一段"旁白"（例如"正在为你规划上海到杭州的 3 天行程..."），单帧长度由 LLM token 决定，**不保证等长**。
* `assistant_done`：助手文本结束。

前端应以最后的 `done.plan` 作为最终结构化行程，以 `assistant_delta` 只作为生成中的可读反馈。`TRAVEL_AGENT_LLM_STREAM_ENABLED=false` 时旁白回退到等长 chunkText 切片（向后兼容）。

## 当前阶段

当前 HTTP API 使用异步任务模式，支持 request hash 去重、任务状态查询、缓存复用、限流和 SSE 流式事件。Redis 可用时使用 Redis；Redis 未配置或不可用时，开发环境降级到内存实现。

## 环境变量

```text
TRAVEL_AGENT_HTTP_ADDR=:8080
TRAVEL_AGENT_PLANNER=mock|eino
TRAVEL_AGENT_REDIS_ADDR=localhost:6379
TRAVEL_AGENT_REDIS_PASSWORD=
TRAVEL_AGENT_REDIS_DB=0
TRAVEL_AGENT_CACHE_TTL_SECONDS=1800
TRAVEL_AGENT_RATE_LIMIT_PER_MINUTE=60
TRAVEL_AGENT_SQL_ENABLED=false
TRAVEL_AGENT_SQL_DSN=
TRAVEL_AGENT_SQL_MAX_OPEN_CONNS=10
TRAVEL_AGENT_SQL_MAX_IDLE_CONNS=5
TRAVEL_AGENT_SQL_CONN_MAX_LIFETIME_SECONDS=1800
```

MySQL 启用后不改变 HTTP API contract。任务创建、查询和 SSE 语义保持不变；区别是任务和最终 plan 可跨服务重启保留。

## 错误响应

```json
{
  "request_id": "20260623193000.000000000",
  "code": "invalid_request",
  "message": "days is required"
}
```

常见状态码：

* `400`：请求无效
* `404`：任务不存在
* `429`：触发限流
* `500`：服务内部错误

## POST /api/v1/travel/plans

创建异步路线规划任务。相同请求会通过 `request_hash` 复用已有任务或命中缓存。

请求：

```json
{
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["自然风光", "美食"],
  "travelers": 2,
  "transport_mode": "高铁 + 打车",
  "pace": "轻松",
  "test_mode": true
}
```

`travelers`、`interests` 等必填字段缺失时返回 `400 invalid_request`。可选 brief 字段缺失时后端会使用默认值：`date_range=任意`、`pace=适中`、`transport_mode=任意`、`walking_tolerance=任意`、`hotel_area=任意`、`traveler_type=无要求`、`budget_type=总预算`、`budget_includes=["住宿","餐饮","门票","市内交通"]`。预算包含项展示时默认说明“不含往返大交通”。

`test_mode` 可选，默认 `false`。前端开启测试模式时传 `true`，后端会在本次请求中强制使用本地规则解析、deterministic plan generator 和 Mock Tools；关闭测试模式时使用服务端环境变量配置的真实 LLM / real tools，如果 key 或 provider 不可用仍会按原有 fallback 策略降级。

响应：

```json
{
  "task_id": "task_xxx",
  "request_hash": "sha256...",
  "status": "pending",
  "cached": false
}
```

状态码：

* `202`：任务已创建或复用
* `400`：请求 JSON 或必填字段无效
* `429`：请求过于频繁
* `500`：任务创建失败

## GET /api/v1/travel/plans/:task_id

查询任务状态和结果。

响应：

```json
{
  "task_id": "task_xxx",
  "request_hash": "sha256...",
  "status": "succeeded",
  "plan": {
    "title": "杭州3日旅行规划",
    "summary": "...",
    "days": [],
    "budget": {
      "transport": 0,
      "food": 0,
      "hotel": 0,
      "ticket": 0,
      "total": 0
    },
    "warnings": []
  },
  "created_at": "2026-06-23T10:00:00Z",
  "updated_at": "2026-06-23T10:00:01Z"
}
```

状态值：

* `pending`
* `running`
* `succeeded`
* `failed`

失败时：

```json
{
  "task_id": "task_xxx",
  "request_hash": "sha256...",
  "status": "failed",
  "error": "planner error"
}
```

## GET /api/v1/travel/plans/:task_id/stream

订阅任务事件流。

响应头：

```text
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

事件类型：

* `progress`
* `node`
* `warning`
* `error`
* `done`
* `heartbeat`

事件 payload：

```json
{
  "type": "progress",
  "task_id": "task_xxx",
  "status": "running",
  "message": "planner started",
  "created_at": "2026-06-23T10:00:00Z"
}
```

节点事件 payload：

```json
{
  "type": "node",
  "request_id": "20260624115200.000000000",
  "task_id": "task_xxx",
  "status": "running",
  "message": "loaded 5 pois",
  "node_name": "SearchPOIsToolNode",
  "node_status": "success",
  "duration_ms": 12,
  "created_at": "2026-06-24T11:52:00Z"
}
```

新增 `node` 事件保持向后兼容；已有前端可继续只处理 `progress`、`warning`、`error` 和 `done`。

如果任务已完成，新连接会立即返回 `done` 或 `error`。

示例：

```bash
curl -N http://localhost:8080/api/v1/travel/plans/{task_id}/stream
```

## 前端接入说明

React H5 客户端复用现有契约，不需要额外接口。UI 是对话式体验，但前端会把已收集的需求转换成同一份结构化旅行任务 payload：

* `POST /api/v1/travel/plans` 创建任务。
* `GET /api/v1/travel/plans/:task_id/stream` 流式返回进度和最终 `done` 事件。
* `GET /api/v1/travel/plans/:task_id` 在 SSE 断开时作为轮询 fallback。

当前端和 API 不在同源服务下时，需要为 web app 设置 `VITE_API_BASE_URL`。本地 Vite 开发时，留空会使用开发服务器的 `/api` 代理。

前端模式开关映射到同一个 `test_mode` 请求字段。测试模式使用本地确定性行为，适合演示和可重复检查；真实模式依赖后端 `TRAVEL_AGENT_LLM_*` 和 `TRAVEL_AGENT_TOOL_MODE=real` / `TRAVEL_AGENT_AMAP_*` 配置。
