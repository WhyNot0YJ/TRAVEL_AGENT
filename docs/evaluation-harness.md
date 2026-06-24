# Evaluation Harness 设计文档

## 1. 背景

本项目的核心能力不是简单调用大模型生成文本，而是让 Agent 根据用户旅游需求，结合 POI、路线、天气、预算等信息，生成结构化、可验证、可复用的旅游路线。因此需要一套 Evaluation Harness 来持续评估 Agent 的输出质量。

## 2. Harness 目标

1. 自动读取测试用例
2. 调用统一 `TravelPlanner` 接口
3. 支持 `MockPlanner` 和 `EinoTravelPlanner`
4. 评估路线结构完整性
5. 评估天数是否匹配
6. 评估预算是否合理
7. 评估关键词是否命中
8. 评估是否存在非法字段
9. 输出 JSON 报告
10. 为后续 Agent 迭代提供可量化指标

## 3. 当前架构

```text
testdata/travel_cases.json
  -> Harness Runner
  -> TravelPlanner Interface
  -> MockPlanner / EinoTravelPlanner
  -> Evaluator
  -> SummaryMetrics
  -> Console Report + reports/eval_report.json
```

Eino 模式的内部流程：

```text
TravelPlanner Interface
  -> EinoTravelPlanner
  -> Eino Graph / Workflow
  -> Mock POI Tool
  -> Mock Weather Tool
  -> Mock Route Tool
  -> Mock Budget Tool
  -> LLM Schema Generator / Mock Plan Generator fallback
```

Eino Tools 默认使用 mock mode。可通过环境变量切换 real tools：

```bash
TRAVEL_AGENT_TOOL_MODE=real
TRAVEL_AGENT_AMAP_API_KEY=your-key
go run ./cmd/harness -planner eino
```

如果未配置 key 或外部 API 调用失败，real tools 会 fallback 到 mock tools，并在 `TravelPlan.warnings` 中记录原因。

real tool fallback warning 使用稳定格式，后续可用于统计：

```text
tool fallback: tool=poi provider=amap stage=request category=provider_error mock_fallback=true reason=...
```

当前 Harness 仍只做基础规则评分，不要求真实外部 API 可用。使用 real mode 跑评估时，缺 key、timeout、非 2xx、provider 失败、无效 JSON、缺字段、坐标缺失等路径应表现为 warning 和 mock fallback，而不是让默认评估崩溃。

## 4. 核心接口

```go
type TravelPlanner interface {
	Plan(ctx context.Context, req domain.TravelRequest) (*domain.TravelPlan, error)
}
```

抽象该接口的原因：

1. Harness 不关心底层是 MockPlanner 还是 EinoTravelPlanner
2. 方便测试
3. 方便替换实现
4. 方便后续做 A/B 测试
5. 方便比较不同模型或不同 Agent Graph 的效果

## 5. 测试集设计

`testdata/travel_cases.json` 用于存放本地评估数据集。每条 case 包含：

1. case id
2. description：说明该 case 测试的功能、范围或边界
3. input 用户请求
4. expectation 期望约束

当前测试集覆盖：

1. 杭州三日游
2. 南京二日游
3. 北京亲子博物馆路线
4. 成都美食休闲路线
5. 广州 citywalk
6. 西安历史文化路线
7. 苏州一日轻松游
8. 未知城市兜底测试
9. 低预算和极低预算路线
10. 高预算深度路线
11. 长天数 POI 循环分配
12. 空兴趣、空交通方式和空节奏默认值
13. 同城游
14. 老人友好、亲子、商务间隙、年轻人高强度等人群或目的场景

## 6. 评估指标

每条 case 评分总分 100 分：

1. 基础成功：20 分
2. 天数匹配：20 分
3. 预算合规：20 分
4. 结构完整：20 分
5. 关键词匹配：10 分
6. 无非法字段：10 分

汇总指标：

* SuccessRate：成功 case 占比
* AverageScore：平均得分
* AverageDurationMs：平均耗时
* BudgetPassRate：预算合规率
* DayMatchRate：天数匹配率
* StructurePassRate：结构完整率

## 7. 报告格式

`reports/eval_report.json` 结构：

```json
{
  "generated_at": "...",
  "planner_type": "eino",
  "summary": {},
  "cases": []
}
```

每个 `CaseResult` 包含：

* CaseID
* Description
* Success
* DurationMs
* Score
* Errors
* Warnings
* Checks
* Plan

## 8. 如何运行

```bash
go test ./...
go run ./cmd/harness
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
go run ./cmd/harness -dataset testdata/travel_cases.json -report reports/eval_report.json
make harness
make harness-mock
make harness-eino
```

可选 real tool 评估：

```bash
set TRAVEL_AGENT_TOOL_MODE=real
set TRAVEL_AGENT_AMAP_API_KEY=your-key
set TRAVEL_AGENT_EXTERNAL_API_TIMEOUT=10s
go run ./cmd/harness -planner eino
```

不要把真实 API Key 写入报告、测试数据或文档。无 Key 时，real mode 会稳定 fallback 到 mock tools。

