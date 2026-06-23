# Travel Agent 分阶段开发提示词

说明：`skills/agent-feature-dev.md` 当前不存在，后续阶段需要创建或补充；`docs/prd.md`、`docs/architecture.md`、`docs/api.md`、`docs/database.md`、`docs/external-apis.md` 当前主要是 TODO，需要在对应阶段逐步补充，不要一次性编造完整功能。

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
