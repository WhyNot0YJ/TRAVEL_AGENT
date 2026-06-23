# Travel Agent

本项目是一个基于 React + Go Gin + CloudWeGo Eino 的高并发智能旅游路线规划应用。当前仓库先搭建后端 Travel Agent Evaluation Harness，用于后续持续评估路线规划 Agent 的稳定性、正确性、耗时和结构化输出质量。

## Travel Agent Evaluation Harness

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
```

说明：

* `mock`：不依赖 Eino，不经过 Graph，只用于基础 Harness 测试。
* `eino`：经过 CloudWeGo Eino Graph / Workflow。默认使用 Mock Tools 和 deterministic plan generator；设置 LLM 环境变量后可调用真实 LLM，但仍不调用真实地图、天气或路线 API。

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
go run ./cmd/harness -planner eino
```

如果 LLM 未启用、配置缺失、provider 不支持 schema 输出、tool call 缺失、返回结构无效或业务校验失败，Eino planner 会自动 fallback 到 deterministic generator，并在 `warnings` 中记录原因。

## 如何添加新的测试用例

在 `testdata/travel_cases.json` 中新增一条 case。每条 case 必须包含唯一 `id`、清晰说明覆盖范围的 `description`、与 `id` 一致的 `input.id`、完整的 `input`，以及包含 `min_days`、`max_budget_ratio`、`required_keywords` 的 `expectation`。

`required_keywords` 至少应包含目的地城市，用于校验标题、摘要或路线内容是否命中核心目的地。

当前数据集包含常规多日游、单日游、低预算、高预算、长天数、空兴趣、未知城市、同城游、不同节奏和不同交通方式等 24 条覆盖 case。

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

单条 case 按 100 分计算：基础成功 20 分、天数匹配 20 分、预算合规 20 分、结构完整 20 分、关键词匹配 10 分、无非法字段 10 分。

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

更多说明见 `docs/evaluation-harness.md`。
