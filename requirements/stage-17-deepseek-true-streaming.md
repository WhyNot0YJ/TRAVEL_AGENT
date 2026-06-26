# Stage 17：DeepSeek 真·流式 SSE 透传
> 本阶段对应 2026-06 用户提出的 "SSE 切块太大，能不能跟 DeepSeek API 同步" 中讨论的 **方案 2（彻底）**。
> 不要采用方案 1 的"调小 chunkText 粒度 + sleep 节流"作为主路径——本阶段就是要把伪流式换成真流式。

## 1. 任务目标

把 `internal/agent/eino` 中的 LLM 调用从**一次性 `io.ReadAll`** 改造成 **`stream=true` 的逐 token SSE 透传**，让前端 `assistant_delta` 事件随 DeepSeek 的 token delta 实时滚动，达到与"直连 DeepSeek API"一致的打字机体验。

具体交付：

1. 在 `internal/agent/eino` 内部新增 streaming chat completion 客户端，能解析 OpenAI / DeepSeek 标准 `text/event-stream` 协议（`data: {...}\n\n` 帧 + `data: [DONE]`）。
2. **聊天链路（链路 A，`/api/v1/travel/chat/stream`）**：把信息收集后的"自然语言回复"换成 LLM 真流式生成，逐 delta 发到 `assistant_delta`。
3. **规划链路（链路 B，`/api/v1/travel/plans/:task_id/stream`）**：保留结构化 `submit_travel_plan` strict tool call 的非流式语义，但额外增加一段"自然语言旁白" stream，让前端能在工具调用完成前就看到行进文字；最终结构化 plan 仍走非流式落库 + `done` 事件。
4. 删除或下线 `chunkText(reply, 18)` / `chunkText(text, 24)` 这两处人为切片；前端不能感知到任何"按字数等距切块"的痕迹。
5. 单元测试覆盖流式解析、断流、客户端断开、重试与降级。
6. 更新 `docs/api.md`、`docs/agent-flow.md`、`docs/external-apis.md`、`README.md`，明确 SSE delta 不再保证等长。

## 2. 当前上下文（必须先读）

### 2.1 现状（伪流式）

* [internal/travel/service.go](../internal/travel/service.go) 的 `ChatStream`（约第 158 行）和 `publishPlanSummary`（约第 224 行）都是**先拿到完整 reply，再 `chunkText` 等距切片**：

  ```go
  // ChatStream
  for _, chunk := range chunkText(resp.Reply, 18) {
      emit(TaskEvent{Type: EventAssistantDelta, Message: chunk, ...})
  }
  emit(TaskEvent{Type: EventAssistantDone, Message: resp.Reply, ...})

  // publishPlanSummary
  for _, chunk := range chunkText(text, 24) { ... }
  ```

* [internal/agent/eino/llm.go](../internal/agent/eino/llm.go) `openAICompatibleClient.GenerateTravelPlan` 走的是 **非流式**：`req.Stream = false`、`io.ReadAll(io.LimitReader(resp.Body, 4<<20))`、整段 unmarshal、从 `tool_calls[0].function.arguments` 拿完整 JSON。
* [internal/agent/eino/chat.go](../internal/agent/eino/chat.go) `chatInfoExtractor.callLLM`（约第 74 行）也是非流式：`e.client.httpClient.Do(req)` → `chatCompletionResponse` 一次性解码。
* `chatCompletionRequest.Stream` 字段已经存在但**强制设为 `false`**（见 `buildChatCompletionRequest`）。
* 前端契约：[web/src/api/types.ts](../web/src/api/types.ts) 已支持 `assistant_delta` / `assistant_done`，[web/src/hooks/useTravelPlanStream.ts](../web/src/hooks/useTravelPlanStream.ts) 通过 `EventSource.addEventListener` 监听并把 `message` 字段拼接为 `assistantText`。前端不需要协议改动，只需要 delta 频率/粒度变化。

### 2.2 已经存在、可以复用

* `EventBus` / `TaskEvent` / `EventAssistantDelta` / `EventAssistantDone` 事件类型。
* `plannerEventReporter`：可以扩展用来接收 LLM stream delta。
* DeepSeek strict tool call 协议、`submit_travel_plan` schema、retry / fallback 框架。

### 2.3 已知的"坑"

