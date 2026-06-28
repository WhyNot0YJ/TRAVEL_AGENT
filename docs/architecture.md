# 架构

## 当前阶段

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

## 前端 H5 流程

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

前端 API 类型放在 `web/src/api/types.ts`，请求 helper 放在 `web/src/api/client.ts`。`VITE_API_BASE_URL` 控制后端基础 URL；为空时，本地 Vite 开发服务器会使用 `/api` 代理到 `http://localhost:8080`。

前端不依赖 Eino、Redis 或 planner 内部实现，只消费任务创建、任务查询和 SSE 事件。

第 15 阶段的 H5 体验把最终计划当作可操作的规划界面，而不是营销落地页。`PlanDetail` 渲染紧凑的站点时间线、warning/fallback 说明、比例预算条，以及可把后续需求送回对话的局部调整动作。`PlanProgress` 消费通用 `node` SSE 事件并展示 planner 节点状态和耗时，同时保持旧的 progress/warning/done 事件兼容。

## 后端分层

* `cmd/harness`：本地评估入口。
* `cmd/server`：Gin HTTP server 入口。
* `internal/config`：读取 HTTP、planner、Redis、限流和 MySQL 配置。
* `internal/server`：router、request id、logging、recovery、CORS。
* `internal/travel`：HTTP DTO、handler、service、任务 store、request_hash 缓存、限流、EventBus。
* `internal/redis`：Redis client 初始化和可用性检查。
* `internal/agent`：`TravelPlanner` 接口、MockPlanner、通用 planner metadata / event reporter。
* `internal/agent/eino`：Eino Graph、LLM、mock/real tools。
* `internal/domain`：稳定业务模型，不依赖 HTTP、Redis、Eino 或外部 API 原始响应。

## 当前存储

MySQL 可选启用。启用并连接成功时，`TravelPlanService` 使用 `MySQLTaskStore` 保存任务状态、请求 hash 和最终 TravelPlan。服务重启后可继续查询已完成任务。

Redis 仍用于限流；当未启用 MySQL 时，Redis 也可继续作为任务缓存和 request hash 索引。Redis 未配置或不可用时，开发环境自动降级为内存任务 store 和内存限流。内存模式下服务重启后任务会丢失。

职责边界：

* MySQL：长期任务状态、最终计划、后续 planner run/event trace。
* Redis：短期缓存、request hash 复用、限流计数。
* 内存：本地开发 fallback。

## SSE 流程

```text
POST /travel/plans
  -> create task
  -> TravelPlanService publishes progress
  -> background planner updates task
  -> EventBus publishes warning/error/done
  -> GET /travel/plans/:task_id/stream writes SSE events
```

SSE handler 只依赖 `TravelPlanService` 和 `EventBus`，不直接依赖 Eino 内部实现。

## 可观测性

Request id 来源于 `X-Request-ID`，未提供时由 middleware 生成，并写回响应头。`travel.Handler` 将 request id 放入 context，`TravelPlanService` 保存到 task 并在后台 planner run 中继续传递。

节点级事件通过 `internal/agent.PlannerEventReporter` 从 context 传入 planner。Eino 只上报通用 `PlannerTraceEvent`，`internal/travel` 将其转换为 SSE `node` 事件和结构化日志，因此 `internal/travel` 不依赖 Eino 包。

结构化日志字段保持稳定：

* `request_id`
* `task_id`
* `node`
* `duration_ms`
* `status`
* `error`
