# Travel Agent

本项目是一个基于 React + Go Gin + CloudWeGo Eino 的高并发智能旅游路线规划应用。当前仓库已经包含后端 Evaluation Harness、Gin 异步任务 API、Eino Travel Planner、Redis/内存任务存储，以及 React H5 对话式前端，用于持续评估和演示路线规划 Agent 的稳定性、正确性、耗时和结构化输出质量。

## Travel Agent 评估 Harness

当前 Harness 用于评估旅游路线规划 Agent 的输出质量。

它通过读取 `testdata/travel_cases.json` 中的测试用例，调用 `TravelPlanner` 接口生成 `TravelPlan`，然后使用 `Evaluator` 计算得分并输出报告。

当前默认实现是 `MockPlanner`，第二版已新增 `EinoTravelPlanner`。

`EinoTravelPlanner` 会经过 CloudWeGo Eino Graph / Workflow 编排，并已支持可选真实 LLM 生成节点。默认不开启 LLM，仍使用 Mock POI、Mock Weather、Mock Route、Mock Budget Tool 和 deterministic fallback，不调用真实外部 API。

## 如何运行

```bash
go test ./...
go run ./cmd/harness
make harness
```

也可以指定参数：

```bash
go run ./cmd/harness -dataset testdata/travel_cases.json -report reports/eval_report.json
```

## Planner 类型

当前 Harness 支持两种 planner：

1. `mock`
2. `eino`

运行 MockPlanner：

```bash
go run ./cmd/harness -planner mock
```

运行 EinoTravelPlanner：

```bash
go run ./cmd/harness -planner eino
```

组合参数：

```bash
go run ./cmd/harness -planner eino -dataset testdata/travel_cases.json -report reports/eval_report.json
go run ./cmd/harness -planner eino -repeat 3 -concurrency 4
```

## HTTP API

启动 Gin server：

```bash
set TRAVEL_AGENT_HTTP_ADDR=:8080
set TRAVEL_AGENT_PLANNER=mock
go run ./cmd/server
```

创建异步旅行规划任务：

```bash
curl -X POST http://localhost:8080/api/v1/travel/plans \
  -H "Content-Type: application/json" \
  -d "{\"departure_city\":\"上海\",\"destination_city\":\"杭州\",\"days\":3,\"budget\":3000,\"interests\":[\"自然风光\",\"美食\"],\"travelers\":2,\"transport_mode\":\"高铁 + 打车\",\"pace\":\"轻松\"}"
```

查询任务：

```bash
curl http://localhost:8080/api/v1/travel/plans/{task_id}
```

当前 HTTP API 返回任务状态。POST 返回 `task_id`、`request_hash`、`status` 和 `cached`，GET 返回 `pending` / `running` / `succeeded` / `failed` 以及最终 plan 或错误。创建任务要求出发地、目的地、天数、预算、兴趣偏好和出行人数；日期、节奏、交通偏好、步行强度、酒店区域、必去地点、避开内容、同行人群和预算口径缺失时会使用 Travel Brief 默认值。

Redis 配置：

```bash
set TRAVEL_AGENT_REDIS_ADDR=localhost:6379
set TRAVEL_AGENT_REDIS_PASSWORD=
set TRAVEL_AGENT_REDIS_DB=0
set TRAVEL_AGENT_CACHE_TTL_SECONDS=1800
set TRAVEL_AGENT_RATE_LIMIT_PER_MINUTE=60
```

Redis 不可用时，开发环境会降级为内存任务 store 和内存限流。真实数据库持久化会在后续阶段接入。

MySQL 持久化是可选能力。先执行迁移：

```bash
mysql -u root -p travel_agent < migrations/mysql/001_travel_persistence.sql
```

启用 MySQL task store：

