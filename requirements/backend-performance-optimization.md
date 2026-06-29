# 后端高可用与性能优化专项需求

## 任务目标

基于当前 Travel Agent 已具备的 Gin API、异步任务、SSE、Redis / memory fallback、可选 MySQL、Eino Planner、LLM / 外部 Tool fallback 和 Harness，本专项从纯后端视角推进高并发、高可用和可扩展能力建设。

本专项的目标不是做局部微优化，而是把 Travel Agent 后端升级为更接近生产系统的架构：

* MySQL 承担长期任务、结果、运行记录和业务数据持久化。
* Redis 承担缓存、限流、分布式锁、热点数据和短期状态加速。
* 消息队列承担异步规划任务解耦、削峰填谷、失败重试和任务恢复。
* Worker 池承担 Planner 执行隔离，避免 HTTP 进程被慢 LLM / 外部 API 拖垮。
* 通过超时、重试、熔断、降级、幂等、死信和补偿机制提升系统高可用。
* 通过结构化日志、错误追踪、benchmark、观测性、告警和容量规划，让性能优化可定位、可度量、可回归、可运营。

## 当前上下文

当前系统已经具备：

* `cmd/server` Gin HTTP server。
* `POST /api/v1/travel/chat/stream` 聊天式需求收集 SSE。
* `POST /api/v1/travel/plans` 创建异步规划任务。
* `GET /api/v1/travel/plans/:task_id/stream` 订阅任务 SSE。
* `GET /api/v1/travel/plans/:task_id` 查询任务状态和结果。
* `TravelPlanService`、`TaskStore`、`EventBus`、`RateLimiter`。
* Redis / MySQL / memory fallback 的任务存储基础。
* request hash 缓存复用。
* Eino Planner、MockPlanner、真实 LLM / 外部 Tool fallback。
* 节点级 `node` SSE 事件和结构化日志。
* Harness 可统计平均耗时、节点耗时、fallback rate、warning rate 等指标。

当前主要问题：

* HTTP 任务创建与后台 planner goroutine 仍偏单进程模型，不利于多实例部署和任务恢复。
* Redis、MySQL 的职责边界需要进一步明确：哪些数据长期保存、哪些只做缓存、哪些需要一致性保障。
* 缺少消息队列层，无法在高峰期稳定削峰，也缺少可靠的失败重试和死信处理。
* 相同 request hash 的并发请求可能重复触发 planner 执行，需要分布式级幂等与 singleflight。
* SSE EventBus 当前更适合单进程内事件分发，多实例部署时需要考虑事件恢复和查询兜底。
* LLM、外部 Tool、数据库、Redis、消息队列都需要统一超时、重试、熔断和降级策略。
* 当前日志仍分散在 `log.Printf` 调用中，缺少稳定字段、错误分类和跨 HTTP / Worker / Planner / Tool 的统一排障入口。
* 目前缺少系统级容量指标，例如 QPS、并发任务数、队列积压、缓存命中率、DB 慢查询、worker 饱和度。

## 后端优化方向总览

| 优先级 | 方向 | 目标 |
| :---: | :--- | :--- |
| P0 | MySQL 持久化强化 | 任务、结果、运行记录可恢复、可审计、可扩展 |
| P0 | Redis 缓存与限流体系 | 降低 DB / Planner 压力，抵御热点和突发流量 |
| P0 | 消息队列异步任务架构 | HTTP 与 Planner 解耦，支持削峰、重试、死信 |
| P0 | 幂等、去重与分布式锁 | 避免重复规划、重复扣费式外部调用和缓存击穿 |
| P1 | Worker 池与任务调度 | 控制并发、隔离慢任务、支持优先级和超时取消 |
| P1 | 熔断、降级与 fallback | 外部依赖异常时保持核心链路可用 |
| P1 | 结构化日志与错误追踪 | 通过 request id、trace id、task id 快速还原错误链路 |
| P1 | 可观测性、告警与容量指标 | 能发现、定位和量化高可用问题 |
| P2 | 数据库性能与归档 | 索引、分页、冷热分层、历史数据治理 |
| P2 | 多实例部署与故障恢复 | 进程重启、实例宕机、队列重放后任务不丢 |
| P2 | 压测与性能回归 | 建立持续容量验证机制 |

