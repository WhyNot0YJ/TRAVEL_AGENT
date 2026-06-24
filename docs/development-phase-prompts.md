# Travel Agent 分阶段开发提示词

说明：本文记录项目分阶段开发提示词和阶段交接上下文。`skills/agent-feature-dev.md` 当前不存在，后续阶段需要创建或补充；核心架构、API、数据库、外部 API 和 Harness 文档已随实现逐步补齐，新增阶段仍应按实际实现同步更新文档，不要一次性编造尚未落地的复杂功能细节。

## 阶段 1：Harness + MockPlanner 提示词

```text
# 阶段 1：Harness + MockPlanner

## 任务目标

搭建 Travel Agent Evaluation Harness 的最小可运行版本：

* 定义 `TravelPlanner` 接口
* 实现 deterministic `MockPlanner`
* 定义基础领域模型 `TravelRequest`、`TravelPlan`
* 读取 `testdata/travel_cases.json`
* 执行 evaluator
* 输出 `reports/eval_report.json`
* 提供 `cmd/harness` 命令行入口

## 当前前置条件

这是第 1 阶段，不依赖 Eino、Gin、Redis、数据库或前端。

如果仓库中已有部分实现，请先阅读并复用现有结构，只补齐缺失能力，不要重写无关代码。

## 本阶段不做什么

* 不接 CloudWeGo Eino
* 不接 Gin
* 不接 Redis
* 不接 MySQL / PostgreSQL
* 不接真实 LLM
* 不接真实 POI / 天气 / 路线 API
* 不做 SSE
* 不做 React 前端
* 不实现异步任务系统
* 不新增复杂评测指标

## 需要阅读的文件

* `AGENTS.md`
* `README.md`
* `docs/prd.md`
* `docs/architecture.md`
* `docs/evaluation-harness.md`
* `docs/api.md`
* `docs/database.md`
* `skills/agent-feature-dev.md`：如果不存在，说明该文件当前不存在，后续阶段需要创建
* 现有 `internal`、`cmd`、`testdata` 目录结构

## 需要新增或修改的文件

预计新增或修改：

* `go.mod`
* `cmd/harness/main.go`
* `internal/domain/travel.go`
* `internal/agent/planner.go`
* `internal/agent/mock_planner.go`
* `internal/agent/mock_planner_test.go`
* `internal/harness/dataset.go`
* `internal/harness/evaluator.go`
* `internal/harness/metrics.go`
* `internal/harness/report.go`
* `internal/harness/runner.go`
* `cmd/harness/main_test.go`
* `testdata/travel_cases.json`
* `reports/eval_report.json`
* `README.md`
* `docs/evaluation-harness.md`
* `Makefile`

## 实现要求

* `internal/domain` 只放通用业务实体，不依赖 harness 或 agent。
* `TravelPlanner` 接口放在 `internal/agent`，签名为：`Plan(ctx context.Context, req domain.TravelRequest) (*domain.TravelPlan, error)`。
* `MockPlanner` 必须稳定、可复现，不调用 LLM 或外部 API。
* `MockPlanner` 输出必须包含 title、summary、days、budget、warnings。
* `testdata/travel_cases.json` 至少包含 5 到 8 条本地评估 case。
* Dataset loader 需要校验 case id、input.id、destination_city、days、budget、expectation。
* Evaluator 至少评估 planner 是否成功、天数是否匹配、预算是否未超出阈值、结构是否完整、required keywords 是否命中、是否存在非法负数值。
* Summary metrics 至少包含 SuccessRate、AverageScore、AverageDurationMs、BudgetPassRate、DayMatchRate、StructurePassRate。
* 报告必须写入 `reports/eval_report.json`。
* CLI 默认读取 `testdata/travel_cases.json`，默认输出 `reports/eval_report.json`。
* 支持参数：`-dataset`、`-report`。
* 保持代码小而清晰，不要提前引入 Eino、Gin 或大型框架。

## 文档更新要求

* 更新 `README.md`：说明 Harness 目标、运行方式、报告位置。
* 更新 `docs/evaluation-harness.md`：说明测试用例、评估指标、报告字段、运行方式。
* 不需要更新 `docs/api.md`，因为本阶段没有 HTTP API。
* 不需要更新 `docs/database.md`，因为本阶段没有数据库结构。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness
make harness
```

如果 `make harness` 尚未存在，需要补充 Makefile target。

## 验收标准

* `go test ./...` 通过。
* `go run ./cmd/harness` 可运行。
* 生成 `reports/eval_report.json`。
* 控制台能看到评估摘要。
* Harness 只依赖 `TravelPlanner` 接口。
* 没有 Eino、Gin、Redis、真实 LLM、真实外部 API 依赖。

## 最终回复要求

完成后请输出：

1. 修改了哪些文件
2. 新增了哪些文件
3. 做了哪些设计决策
4. 跑了哪些测试
5. 测试是否通过
6. 报告如何验证
7. 风险和未完成事项
```

## 阶段 2：EinoTravelPlanner + Mock Tools 提示词

```text
# 阶段 2：EinoTravelPlanner + Mock Tools

## 任务目标

在已有 Harness + MockPlanner 基础上接入 CloudWeGo Eino：

* 新增 `EinoTravelPlanner`
* `EinoTravelPlanner` 实现 `TravelPlanner` 接口
* 使用 Eino Graph / Workflow 编排旅行规划流程
* 接入 Mock POI Tool、Mock Weather Tool、Mock Route Tool、Mock Budget Tool
* Harness 支持 `-planner mock` 和 `-planner eino`
* 保持默认 planner 为 `mock`

