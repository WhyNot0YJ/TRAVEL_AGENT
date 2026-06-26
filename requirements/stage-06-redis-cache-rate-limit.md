# Stage 6：Redis 缓存、限流、任务状态

## 任务目标

在 Gin API 基础上接入 Redis 能力：

* 增加请求缓存
* 增加 `request_hash` 去重
* 增加用户 / IP 限流
* 增加任务状态管理
* 支持异步任务创建
* 支持 `task_id`
* Gin API 返回任务状态

## 当前前置条件

第 1 到 5 阶段已完成：Harness、Gin API、`TravelPlanService`、POST / GET plan API 均可运行，当前仍是同步返回。

## 本阶段不做什么

* 不做 SSE
* 不做 React 前端
* 不接真实数据库持久化
* 不实现用户登录系统
* 不实现分布式任务队列
* 不破坏同步 Harness
* 不让 Redis 逻辑进入 `internal/domain`

## 需要阅读的文件

* `AGENTS.md`
* `README.md`
* `docs/api.md`
* `docs/architecture.md`
* `docs/database.md`
* `internal/travel/*`
* `internal/server/*`
* `internal/config/*`
* `internal/agent/planner.go`
* `skills/agent-feature-dev.md`：如果不存在，说明该文件当前不存在，后续阶段需要创建

## 需要新增或修改的文件

预计新增或修改：

* `internal/config/config.go`
* `internal/redis/client.go`
* `internal/travel/task.go`
* `internal/travel/task_store.go`
* `internal/travel/cache.go`
* `internal/travel/rate_limiter.go`
* `internal/travel/service.go`
* `internal/travel/handler.go`
* `internal/travel/dto.go`
* `internal/server/router.go`
* `internal/server/middleware.go`
* `internal/travel/*_test.go`
* `README.md`
* `docs/api.md`
* `docs/architecture.md`
* `docs/database.md`
* `Makefile`

## 实现要求

* Redis 配置来自环境变量：`TRAVEL_AGENT_REDIS_ADDR`、`TRAVEL_AGENT_REDIS_PASSWORD`、`TRAVEL_AGENT_REDIS_DB`、`TRAVEL_AGENT_CACHE_TTL_SECONDS`、`TRAVEL_AGENT_RATE_LIMIT_PER_MINUTE`。
* Redis 不可用时，开发环境可降级为内存实现，但要在日志和文档中说明。
* 生成稳定 `request_hash`：基于规范化后的请求 JSON，相同请求命中缓存或复用任务。
* 新增任务状态：`pending`、`running`、`succeeded`、`failed`。
* POST API 改为创建任务，返回 `task_id`、`request_hash`、`status`、可选 cached 标记。
* GET API 根据 `task_id` 返回任务状态和结果。
* 后台 goroutine 执行 planner，注意 context、panic recovery、状态更新。
* 增加 IP 限流 middleware 或 service 级限流。
* 错误响应保持统一。
* 不要把任务状态管理写进 Harness。
* 单元测试覆盖 request_hash、缓存命中、任务状态转换、限流、Redis fallback。

## 文档更新要求

* 更新 `docs/api.md`：说明异步任务 API、状态字段、错误码、限流响应。
* 更新 `docs/architecture.md`：说明 Gin、Service、Redis、Planner 的关系。
* 更新 `docs/database.md`：说明 Redis key 设计和 TTL；如果没有 SQL 表，不要编造 SQL 表。
* 更新 `README.md`：说明 Redis 环境变量和运行方式。
* 如果 Agent 流程未变化，不需要改 `docs/agent-flow.md`。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
```

如果本地有 Redis，可运行 server 并手动验证创建任务、查询任务、重复请求缓存和限流。

## 验收标准

* Redis 配置来自环境变量。
* Redis 不可用时有清晰降级或错误策略。
* POST API 返回 `task_id` 和任务状态。
* GET API 可查询 pending/running/succeeded/failed。
* 相同请求可通过 `request_hash` 去重或缓存。
* 限流生效。
* Harness 不受影响。

## 最终回复要求

完成后请输出：

1. 修改了哪些文件
2. 新增了哪些文件
3. 做了哪些设计决策
4. 跑了哪些测试
5. 测试是否通过
6. 接口如何验证
7. 风险和未完成事项