## 本专项不做什么

* 不做前端 UI、交互、样式或浏览器兼容优化。
* 不改动 TravelPlan 的核心业务语义，除非后端持久化确实需要补充内部字段。
* 不让 `internal/harness` 直接依赖 MySQL、Redis、消息队列或 Eino 具体实现。
* 不硬编码 API Key、数据库密码、Redis 密码或 MQ 连接串。
* 不为了追求吞吐牺牲输出正确性、任务幂等性和数据一致性。
* 不一次性引入复杂微服务体系；优先在当前单仓库内演进出清晰边界。
* 不在文档中承诺尚未实现的多地域容灾、自动扩缩容或云厂商专有能力。

## 目标架构

```text
HTTP Client
  -> Gin Router / Middleware
  -> Request Context / Structured Logger
  -> Travel Handler
  -> TravelPlanService
  -> MySQL Task Repository
  -> Redis Cache / RateLimit / Lock
  -> Message Queue Producer
  -> Planner Worker Pool
  -> agent.TravelPlanner
  -> Eino / Mock Planner
  -> MySQL Result / RunLog Repository
  -> Error Trace / Audit Log Repository
  -> Redis Hot Result Cache
  -> SSE Query / Event Recovery
```

推荐演进原则：

* HTTP 进程只负责参数校验、幂等创建任务、写入 MySQL、写入 Redis 缓存 / 锁、投递 MQ。
* Worker 从 MQ 消费任务并执行 Planner，完成后写 MySQL，并刷新 Redis 热缓存。
* SSE 优先消费内存事件；断线、跨实例或事件丢失时通过 MySQL 任务状态和结果查询兜底。
* Redis 不是权威数据源，权威状态以 MySQL 为准；Redis 失效不应导致任务数据丢失。
* MQ 不是业务数据库，消息可重放、可重复投递，消费者必须幂等。
* 日志上下文从 HTTP middleware 创建并注入 `context.Context`，在 service、repository、queue、worker、planner、tool 和 SSE 中延续同一组追踪字段。
* 错误日志既要写标准输出供容器采集，也要在关键任务失败、重试和死信场景写入 MySQL，保证服务重启后仍可按 `request_id`、`trace_id`、`task_id` 复盘。

## 数据分层要求

### MySQL

MySQL 作为后端权威持久化层，至少承载：

* `travel_tasks`：任务主表，记录 `task_id`、`request_hash`、`status`、`planner_type`、`agent_mode`、`test_mode`、创建时间、更新时间、失败原因。
* `travel_task_requests`：规范化后的请求快照，便于重试、审计和问题复现。
* `travel_plan_results`：最终结构化 `TravelPlan` JSON、结果版本、生成耗时。
* `travel_planner_runs`：每次 planner run 的执行记录，包括 worker id、attempt、started_at、finished_at、duration_ms、fallback 信息。
* `travel_node_traces`：可选，记录 Eino 节点级耗时、状态和 warning，用于性能归因。
* `travel_error_logs`：记录关键错误事件，包含 `request_id`、`trace_id`、`task_id`、`run_id`、`component`、`operation`、`error_category`、`error_code`、`retryable`、`attempt`、`message`、`stack_hash`、`created_at`。

实现要求：

* `task_id` 唯一索引。
* `request_hash` 建唯一或组合唯一约束，支持幂等创建。
* `status + updated_at` 建索引，支持扫描超时 / 卡住任务。
* `trace_id + created_at`、`task_id + created_at`、`error_category + created_at` 建索引，支持按链路、任务和错误类型检索。
* 大 JSON 字段与高频状态字段分表或至少避免在状态轮询路径反复读取大字段。
* 所有写入必须考虑事务边界：任务创建、请求快照、MQ outbox 记录应保持一致。
* 数据库连接池配置来自环境变量，不能写死。

### Redis

Redis 作为性能加速和分布式协调层，至少承载：