* DeepSeek `submit_travel_plan` 是 strict JSON Schema tool call，**逐 token 透传 `tool_calls.arguments` 给前端没意义**——前端拿到的是半截 JSON，没法渲染。所以规划链路不能简单把 LLM 流原样转发。
* DeepSeek streaming 模式下 `tool_calls` 的 `arguments` 是分片到达的，需要按 `tool_calls[idx].function.arguments` 累加；不能假设一次到位。
* SSE 帧可能跨 TCP 包断裂；解析必须支持"按 `\n\n` 分隔事件 + 行内可能有多条 `data:`"。

## 3. 本阶段不做什么

* **不做** WebSocket / gRPC streaming。
* **不动** 前端 `assistant_delta` / `assistant_done` 协议字段（保持 `{message: string}` 语义）。
* **不动** DeepSeek strict tool call 的 schema 结构。
* **不引入**新的 LLM SDK。坚持使用 `net/http` 手写 SSE 解析。
* **不改**外部 Tool（POI / 天气 / 路线）链路。
* **不接**新的 LLM provider。仍只支持 DeepSeek 与 OpenAI-compatible。
* **不为了流式放宽** business validation —— 流结束后仍要跑 `parseTravelPlanArguments` + 业务校验。
* **不在 `internal/domain` 引入** SSE / streaming 相关类型。
* **不影响** harness（`cmd/harness`）—— harness 没有 HTTP 层，应当继续走非流式或在内部直接拿到完整 plan。

## 4. 必读文件清单

Codex 在动手前必须阅读以下文件。括号里是阅读重点：

* `AGENTS.md`（项目通用规约）
* `requirements/README.md`（本目录索引）
* `requirements/stage-07-sse.md`（SSE 基础协议）
* `requirements/stage-10-llm-pipeline-hardening.md`（LLM fallback 分类，必须保持兼容）
* `docs/api.md`（SSE 当前 contract）
* `docs/agent-flow.md`（Eino Graph 节点）
* `docs/external-apis.md`（DeepSeek 配置）
* `internal/agent/eino/llm.go`（要改造的核心文件）
* `internal/agent/eino/chat.go`（聊天 LLM 调用）
* `internal/agent/eino/config.go`（LLM 配置项）
* `internal/agent/eino/prompt.go`、`internal/agent/eino/schema.go`（不需要改但要理解）
* `internal/agent/eino/llm_test.go`、`internal/agent/eino/chat_test.go`（要扩展）
* `internal/agent/eino/callbacks.go`（事件上报抽象点）
* `internal/agent/metadata.go`（`PlannerOptions` / `WithPlannerEventReporter`）
* `internal/travel/service.go`（`ChatStream` / `publishPlanSummary` / `runTask`）
* `internal/travel/events.go`、`internal/travel/event_bus.go`（事件协议）
* `internal/travel/handler.go`（SSE handler）
* `web/src/hooks/useTravelPlanStream.ts`、`web/src/components/PlanProgress.tsx`、`web/src/api/client.ts`（前端消费方）

## 5. 预期改动文件清单

新增：

* `internal/agent/eino/llm_stream.go` —— SSE 帧解析、`chatCompletionStream` 方法
* `internal/agent/eino/llm_stream_test.go` —— 用 `httptest.Server` 模拟 DeepSeek 流式响应
* `internal/agent/eino/chat_stream.go` —— `chatInfoExtractor.ExtractStream` 或 `Stream(...)` 方法
* `internal/agent/eino/chat_stream_test.go`

修改：

* `internal/agent/eino/llm.go`：把 `openAICompatibleClient` 改造成既能非流式（用于 strict tool call 落库）也能流式（用于自然语言旁白）；保留 `GenerateTravelPlan` 非流式入口
* `internal/agent/eino/chat.go`：聊天 reply 改为流式 LLM；保留 fallback
* `internal/agent/eino/config.go`：新增配置项 `TRAVEL_AGENT_LLM_STREAM_ENABLED`（默认 `true`），用于一键回滚
* `internal/agent/eino/callbacks.go` 或 `internal/agent/metadata.go`：新增 `LLMDeltaReporter` 抽象，让 stream client 不直接依赖 `internal/travel`
* `internal/agent/planner.go` / `internal/agent/eino/planner.go`：让规划链路在 LLM 调用阶段也能上报 delta
* `internal/travel/service.go`：
  * 删除 `ChatStream` 中的 `chunkText(resp.Reply, 18)`，改为接 LLM stream delta 回调
  * 删除 / 改造 `publishPlanSummary` 中的 `chunkText(text, 24)`
  * 把 `chunkText` 函数本身保留为最后兜底（fallback / 测试模式可用），但不再是默认路径