## 当前前置条件

第 1 阶段已完成：`TravelPlanner` 接口已存在，`MockPlanner` 已存在且可运行，`cmd/harness` 已可读取 dataset 并输出报告，`docs/evaluation-harness.md` 已描述基础 Harness。

如果仓库中已有 `internal/agent/eino`，请先阅读现有实现，只补齐缺失能力，不要破坏已通过的 MockPlanner 和 Harness。

## 本阶段不做什么

* 不接真实 LLM
* 不接真实高德 / 天气 / POI API
* 不接 Gin
* 不接 Redis
* 不接数据库
* 不做 SSE
* 不做 React 前端
* 不让 `internal/harness` 直接依赖 Eino 具体实现

## 需要阅读的文件

* `AGENTS.md`
* `README.md`
* `docs/agent-flow.md`
* `docs/evaluation-harness.md`
* `docs/architecture.md`
* `internal/domain/travel.go`
* `internal/agent/planner.go`
* `internal/agent/mock_planner.go`
* `internal/harness/*`
* `cmd/harness/main.go`
* `testdata/travel_cases.json`
* `skills/agent-feature-dev.md`：如果不存在，说明该文件当前不存在，后续阶段需要创建

## 需要新增或修改的文件

预计新增或修改：

* `go.mod`
* `go.sum`
* `cmd/harness/main.go`
* `internal/agent/eino/planner.go`
* `internal/agent/eino/graph.go`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/tools.go`
* `internal/agent/eino/types.go`
* `internal/agent/eino/callbacks.go`
* `internal/agent/eino/planner_test.go`
* `README.md`
* `docs/agent-flow.md`
* `docs/evaluation-harness.md`
* `Makefile`

## 实现要求

* Eino 相关代码只能放在 `internal/agent/eino` 或其子目录。
* `EinoTravelPlanner` 必须实现 `agent.TravelPlanner`。
* `cmd/harness` 负责根据 `-planner` 选择 `mock` 或 `eino`。
* `internal/harness` 不能 import `internal/agent/eino`。
* Eino Graph 建议包含 ParseTravelRequestNode、SearchPOIsToolNode、GetWeatherToolNode、ComputeRouteToolNode、EstimateBudgetToolNode、OptimizeItineraryNode、GenerateTravelPlanNode、ValidatePlanNode。
* Mock Tools 必须稳定、可复现，不调用任何外部 API。
* Eino 输出必须仍然是 `domain.TravelPlan`。
* 保留 `MockPlanner` 不变或只做必要兼容调整。
* Harness 报告需要记录 planner type。
* `go run ./cmd/harness -planner mock` 和 `go run ./cmd/harness -planner eino` 都应可运行。

## 文档更新要求

* 更新 `docs/agent-flow.md`：说明 Eino Graph 节点、输入输出、Mock Tools。
* 更新 `docs/evaluation-harness.md`：说明 `-planner mock/eino`。
* 更新 `README.md`：补充 Eino 模式运行方式。
* 不更新 `docs/api.md`，因为本阶段没有 HTTP API。
* 不更新 `docs/external-apis.md`，因为仍然没有真实外部 API。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
make harness
make harness-mock
make harness-eino
```

## 验收标准

* MockPlanner 仍可运行。
* EinoTravelPlanner 可通过 Harness 运行。
* `reports/eval_report.json` 可生成且包含 `planner_type`。
* `internal/harness` 没有直接依赖 `internal/agent/eino`。
* Eino 模式不调用真实 LLM 或外部 API。
* 文档准确说明当前只是 Eino Graph + Mock Tools。

## 最终回复要求

完成后请输出：

1. 修改了哪些文件
2. 新增了哪些文件
3. 做了哪些设计决策
4. 跑了哪些测试
5. 测试是否通过
6. 报告如何验证
7. 风险和未完成事项
```

## 阶段 3：真实 LLM 提示词

```text
# 阶段 3：接入真实 LLM

## 任务目标

在 `EinoTravelPlanner` 中接入真实 ChatModel：

* 通过环境变量配置 LLM API Key、BaseURL、Model
* 新增 LLM 生成 TravelPlan 的节点或组件
* LLM 输出必须解析为结构化 `domain.TravelPlan`
* 增加 JSON 解析、校验、重试、降级
* 保留 Mock Plan Generator 作为 fallback
* Harness 支持真实 LLM 模式

## 当前前置条件

第 1、2 阶段已完成：Harness、MockPlanner、EinoTravelPlanner 均可运行，EinoTravelPlanner 当前使用 Eino Graph + Mock Tools。

## 本阶段不做什么

* 不接真实高德 / 天气 / POI API
* 不接 Gin
* 不接 Redis
* 不接数据库
* 不做 SSE
* 不做 React 前端
* 不破坏 Mock Tools
* 不硬编码任何 API Key
* 不让 Harness 直接依赖 Eino 或 LLM 实现

## 需要阅读的文件

* `AGENTS.md`
* `README.md`
* `docs/agent-flow.md`
* `docs/evaluation-harness.md`
* `docs/external-apis.md`
* `internal/domain/travel.go`
* `internal/agent/planner.go`
* `internal/agent/eino/*`
* `internal/harness/*`
* `cmd/harness/main.go`
* `skills/agent-feature-dev.md`：如果不存在，说明该文件当前不存在，后续阶段需要创建