```bash
set TRAVEL_AGENT_SQL_ENABLED=true
set TRAVEL_AGENT_SQL_DSN=user:pass@tcp(localhost:3306)/travel_agent?parseTime=true&charset=utf8mb4&loc=UTC
set TRAVEL_AGENT_SQL_MAX_OPEN_CONNS=10
set TRAVEL_AGENT_SQL_MAX_IDLE_CONNS=5
set TRAVEL_AGENT_SQL_CONN_MAX_LIFETIME_SECONDS=1800
go run ./cmd/server
```

未配置 SQL 或连接失败时，server 会继续使用 Redis/内存 store。Redis 仍可用于限流和无 SQL 模式下的短期任务缓存；MySQL 用于长期保存任务和最终计划。

订阅任务事件流：

```bash
curl -N http://localhost:8080/api/v1/travel/plans/{task_id}/stream
```

事件类型包括 `progress`、`node`、`warning`、`assistant_delta`、`assistant_done`、`brief_delta`、`poi_batch`、`weather_delta`、`route_delta`、`budget_delta`、`day_delta`、`error`、`done`。任务已完成时，新连接会先回放最近的历史事件；如果历史缓存不存在，则立即收到最终事件。

`node` 事件用于观测 planner 节点耗时，包含 `request_id`、`task_id`、`node_name`、`node_status` 和 `duration_ms`。不需要节点级进度的前端可以忽略该事件。

## React H5 前端

H5 客户端位于 `web` 目录，基于 Vite、React 和 TypeScript 构建。它通过同一套异步任务契约与 Gin API 通信：

Windows 快速启动：

```powershell
.\quick-start.ps1
```

停止脚本启动的开发服务：

```powershell
.\quick-start.ps1 -Stop
```

可选端口和 planner：

```powershell
.\quick-start.ps1 -BackendPort 18085 -FrontendPort 5175 -Planner eino
```

```bash
cd web
npm install
npm run dev
```

默认情况下，Vite 会把 `/api` 代理到 `http://localhost:8080`。如果要直接调用其他后端：

```bash
set VITE_API_BASE_URL=http://localhost:8080
npm run dev
```

生产构建：

```bash
cd web
npm run typecheck
npm run lint
npm run build
```

前端 UI Harness：

```bash
cd web
npm run harness:ui
```

UI Harness 使用 Playwright，在浏览器测试中 mock 旅行 API/SSE 契约，分别运行桌面和移动端 Chromium 项目，并写入 `reports/ui_eval_report.json`。

首屏是对话式旅行 Agent。聊天流程会实时收集旅行需求；必填信息齐全后，通过 `POST /api/v1/travel/plans` 创建任务，订阅 `GET /api/v1/travel/plans/{task_id}/stream`，当 SSE 断开时回退到 `GET /api/v1/travel/plans/{task_id}` 轮询，并渲染最终 `TravelPlan`。

路线详情页针对移动端快速浏览做了产品化处理：每天以站点式时间线展示，warning 和 fallback 信息按易读标签分组，预算面板展示“已知预算”和缺失项，本地调整动作可以把后续需求回填到聊天中。进度面板会消费 `node` 和业务级 SSE 事件，让用户在最终计划到达前看到 POI、天气、路线、预算和按天草稿。

说明：

* `mock`：不依赖 Eino，不经过 Graph，只用于基础 Harness 测试。
* `eino`：经过 CloudWeGo Eino Graph / Workflow。默认使用 Mock Tools 和 deterministic plan generator；设置 LLM 环境变量后可调用真实 LLM，设置 Tool 环境变量后可调用真实高德 POI、路线和天气 API。

## LLM 模式

当前 LLM 模式默认面向 DeepSeek OpenAI-compatible API，并使用 JSON Schema 约束输出：

* DeepSeek 默认 provider：`deepseek`
* 默认 Base URL：`https://api.deepseek.com/beta`
* 默认模型：`deepseek-v4-flash`
* DeepSeek 输出方式：强制调用 `submit_travel_plan` tool，tool parameters 使用 `TravelPlan` JSON Schema，`strict=true`
* API Key 只从环境变量读取，不要写入代码、测试或文档

