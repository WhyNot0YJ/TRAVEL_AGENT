# Architecture

## Current Stage

当前阶段包含 Evaluation Harness、Eino Planner、可选外部 Tools，以及 Gin 同步 HTTP API。

```text
HTTP Client
  -> Gin Router
  -> Travel Handler
  -> TravelPlanService
  -> agent.TravelPlanner
  -> MockPlanner / EinoTravelPlanner
  -> MemoryPlanStore
```

## Backend Layers

* `cmd/harness`：本地评估入口。
* `cmd/server`：Gin HTTP server 入口。
* `internal/config`：读取 HTTP 地址和 planner 类型。
* `internal/server`：router、request id、logging、recovery、CORS。
* `internal/travel`：HTTP DTO、handler、service、内存 store。
* `internal/agent`：`TravelPlanner` 接口和 MockPlanner。
* `internal/agent/eino`：Eino Graph、LLM、mock/real tools。
* `internal/domain`：稳定业务模型，不依赖 HTTP、Redis、Eino 或外部 API 原始响应。

## Current Storage

阶段 5 只使用内存 store 保存最近生成的 plan。服务重启后数据会丢失。

Redis、异步任务状态、SSE 和真实数据库持久化将在后续阶段接入。
