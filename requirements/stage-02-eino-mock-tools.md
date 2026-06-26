# Stage 2：EinoTravelPlanner + Mock Tools

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