示例：

```bash
set TRAVEL_AGENT_LLM_ENABLED=true
set TRAVEL_AGENT_LLM_PROVIDER=deepseek
set TRAVEL_AGENT_LLM_API_KEY=your-api-key
set TRAVEL_AGENT_LLM_MODEL_QUICK=deepseek-v4-flash
set TRAVEL_AGENT_LLM_MODEL_EXPERT=deepseek-v4-pro
set TRAVEL_AGENT_LLM_STREAM_ENABLED=true
go run ./cmd/harness -planner eino
```

`TRAVEL_AGENT_LLM_MODEL_QUICK` / `TRAVEL_AGENT_LLM_MODEL_EXPERT` 控制 `agent_mode=quick` / `agent_mode=expert` 实际调用的模型。两者未设置时，`TRAVEL_AGENT_LLM_MODEL` 既作为 quick 也作为 expert 的回退（向后兼容）。`request_hash` 包含 agent_mode，所以两种模式的缓存互不污染。

`TRAVEL_AGENT_LLM_STREAM_ENABLED=true`（默认）打开 DeepSeek 真·流式：信息抽取、规划生成和聊天 reply 都以 `stream=true` 调用 Chat Completions。结构化 tool args 会在后端聚合后解析；规划生成会同步扫描 `summary` 字段并通过 `assistant_delta` 增量推送，单帧长度不再保证等长。设为 `false` 时整体回退到非流式 + `chunkText` 等长切片（生产回滚开关）。详细 wire format 见 `docs/external-apis.md`。

如果 LLM 未启用、配置缺失、provider 不支持 schema 输出、tool call 缺失、返回结构无效或业务校验失败，Eino planner 会自动 fallback 到 deterministic generator，并在 `warnings` 中记录原因。

LLM prompt 当前版本为 `travel-plan-v1`。成功调用会记录耗时和 token usage；provider 未返回 usage 时显示 `unknown`。fallback warning 使用稳定分类：

```text
LLM trace: prompt_version=travel-plan-v1 duration_ms=123 prompt_tokens=unknown completion_tokens=unknown total_tokens=unknown
LLM fallback: prompt_version=travel-plan-v1 category=invalid_json attempts=1 duration_ms=123 reason=...
```

fallback category 包括 `disabled`、`missing_api_key`、`provider_error`、`timeout`、`invalid_json`、`business_validation_failed` 和 `retry_exhausted`。

## 外部 Tool 模式

Eino Tools 默认使用本地 mock：

```bash
go run ./cmd/harness -planner eino
```

如需使用真实高德 POI、路线和天气 API：

```bash
set TRAVEL_AGENT_TOOL_MODE=real
set TRAVEL_AGENT_AMAP_API_KEY=your-amap-key
go run ./cmd/harness -planner eino
```

可选配置：

* `TRAVEL_AGENT_AMAP_BASE_URL`
* `TRAVEL_AGENT_WEATHER_API_KEY`
* `TRAVEL_AGENT_WEATHER_BASE_URL`
* `TRAVEL_AGENT_EXTERNAL_API_TIMEOUT`
* `TRAVEL_AGENT_EXTERNAL_API_CONCURRENCY`
* `TRAVEL_AGENT_EXTERNAL_API_QPS`

real tool 初始化失败、请求失败或响应缺字段时，会自动 fallback 到 mock tool，并在 `warnings` 中说明原因。
高德 real tools 默认在后端进程内限流：最多 2 个并发外部 API 请求，且每秒最多发起 2 次请求；可通过 `TRAVEL_AGENT_EXTERNAL_API_CONCURRENCY` 和 `TRAVEL_AGENT_EXTERNAL_API_QPS` 调整。

