# 数据库

## 当前阶段

当前阶段已支持可选 MySQL 持久化。未配置 SQL 时，系统继续使用 Redis 或内存开发模式。

* MySQL：长期保存 travel task、最终 TravelPlan，以及后续 planner run / event trace 表。
* Redis：继续用于无 SQL 模式下的任务缓存、request hash 去重和限流。
* 内存：Redis 和 MySQL 都不可用时的本地开发 fallback，服务重启后数据丢失。

迁移文件：

```text
migrations/mysql/001_travel_persistence.sql
```

## MySQL 配置

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `TRAVEL_AGENT_SQL_ENABLED` | 是否启用 SQL task store | `false` |
| `TRAVEL_AGENT_SQL_DSN` | MySQL DSN，例如 `user:pass@tcp(localhost:3306)/travel_agent?parseTime=true&charset=utf8mb4&loc=UTC` | 空 |
| `TRAVEL_AGENT_SQL_MAX_OPEN_CONNS` | 最大打开连接数 | `10` |
| `TRAVEL_AGENT_SQL_MAX_IDLE_CONNS` | 最大空闲连接数 | `5` |
| `TRAVEL_AGENT_SQL_CONN_MAX_LIFETIME_SECONDS` | 连接最长生命周期 | `1800` |

如果 `TRAVEL_AGENT_SQL_ENABLED=true` 但 DSN 为空、连接失败或 ping 失败，server 会保留 Redis/内存 store 并记录日志。

## MySQL 表

### `travel_tasks`

保存异步任务的稳定状态。

| 字段 | 说明 |
| --- | --- |
| `id` | task id，主键 |
| `request_hash` | 请求 hash，唯一索引，用于复用相同请求 |
| `status` | `pending`、`running`、`succeeded`、`failed` |
| `request_json` | `domain.TravelRequest` JSON |
| `error_text` | 失败错误摘要 |
| `created_at` / `updated_at` | UTC 时间 |

索引：

* `PRIMARY KEY (id)`
* `UNIQUE KEY ux_travel_tasks_request_hash (request_hash)`
* `KEY idx_travel_tasks_status_updated_at (status, updated_at)`

### `travel_plans`

保存最终 `domain.TravelPlan` JSON 和摘要字段。

| 字段 | 说明 |
| --- | --- |
| `task_id` | task id，主键并引用 `travel_tasks.id` |
| `plan_json` | `domain.TravelPlan` JSON |
| `budget_total` | 总预算 |
| `day_count` | 天数 |
| `warning_count` | warning 数量 |
| `updated_at` | UTC 时间 |

### `planner_runs`

预留给 planner 运行摘要。当前迁移已建表，后续阶段会把通用 trace 写入该表。

字段包括 planner type、prompt version、tool mode、耗时、fallback、token usage 等。

### `planner_events`

预留给节点/tool/LLM 事件。当前迁移已建表，后续阶段会写入节点名、tool、provider、耗时、状态和 fallback reason。

## Redis Key

| Key | Value | TTL |
| --- | --- | --- |
| `travel:task:{task_id}` | JSON encoded task，包括状态、请求、结果、错误、时间戳 | `TRAVEL_AGENT_CACHE_TTL_SECONDS` |
| `travel:request_hash:{hash}` | 对应的 `task_id` | `TRAVEL_AGENT_CACHE_TTL_SECONDS` |
| `travel:rate:{client}` | 当前窗口内请求计数 | 1 minute |

## 任务状态

任务状态：

* `pending`
* `running`
* `succeeded`
* `failed`

## 说明

* `internal/domain` 不包含 Redis key 或任务状态管理。
* SQL 原始表结构不污染 `internal/domain`。
* 不持久化 API Key、完整敏感请求头或外部 provider 原始大响应。
* 当前没有自动 TTL 删除 SQL 行；生产部署应按业务保留周期定期清理旧任务和 trace。
