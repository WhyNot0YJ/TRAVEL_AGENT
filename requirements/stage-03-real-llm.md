# Stage 3：接入真实 LLM

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