真实价格数据采用“能拿到才展示”的规则：高德 POI `extensions=all` 的 `biz_ext.cost` 可用于餐饮/景点参考金额；路线接口可用的 `taxi_cost`、`tolls`、`transits[].cost` 可用于市内交通。住宿、跨城高铁/飞机等尚未接入真实票价的数据展示为“暂无信息”，不计入 `known_total`。

fallback warning 使用稳定格式，便于后续报告统计：

```text
tool fallback: tool=poi provider=amap stage=request category=provider_error mock_fallback=true reason=...
```

当前分类包括 `configuration`、`timeout`、`rate_limit`、`provider_error`、`invalid_json`、`missing_field`、`request_error` 和 `unknown`。默认 `TRAVEL_AGENT_TOOL_MODE=mock` 不会调用任何外部 API；`real` mode 配置不完整、触发限流或 provider 异常时会降级到 mock，避免本地 Harness 因外部依赖不可用而失败。

## 路线真实性校验

Eino planner 会在生成最终计划前执行轻量路线真实性校验。该校验只写入内部状态和 `TravelPlan.warnings`，不改变 `domain.TravelPlan` schema。

当前检查：

* 每日 POI 数量是否匹配 `pace`
* 相邻 POI route duration 是否过长
* POI 坐标是否缺失
* 雨天是否有室内友好备选
* 已知预算拆分是否与 `known_total` 一致
* 同一天是否重复明显相同 POI

warning 示例：

```text
route feasibility: check=poi_coordinates score=90 message=some POIs do not have coordinates; route duration may use mock fallback
```

这不是地图级精准排程；真实 API 不可用时仍允许 mock Harness 稳定通过。

## 如何添加新的测试用例

在 `testdata/travel_cases.json` 中新增一条 case。每条 case 必须包含唯一 `id`、清晰说明覆盖范围的 `description`、与 `id` 一致的 `input.id`、完整的 `input`，以及包含 `min_days`、`max_budget_ratio`、`required_keywords` 的 `expectation`。

`required_keywords` 至少应包含目的地城市，用于校验标题、摘要或路线内容是否命中核心目的地。

新增、删除或重命名后端 case 时，必须同步更新 `testdata/harness-cases.md`；新增、删除或重命名前端 UI Harness test 时，必须同步更新 `web/e2e/harness-cases.md`。两份文档分别说明 case 覆盖的场景、输入摘要和主要验证点。

当前数据集包含常规多日游、单日游、低预算、高预算、长天数、未知城市、同城游、不同节奏、不同交通方式，以及 Stage 18 的人数、避开内容、必去地点和默认值路径等 29 条覆盖 case。

## 报告输出

Harness 会输出控制台摘要，并生成：

```text
reports/eval_report.json
```

JSON 报告包含 `generated_at`、`planner_type`、`summary` 和 `cases`。每个 case result 会包含 `description`，适合后续脚本、CI 或看板读取。

## 当前评估指标

* SuccessRate
* AverageScore
* AverageDurationMs
* BudgetPassRate
* DayMatchRate
* StructurePassRate
* ToolFallbackRate
* LLMFallbackRate
* ExternalAPISuccessRate
* AverageNodeDurationMs
* AverageTokenUsage
* RouteFeasibilityPassRate
* WarningRate

单条 case 按 100 分计算：基础成功 20 分、天数匹配 20 分、预算合规 20 分、结构完整 20 分、关键词匹配 10 分、无非法字段 10 分。

报告中的每条 case 还包含 `diagnostics` 和失败时的 `failure` 快照。当前 diagnostics 基于稳定 warning 格式聚合 tool fallback、LLM fallback、token usage、节点耗时和路线真实性信号；Harness 仍只依赖 `TravelPlanner` 接口，不 import Eino 实现。

## 如何接入真实 Eino Agent

当前已有 `EinoTravelPlanner`，它实现了 `TravelPlanner` 接口：

```go
type EinoTravelPlanner struct {
	// graph / model / tools
}

func (p *EinoTravelPlanner) Plan(ctx context.Context, req domain.TravelRequest) (*domain.TravelPlan, error) {
	// 调用 Eino Graph
}
```