* request hash 到 task id 的短期映射。
* 已完成任务结果热缓存。
* IP / 用户维度限流计数。
* 分布式锁，避免同 request hash 并发重复创建和重复执行。
* singleflight / in-flight 标记，避免缓存击穿。
* 外部 Tool 热点查询缓存，例如热门目的地 POI、短 TTL 天气、路线片段。

实现要求：

* 所有 key 必须有命名规范和 TTL 说明。
* Redis 不可用时，开发环境可降级为内存；生产环境策略必须明确是 fail-close 还是 fail-open。
* 分布式锁必须设置过期时间和安全释放逻辑，避免死锁。
* 缓存 value 需要控制大小，避免把过大的 plan / trace 长期塞入 Redis。
* 高并发热点 key 要考虑随机 TTL、互斥重建或异步刷新，避免缓存雪崩。

### 消息队列

消息队列用于解耦 HTTP 创建任务和 Planner 执行。首选实现可以是 Redis Streams、RabbitMQ、NATS、Kafka 或云厂商 MQ，但本项目初期推荐优先选轻量方案，避免架构过度膨胀。

队列至少需要：

* `travel.plan.requested`：规划任务创建事件。
* `travel.plan.retry`：重试任务。
* `travel.plan.dead_letter`：死信任务。
* 可选 `travel.plan.completed`：任务完成事件，用于后续通知或业务闭环。

消息内容至少包含：

* `message_id`
* `task_id`
* `request_hash`
* `planner_type`
* `agent_mode`
* `test_mode`
* `attempt`
* `created_at`
* `trace_id` / `request_id`

实现要求：

* Producer 写消息前后要有可靠性策略。推荐使用 MySQL outbox 模式：先在同事务写任务和 outbox，再由 relay 投递 MQ。
* Consumer 必须幂等：重复消费同一 `task_id + attempt` 不应重复生成多个最终结果。
* 支持最大重试次数、指数退避和死信队列。
* 对不可重试错误直接失败，例如请求参数无效、业务校验失败。
* 对可重试错误进入 retry，例如 provider timeout、429、临时网络错误、数据库短暂不可用。
* 队列积压、消费延迟、重试次数和死信数量必须纳入指标。

## 实现要求

### G1：任务状态机与幂等

任务状态建议扩展为：

```text
pending -> queued -> running -> succeeded
                         -> failed
                         -> retrying
                         -> canceled
                         -> dead_letter
```

要求：

* 所有状态流转必须集中定义，不允许各处随意写字符串。
* `request_hash` 相同且请求语义一致时，优先复用已有任务。
* 已成功任务可直接返回缓存或历史结果。
* 运行中任务应返回同一个 `task_id`，不要重复投递 MQ。
* 失败任务是否复用必须明确：默认不复用业务失败；短期 provider 故障可负缓存。
* 状态更新需要乐观锁或条件更新，避免多个 worker 抢同一任务导致状态回退。

### G2：MySQL Repository 与事务

* 引入清晰 Repository 层，避免 handler / service 直接拼 SQL。
* 任务创建应在一个事务内完成：创建任务、保存请求快照、写 outbox 事件。
* Planner 完成应在一个事务内完成：写结果、写 run log、更新任务状态。
* 失败路径必须保存错误分类、错误摘要和 attempt。
* 查询接口应区分轻量状态查询和完整结果查询，避免每次轮询读取大 JSON。
* migration 必须可重复执行、可回滚或至少有明确升级说明。

### G3：Redis 缓存、限流与锁

* request hash 去重先查 Redis，再查 MySQL，Redis miss 不代表任务不存在。
* 已完成 plan 的 Redis 缓存需要包含结果版本，避免 schema 变化后读到旧结构。
* 限流至少支持 IP 维度，预留 user id / api key 维度。
* 限流响应需要稳定错误码，避免前端和调用方误判为 500。
* 分布式锁只保护关键区，不能长时间包裹 LLM / Tool 调用。
* Redis key 和 TTL 必须写入 `docs/database.md` 或 `docs/performance.md`。

### G4：消息队列与 Worker 池

