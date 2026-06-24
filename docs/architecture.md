# Architecture

## Current Stage

当前阶段包含 Evaluation Harness、Eino Planner、可选外部 Tools，以及 Gin 异步任务 HTTP API。

```text
HTTP Client
  -> Gin Router
  -> Travel Handler
  -> TravelPlanService
  -> EventBus
  -> TaskStore / RateLimiter (Redis or memory fallback)
  -> agent.TravelPlanner
  -> MockPlanner / EinoTravelPlanner
```

## Frontend H5 Flow

当前仓库已在 `web` 下接入 React + TypeScript H5 client。前端采用 conversation-first 体验：通过聊天收集旅行意图，维护实时 brief panel，并在必要字段齐全后提交结构化任务。

```text
React H5
  -> typed API client
  -> POST /api/v1/travel/plans
  -> GET /api/v1/travel/plans/:task_id/stream
  -> optional GET /api/v1/travel/plans/:task_id polling fallback
  -> Gin Router
  -> TravelPlanService
  -> EventBus / TaskStore / agent.TravelPlanner
```

The frontend keeps API types in `web/src/api/types.ts` and request helpers in `web/src/api/client.ts`. `VITE_API_BASE_URL` controls the backend base URL. When it is empty, local Vite dev uses its `/api` proxy to `http://localhost:8080`.

The frontend does not depend on Eino, Redis, or planner internals. It only consumes task creation, task lookup, and SSE events.

## Backend Layers

* `cmd/harness`：本地评估入口。
* `cmd/server`：Gin HTTP server 入口。
* `internal/config`：读取 HTTP 地址和 planner 类型。
* `internal/server`：router、request id、logging、recovery、CORS。
* `internal/travel`：HTTP DTO、handler、service、任务 store、request_hash 缓存、限流、EventBus。
* `internal/redis`：Redis client 初始化和可用性检查。
* `internal/agent`：`TravelPlanner` 接口和 MockPlanner。
* `internal/agent/eino`：Eino Graph、LLM、mock/real tools。
* `internal/domain`：稳定业务模型，不依赖 HTTP、Redis、Eino 或外部 API 原始响应。

## Current Storage

Redis 可用时，任务、request_hash 索引和限流计数写入 Redis。

Redis 未配置或不可用时，开发环境自动降级为内存任务 store 和内存限流。内存模式下服务重启后任务会丢失。

真实数据库持久化将在后续阶段接入。

## SSE Flow

```text
POST /travel/plans
  -> create task
  -> TravelPlanService publishes progress
  -> background planner updates task
  -> EventBus publishes warning/error/done
  -> GET /travel/plans/:task_id/stream writes SSE events
```

SSE handler 只依赖 `TravelPlanService` 和 `EventBus`，不直接依赖 Eino 内部实现。
