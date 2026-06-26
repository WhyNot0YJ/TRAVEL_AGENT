# Stage 13：Evaluation Harness 升级

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