* `internal/travel/dto.go`：如果 `ChatRequest` 需要新增 `Stream bool`，按需更新
* `internal/agent/eino/llm_test.go`、`internal/agent/eino/chat_test.go`：保证非流式分支仍覆盖
* `docs/api.md`：在 `assistant_delta` 段落补充"delta 长度不再保证等长，按 LLM token 分片"
* `docs/agent-flow.md`：补充流式 LLM 节点
* `docs/external-apis.md`：补充 `stream=true` 的 wire format 和限制
* `README.md`：补充 `TRAVEL_AGENT_LLM_STREAM_ENABLED` 环境变量

可选：

* `web/src/components/PlanProgress.tsx` / `useTravelPlanStream.ts`：如果发现 token 级 delta 导致 React state 更新过频，加 `requestAnimationFrame` 节流（**只在确认有性能问题时再做**）

## 6. 实现要求（核心约束）

### 6.1 Stream HTTP Client（`llm_stream.go`）

* 新增方法 `(c *openAICompatibleClient) chatCompletionStream(ctx, payload, onDelta) (*streamResult, error)`。
* `payload.Stream = true`；`tool_choice` / `response_format` 等其余字段保持不变。
* 请求头额外加 `Accept: text/event-stream`。
* 响应必须**按 SSE 帧解析**，不要 `io.ReadAll`：用 `bufio.Reader.ReadBytes('\n')` 累积，遇到空行（`\n\n`）后取出该 event；遇到 `data: [DONE]` 立即结束。
* 对每个 `data:` JSON 帧解码为 `chatCompletionStreamChunk`：

  ```go
  type chatCompletionStreamChunk struct {
      Choices []struct {
          Delta struct {
              Content   string         `json:"content"`
              ToolCalls []chatToolCall `json:"tool_calls"`
          } `json:"delta"`
          FinishReason string `json:"finish_reason"`
      } `json:"choices"`
      Usage *tokenUsage `json:"usage"`
  }
  ```

* 对每个 `Delta.Content` 非空片段调用 `onDelta(content)`；对 `Delta.ToolCalls[i].Function.Arguments` 片段，按 index 累加到内部 `toolCallBuffers[i]`。
* `streamResult` 必须返回：累积 content、累积 toolCallArguments（map[index]string）、token usage（如有）、耗时、finishReason。
* `ctx.Done()` 时立刻 `resp.Body.Close()` 并返回 `ctx.Err()`，**不能**继续读 body。
* 必须设置 streaming 专用 timeout（单帧间隔 timeout > 整体 timeout 一段）；推荐用 `http.Client{Timeout: 0}` + 在外层用 `context.WithTimeout` 控制总时长，避免误把空闲帧间隔当成超时。

### 6.2 LLMDeltaReporter 抽象

为了避免 `internal/agent/eino` 反向 import `internal/travel`，沿用现有 `WithPlannerEventReporter` 模式：

* 在 `internal/agent/metadata.go` 新增：

  ```go
  type LLMDeltaReporter interface {
      ReportLLMDelta(ctx context.Context, delta string)
  }
  func WithLLMDeltaReporter(ctx context.Context, r LLMDeltaReporter) context.Context
  func LLMDeltaReporterFromContext(ctx context.Context) LLMDeltaReporter
  ```

* `internal/travel/service.go` 在 `runTask` / `Chat` 入口注入一个把 delta 转成 `EventAssistantDelta` 事件 publish 的实现。
* `internal/agent/eino/llm.go` 与 `chat.go` 在调用 stream client 时，把 reporter 作为 `onDelta`。

### 6.3 链路 A：聊天信息收集（最重要）

* `chatInfoExtractor.Extract` 当前先 LLM tool call 拿结构化字段，再用本地话术拼 `result.Reply`。
* 改造方案：
  1. **第一次调用** LLM 仍是 strict tool call，**保持非流式**，拿到结构化 `extract_travel_info` 字段。原因：tool 参数也是 JSON，不能逐 token 透传给前端。
  2. **第二次调用** LLM 是 streaming chat completion，prompt 让模型基于已知字段 + 用户最新消息生成"自然语言回复"，`stream=true`，逐 delta 通过 `LLMDeltaReporter` 推送。
  3. 最终 reply = 累积 content。把 `result.Reply` 设为这段 stream 累积。