## 需要新增或修改的文件

预计新增或修改：

* `internal/agent/eino/config.go`
* `internal/agent/eino/llm.go`
* `internal/agent/eino/prompt.go`
* `internal/agent/eino/json_parser.go`
* `internal/agent/eino/graph.go`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/planner.go`
* `internal/agent/eino/*_test.go`
* `cmd/harness/main.go`
* `README.md`
* `docs/agent-flow.md`
* `docs/evaluation-harness.md`
* `docs/external-apis.md`

## 实现要求

* LLM 配置必须来自环境变量，例如 `TRAVEL_AGENT_LLM_ENABLED`、`TRAVEL_AGENT_LLM_API_KEY`、`TRAVEL_AGENT_LLM_BASE_URL`、`TRAVEL_AGENT_LLM_MODEL`、`TRAVEL_AGENT_LLM_TIMEOUT`、`TRAVEL_AGENT_LLM_MAX_RETRIES`。
* 未启用 LLM 或配置缺失时，必须自动使用现有 mock generator fallback。
* 不允许在代码、测试或文档示例中写真实 API Key。
* LLM prompt 要要求输出严格 JSON，字段匹配 `domain.TravelPlan`。
* 增加解析逻辑：去除 markdown code fence、解析 JSON、校验 title/summary/days/items/budget、校验 days 数量、校验预算非负且尽量不超预算。
* 增加重试：JSON 解析失败可重试，结构校验失败可重试，重试后仍失败则 fallback 到 mock generator。
* fallback 必须在 warnings 中标记原因。
* Harness 可通过参数或环境变量启用 LLM 模式。
* 单元测试必须覆盖 LLM disabled fallback、无效 JSON fallback、合法 JSON 转换成功、校验失败重试或 fallback。

## 文档更新要求

* 更新 `docs/agent-flow.md`：说明 LLM 生成节点、fallback、校验流程。
* 更新 `docs/evaluation-harness.md`：说明如何运行 LLM 模式、报告仍为 `reports/eval_report.json`。
* 更新 `docs/external-apis.md`：说明 LLM provider 配置项，但不要写真实 key。
* 更新 `README.md`：补充 LLM 模式运行示例。
* 不更新 `docs/api.md`，因为本阶段没有 HTTP API。
* 不更新 `docs/database.md`，因为本阶段没有数据库结构。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
make harness
make harness-eino
```

如果有可用测试 key，可额外手动运行真实 LLM 模式，但不要把真实 key 写入仓库。

## 验收标准

* 未配置 LLM 时，Eino 模式仍可稳定运行。
* 配置 LLM 时，能够调用 ChatModel 生成结构化 TravelPlan。
* LLM 输出解析失败时不会让 Harness 崩溃。
* fallback 机制可用。
* MockPlanner 不受影响。
* 文档说明真实 LLM 的配置、限制和运行方式。

## 最终回复要求

完成后请输出：

1. 修改了哪些文件
2. 新增了哪些文件
3. 做了哪些设计决策
4. 跑了哪些测试
5. 测试是否通过
6. 报告如何验证
7. 风险和未完成事项
```

## 阶段 4：真实高德 / 天气 / POI API 提示词

```text
# 阶段 4：接入真实高德 / 天气 / POI API

## 任务目标

为 Eino Tools 增加真实外部 API 能力：

* 新增外部 API client
* 接入真实 POI API
* 接入真实路线规划 API
* 接入真实天气 API
* 通过环境变量配置 API Key
* 保留 Mock Tools 作为 fallback
* Eino Tools 支持 mock 和 real 两种模式
* Harness 可用 mock tools 或 real tools 运行

## 当前前置条件

第 1、2、3 阶段已完成：Harness、MockPlanner、EinoTravelPlanner、LLM 模式与 fallback 均可运行。

## 本阶段不做什么

* 不接 Gin
* 不接 Redis
* 不接数据库
* 不做 SSE
* 不做 React 前端
* 不实现酒店、门票、支付或用户系统
* 不移除 Mock Tools
* 不硬编码任何 API Key
* 不让 `internal/harness` 直接依赖具体 API client

## 需要阅读的文件

* `AGENTS.md`
* `README.md`
* `docs/agent-flow.md`
* `docs/evaluation-harness.md`
* `docs/external-apis.md`
* `internal/agent/eino/*`
* `internal/domain/travel.go`
* `cmd/harness/main.go`
* `skills/agent-feature-dev.md`：如果不存在，说明该文件当前不存在，后续阶段需要创建

## 需要新增或修改的文件

预计新增或修改：

* `internal/agent/eino/config.go`
* `internal/agent/eino/tools.go`
* `internal/agent/eino/real_poi_tool.go`
* `internal/agent/eino/real_weather_tool.go`
* `internal/agent/eino/real_route_tool.go`
* `internal/agent/eino/api_client.go`
* `internal/agent/eino/tool_mode.go`
* `internal/agent/eino/*_test.go`
* `cmd/harness/main.go`
* `README.md`
* `docs/agent-flow.md`
* `docs/evaluation-harness.md`
* `docs/external-apis.md`

## 实现要求

* 外部 API 配置必须来自环境变量，例如 `TRAVEL_AGENT_TOOL_MODE=mock|real`、`TRAVEL_AGENT_AMAP_API_KEY`、`TRAVEL_AGENT_AMAP_BASE_URL`、`TRAVEL_AGENT_WEATHER_API_KEY`、`TRAVEL_AGENT_WEATHER_BASE_URL`、`TRAVEL_AGENT_EXTERNAL_API_TIMEOUT`。
* 默认 tool mode 必须是 `mock`。
* real tools 初始化失败或请求失败时，应按配置 fallback 到 mock tools，并在 warnings 中说明。
* 抽象 Tool 接口，避免 Graph 节点绑定具体 mock 类型。
* HTTP client 必须设置 timeout。
* API 响应解析要有错误处理，不假设字段永远存在。
* 不要把真实 API 返回结构污染到 `internal/domain`。
* `internal/domain` 只保留稳定业务模型。
* Harness 可通过环境变量或 CLI 参数选择 mock / real tools。
* 单元测试用 fake HTTP server 或 mock client，不调用真实外部网络。
* 保持 `go run ./cmd/harness -planner eino` 在无 key 环境下仍可运行。

## 文档更新要求

* 更新 `docs/external-apis.md`：说明 POI、路线、天气 API 的配置项、用途、fallback 策略、限制。
* 更新 `docs/agent-flow.md`：说明 Eino Tools 支持 mock / real。
* 更新 `docs/evaluation-harness.md`：说明如何使用 mock tools 或 real tools 运行。
* 更新 `README.md`：补充环境变量示例。
* 不更新 `docs/api.md`，因为本阶段没有 HTTP API。
* 不更新 `docs/database.md`，因为本阶段没有数据库结构。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
make harness
make harness-eino
```

如有真实 API Key，可手动运行 real tools；不要把真实 key 写入仓库。

## 验收标准

* 默认 mock tools 仍可运行。
* real tools 配置齐全时可以被 Eino Graph 使用。
* real tools 出错时可 fallback 到 mock。
* 所有 API Key 均来自环境变量。
* Harness 不直接依赖具体外部 API client。
* 文档清楚说明外部 API 配置与限制。

## 最终回复要求

完成后请输出：

1. 修改了哪些文件
2. 新增了哪些文件
3. 做了哪些设计决策
4. 跑了哪些测试
5. 测试是否通过
6. 报告如何验证
7. 风险和未完成事项
```

## 阶段 5：Gin API 提示词

```text
# 阶段 5：接入 Gin API

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
```

## 阶段 6：Redis 缓存、限流、任务状态提示词

```text
# 阶段 6：Redis 缓存、限流、任务状态

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
```

## 阶段 7：SSE 提示词

```text
# 阶段 7：SSE 流式接口

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
```

## 阶段 8：React 前端提示词

```text
# 阶段 8：React + TypeScript 前端

## 任务目标

新增 React + TypeScript H5 前端：

* 接入 Gin API
* 支持创建旅游路线规划请求
* 支持 SSE 流式展示生成过程
* 支持路线详情展示
* 支持 loading / error / empty 状态
* 支持 typed API client
* 先做 H5，后续再套壳 App

## 当前前置条件

第 1 到 7 阶段已完成：后端 Gin API、POST 创建任务、GET 查询任务、SSE stream 均可用，`docs/api.md` 已描述接口。

## 本阶段不做什么

* 不改后端核心 Planner 逻辑，除非发现前端必须依赖的小型 API bug
* 不接登录注册
* 不做支付、订单、收藏、分享
* 不接移动端原生壳
* 不引入大型 UI 框架，除非项目已有明确选择
* 不做复杂地图渲染
* 不硬编码后端地址，配置来自环境变量

## 需要阅读的文件

* `AGENTS.md`
* `README.md`
* `docs/api.md`
* `docs/architecture.md`
* `cmd/server/main.go`
* `internal/travel/dto.go`
* `internal/travel/handler.go`
* 现有前端目录：如果不存在，需要创建
* `skills/agent-feature-dev.md`：如果不存在，说明该文件当前不存在，后续阶段需要创建

## 需要新增或修改的文件

预计新增或修改：

* `web/package.json`
* `web/tsconfig.json`
* `web/vite.config.ts`
* `web/index.html`
* `web/src/main.tsx`
* `web/src/App.tsx`
* `web/src/api/client.ts`
* `web/src/api/types.ts`
* `web/src/hooks/useTravelPlanStream.ts`
* `web/src/components/TravelPlanForm.tsx`
* `web/src/components/PlanProgress.tsx`
* `web/src/components/PlanDetail.tsx`
* `web/src/components/StateView.tsx`
* `web/src/styles.css`
* `README.md`
* `docs/api.md`：如发现前后端契约需要补充
* `docs/architecture.md`

## 实现要求

* 使用 React + TypeScript。
* 建议使用 Vite，除非项目已有前端构建方案。
* API base URL 来自环境变量，例如 `VITE_API_BASE_URL`。
* typed API client 必须覆盖创建任务、查询任务、连接 SSE。
* 表单至少包含出发城市、目的地城市、天数、预算、兴趣标签、交通方式、节奏。
* 提交后调用 POST 创建任务，展示 loading/progress，连接 SSE stream，收到 done 后展示路线详情。
* 必须实现 loading、error、empty 状态，以及 SSE 断开或失败降级提示。
* UI 以移动端 H5 为主，布局紧凑、清晰、适合反复使用。
* 不要做营销 landing page，第一屏就是可用的路线规划工具。
* 不要用大段说明文字描述功能。
* 表单、按钮、状态提示在移动端不能溢出或重叠。
* 路线详情展示 days、items、budget、warnings。
* TypeScript 类型不要用大量 `any`。
* 如后端跨域未配置，可最小化补充 CORS 配置，并同步更新 docs。

## 文档更新要求

* 更新 `README.md`：说明前端启动、环境变量、后端依赖。
* 更新 `docs/architecture.md`：说明 React -> Gin API -> SSE 的关系。
* 如 API 契约有补充，更新 `docs/api.md`。
* 不更新 `docs/database.md`，除非新增持久化结构。
* 不更新 `docs/agent-flow.md`，除非改变 Agent 流程。

## 测试要求

后端尽量运行：

```bash
go test ./...
go vet ./...
```

前端尽量运行：

```bash
cd web
npm install
npm run typecheck
npm run lint
npm run build
```

如果项目未配置 lint，需要在最终回复中说明未运行原因，或补充合理 lint 脚本。

本地联调：

```bash
go run ./cmd/server
cd web
npm run dev
```

## 验收标准

* 前端可启动。
* 可以提交旅行规划请求。
* 可以看到 SSE progress。
* done 后展示路线详情。
* loading / error / empty 状态完整。
* typed API client 存在。
* 移动端 H5 布局可用。
* 后端 Harness 和 API 不被破坏。

## 最终回复要求

完成后请输出：

1. 修改了哪些文件
2. 新增了哪些文件
3. 做了哪些设计决策
4. 跑了哪些测试
5. 测试是否通过
6. 前端和接口如何验证
7. 风险和未完成事项
```
## 阶段 9：真实外部 Tool 稳定化提示词

```text
# 阶段 9：真实外部 Tool 稳定化

## 任务目标

把现有高德 POI、路线、天气 adapter 从“可配置可调用”推进到“可验证、可观测、可降级”的工程能力。默认 mock tool 行为必须保持稳定；real tool 只在显式配置时启用，并且所有失败都要可解释地 fallback。

## 当前上下文

项目当前已经实现：

* `internal/agent/eino/real_poi_tool.go`
* `internal/agent/eino/real_route_tool.go`
* `internal/agent/eino/real_weather_tool.go`
* `internal/agent/eino/tool_mode.go`
* mock/real tool mode
* real tool fallback 到 mock tool
* `TravelPlan.warnings` 记录 fallback 原因

但当前还缺少真实外部 API 稳定性验证、失败分类、调用轨迹和生产数据下的边界覆盖。

## 不做什么

* 不引入新的大型外部 API SDK。
* 不把外部 API 原始响应放进 `internal/domain`。
* 不破坏 mock tools。
* 不把 API Key 写入代码、测试、文档或报告。
* 不做酒店、票务、支付、用户系统。

## 需要阅读的文件

* `AGENTS.md`
* `README.md`
* `docs/agent-flow.md`
* `docs/external-apis.md`
* `docs/evaluation-harness.md`
* `internal/agent/eino/tool_mode.go`
* `internal/agent/eino/tools.go`
* `internal/agent/eino/real_poi_tool.go`
* `internal/agent/eino/real_route_tool.go`
* `internal/agent/eino/real_weather_tool.go`
* `internal/agent/eino/real_tools_test.go`
* `internal/agent/eino/nodes.go`

## 实现要求

* 标准化 Tool fallback warning，例如包含 tool name、失败阶段、外部 provider、是否使用 mock fallback。
* 增加或完善 fake HTTP server 测试，覆盖：
  * 未配置 API Key
  * HTTP 超时
  * 非 2xx 或 provider 返回失败状态
  * JSON 无效
  * 必填字段缺失
  * POI 坐标缺失导致路线 fallback
  * 天气城市编码失败
* real tools 的 HTTP timeout 必须来自 `TRAVEL_AGENT_EXTERNAL_API_TIMEOUT`。
* 所有外部响应解析必须留在 `internal/agent/eino` 内部。
* 保证 `TRAVEL_AGENT_TOOL_MODE=mock` 时不会发起外部请求。
* 保证 `TRAVEL_AGENT_TOOL_MODE=real` 但配置不完整时可以稳定 fallback。

## 文档更新要求

* 更新 `docs/external-apis.md`：说明 real tool 稳定性、fallback 分类和限制。
* 更新 `docs/agent-flow.md`：如新增 warning 或 trace 字段，需要同步说明。
* 更新 `docs/evaluation-harness.md`：说明如何用 real tool mode 跑评估，以及默认不依赖真实外部 API。
* 更新 `README.md`：只补充实际可运行命令和环境变量，不夸大真实 API 覆盖能力。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
```

如有条件配置测试 Key，可手动运行：

```bash
set TRAVEL_AGENT_TOOL_MODE=real
set TRAVEL_AGENT_AMAP_API_KEY=your-key
go run ./cmd/harness -planner eino
```

## 验收标准

* 默认 mock mode 全部测试通过。
* real mode 的异常路径都有测试覆盖。
* real mode 不可用时不会导致 planner 整体失败，除非输入本身非法。
* fallback warning 可读、可分类、可用于报告统计。
* 文档不再把 adapter 能力描述成未经验证的生产级能力。
```

## 阶段 10：真实 LLM 规划链路硬化提示词

```text
# 阶段 10：真实 LLM 规划链路硬化

## 任务目标

让 `TRAVEL_AGENT_LLM_ENABLED=true` 模式成为可评估、可回退、可追踪的真实规划链路。LLM 输出仍必须经过 JSON Schema 结构约束和本地业务校验；任何失败都不能污染 `internal/domain` 或破坏 deterministic fallback。

## 当前上下文

项目当前已经实现：

* DeepSeek/OpenAI-compatible 配置
* provider-native JSON Schema / strict tool call 输出
* `submit_travel_plan` schema
* LLM disabled fallback
* 无效 JSON / 校验失败 fallback
* `TravelPlan.warnings` 记录 LLM fallback 原因

但还缺少 prompt version 管理、token/耗时统计、失败样本快照、多 provider 对比和 LLM 模式下的评估报告字段。

## 不做什么

* 不使用 prompt-only JSON 作为主路径。
* 不把 API Key 写入代码、测试、文档或报告。
* 不让 Harness 直接依赖 Eino 或 LLM provider。
* 不移除 deterministic generator。
* 不为了 LLM 输出放宽 domain 校验。

## 需要阅读的文件

* `AGENTS.md`
* `docs/agent-flow.md`
* `docs/external-apis.md`
* `docs/evaluation-harness.md`
* `internal/agent/eino/llm.go`
* `internal/agent/eino/llm_test.go`
* `internal/agent/eino/prompt.go`
* `internal/agent/eino/schema.go`
* `internal/agent/eino/json_parser.go`
* `internal/agent/eino/config.go`
* `internal/agent/eino/nodes.go`
* `internal/domain/travel.go`

## 实现要求

* 为 prompt 增加显式版本标识，记录到内部 trace 或 warning/metadata 中。
* 标准化 LLM fallback reason，至少区分：
  * disabled
  * missing_api_key
  * provider_error
  * timeout
  * invalid_json
  * schema_violation
  * business_validation_failed
  * retry_exhausted
* 如果 provider 返回 token usage，记录 prompt/completion/total token。
* 如果 provider 不返回 token usage，不要伪造；记录 unknown。
* 记录 LLM 调用耗时。
* 增加 fake LLM HTTP server 测试，覆盖成功、无 tool call、坏 JSON、业务校验失败、重试后成功、重试耗尽 fallback。
* 保持 `TravelPlanner` 接口不变。

## 文档更新要求

* 更新 `docs/external-apis.md`：补充 LLM provider、token usage、fallback 分类。
* 更新 `docs/agent-flow.md`：补充 prompt version、LLM trace 和 fallback 流程。
* 更新 `docs/evaluation-harness.md`：如新增指标或报告字段，必须同步说明。
* 更新 `README.md`：补充 LLM 模式运行方式和限制。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
```

如有真实 Key，可手动运行：

```bash
set TRAVEL_AGENT_LLM_ENABLED=true
set TRAVEL_AGENT_LLM_PROVIDER=deepseek
set TRAVEL_AGENT_LLM_API_KEY=your-key
go run ./cmd/harness -planner eino
```

## 验收标准

* LLM 模式成功时输出结构化 `TravelPlan`。
* LLM 模式失败时 deterministic fallback 可用。
* token/耗时/失败原因可被后续 Harness 汇总。
* 所有新增字段有文档说明。
```

## 阶段 11：SQL 持久化与 Repository 层提示词

```text
# 阶段 11：SQL 持久化与 Repository 层

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
```

## 阶段 12：路线真实性校验提示词

```text
# 阶段 12：路线真实性校验

## 任务目标

把当前“结构正确”的路线规划升级为“基本可信”的路线规划。校验应覆盖 POI 坐标、相邻点交通耗时、营业时间、天气影响、预算拆分和每日强度，并以 warning、score 或 validation error 的形式反馈。

## 当前上下文

当前 `ValidatePlanNode` 主要校验：

* title/summary/days/items/budget 结构完整
* 天数匹配
* 预算阈值
* 非负字段
* 目的地关键词

当前不校验真实路线距离、营业时间、实时天气或 POI 可达性。

## 不做什么

* 不要求一次性做到地图级精准排程。
* 不把验证规则写死到 Harness 里绕过 Planner。
* 不让真实 API 不可用时导致默认 mock harness 失败。
* 不引入复杂地图渲染。

## 需要阅读的文件

* `docs/agent-flow.md`
* `docs/evaluation-harness.md`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/types.go`
* `internal/agent/eino/tools.go`
* `internal/agent/eino/real_route_tool.go`
* `internal/domain/travel.go`
* `internal/harness/evaluator.go`
* `internal/harness/metrics.go`
* `testdata/travel_cases.json`

## 实现要求

* 为路线真实性定义清晰的内部校验结果，不污染稳定 domain 模型；如需进入报告，放在 Harness result 或 planner metadata。
* 校验项建议包括：
  * 每天 POI 数量与 pace 是否匹配
  * 相邻 POI route duration 是否过长
  * POI 缺坐标时是否明确降级
  * 雨天是否增加室内备选 warning
  * 预算拆分是否和天数、交通模式大致一致
  * 同一天是否出现明显重复 POI
* 对真实数据不可得的情况输出 warning，不要伪造精确信息。
* 保持 mock 数据可稳定通过基础 Harness。

## 文档更新要求

* 更新 `docs/agent-flow.md`：说明 ValidatePlanNode 新增真实性校验。
* 更新 `docs/evaluation-harness.md`：如新增真实性评分或检查项，必须说明计算方式。
* 更新 `README.md`：补充当前真实性校验的能力边界。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
```

## 验收标准

* 基础结构校验保持兼容。
* 明显不可达、重复、预算异常或天气冲突能被识别。
* 新增 warning 或 score 字段有测试和文档。
* Harness 报告能体现真实性校验结果。
```

## 阶段 13：Evaluation Harness 升级提示词

```text
# 阶段 13：Evaluation Harness 升级

## 任务目标

把当前规则评分 Harness 升级为可比较不同 planner、LLM、tool mode、prompt version 的评估平台。报告应支持质量、耗时、成本、fallback 和真实性维度。

## 当前上下文

当前 Harness 已支持：

* `TravelPlanner` 接口
* `MockPlanner`
* `EinoTravelPlanner`
* 24 条本地测试 case
* SuccessRate、AverageScore、AverageDurationMs、BudgetPassRate、DayMatchRate、StructurePassRate
* JSON report 输出到 `reports/eval_report.json`

当前缺少：

* Tool trace 指标
* LLM fallback 指标
* Token 成本指标
* 外部 API 成功率
* 节点级耗时
* 并发压测和 benchmark
* 失败 case 快照
* 多 planner 对比报告

## 不做什么

* 不让 `internal/harness` 直接依赖 `internal/agent/eino`。
* 不在报告中保存 API Key 或敏感原始响应。
* 不用主观大模型打分替代基础规则评分，除非另起独立 optional evaluator。
* 不破坏现有 `reports/eval_report.json` 的基础可读性。

## 需要阅读的文件

* `docs/evaluation-harness.md`
* `README.md`
* `cmd/harness/main.go`
* `internal/harness/dataset.go`
* `internal/harness/runner.go`
* `internal/harness/evaluator.go`
* `internal/harness/metrics.go`
* `internal/harness/report.go`
* `internal/agent/planner.go`
* `testdata/travel_cases.json`

## 实现要求

* 新增指标前，先定义字段语义和默认值。
* 建议新增：
  * `ToolFallbackRate`
  * `LLMFallbackRate`
  * `ExternalAPISuccessRate`
  * `AverageNodeDurationMs`
  * `AverageTokenUsage`
  * `RouteFeasibilityPassRate`
  * `WarningRate`
* 支持 benchmark 或 concurrency 参数，例如并发数、重复次数、planner 类型。
* 失败 case 快照应保存输入、错误、warnings、必要的脱敏 trace。
* 多 planner 对比可以作为新增命令或报告模式，不要破坏默认命令。
* 保持 Harness 只依赖 `TravelPlanner` 接口；Eino 细节通过通用 metadata/trace 暴露。

## 文档更新要求

每次新增评估指标，必须同步更新：

* `internal/harness/evaluator.go`
* `internal/harness/metrics.go`
* `docs/evaluation-harness.md`
* `README.md`

如果报告 schema 变化，必须在 `docs/evaluation-harness.md` 中给出示例。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness
go run ./cmd/harness -planner eino
make harness
make harness-eino
```

## 验收标准

* 默认 harness 仍能输出 `reports/eval_report.json`。
* 新指标在 mock/eino 默认模式下有稳定值。
* 并发或 benchmark 模式不会破坏默认模式。
* 报告字段文档齐全。
```

## 阶段 14：观测性与可靠性提示词

```text
# 阶段 14：观测性与可靠性

## 任务目标

让一次旅行规划请求可以从 HTTP request、任务状态、Eino 节点、Tool/LLM 调用到最终 SSE 事件完整追踪。新增结构化日志、节点级耗时、错误分类和基础指标，同时保持系统轻量。

## 当前上下文

当前已有：

* Gin middleware：request id、logging、recovery、CORS
* TravelPlanService 任务状态转换
* EventBus
* SSE progress/warning/error/done/heartbeat
* Eino callbacks 文件存在，但节点级事件尚未完整推送到 SSE

## 不做什么

* 不引入重量级可观测平台，除非用户明确批准。
* 不输出 API Key、用户敏感信息或完整外部原始响应。
* 不让日志格式随意散落在各层。
* 不破坏现有 SSE contract。

## 需要阅读的文件

* `docs/architecture.md`
* `docs/api.md`
* `docs/agent-flow.md`
* `internal/server/middleware.go`
* `internal/travel/events.go`
* `internal/travel/event_bus.go`
* `internal/travel/service.go`
* `internal/travel/handler.go`
* `internal/agent/eino/callbacks.go`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/types.go`

## 实现要求

* request id 应贯穿 HTTP handler、service、planner run 和日志。
* 定义节点级事件类型或内部 event reporter，避免 `internal/travel` 直接依赖 Eino。
* 节点事件可包含：
  * node name
  * status
  * started/finished time
  * duration
  * warning/fallback reason
* SSE 对外新增事件时必须保持向后兼容。
* 结构化日志字段命名稳定，例如 `request_id`、`task_id`、`planner`、`node`、`duration_ms`、`status`。
* panic recovery 应继续更新任务失败状态并推送 error 事件。

## 文档更新要求

* 更新 `docs/api.md`：如新增 SSE event type，需要说明。
* 更新 `docs/architecture.md`：说明 observability/event reporter 的位置。
* 更新 `docs/agent-flow.md`：说明节点级事件与 Eino callback 的关系。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner eino
```

如果改动 SSE，需要尽量运行：

```bash
cd web
npm run typecheck
npm run lint
npm run harness:ui
```

## 验收标准

* 单次请求能通过 request id/task id 串起日志。
* Eino 节点耗时可观测。
* SSE 仍兼容前端。
* 失败和 panic 路径可被稳定记录。
```

## 阶段 15：前端产品化提示词

```text
# 阶段 15：前端产品化

## 任务目标

把当前 React H5 对话式 demo 打磨成可反复使用的旅行规划工具。重点是路线查看、局部调整、错误恢复、移动端体验和真实 API 联调，而不是做营销落地页。

## 当前上下文

当前前端已有：

* Vite + React + TypeScript
* 对话式 AgentConversation
* TravelBriefPanel
* PlanProgress
* PlanDetail
* typed API client
* SSE + polling fallback hook
* Playwright UI harness

## 不做什么

* 不引入大型 UI 框架，除非用户明确批准。
* 不做营销 landing page。
* 不做复杂地图渲染，除非后端已提供稳定坐标和路线数据。
* 不接登录、支付、订单、收藏、分享，除非进入阶段 16。
* 不在前端硬编码后端地址。

## 必须使用的技能

* 前端 UI 改造前使用 `$frontend-design`。
* 前端实现后使用 `$playwright-skill` 或项目已有 Playwright harness 进行验证。

## 需要阅读的文件

* `docs/frontend-skills.md`
* `docs/api.md`
* `docs/architecture.md`
* `web/src/App.tsx`
* `web/src/styles.css`
* `web/src/api/client.ts`
* `web/src/api/types.ts`
* `web/src/hooks/useTravelPlanStream.ts`
* `web/src/components/AgentConversation.tsx`
* `web/src/components/PlanDetail.tsx`
* `web/src/components/PlanProgress.tsx`
* `web/e2e/chat-agent.spec.ts`

## 实现方向

优先考虑以下能力：

* 路线详情从纯展示升级为可扫描的时间轴。
* 支持“重新生成某一天”或“调整预算/节奏/兴趣”的入口。
* 展示 warning/fallback 的温和解释。
* 预算拆分更清晰。
* 移动端输入、按钮、进度、路线详情不能溢出或重叠。
* SSE 断开、任务失败、后端限流要有明确恢复路径。
* 增加真实后端联调模式文档。

## 文档更新要求

* 更新 `README.md`：前端运行、构建、UI harness、联调方式。
* 更新 `docs/architecture.md`：如前端状态流变化，需要同步。
* 更新 `docs/api.md`：如前端推动 API 契约变化，必须同步。

## 测试要求

尽量运行：

```bash
cd web
npm run typecheck
npm run lint
npm run build
npm run harness:ui
```

如涉及后端联调：

```bash
go test ./...
go run ./cmd/server
cd web
npm run dev
```

## 验收标准

* 第一屏仍是可用工具，不是 landing page。
* 移动端和桌面端核心流程可用。
* Playwright 覆盖生成、进度、done 渲染和错误/断线至少一种路径。
* UI 文案不夸大真实 LLM 或真实 API 能力。
```

## 阶段 16：用户与业务闭环提示词

```text
# 阶段 16：用户与业务闭环

## 任务目标

在核心规划能力稳定后，补齐应用层闭环：用户身份、历史行程、收藏、分享、导出和权限边界。酒店、票务、支付等高风险外部 API 只作为后续可选扩展，不在本阶段默认接入。

## 当前上下文

当前系统没有：

* 用户系统
* 登录鉴权
* 历史行程归属
* 收藏/分享/导出
* 订单/支付
* 酒店/票务 API

当前 API 是匿名任务模式，适合 demo 和评估，不适合长期用户数据管理。

## 不做什么

* 不直接接支付。
* 不保存明文密码。
* 不把匿名历史任务直接暴露给任意用户。
* 不接酒店、票务、支付 API，除非另开安全和合规评估。
* 不在文档中编造未实现的商业闭环。

## 需要阅读的文件

* `docs/prd.md`
* `docs/api.md`
* `docs/database.md`
* `docs/architecture.md`
* `internal/server/router.go`
* `internal/server/middleware.go`
* `internal/travel/handler.go`
* `internal/travel/service.go`
* `web/src/api/client.ts`
* `web/src/App.tsx`

## 推荐能力拆分

先做低风险用户闭环：

1. 用户注册/登录或外部身份 provider 接入方案。
2. 用户与 travel task/plan 的归属关系。
3. 历史行程列表。
4. 行程详情复查。
5. 收藏或归档。
6. 导出 Markdown/JSON/PDF 的可选能力。
7. 分享链接，默认只读并可撤销。

酒店、票务、支付 API 应独立设计，不与基础用户系统混在一个阶段。

## 安全要求

* 所有鉴权配置来自环境变量或配置文件。
* 密码必须哈希存储；优先考虑成熟方案。
* API 必须校验当前用户是否有权访问 task/plan。
* 分享链接应使用不可预测 token，并支持过期或撤销。
* 文档需要说明隐私和数据保留策略。

## 文档更新要求

* 更新 `docs/prd.md`：补充用户故事和范围。
* 更新 `docs/api.md`：补充 auth、history、share/export 接口。
* 更新 `docs/database.md`：补充 users、sessions、plan ownership、share links 等表。
* 更新 `docs/architecture.md`：说明鉴权层和数据归属。
* 更新 `README.md`：补充本地运行所需环境变量。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
```

如涉及前端：

```bash
cd web
npm run typecheck
npm run lint
npm run build
npm run harness:ui
```

## 验收标准

* 匿名任务模式是否保留有明确决策。
* 用户只能访问自己的行程。
* 历史行程可查询。
* 分享链接权限边界清晰。
* 安全文档和数据库文档同步。
```
