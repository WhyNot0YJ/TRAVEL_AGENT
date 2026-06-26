# Stage 4：接入真实 高德 / 天气 / POI API

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
