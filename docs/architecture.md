# Architecture

## Current Stage

当前阶段包含 Evaluation Harness、Eino Planner、可选外部 Tools，以及 Gin 异步任务 HTTP API。

```text
HTTP Client
  -> Gin Router
  -> Travel Handler
  -> TravelPlanService
  -> EventBus
  -> TaskStore (MySQL, Redis, or memory fallback)
  -> RateLimiter (Redis or memory fallback)
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
* `internal/config`：读取 HTTP、planner、Redis、限流和 MySQL 配置。
* `internal/server`：router、request id、logging、recovery、CORS。
* `internal/travel`：HTTP DTO、handler、service、任务 store、request_hash 缓存、限流、EventBus。
* `internal/redis`：Redis client 初始化和可用性检查。
* `internal/agent`：`TravelPlanner` 接口和 MockPlanner。
* `internal/agent/eino`：Eino Graph、LLM、mock/real tools。
* `internal/domain`：稳定业务模型，不依赖 HTTP、Redis、Eino 或外部 API 原始响应。

## Current Storage

MySQL 可选启用。启用并连接成功时，`TravelPlanService` 使用 `MySQLTaskStore` 保存任务状态、请求 hash 和最终 TravelPlan。服务重启后可继续查询已完成任务。

Redis 仍用于限流；当未启用 MySQL 时，Redis 也可继续作为任务缓存和 request hash 索引。Redis 未配置或不可用时，开发环境自动降级为内存任务 store 和内存限流。内存模式下服务重启后任务会丢失。

职责边界：

* MySQL：长期任务状态、最终计划、后续 planner run/event trace。
* Redis：短期缓存、request hash 复用、限流计数。
* 内存：本地开发 fallback。

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
