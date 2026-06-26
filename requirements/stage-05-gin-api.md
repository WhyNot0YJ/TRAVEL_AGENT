# Stage 5：接入 Gin API

## 任务目标

新增 Go Gin HTTP API：

* 新增 Gin 服务入口
* 新增 `TravelPlanService`
* Gin Handler 调用 `TravelPlanner` 接口
* 实现 `POST /api/v1/travel/plans`
* 实现 `GET /api/v1/travel/plans/:id`
* 本阶段同步返回结果
* 暂时使用内存存储保存最近生成的 plan

## 当前前置条件

第 1 到 4 阶段已完成：Harness、MockPlanner、EinoTravelPlanner、mock/real tools、LLM/fallback 均可运行。

## 本阶段不做什么

* 不接 Redis
* 不接数据库
* 不做 SSE
* 不做 React 前端
* 不实现异步任务状态
* 不实现用户登录
* 不实现复杂权限系统
* 不把 Gin handler 直接绑定 Eino 具体实现
* 不破坏 Harness

## 需要阅读的文件

* `AGENTS.md`
* `README.md`
* `docs/api.md`
* `docs/architecture.md`
* `docs/agent-flow.md`
* `internal/domain/travel.go`
* `internal/agent/planner.go`
* `cmd/harness/main.go`
* `internal/agent/eino/*`
* `skills/agent-feature-dev.md`：如果不存在，说明该文件当前不存在，后续阶段需要创建

## 需要新增或修改的文件

预计新增或修改：

* `cmd/server/main.go`
* `internal/config/config.go`
* `internal/server/router.go`
* `internal/server/middleware.go`
* `internal/travel/dto.go`
* `internal/travel/service.go`
* `internal/travel/handler.go`
* `internal/travel/store_memory.go`
* `internal/travel/*_test.go`
* `go.mod`
* `go.sum`
* `README.md`
* `docs/api.md`
* `docs/architecture.md`
* `Makefile`

## 实现要求

* 引入 Gin，但不要引入额外大型框架。
* `TravelPlanService` 依赖 `agent.TravelPlanner` 接口。
* Handler 不直接依赖 `MockPlanner` 或 `EinoTravelPlanner`。
* 支持通过环境变量选择 planner：`TRAVEL_AGENT_PLANNER=mock|eino`。
* HTTP 请求 DTO 与 domain 模型分离，但字段可以映射到 `domain.TravelRequest`。
* `POST /api/v1/travel/plans` 接收旅行规划请求、校验必填字段、同步调用 planner、返回 plan id 和 plan 内容。
* `GET /api/v1/travel/plans/:id` 从内存 store 查询已生成 plan，找不到返回 404。
* 增加基础 request id / recovery / logging middleware。
* 错误响应结构统一。
* 服务端口来自环境变量，例如 `TRAVEL_AGENT_HTTP_ADDR`，默认 `:8080`。
* 不要改坏 `cmd/harness`。
* 单元测试覆盖 service、handler、参数校验、404。

## 文档更新要求

* 更新 `docs/api.md`：说明 endpoints、请求响应 JSON、错误码。
* 更新 `docs/architecture.md`：说明 Gin -> Service -> TravelPlanner 的调用关系。
* 更新 `README.md`：说明如何启动 server 和调用 API。
* 如果 Agent 流程未变化，不需要改 `docs/agent-flow.md`。
* 不更新 `docs/database.md`，因为本阶段没有数据库结构。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
```

可手动启动服务：

```bash
go run ./cmd/server
```

并用 curl 验证 POST / GET API。

## 验收标准

* Harness 仍可运行。
* Gin server 可启动。
* POST API 可同步返回 TravelPlan。
* GET API 可查询已生成 plan。
* Handler 和 Service 复用 `TravelPlanner` 接口。
* `docs/api.md` 已更新。

## 最终回复要求

完成后请输出：

1. 修改了哪些文件
2. 新增了哪些文件
3. 做了哪些设计决策
4. 跑了哪些测试
5. 测试是否通过
6. 接口如何验证
7. 风险和未完成事项