* HTTP handler 不直接启动长期 planner goroutine，改为投递 MQ。
* Worker 数量、单 worker 并发、队列拉取 batch size、任务 timeout 均来自配置。
* Worker 启动时可以扫描 MySQL 中卡在 `running/queued/retrying` 的超时任务，并按策略恢复。
* Worker 执行前必须抢占任务状态，例如 `queued -> running` 条件更新成功才执行。
* Worker panic 必须 recover，记录失败并按错误类型决定 retry / dead letter。
* 任务执行 context 必须带全链路 timeout，并传递到 Eino、LLM、外部 Tool。

### G5：外部依赖熔断与降级

外部依赖包括：

* LLM provider。
* 高德 / 天气 / POI / 路线 API。
* MySQL。
* Redis。
* MQ。

要求：

* 每类外部依赖都有 timeout、最大重试次数、熔断阈值和恢复窗口。
* LLM / Tool 失败时优先走 deterministic / mock fallback，保证任务尽量完成。
* Redis 失败时不能丢失 MySQL 权威数据；可降级为 DB 查询和本地限流。
* MQ 短暂不可用时，outbox relay 应持续重试，HTTP 不应把未投递消息误判为已执行。
* MySQL 不可用时，创建任务应失败并返回稳定错误，不应只写 Redis 造成数据不一致。

### G6：SSE 与跨实例可用性

* SSE 连接仍可使用进程内 EventBus 推送实时事件。
* 多实例部署时，SSE 不应依赖用户永远连接到同一实例。
* 新连接应先查 MySQL 当前任务状态；已完成任务直接返回 done / error。
* 运行中任务如果没有内存事件，也应通过状态轮询、Redis pub/sub 或 MQ completed 事件恢复进度。
* EventBus subscriber 必须有容量上限、慢消费者策略和断开清理测试。
* SSE 事件不应成为权威数据源，最终结果以 MySQL 查询为准。

### G7：性能与容量治理

* 新增后端 benchmark 或 harness 性能模式，输出 QPS、平均耗时、P50、P95、P99、错误率。
* 指标至少覆盖：
  * 任务创建 QPS。
  * request hash 命中率。
  * Redis 命中率。
  * MQ 生产 / 消费延迟。
  * 队列积压量。
  * Worker 并发与饱和度。
  * Planner 平均耗时和分位耗时。
  * LLM / Tool fallback rate。
  * MySQL 慢查询数量。
  * SSE 当前连接数。
* 默认压测不能依赖真实 API Key；真实外部 API 压测必须有显式环境变量开关。

### G8：观测性与告警

* request id / trace id 必须贯穿 HTTP、MySQL、Redis、MQ、Worker、Planner 和 SSE。
* 统一日志入口应封装在独立包中，业务代码只写稳定字段和事件语义，不直接拼接自由格式错误文本。
* 结构化日志字段保持稳定：`request_id`、`trace_id`、`span_id`、`parent_span_id`、`task_id`、`request_hash`、`status`、`attempt`、`worker_id`、`component`、`operation`、`duration_ms`、`error_category`、`error_code`、`retryable`。
* 日志事件命名保持有限集合，例如 `http.request`、`task.created`、`task.queued`、`worker.started`、`planner.node.finished`、`tool.call.failed`、`store.query.failed`、`sse.disconnected`，避免随意扩散。
* 错误分类至少覆盖：`validation`、`rate_limited`、`db`、`redis`、`mq`、`llm`、`external_tool`、`timeout`、`canceled`、`panic`、`serialization`、`unknown`。
* HTTP middleware 负责生成或接收 `X-Request-ID`，并生成内部 `trace_id`；异步任务、MQ 消息、Worker 执行和 Eino callback 必须继承同一个 `trace_id`。
* `TaskEvent`、planner trace、run log 和 error log 中的 `request_id` / `trace_id` / `task_id` 必须一致，方便从 SSE 用户问题反查后台执行链路。
* 必须记录关键业务事件：任务创建、入队、出队、开始执行、成功、失败、重试、死信、取消。
* 必须记录关键依赖事件：DB error、Redis error、MQ publish / consume error、LLM timeout、Tool fallback。
* panic recovery 必须记录 `stack_hash` 和有限长度 stack 摘要；标准输出可保留完整 stack，持久化日志只保存可检索摘要，避免大字段膨胀。
* 外部 API 日志不得记录完整 URL、API Key、Authorization、Cookie、用户敏感原文和未脱敏响应体；响应体只允许在显式 debug 开关下截断采样。
* 任务失败、重试、死信、panic、依赖熔断打开和 fallback 触发必须写入可持久化错误日志；普通成功访问只写标准输出和指标。
* 日志级别建议保持 `debug`、`info`、`warn`、`error` 四档，生产默认 `info`，debug 需要环境变量显式打开并限制敏感字段。
* 提供按 `request_id`、`trace_id`、`task_id`、`error_category` 检索错误链路的内部查询能力，可以先以 repository / CLI / harness report 形式落地，不要求第一阶段暴露公网 API。
* 告警建议覆盖：队列积压过高、死信增长、任务失败率升高、P95 延迟升高、DB 连接池耗尽、Redis 错误率升高、worker 长时间无消费。