## 8.1 Frontend UI Harness

前端 UI Harness 与 Go Agent Harness 分开运行，避免让 `internal/harness` 依赖浏览器、HTTP server 或 Playwright。

```bash
cd web
npm run harness:ui
```

UI Harness 使用 Playwright：

1. 在浏览器中打开 React H5。
2. Mock `POST /api/v1/travel/plans`、SSE stream 和任务查询接口。
3. 模拟用户在聊天窗口输入旅行需求。
4. 校验 `生成行程` 按钮从禁用到可用。
5. 校验规划进度卡出现。
6. 校验最终路线详情在聊天窗口中展开。
7. 分别覆盖 desktop 和 mobile Chromium viewport。

报告输出：

```text
reports/ui_eval_report.json
```

UI Harness 关注交互、状态展示、SSE/done 渲染和移动端可用性；Go Harness 继续关注 `TravelPlanner` 输出质量。两者报告彼此独立，后续可由 CI 或聚合脚本统一运行。

## 9. 如何新增测试用例

在 `testdata/travel_cases.json` 中新增 case。要求：

1. `id` 必须唯一
2. `input.id` 必须与 case id 一致
3. `input` 必须包含 `destination_city`、`days`、`budget` 等核心字段
4. `expectation` 必须包含 `min_days`、`max_budget_ratio`、`required_keywords`
5. 新增边界 case 时要说明目的

## 10. EinoTravelPlanner

第二版已新增 `EinoTravelPlanner`。阶段 3 已支持可选 LLM 生成节点。Harness 可以通过 `-planner` 参数选择 planner：

```bash
-planner mock
-planner eino
```

当前 `EinoTravelPlanner` 使用 CloudWeGo Eino Graph / Workflow 串联以下节点：

1. ParseTravelRequestNode
2. SearchPOIsToolNode
3. GetWeatherToolNode
4. ComputeRouteToolNode
5. EstimateBudgetToolNode
6. OptimizeItineraryNode
7. GenerateTravelPlanNode
8. ValidatePlanNode

当前 Eino 模式默认使用 Mock Tools；启用 `TRAVEL_AGENT_TOOL_MODE=real` 后可调用高德 POI、路线和天气 API。LLM 默认不启用；启用后，GenerateTravelPlanNode 会优先使用 provider-native JSON Schema 结构化输出，失败时 fallback 到 deterministic generator。

LLM 相关 warning 当前仍作为 case warnings 输出，后续 Harness 可基于稳定字段聚合：

```text
LLM trace: prompt_version=travel-plan-v1 duration_ms=123 prompt_tokens=unknown completion_tokens=unknown total_tokens=unknown
LLM fallback: prompt_version=travel-plan-v1 category=business_validation_failed attempts=1 duration_ms=123 reason=...
```

fallback category 包括 `disabled`、`missing_api_key`、`provider_error`、`timeout`、`invalid_json`、`business_validation_failed` 和 `retry_exhausted`。如果 provider 返回 token usage，trace 中记录 prompt/completion/total token；如果未返回，则写 `unknown`。

DeepSeek 模式使用 strict tool calling beta：

```text
TRAVEL_AGENT_LLM_ENABLED=true
TRAVEL_AGENT_LLM_PROVIDER=deepseek
TRAVEL_AGENT_LLM_API_KEY=your-api-key
go run ./cmd/harness -planner eino
```

DeepSeek 默认使用 `https://api.deepseek.com/beta` 和 `deepseek-v4-flash`。输出结构由 `submit_travel_plan` tool 的 JSON Schema 约束，并设置 `strict=true`。Harness 不依赖 prompt-only JSON 约束。

后续接入真实 Eino Agent 时，可以继续扩展已有 `EinoTravelPlanner`：

```go
type EinoTravelPlanner struct {
	// graph / model / tools
}

func (p *EinoTravelPlanner) Plan(ctx context.Context, req domain.TravelRequest) (*domain.TravelPlan, error) {
	// 调用 Eino Graph
}
```

真实 LLM、POI、路线、天气工具应在 `internal/agent/eino` 中扩展，不能让 harness 包直接依赖具体 Eino 实现。

## 11. 后续扩展方向

1. 接入高德地图 POI API
2. 接入高德路线规划 API
3. 增加 Tool 调用轨迹评估
4. 增加 Token 消耗统计
5. 增加外部 API 调用成功率统计
6. 增加并发压测模式
7. 增加 benchmark 命令
8. 增加人工评分字段
9. 增加多模型对比能力
10. 增加不同 Prompt 版本对比
11. 增加 Eino Graph 节点级耗时统计
12. 增加失败 case 自动保存输入和输出快照

## 12. 当前限制

1. 当前使用 MockPlanner，不代表真实 Agent 效果
2. 当前不评估真实路线距离
3. 当前不校验真实景点营业时间
4. 当前不调用真实天气 API
5. 当前不统计 Token 消耗
6. 当前只做基础规则评估，不做语义质量评估
7. 当前 LLM 输出会做结构和业务校验，但不评价真实路线可达性
