# Stage 14：观测性与可靠性

## 任务目标

让一次旅行规划请求可以从 HTTP request、任务状态、Eino 节点、Tool/LLM 调用到最终 SSE 事件完整追踪。新增结构化日志、节点级耗时、错误分类和基础指标，同时保持系统轻量。

## 当前上下文

当前已有：

* Gin middleware：request id、logging、recovery、CORS
* TravelPlanService 任务状态转换
* EventBus
* SSE progress/warning/error/done/heartbeat
* Eino callbacks 文件存在，但节点级事件尚未完整推送到 SSE

## 不做什么

* 不引入重量级可观测平台，除非用户明确批准。
* 不输出 API Key、用户敏感信息或完整外部原始响应。
* 不让日志格式随意散落在各层。
* 不破坏现有 SSE contract。

## 需要阅读的文件

* `docs/architecture.md`
* `docs/api.md`
* `docs/agent-flow.md`
* `internal/server/middleware.go`
* `internal/travel/events.go`
* `internal/travel/event_bus.go`
* `internal/travel/service.go`
* `internal/travel/handler.go`
* `internal/agent/eino/callbacks.go`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/types.go`

## 实现要求

* request id 应贯穿 HTTP handler、service、planner run 和日志。
* 定义节点级事件类型或内部 event reporter，避免 `internal/travel` 直接依赖 Eino。
* 节点事件可包含：
  * node name
  * status
  * started/finished time
  * duration
  * warning/fallback reason
* SSE 对外新增事件时必须保持向后兼容。
* 结构化日志字段命名稳定，例如 `request_id`、`task_id`、`planner`、`node`、`duration_ms`、`status`。
* panic recovery 应继续更新任务失败状态并推送 error 事件。

## 文档更新要求

* 更新 `docs/api.md`：如新增 SSE event type，需要说明。
* 更新 `docs/architecture.md`：说明 observability/event reporter 的位置。
* 更新 `docs/agent-flow.md`：说明节点级事件与 Eino callback 的关系。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner eino
```

如果改动 SSE，需要尽量运行：

```bash
cd web
npm run typecheck
npm run lint
npm run harness:ui
```

## 验收标准

* 单次请求能通过 request id/task id 串起日志。
* Eino 节点耗时可观测。
* SSE 仍兼容前端。
* 失败和 panic 路径可被稳定记录。