* 失败降级：
  * 如果第二次（流式）调用失败 → 降级为现有的本地话术，但**仍然走流式发出**：把整段 reply 一次 emit 一个 delta（或保留原 `chunkText` 作为 emergency fallback，并在 warning 中记录 `llm_stream_unavailable`）。
  * 如果第一次（结构化）调用失败 → 维持现有 fallback 行为不变。
* `service.ChatStream` 需要把 `LLMDeltaReporter` 注入 ctx，且去掉对 `chunkText(resp.Reply, 18)` 的依赖；最后仍然 emit `EventAssistantDone{Message: fullReply}`。

### 6.4 链路 B：行程规划

* `EinoTravelPlanner` 内部 LLM 节点（`GenerateTravelPlanNode`）原本通过 strict tool call 一次性生成 `TravelPlan`。**不要**逐 token 透传 `submit_travel_plan` 的 arguments。
* 增加一段"旁白" stream：在调用 strict tool call 之前或并行，**额外发起一次** streaming chat completion，prompt 简短，要求模型"用一两句话给用户播报当前正在规划什么"。这段 stream 作为 `EventAssistantDelta` 推送。
* 也可以更轻量地直接复用 `publishPlanSummary` 的入口，但把 summary 的"切片源"换成"流式生成的旁白"。具体取舍由实现者评估，**只要不出现等距切块就行**。
* 最终结构化 plan 仍由原非流式 `GenerateTravelPlan` 产出 → `EventDone`。
* 旁白 stream 失败/被禁用 → 静默跳过，不能影响 plan 主路径。

### 6.5 配置项

* 新增 `TRAVEL_AGENT_LLM_STREAM_ENABLED`，默认 `true`。
* 当为 `false` 时，全部走旧路径（`chunkText` + 非流式 LLM），用于**生产回滚**。
* `TRAVEL_AGENT_LLM_TIMEOUT` 现有字段含义改为"非流式整体 / 流式累积总耗时上限"。

### 6.6 删除 / 标注废弃

* `chunkText` **不要直接删除**（部分测试和 fallback 仍依赖），但：
  * 在 `service.ChatStream` 主路径不再调用。
  * 在 `publishPlanSummary` 主路径不再调用。
  * 加注释 `// Deprecated: 仅用于 LLM 流式不可用时的兜底。`

### 6.7 兼容性硬约束

* `cmd/harness` 不允许因为本阶段改动而变慢、变非确定性或失败。Harness 路径里 `LLMDeltaReporter` 必须可以为 `nil`，stream client 必须能在没有 reporter 时无副作用运行。
* 现有所有 `*_test.go` 必须继续通过；如果某个测试断言 `assistant_delta` 等距切块，需要改成"累积内容等于完整 reply"这种宽松断言。
* 前端 `web/e2e/chat-agent.spec.ts` 必须仍然通过；如断言 chunk 数量则需放宽。

## 7. 测试要求

### 7.1 必须新增的单元测试

`internal/agent/eino/llm_stream_test.go` 覆盖：

1. **正常 stream**：fake server 按 5 帧返回 `data: {"choices":[{"delta":{"content":"你"}}]}` 风格 → 累积 content 正确，onDelta 被调用 5 次。
2. **DeepSeek tool call streaming**：fake server 按多帧返回 `tool_calls[0].function.arguments` 分片 → 累积 arguments 等于完整 JSON。
3. **`data: [DONE]`** 结束帧识别。
4. **半截帧 + 跨包**：手工写入 `data: {"choices":[{"delta":{"con` + flush + `tent":"x"}}]}\n\n` → 解析仍正确。
5. **HTTP 5xx**：返回正确 error。
6. **客户端 ctx cancel**：fake server 故意慢 → ctx 超时后 `chatCompletionStream` 立刻返回 `ctx.Err()`，不阻塞。
7. **空 delta / 仅 finish_reason 帧**：不应 panic。

`internal/agent/eino/chat_stream_test.go` 覆盖：

1. 流式分支成功 → reply 等于累积 delta，且 `onDelta` 调用次数 > 1。
2. 流式分支失败 → 走 deterministic 话术 fallback，warning 包含 `llm_stream_unavailable`。
3. `TRAVEL_AGENT_LLM_STREAM_ENABLED=false` → 完全走旧 `chunkText` 路径。