### G9：安全、配置与运维

* 所有 MySQL、Redis、MQ、LLM、外部 API 配置必须来自环境变量或配置文件。
* 配置必须有默认值、说明和校验；生产缺失关键配置时应启动失败或明确降级。
* 日志、报告和 dead letter 中不得包含 API Key、数据库密码或用户敏感原文。
* 支持优雅关闭：停止接收新请求、停止拉取新消息、等待运行中任务完成或超时取消。
* 支持健康检查：
  * liveness：进程存活。
  * readiness：MySQL / Redis / MQ 基础可用。
  * worker readiness：可消费队列且未处于熔断。

## 需要阅读的文件

动手前必须阅读：

* `AGENTS.md`
* `requirements/README.md`
* `requirements/stage-06-redis-cache-rate-limit.md`
* `requirements/stage-07-sse.md`
* `requirements/stage-09-external-tool-hardening.md`
* `requirements/stage-10-llm-pipeline-hardening.md`
* `requirements/stage-11-sql-persistence.md`
* `requirements/stage-14-observability.md`
* `requirements/stage-17-deepseek-true-streaming.md`
* `requirements/stage-18-travel-brief-confirmation.md`
* `docs/api.md`
* `docs/architecture.md`
* `docs/database.md`
* `docs/agent-flow.md`
* `docs/evaluation-harness.md`
* `README.md`
* `internal/config/config.go`
* `internal/server/*`
* `internal/travel/*`
* `internal/redis/*`
* `internal/agent/planner.go`
* `internal/agent/eino/*`
* `internal/harness/*`
* `cmd/server/main.go`
* `cmd/harness/main.go`
* `migrations/*`

## 需要新增或修改的文件

预期修改：

* `internal/config/config.go`
* `internal/server/router.go`
* `internal/server/middleware.go`
* `internal/observability/logger.go`
* `internal/observability/context.go`
* `internal/observability/errors.go`
* `internal/travel/service.go`
* `internal/travel/handler.go`
* `internal/travel/task.go`
* `internal/travel/task_store.go`
* `internal/travel/mysql_task_store.go`
* `internal/travel/cache.go`
* `internal/travel/rate_limiter.go`
* `internal/travel/event_bus.go`
* `internal/redis/client.go`
* `internal/agent/eino/planner.go`
* `internal/harness/metrics.go`
* `internal/harness/runner.go`
* `cmd/server/main.go`
* `cmd/harness/main.go`
* `migrations/mysql/*`
* `docs/api.md`
* `docs/architecture.md`
* `docs/database.md`
* `docs/evaluation-harness.md`
* `README.md`

按实际实现可新增：

* `internal/travel/repository.go`
* `internal/travel/outbox.go`
* `internal/travel/idempotency.go`
* `internal/travel/locks.go`
* `internal/travel/worker.go`
* `internal/travel/worker_pool.go`
* `internal/travel/retry_policy.go`
* `internal/travel/dead_letter.go`
* `internal/travel/circuit_breaker.go`
* `internal/travel/error_log_store.go`
* `internal/queue/queue.go`
* `internal/queue/redis_streams.go`
* `internal/queue/memory_queue.go`
* `cmd/worker/main.go`
* `cmd/bench/main.go`
* `docs/performance.md`
* `reports/performance_report.json`

## 文档更新要求

