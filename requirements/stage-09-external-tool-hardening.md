# Stage 9：真实外部 Tool 稳定化

## 任务目标

把现有高德 POI、路线、天气 adapter 从"可配置可调用"推进到"可验证、可观测、可降级"的工程能力。默认 mock tool 行为必须保持稳定；real tool 只在显式配置时启用，并且所有失败都要可解释地 fallback。

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