### 7.2 必须运行

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
```

如果改了前端：

```bash
cd web
npm run typecheck
npm run lint
npm run build
npm run harness:ui
```

### 7.3 手动联调（可选但强烈建议）

```bash
set TRAVEL_AGENT_LLM_ENABLED=true
set TRAVEL_AGENT_LLM_PROVIDER=deepseek
set TRAVEL_AGENT_LLM_API_KEY=<your-key>
set TRAVEL_AGENT_LLM_STREAM_ENABLED=true
go run ./cmd/server
```

然后：

```bash
curl -N -X POST http://localhost:8080/api/v1/travel/chat/stream \
  -H 'Content-Type: application/json' \
  -d '{"message":"我想去成都玩三天，预算3000"}'
```

预期：能看到 `event: assistant_delta` 一帧一个/几个汉字、密集滚动，而不是 18 字一块的均匀块。

## 8. 文档更新要求

* `docs/api.md` —— `assistant_delta` 段加一句："`message` 字段为 LLM token 级增量；前端应累加而非替换；长度不保证等长。"
* `docs/agent-flow.md` —— 新增 "LLM 流式生成节点" 与 `LLMDeltaReporter` 的关系图。
* `docs/external-apis.md` —— 新增 "Streaming Chat Completion wire format（OpenAI 兼容）" 段落，给出帧示例。
* `README.md` —— 在环境变量列表加 `TRAVEL_AGENT_LLM_STREAM_ENABLED`。
* `requirements/README.md` —— **不需要改**，本阶段已经在索引里。

## 9. 验收标准

* `go test ./...` 全部通过。
* 默认配置（`TRAVEL_AGENT_LLM_ENABLED=true` + `TRAVEL_AGENT_LLM_STREAM_ENABLED=true`）下：
  * 聊天 `/api/v1/travel/chat/stream` 输出的 `assistant_delta` **不再呈现 18 字等长切片特征**；可观察到 token 级粒度（通常 1~6 字）。
  * 规划 `/api/v1/travel/plans/:task_id/stream` 在 `done` 事件之前可观察到非等长的 `assistant_delta`（旁白）。
* `TRAVEL_AGENT_LLM_STREAM_ENABLED=false` 时行为与改造前一致（向后兼容、可回滚）。
* `LLM_ENABLED=false` 时所有路径仍可降级跑通（不依赖 LLM 的部署不被本阶段破坏）。
* `cmd/harness` 性能与确定性不退化。
* 所有新增能力有单元测试，且没有真实 API Key 写入仓库。
* 文档与实现一致，不夸大流式能力（例如不要写"完全等同于 DeepSeek 官方 SDK"）。

## 10. 最终回复要求

完成后请输出：

1. 修改了哪些文件
2. 新增了哪些文件
3. 关键设计决策（特别是：链路 B 旁白方案怎么选的，timeout/取消怎么处理的）
4. 跑了哪些测试，分别在什么模式下
5. 测试是否全部通过；如未通过，原因
6. 手动 curl 联调结果摘要（如果跑了）
7. 风险和未完成事项（特别是：DeepSeek tool call streaming 的 arguments 分片是否在生产 key 上验证过）

## 11. 计划方案（建议执行顺序）

> 这一节是给执行方（Codex / 工程师）的**实施路线**，不是规约。可调整顺序，但每一步的产物应是可独立 review 的。

| Step | 内容 | 产物 |
| :---: | :--- | :--- |
| S1 | 抽象 `LLMDeltaReporter`，加配置项 `TRAVEL_AGENT_LLM_STREAM_ENABLED` | `metadata.go` / `config.go` 改动 + 单测 |
| S2 | 实现 `llm_stream.go`：SSE 帧解析 + `chatCompletionStream` | 新增文件 + 7 个单测 |
| S3 | 改造 `chat.go`：信息收集后用流式生成 reply；保留 fallback | `chat.go` / `chat_stream.go` + 测试 |
| S4 | 改造 `service.ChatStream`：注入 reporter、移除 `chunkText` 主路径 | `service.go` 改动 + 测试 |
| S5 | 规划链路：在 `runTask` 中追加旁白流（最低实现：在 LLM 调用前发一次 streaming chat 简短描述） | `service.go` / `planner.go` 改动 |
| S6 | 删除 `publishPlanSummary` 中 `chunkText` 主路径，保留 fallback | `service.go` 改动 |
| S7 | 文档同步：`api.md` / `agent-flow.md` / `external-apis.md` / `README.md` | docs 改动 |
| S8 | 跑全套测试 + 手动 curl 验证 + 截图 / 日志附在最终回复 | 验收 |

每步建议独立 commit，方便回滚。
