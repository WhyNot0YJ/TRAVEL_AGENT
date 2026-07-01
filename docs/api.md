# API 文档

## 2026-06 聊天优先契约

前端现在以聊天为主入口：先收集旅行需求，信息齐全后展示 Travel Brief 确认卡和“生成行程”按钮。必填项为出发地、目的地、天数、预算、兴趣偏好和出行人数；可选项缺失时使用稳定默认值。

### POST /api/v1/travel/chat/stream

用于用户信息收集阶段，返回 `text/event-stream`。测试模式使用本地规则提取；真实模式使用配置的 LLM 提取，失败时 fallback 到规则提取。

请求：

```json
{
  "message": "上海出发，杭州 3 天，2 人，预算 3000，喜欢美食和自然风光",
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["美食"],
  "travelers": 2,
  "date_range": "任意",
  "transport_mode": "任意",
  "pace": "适中",
  "walking_tolerance": "任意",
  "hotel_area": "任意",
  "must_visit": [],
  "avoid": [],
  "traveler_type": "无要求",
  "budget_type": "总预算",
  "budget_includes": ["住宿", "餐饮", "门票", "市内交通"],
  "test_mode": true,
  "agent_mode": "quick"
}
```

`agent_mode` 可选：`quick` 为快速模式，`expert` 为专家模式。`test_mode=true` 时始终走本地规则、Mock tools 和 deterministic planner；`test_mode=false` 时按服务端 LLM 与真实工具配置执行。

流式事件：

* `assistant_delta`：助手回复增量文本。`message` 字段长度**不再保证等长**——启用流式（`TRAVEL_AGENT_LLM_STREAM_ENABLED=true`，默认）时按 LLM token 增量到达，单帧通常 1~6 个汉字；前端必须**累加**而不是替换。
* `assistant_done`：助手回复文本结束。
* `done`：最终结构化 `ChatResponse`。
* `error`：处理失败。

最终 `done` payload：

```json
{
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["美食", "自然风光"],
  "travelers": 2,
  "date_range": "任意",
  "transport_mode": "高铁 + 打车",
  "pace": "轻松",
  "walking_tolerance": "任意",
  "hotel_area": "任意",
  "must_visit": [],
  "avoid": [],
  "traveler_type": "无要求",
  "budget_type": "总预算",
  "budget_includes": ["住宿", "餐饮", "门票", "市内交通"],
  "reply": "信息已经齐全，可以生成行程了。",
  "missing": [],
  "is_complete": true,
  "agent_mode": "quick"
}
```

### POST /api/v1/travel/plans

新增 `agent_mode` 字段。`request_hash` 会同时包含 `test_mode`、`agent_mode` 和完整 Travel Brief，避免不同人数、避开内容、必去地点或步行强度复用同一结果。

```json
{
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["美食", "自然风光"],
  "travelers": 2,
  "date_range": "任意",
  "transport_mode": "高铁 + 打车",
  "pace": "轻松",
  "walking_tolerance": "任意",
  "hotel_area": "任意",
  "must_visit": ["西湖"],
  "avoid": ["网红店"],
  "traveler_type": "情侣",
  "budget_type": "总预算",
  "budget_includes": ["住宿", "餐饮", "门票", "市内交通"],
  "test_mode": false,
  "agent_mode": "expert"
}
```

`POST /api/v1/travel/chat` 仍保留为兼容接口，但前端主链路使用 `POST /api/v1/travel/chat/stream`。`GET /api/v1/travel/plans/:task_id` 继续作为 SSE 断线后的任务查询兜底。

### GET /api/v1/travel/plans/:task_id/stream

规划任务 SSE 继续返回 `progress`、`node`、`warning`、`error`、`done`、`heartbeat`，并新增业务事件：

* `assistant_delta`：生成过程中的助手文本增量。启用流式时，规划链路在结构化 plan 落库前会通过 LLM streaming 推送一段"旁白"（例如"正在为你规划上海到杭州的 3 天行程..."），单帧长度由 LLM token 决定，**不保证等长**。
* `assistant_done`：助手文本结束。
* `brief_delta`：已规范化的 Travel Brief。
* `poi_batch`：POI 查询阶段候选地点，包含地点、地址、坐标、来源和可得价格。
* `weather_delta`：天气查询阶段结果。
* `route_delta`：路线计算阶段结果，包含距离、时长和可得交通费用。
* `budget_delta`：预算阶段结果，只包含真实可得金额合计和缺失项。
* `day_delta`：初版 itinerary 的单日草稿，`draft=true`。