后续接入真实 LLM 或外部 API 时，应在 `internal/agent/eino` 内替换或扩展节点和工具，继续保持 Harness 只依赖 `TravelPlanner` 接口。

## 后续扩展方向

* 接入真实外部 API
* 接入高德地图 POI 和路线 API
* 增加 Agent Tool 调用轨迹评估
* 增加 Token 消耗统计
* 增加外部 API 调用成功率统计
* 增加路线合理性人工评分
* 增加并发压测模式
* 增加 benchmark 命令
* 增加 JSON Schema 校验
* 增加多模型对比
* 增加不同 Prompt 版本对比
* 增加 Eino Graph 节点级耗时统计
* 增加失败 case 自动保存输入和输出快照

更多说明见 `docs/evaluation-harness.md`、`testdata/harness-cases.md` 和 `web/e2e/harness-cases.md`。

## 用户与计划库

Travel Agent 在异步规划之外提供一层用户资产能力：注册登录、计划保存、用户中心 CRUD、发布/取消发布，以及首页公开计划与搜索。

开启方式（`.env.example` 已同步）：

```bash
TRAVEL_AGENT_AUTH_ENABLED=true
TRAVEL_AGENT_SESSION_COOKIE_NAME=travel_agent_session
TRAVEL_AGENT_SESSION_TTL_HOURS=168
TRAVEL_AGENT_PASSWORD_MIN_LENGTH=8
TRAVEL_AGENT_PUBLIC_PLAN_PAGE_SIZE=20
TRAVEL_AGENT_ALLOW_ANONYMOUS_PLAN_GENERATION=false
TRAVEL_AGENT_COOKIE_SECURE=false
TRAVEL_AGENT_COOKIE_DOMAIN=
TRAVEL_AGENT_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
```

`TRAVEL_AGENT_ALLOW_ANONYMOUS_PLAN_GENERATION=false` 是默认值，意味着 `/api/v1/travel/plans*` 也要登录。本地希望保留旧的匿名链路时把它改成 `true`。

执行迁移（MySQL 启用时）：

```bash
mysql -u root -p travel_agent < migrations/mysql/003_users_and_plan_library.sql
```

未启用 MySQL 也能开起来：服务会回落到 `auth.MemoryUserStore` / `plans.MemoryPlanStore` 等内存实现，进程重启后数据丢失，仅供本地体验与 e2e。

新增接口（详见 `docs/api.md`「Auth & Plan Library API」章节）：

```text
POST /api/v1/auth/register | login | logout
GET  /api/v1/auth/me
POST /api/v1/me/plans                        保存计划
GET  /api/v1/me/plans                        我的计划列表
GET  /api/v1/me/plans/:id                    详情（含完整 TravelPlan）
PATCH/DELETE /api/v1/me/plans/:id            编辑 / 软删除
POST /api/v1/me/plans/:id/publish | unpublish
GET  /api/v1/me/current                      首页"当前计划"入口
GET  /api/v1/public/plans                    公开排行 / 搜索
GET  /api/v1/public/plans/:id                公开详情
POST /api/v1/public/plans/:id/save           保存为我的副本
```

前端：

* 路由由 `react-router-dom` 接管：`/`、`/login`、`/planner`、`/me`、`/me/plans/:planId`、`/public`、`/public/:publicPlanId`。
* `useAuth` Provider 在 `main.tsx` 顶层注入；`RequireAuth` 包装私有路由，未登录跳 `/login?return_to=...`。
* 设计 token 在 `web/src/styles.css` 顶部的 `:root` 注入；旧的聊天界面保持原视觉。

测试：

```bash
go test ./...
go vet ./...
cd web
npm run typecheck
npm run lint
npm run build
npm run harness:ui
```

后续优化路线见 `requirements/stage-21-auth-home-plan-community.md`。
