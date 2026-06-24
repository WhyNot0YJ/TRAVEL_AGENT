# API

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
  "pace": "relaxed"
}
```

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
