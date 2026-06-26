# Stage 1：Harness + MockPlanner

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
