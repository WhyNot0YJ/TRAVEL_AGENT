# Stage 7：SSE 流式接口

## 任务目标

为后端新增 SSE 流式接口：

* 实现 `GET /api/v1/travel/plans/:task_id/stream`
* 将任务状态、Eino 执行过程、Tool 执行过程推送给前端
* 支持事件类型：`progress`、`warning`、`error`、`done`
* 支持客户端断开
* 支持超时和错误处理
* 提供简单测试方式

## 当前前置条件

第 1 到 6 阶段已完成：Gin API、Redis 缓存、限流、任务状态、异步任务创建与查询均可运行。

## 本阶段不做什么

* 不做 React 前端页面
* 不接数据库
* 不实现 WebSocket
* 不引入大型消息队列
* 不破坏现有 POST / GET API
* 不让 SSE handler 直接依赖 Eino 具体实现细节
* 不移除普通任务查询接口

## 需要阅读的文件

* `AGENTS.md`
* `README.md`
* `docs/api.md`
* `docs/architecture.md`
* `docs/agent-flow.md`
* `internal/server/*`
* `internal/travel/*`
* `internal/agent/eino/*`
* `skills/agent-feature-dev.md`：如果不存在，说明该文件当前不存在，后续阶段需要创建

## 需要新增或修改的文件

预计新增或修改：

* `internal/travel/events.go`
* `internal/travel/event_bus.go`
* `internal/travel/task.go`
* `internal/travel/service.go`
* `internal/travel/handler.go`
* `internal/server/router.go`
* `internal/agent/eino/callbacks.go`
* `internal/agent/eino/planner.go`
* `internal/agent/eino/nodes.go`
* `internal/travel/*_test.go`
* `README.md`
* `docs/api.md`
* `docs/architecture.md`
* `docs/agent-flow.md`

## 实现要求

* SSE endpoint：`GET /api/v1/travel/plans/:task_id/stream`。
* 响应 header 必须适合 SSE：`Content-Type: text/event-stream`、`Cache-Control: no-cache`、`Connection: keep-alive`。
* 事件 payload 使用 JSON。
* 至少支持 `progress`、`warning`、`error`、`done`。
* 客户端断开时要停止写入并清理订阅。
* 任务已完成时，新连接应至少返回当前状态和 done/error。
* 增加 stream timeout 或 heartbeat，避免连接无声挂死。
* Eino 执行过程可以通过 callback / event reporter 抽象上报，不要把 HTTP SSE 逻辑写进 `internal/agent/eino`。
* MockPlanner 路径也应能产生基本 progress/done 事件。
* 单元测试覆盖事件格式、订阅清理、已完成任务 stream。

## 文档更新要求

* 更新 `docs/api.md`：说明 SSE endpoint、事件类型、payload 示例、错误处理。
* 更新 `docs/agent-flow.md`：说明 Agent/Tool 事件如何上报。
* 更新 `docs/architecture.md`：说明 EventBus / TaskStore / SSE Handler。
* 更新 `README.md`：给出 curl 测试方式。
* 不更新 `docs/database.md`，除非新增 Redis key 设计。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
```

手动验证：

```bash
go run ./cmd/server
curl -N http://localhost:8080/api/v1/travel/plans/{task_id}/stream
```

## 验收标准

* SSE endpoint 可连接。
* 能收到 progress / done 事件。
* 任务失败时能收到 error 事件。
* 客户端断开不会导致 goroutine 泄漏或持续写入。
* 现有 POST / GET API 仍可用。
* Harness 不受影响。
* 文档包含可复制的 SSE 测试方式。

## 最终回复要求

完成后请输出：

1. 修改了哪些文件
2. 新增了哪些文件
3. 做了哪些设计决策
4. 跑了哪些测试
5. 测试是否通过
6. SSE 接口如何验证
7. 风险和未完成事项
