# 数据库

## 当前阶段

当前阶段已支持可选 MySQL 持久化。未配置 SQL 时，系统继续使用 Redis 或内存开发模式。

* MySQL：长期保存 travel task、最终 TravelPlan，以及后续 planner run / event trace 表。
* Redis：继续用于无 SQL 模式下的任务缓存、request hash 去重和限流。
* 内存：Redis 和 MySQL 都不可用时的本地开发 fallback，服务重启后数据丢失。

迁移文件：

```text
migrations/mysql/001_travel_persistence.sql
migrations/mysql/002_observability_request_id.sql
migrations/mysql/003_users_and_plan_library.sql
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

## 用户与计划库表

迁移 003 在原有的 task/plan 表之外，新增了用户、session、计划库与公开计划，并给 `travel_tasks` 加了一个 `user_id` 列，便于把任务归属到登录用户。匿名 task 仍然允许 `user_id = NULL`。

### `users`

| 字段 | 说明 |
| --- | --- |
| `id` | `user_<hex>`，主键 |
| `email` | 唯一索引，登录凭据 |
| `display_name` | 用户在公开计划上展示的昵称 |
| `password_hash` | bcrypt 哈希，永远不返回到 API |
| `status` | `active` / `disabled` |
| `created_at` / `updated_at` | UTC 时间 |

### `user_sessions`

| 字段 | 说明 |
| --- | --- |
| `id` | `sess_<hex>` |
| `user_id` | 外键 |
| `token_hash` | 仅存哈希，唯一索引 |
| `expires_at` | TTL 默认 168 小时 |
| `revoked_at` | 登出/吊销时间 |

### `user_plans`

| 字段 | 说明 |
| --- | --- |
| `id` | `plan_<hex>` |
| `user_id` | 所属用户 |
| `task_id` | 来源生成任务，可空 |
| `source_public_plan_id` | 副本来源，可空 |
| `title` / `note` / `summary` / `tags_json` | 用户可编辑元信息 |
| `plan_json` | 保存时刻的 `TravelPlan` 快照 |
| `destination_city` / `days` | 冗余字段，方便筛选 |
| `visibility` | `private` / `public` |
| `publish_status` | `draft` / `published` / `unpublished` |
| `created_at` / `updated_at` / `deleted_at` | 软删除字段 |

索引：`(user_id, updated_at)`、`(user_id, deleted_at)`、`destination_city`、`(visibility, publish_status, updated_at)`、`task_id`。

### `plan_conversation_archives`

| 字段 | 说明 |
| --- | --- |
| `id` | `arc_<hex>` |
| `plan_id` | 关联用户计划 |
| `user_id` | 归档所有者 |
| `task_id` | 来源 task |
| `brief_json` | 最终 Travel Brief 快照 |
| `messages_json` / `events_json` | 可选对话与事件摘要 |

### `public_plans`

| 字段 | 说明 |
| --- | --- |
| `id` | `pub_<hex>` |
| `plan_id` | 来源 user_plan，唯一索引 |
| `user_id` | 发布者，仅用于内部权限校验 |
| `title` / `summary` / `tags_json` / `plan_json` | 公开内容快照 |
| `status` | `published` / `unpublished` / `removed` |
| `view_count` / `save_count` / `copy_count` | 计数器 |
| `hot_score` | 增量更新公式：`view + save*5 + copy*3` |
| `published_at` / `updated_at` | 时间 |

索引：`(status, hot_score, published_at)`、`(status, published_at)`、`(destination_city, status)`、`(user_id, published_at)`。

### `public_plan_events`

记录浏览/保存/复制事件，便于后续做反作弊与去重。当前主链路只用它做 hot_score 增量更新。

### `analytics_events`

预留产品事件表（注册转化、保存率、搜索无结果等）。当前 server 还没自动写入，只是建表占位，后续阶段会启用。

### `travel_tasks` 变更

```sql
ALTER TABLE travel_tasks
  ADD COLUMN user_id VARCHAR(64) NULL AFTER request_id,
  ADD KEY idx_travel_tasks_user_id (user_id);
```

匿名 task 保持 `user_id = NULL`；登录用户创建的 task 写入对应 `user_id`，并参与 `request_hash`，避免跨用户复用。

## 内存 Fallback

MySQL 不可用时统一退回 in-memory store：`auth.MemoryUserStore` / `auth.MemorySessionStore` / `plans.MemoryPlanStore` / `plans.MemoryPublicPlanStore`。它们与 MySQL 实现共享同一份接口（`UserStore`、`SessionStore`、`PlanStore`、`PublicPlanStore`），方便单元测试用 stub 注入，也让本地开发无需 MySQL 即可跑完整闭环。重启会丢失数据。