实现本专项任一能力时，必须同步更新：

* `docs/api.md`：如任务状态、错误码、响应字段、健康检查接口变化，必须更新。
* `docs/architecture.md`：说明 MySQL、Redis、MQ、Worker、Service、Planner 的关系。
* `docs/database.md`：说明 MySQL 表结构、索引、Redis key、TTL、MQ topic / stream。
* `docs/evaluation-harness.md`：如新增性能指标、队列指标或报告字段，必须更新。
* `docs/performance.md`：记录压测方法、容量目标、基线结果和调优建议。
* `docs/observability.md`：记录日志字段、事件名、错误分类、脱敏规则、查询方式和告警建议。
* `README.md`：补充 MySQL、Redis、MQ、worker、benchmark 的运行方式和环境变量。

如新增外部 API 或 provider 配置，必须更新 `docs/external-apis.md`。

## 测试要求

后端尽量运行：

```bash
go test ./...
go vet ./...
```

Harness 尽量运行：

```bash
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
make harness
make harness-eino
```

如果新增 worker / MQ：

```bash
go test ./internal/queue ./internal/travel
go run ./cmd/server
go run ./cmd/worker
```

如果新增 benchmark：

```bash
go run ./cmd/bench -planner mock -repeat 100 -concurrency 20
go run ./cmd/bench -planner eino -repeat 30 -concurrency 5
```

如果改动 MySQL：

```bash
go test ./internal/travel -run MySQL
```

如果改动 Redis：

```bash
go test ./internal/travel -run Redis
```

如果改动 SSE / EventBus：

```bash
go test ./internal/travel -run EventBus
```

## 验收标准

* MySQL 成为任务和结果的权威存储，服务重启后已完成任务仍可查询。
* Redis 仅作为缓存、限流和协调层，Redis 故障不会造成权威数据丢失。
* 任务创建具备 request hash 幂等，相同请求不会重复创建多个有效任务。
* 消息队列可承接 Planner 异步执行，HTTP handler 不再直接承担长期 planner run。
* Worker 消费任务具备幂等、超时、panic recovery、重试和死信机制。
* 外部 LLM / Tool 故障时可熔断、降级或 fallback，不拖垮整体服务。
* SSE 在任务完成、断线重连、跨实例查询时仍能通过 MySQL 兜底。
* 任一失败任务都能通过 `request_id`、`trace_id` 或 `task_id` 追查到 HTTP 入口、任务状态流转、Worker attempt、Planner 节点、外部依赖和最终错误分类。
* 日志和错误表不包含 API Key、数据库密码、Redis 密码、Authorization、Cookie 或用户敏感原文。
* 关键后端指标可观测：DB、Redis、MQ、Worker、Planner、SSE、fallback。
* 所有新增配置都有环境变量说明和合理默认值。
* Harness 不直接依赖 MySQL、Redis、MQ 或 Eino 具体实现。

## 推荐拆分顺序

1. **MySQL 权威存储 PR**：表结构、Repository、事务、查询路径、迁移文档。
2. **Redis 幂等与缓存 PR**：request hash、分布式锁、热点缓存、限流强化。
3. **MQ Outbox PR**：outbox 表、队列接口、producer relay、memory queue 测试实现。
4. **Worker Pool PR**：consumer、并发控制、任务状态机、重试和死信。
5. **熔断降级 PR**：LLM / Tool / Redis / MQ timeout、retry、circuit breaker。
6. **日志与错误追踪 PR**：结构化日志包、错误分类、trace context、错误日志持久化、脱敏规则和查询入口。
7. **多实例与 SSE 兜底 PR**：EventBus 清理、状态查询恢复、completed event。
8. **容量与观测 PR**：benchmark、性能报告、指标字段、告警建议和 docs。

## 最终回复要求

完成任一后端高可用优化 PR 后请输出：

1. 修改了哪些文件
2. 本次覆盖了哪些后端能力：MySQL、Redis、MQ、Worker、熔断、观测等
3. 做了哪些设计决策
4. 跑了哪些测试和 benchmark
5. 数据一致性、幂等和失败恢复如何保证
6. 还有哪些风险或未完成事项
