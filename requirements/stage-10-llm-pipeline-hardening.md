# Stage 10：真实 LLM 规划链路硬化

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
