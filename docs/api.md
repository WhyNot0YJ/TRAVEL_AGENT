# API

## 2026-06 Chat-First Contract

前端现在以聊天为主入口：先收集旅行需求，信息齐全后在助手消息里展示“生成行程”按钮。

### POST /api/v1/travel/chat/stream

用于用户信息收集阶段，返回 `text/event-stream`。测试模式使用本地规则提取；真实模式使用配置的 LLM 提取，失败时 fallback 到规则提取。

Request:

```json
{
  "message": "上海出发，杭州 3 天，预算 3000，喜欢美食和自然风光",
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["美食"],
  "transport_mode": "train_taxi",
  "pace": "relaxed",
  "test_mode": true,
  "agent_mode": "quick"
}
```

`agent_mode` 可选：`quick` 为快速模式，`expert` 为专家模式。`test_mode=true` 时始终走本地规则、Mock tools 和 deterministic planner；`test_mode=false` 时按服务端 LLM 与真实工具配置执行。

Stream events:

* `assistant_delta`：助手回复增量文本。`message` 字段长度**不再保证等长**——启用流式（`TRAVEL_AGENT_LLM_STREAM_ENABLED=true`，默认）时按 LLM token 增量到达，单帧通常 1~6 个汉字；前端必须**累加**而不是替换。
* `assistant_done`：助手回复文本结束。
* `done`：最终结构化 `ChatResponse`。
* `error`：处理失败。

Final `done` payload:

```json
{
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["美食", "自然风光"],
  "transport_mode": "train_taxi",
  "pace": "relaxed",
  "reply": "信息已经齐全，可以生成行程了。",
  "missing": [],
  "is_complete": true,
  "agent_mode": "quick"
}
```

### POST /api/v1/travel/plans

新增 `agent_mode` 字段。`request_hash` 会同时包含 `test_mode` 和 `agent_mode`，避免测试/真实、快速/专家结果互相复用。

```json
{
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["美食", "自然风光"],
  "transport_mode": "train_taxi",
  "pace": "relaxed",
  "test_mode": false,
  "agent_mode": "expert"
}
```

### GET /api/v1/travel/plans/:task_id/stream

规划任务 SSE 继续返回 `progress`、`node`、`warning`、`error`、`done`、`heartbeat`，并新增：

* `assistant_delta`：生成过程中的助手文本增量。启用流式时，规划链路在结构化 plan 落库前会通过 LLM streaming 推送一段"旁白"（例如"正在为你规划上海到杭州的 3 天行程..."），单帧长度由 LLM token 决定，**不保证等长**。
* `assistant_done`：助手文本结束。

前端应以最后的 `done.plan` 作为最终结构化行程，以 `assistant_delta` 只作为生成中的可读反馈。`TRAVEL_AGENT_LLM_STREAM_ENABLED=false` 时旁白回退到等长 chunkText 切片（向后兼容）。

## Current Stage

当前 HTTP API 使用异步任务模式，支持 request hash 去重、任务状态查询、缓存复用、限流和 SSE 流式事件。Redis 可用时使用 Redis；Redis 未配置或不可用时，开发环境降级到内存实现。

## Environment

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

## Error Response

```json
{
  "request_id": "20260623193000.000000000",
  "code": "invalid_request",
  "message": "days is required"
}
```

Common status:

* `400`：请求无效
* `404`：任务不存在
* `429`：触发限流
* `500`：服务内部错误

## POST /api/v1/travel/plans

创建异步路线规划任务。相同请求会通过 `request_hash` 复用已有任务或命中缓存。

Request:

```json
{
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["自然风光", "美食"],
  "transport_mode": "train_taxi",
  "pace": "relaxed",
  "test_mode": true
}
```

`test_mode` 可选，默认 `false`。前端开启测试模式时传 `true`，后端会在本次请求中强制使用本地规则解析、deterministic plan generator 和 Mock Tools；关闭测试模式时使用服务端环境变量配置的真实 LLM / real tools，如果 key 或 provider 不可用仍会按原有 fallback 策略降级。

Response:

```json
{
  "task_id": "task_xxx",
  "request_hash": "sha256...",
  "status": "pending",
  "cached": false
}
```

Status:

* `202`：任务已创建或复用
* `400`：请求 JSON 或必填字段无效
* `429`：请求过于频繁
* `500`：任务创建失败

## GET /api/v1/travel/plans/:task_id

查询任务状态和结果。

Response:

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

Status values:

* `pending`
* `running`
* `succeeded`
* `failed`

When failed:

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

Headers:

```text
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

Event types:

* `progress`
* `node`
* `warning`
* `error`
* `done`
* `heartbeat`

Event payload:

```json
{
  "type": "progress",
  "task_id": "task_xxx",
  "status": "running",
  "message": "planner started",
  "created_at": "2026-06-23T10:00:00Z"
}
```

Node event payload:

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

Example:

```bash
curl -N http://localhost:8080/api/v1/travel/plans/{task_id}/stream
```

## Frontend Integration Notes

The React H5 client uses the existing contract without additional endpoints. The UI is conversational, but the frontend converts the collected brief into the same structured travel task payload:

* `POST /api/v1/travel/plans` creates a task.
* `GET /api/v1/travel/plans/:task_id/stream` streams progress and the final `done` event.
* `GET /api/v1/travel/plans/:task_id` is used as a polling fallback when SSE disconnects.

Set `VITE_API_BASE_URL` for the web app when the frontend is not served behind the same origin as the API. In local Vite development, leaving it empty uses the dev-server `/api` proxy.

The frontend mode switch maps to the same `test_mode` request field. Test mode uses local deterministic behavior for demos and repeatable checks; real mode depends on backend `TRAVEL_AGENT_LLM_*` and `TRAVEL_AGENT_TOOL_MODE=real` / `TRAVEL_AGENT_AMAP_*` configuration.