前端应以最后的 `done.plan` 作为最终结构化行程，以 `day_delta` 作为生成中的草稿展示，以 `assistant_delta` 只作为生成中的可读反馈。`TRAVEL_AGENT_LLM_STREAM_ENABLED=false` 时旁白回退到等长 chunkText 切片（向后兼容）。服务端会为每个 task 缓存最近 100 条事件并添加递增 `sequence`，新连接会先收到历史事件；历史缺失时仍会返回最终 `done` 或 `error`。

### TravelPlan 价格与预算字段

`TravelItem` 保留兼容字段 `estimated_cost`，并新增 `cost` 和 `poi`。前端应优先展示 `cost`：

```json
{
  "estimated_cost": 0,
  "cost": {
    "amount": null,
    "currency": "CNY",
    "unit": "per_person",
    "status": "unavailable",
    "display": "暂无信息",
    "included": false
  }
}
```

`cost.status` 可为：

* `available`：`amount` 为真实可得金额，可按 `included=true` 计入预算。
* `unavailable`：`amount=null`，页面展示“暂无信息”，不计入预算。
* `not_applicable`：天然无需费用，例如纯步行/骑行。

`TravelBudget` 保留 `transport`、`food`、`hotel`、`ticket`、`total`，并新增：

```json
{
  "known_total": 1032,
  "complete": false,
  "currency": "CNY",
  "items": [
    {"key": "food", "label": "餐饮", "amount": 900, "currency": "CNY", "status": "available", "source": "amap.poi.biz_ext.cost", "included": true},
    {"key": "hotel", "label": "住宿", "amount": null, "currency": "CNY", "status": "unavailable", "display": "暂无信息", "included": false}
  ],
  "missing": ["hotel"]
}
```

`total` 与 `known_total` 表示“已知预算”，不代表完整旅行总价；`complete=false` 时页面不得把缺失项显示为 `¥0`。

## 当前阶段

当前 HTTP API 使用异步任务模式，支持 request hash 去重、任务状态查询、缓存复用、限流和 SSE 流式事件。Redis 可用时使用 Redis；Redis 未配置或不可用时，开发环境降级到内存实现。

## 环境变量

```text
TRAVEL_AGENT_HTTP_ADDR=:8080
TRAVEL_AGENT_PLANNER=mock|eino
TRAVEL_AGENT_REDIS_ADDR=localhost:6379
TRAVEL_AGENT_REDIS_PASSWORD=
TRAVEL_AGENT_REDIS_DB=0
TRAVEL_AGENT_CACHE_TTL_SECONDS=1800
TRAVEL_AGENT_REDIS_LOCK_TTL_SECONDS=15
TRAVEL_AGENT_RATE_LIMIT_PER_MINUTE=60
TRAVEL_AGENT_SQL_ENABLED=false
TRAVEL_AGENT_SQL_DSN=
TRAVEL_AGENT_SQL_MAX_OPEN_CONNS=10
TRAVEL_AGENT_SQL_MAX_IDLE_CONNS=5
TRAVEL_AGENT_SQL_CONN_MAX_LIFETIME_SECONDS=1800
```

MySQL 启用后不改变 HTTP API contract。任务创建、查询和 SSE 语义保持不变；区别是任务、请求快照、最终 plan、planner run 摘要和失败错误日志可跨服务重启保留。Redis 同时启用时只作为 request hash 映射、终态结果热缓存、限流和短锁。

## 错误响应

```json
{
  "request_id": "20260623193000.000000000",
  "code": "invalid_request",
  "message": "days is required"
}
```

常见状态码：

* `400`：请求无效
* `404`：任务不存在
* `429`：触发限流
* `500`：服务内部错误

## POST /api/v1/travel/plans

创建异步路线规划任务。相同请求会通过 `request_hash` 复用已有任务或命中缓存。

请求：

```json
{
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["自然风光", "美食"],
  "travelers": 2,
  "transport_mode": "高铁 + 打车",
  "pace": "轻松",
  "test_mode": true
}
```

`travelers`、`interests` 等必填字段缺失时返回 `400 invalid_request`。可选 brief 字段缺失时后端会使用默认值：`date_range=任意`、`pace=适中`、`transport_mode=任意`、`walking_tolerance=任意`、`hotel_area=任意`、`traveler_type=无要求`、`budget_type=总预算`、`budget_includes=["住宿","餐饮","门票","市内交通"]`。预算包含项展示时默认说明“不含往返大交通”。

`test_mode` 可选，默认 `false`。前端开启测试模式时传 `true`，后端会在本次请求中强制使用本地规则解析、deterministic plan generator 和 Mock Tools；关闭测试模式时使用服务端环境变量配置的真实 LLM / real tools，如果 key 或 provider 不可用仍会按原有 fallback 策略降级。

响应：

```json
{
  "task_id": "task_xxx",
  "request_hash": "sha256...",
  "status": "pending",
  "cached": false
}
```

