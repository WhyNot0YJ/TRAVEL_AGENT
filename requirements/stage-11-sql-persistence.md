# Stage 11：SQL 持久化与 Repository 层

## 任务目标

接入 MySQL 或 PostgreSQL 持久化能力，让任务、规划结果、planner run、tool/LLM trace 可以跨服务重启保留。Redis 继续用于缓存、request hash 和限流，不作为唯一持久层。

## 当前上下文

当前存储状态：

* Redis 可用时保存任务、request hash 和 rate limit。
* Redis 不可用时降级到内存。
* `docs/database.md` 明确当前没有 SQL 数据库表。
* `internal/domain` 不包含 Redis key 或任务状态管理。

## 不做什么

* 不把 SQL 细节污染 `internal/domain`。
* 不移除现有 Redis/内存 fallback。
* 不一次性引入复杂 ORM，除非用户明确批准。
* 不保存 API Key、完整敏感请求头或外部 provider 密钥。
* 不做用户系统，除非阶段 16 已开始。

## 需要阅读的文件

* `AGENTS.md`
* `docs/database.md`
* `docs/architecture.md`
* `docs/api.md`
* `internal/travel/task.go`
* `internal/travel/task_store.go`
* `internal/travel/service.go`
* `internal/redis/client.go`
* `internal/config/config.go`
* `cmd/server/main.go`

## 推荐设计

新增轻量 repository/store 层，优先保持接口清晰：

* `travel_tasks`：任务状态、request hash、请求摘要、错误、时间戳。
* `travel_plans`：最终 `TravelPlan` JSON、预算总额、天数、warning 数。
* `planner_runs`：planner 类型、开始/结束时间、耗时、是否 fallback。
* `planner_events` 或 `tool_calls`：节点名、tool 名、provider、耗时、状态、fallback reason。

迁移文件可以放在 `migrations/` 或项目已有约定目录；如果新增目录，需要更新 README 和 docs。

## 实现要求

* 数据库连接配置来自环境变量或配置文件。
* 支持无 SQL 配置时继续用 Redis/内存开发模式。
* 新增 SQL store 必须有单元测试或集成测试，优先用接口测试覆盖行为。
* 序列化 `TravelPlan` 时要保持 schema 稳定。
* 不把外部 API 原始大响应默认持久化；如保存快照，需要脱敏并限制大小。
* 所有新增表必须同步文档。

## 文档更新要求

* 更新 `docs/database.md`：表结构、字段说明、索引、TTL/清理策略。
* 更新 `docs/architecture.md`：说明 Redis 与 SQL 的职责边界。
* 更新 `docs/api.md`：如果任务查询语义或错误码变化，需要同步。
* 更新 `README.md`：补充数据库环境变量和本地运行方式。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness
go run ./cmd/harness -planner eino
```

如果新增数据库集成测试，说明需要的本地服务和环境变量。

## 验收标准

* 未配置 SQL 时，现有测试和本地开发模式不受影响。
* 配置 SQL 后，任务和结果可持久化。
* 服务重启后可查询已完成任务。
* Redis 与 SQL 失败边界清晰。
* 数据库文档与实际表结构一致。
