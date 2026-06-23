# Database

## Current Stage

当前阶段没有 SQL 数据库表，也没有真实持久化模型。

阶段 6 只接入 Redis，用于任务状态、request hash 去重和限流。Redis 不可用时，开发环境降级为内存实现。

## Redis Keys

| Key | Value | TTL |
| --- | --- | --- |
| `travel:task:{task_id}` | JSON encoded task，包括状态、请求、结果、错误、时间戳 | `TRAVEL_AGENT_CACHE_TTL_SECONDS` |
| `travel:request_hash:{hash}` | 对应的 `task_id` | `TRAVEL_AGENT_CACHE_TTL_SECONDS` |
| `travel:rate:{client}` | 当前窗口内请求计数 | 1 minute |

## Task Status

任务状态：

* `pending`
* `running`
* `succeeded`
* `failed`

## Notes

* `internal/domain` 不包含 Redis key 或任务状态管理。
* 当前没有 MySQL / PostgreSQL 表。
* 后续如新增 SQL 持久化，需要另行补充表结构和迁移说明。
