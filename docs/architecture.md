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