状态码：

* `202`：任务已创建或复用
* `400`：请求 JSON 或必填字段无效
* `429`：请求过于频繁
* `500`：任务创建失败

## GET /api/v1/travel/plans/:task_id

查询任务状态和结果。

响应：

```json
{
  "task_id": "task_xxx",
  "request_hash": "sha256...",
  "status": "succeeded",
  "plan": {
    "title": "杭州3日旅行规划",
    "summary": "...",
    "days": [],
    "budget": {
      "transport": 0,
      "food": 0,
      "hotel": 0,
      "ticket": 0,
      "total": 0,
      "known_total": 0,
      "complete": false,
      "currency": "CNY",
      "items": [],
      "missing": ["hotel", "intercity_transport"]
    },
    "warnings": []
  },
  "created_at": "2026-06-23T10:00:00Z",
  "updated_at": "2026-06-23T10:00:01Z"
}
```

状态值：

* `pending`
* `queued`
* `running`
* `succeeded`
* `failed`
* `retrying`
* `canceled`
* `dead_letter`

当前 HTTP 内联执行路径实际使用 `pending`、`running`、`succeeded`、`failed`；其余状态为后续 MQ / Worker 阶段预留。

失败时：

```json
{
  "task_id": "task_xxx",
  "request_hash": "sha256...",
  "status": "failed",
  "error": "planner error"
}
```

## GET /api/v1/travel/plans/:task_id/stream

订阅任务事件流。

响应头：

```text
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

事件类型：

* `progress`
* `node`
* `warning`
* `error`
* `done`
* `heartbeat`
* `assistant_delta`
* `assistant_done`
* `brief_delta`
* `poi_batch`
* `weather_delta`
* `route_delta`
* `budget_delta`
* `day_delta`

事件 payload：

```json
{
  "type": "progress",
  "task_id": "task_xxx",
  "status": "running",
  "message": "planner started",
  "created_at": "2026-06-23T10:00:00Z"
}
```

节点事件 payload：

```json
{
  "type": "node",
  "request_id": "20260624115200.000000000",
  "task_id": "task_xxx",
  "status": "running",
  "message": "loaded 5 pois",
  "node_name": "SearchPOIsToolNode",
  "node_status": "success",
  "duration_ms": 12,
  "created_at": "2026-06-24T11:52:00Z"
}
```

新增 `node` 事件保持向后兼容；已有前端可继续只处理 `progress`、`warning`、`error` 和 `done`。

如果任务已完成，新连接会先回放缓存的关键过程事件，再返回历史 `done` 或 `error`；如果缓存已不存在，则直接返回合成的 `done` 或 `error`。

示例：

```bash
curl -N http://localhost:8080/api/v1/travel/plans/{task_id}/stream
```

## 前端接入说明

React H5 客户端复用现有契约，不需要额外接口。UI 是对话式体验，但前端会把已收集的需求转换成同一份结构化旅行任务 payload：

* `POST /api/v1/travel/plans` 创建任务。
* `GET /api/v1/travel/plans/:task_id/stream` 流式返回进度和最终 `done` 事件。
* `GET /api/v1/travel/plans/:task_id` 在 SSE 断开时作为轮询 fallback。

当前端和 API 不在同源服务下时，需要为 web app 设置 `VITE_API_BASE_URL`。本地 Vite 开发时，留空会使用开发服务器的 `/api` 代理。

前端模式开关映射到同一个 `test_mode` 请求字段。测试模式使用本地确定性行为，适合演示和可重复检查；真实模式依赖后端 `TRAVEL_AGENT_LLM_*` 和 `TRAVEL_AGENT_TOOL_MODE=real` / `TRAVEL_AGENT_AMAP_*` 配置。

## Auth & Plan Library API

Travel Agent 在异步规划之外提供用户资产层：注册登录、计划保存、用户中心 CRUD、发布/取消发布、公开计划列表与详情。这一组接口默认开启（`TRAVEL_AGENT_AUTH_ENABLED=true`），并通过 HttpOnly Cookie 维持 session。

`TRAVEL_AGENT_ALLOW_ANONYMOUS_PLAN_GENERATION` 默认 `false`，意味着 `/api/v1/travel/plans*` 也会要求登录。本地开发若想保留匿名生成体验，可在 `.env` 中显式设置为 `true`。

### Auth

```text
POST   /api/v1/auth/register   body: {email, password, display_name}     → 201 {user}
POST   /api/v1/auth/login      body: {email, password}                    → 200 {user}
POST   /api/v1/auth/logout                                                → 204
GET    /api/v1/auth/me                                                    → 200 {user} | 401
```

注册成功会写入 HttpOnly cookie（默认 `travel_agent_session`，TTL 168 小时）。错误返回稳定 code：`email_exists`、`password_too_short`、`invalid_email`、`invalid_credentials`、`unauthenticated`。Cookie 必须随每次私有请求发送（前端 `fetch` 已统一设置 `credentials: "include"`）。

### 我的计划

```text
POST   /api/v1/me/plans                          body: {task_id, title?, note?}                → 201 {plan}
GET    /api/v1/me/plans                          ?q&visibility&publish_status&page&page_size   → 200 {items, total, page, page_size}
GET    /api/v1/me/plans/:plan_id                                                                → 200 {plan}
PATCH  /api/v1/me/plans/:plan_id                 body: {title?, note?, summary?, tags?, visibility?} → 200 {plan}
DELETE /api/v1/me/plans/:plan_id                                                                → 204
GET    /api/v1/me/plans/:plan_id/conversation                                                   → 200 {conversation}
POST   /api/v1/me/plans/:plan_id/publish         body: {title?, summary?, tags?}                → 200 {public_plan}
POST   /api/v1/me/plans/:plan_id/unpublish                                                      → 204
GET    /api/v1/me/current                                                                       → 200 {running_task?, latest_plan?}
```

请求要点：

* 私有 `/me/*` 必须登录，否则返回 `401 unauthenticated`。
* 访问他人 plan_id 一律返回 `404 not_found`，不暴露资源是否存在。
* 重复对同一 `task_id` 调用 `POST /me/plans` 会复用已有 plan_id，不创建副本。
* `task_id` 必须处于 `succeeded` 状态且属于当前用户（或匿名 task）；否则返回 `400 task_not_found` / `409 task_not_ready` / `403 forbidden`。
* `PATCH` 字段全部可选；标题长度需在 2-80 字符之间，否则 `400 invalid_title`。

### 公开计划

```text
GET    /api/v1/public/plans                     ?q&destination_city&days&interest&sort&page    → 200 {items, total, page, page_size}
GET    /api/v1/public/plans/:public_plan_id                                                    → 200 {public_plan}
POST   /api/v1/public/plans/:public_plan_id/save                                              → 201 {plan}
```

* `GET /public/plans/:id` counts a view at most once per public plan and viewer key in a 1-hour window. Redis SETNX is used when Redis is available; otherwise the server uses an in-memory TTL fallback.
* `sort` 取值 `hot`（默认）/ `latest`。`hot_score = view_count + save_count*5 + copy_count*3`，在每次浏览/保存事件触发时实时更新。
* 浏览和保存计数更新成功后会写入 `public_plan_events`；计数/事件写入属于 best-effort，不影响公开详情或保存副本的主流程。
* 公开列表只返回 `status=published`。取消发布或被作者删除后，`/public/plans/:id` 也返回 `404`。
* `POST /public/plans/:id/save` 会为当前用户创建私有副本，并自增公开计划的 `save_count`；不复制作者的 note 与对话归档。

### 字段约定

* `UserPlan` 暴露 `plan_id`、`title`、`note`、`summary`、`tags`、`visibility`、`publish_status`、`destination_city`、`days`、`created_at`、`updated_at`，详情 GET 还返回 `plan`（完整 `TravelPlan`）。
* `PublicPlan` 暴露 `public_plan_id`、`title`、`summary`、`tags`、`destination_city`、`days`、`author.display_name`、`hot_score`、`view_count`、`save_count`、`published_at`，详情 GET 还返回 `plan`。
* 公开接口永远不返回 `user_id`、私密 `note`、`task_id`、`request_hash`、归档对话或内部内部 ID。

### 错误码（Auth & Plan Library 新增）

| code | 状态 | 说明 |
| --- | --- | --- |
| `unauthenticated` | 401 | 无 cookie 或 session 已失效 |
| `invalid_credentials` | 401 | 登录时邮箱或密码不匹配（统一文案） |
| `email_exists` | 409 | 注册邮箱已存在 |
| `password_too_short` | 400 | 密码长度未达 `TRAVEL_AGENT_PASSWORD_MIN_LENGTH` |
| `invalid_email` | 400 | 邮箱格式无效 |
| `display_name_required` | 400 | 注册时未提供昵称 |
| `not_found` | 404 | 计划不存在或不可见 |
| `task_not_found` | 400 | 保存计划时引用了不存在的 task |
| `task_not_ready` | 409 | task 尚未完成生成 |
| `forbidden` | 403 | task 属于其他用户 |
| `already_published` | 409 | 计划当前已发布 |
| `not_published` | 409 | 计划当前不在发布状态 |
| `invalid_title` | 400 | 标题为空或超长 |
| `invalid_visibility` | 400 | visibility 字段非法 |
| `public_plan_unavailable` | 409 | 试图保存已下架的公开计划 |
