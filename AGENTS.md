# AGENTS.md

## 项目目标

本项目是一个高并发智能旅游路线规划应用。

前端：

* React 移动端套壳 / H5
* TypeScript
* 支持 SSE 流式展示路线规划结果

后端：

* Go
* Gin
* CloudWeGo Eino Agent 框架
* Redis 缓存
* MySQL / PostgreSQL 持久化

AI 编排：

* 使用 Eino 构建 Travel Agent Workflow
* 支持旅游需求解析、POI 查询、路线计算、天气查询、预算估算、路线生成和结果校验

Harness：

* 用于评估 Travel Agent 的稳定性、正确性、耗时和输出质量
* 当前使用 MockPlanner
* 第二版已支持 EinoTravelPlanner
* EinoTravelPlanner 当前使用 Eino Graph + Mock Tools，不调用真实 LLM 或外部 API

## 目录结构约定

* `cmd/harness`：评估命令行入口
* `internal/domain`：通用业务实体，不依赖 agent 或 harness
* `internal/agent`：Planner 接口和具体 Planner 实现
* `internal/agent/eino`：EinoTravelPlanner、Eino Graph 节点、Mock Tools 和内部状态
* `internal/harness`：测试集加载、执行、评分、指标和报告
* `testdata`：本地评估数据集
* `reports`：Harness 生成报告
* `docs`：工程设计文档

## 后端开发规则

1. 不要随意引入新的大型框架。
2. 所有后端接口变更必须更新 `docs/api.md`。
3. 所有 Agent 流程变更必须更新 `docs/agent-flow.md`。
4. 不允许硬编码 API Key。
5. 配置必须来自环境变量或配置文件。
6. 新增后端功能时，应尽量包含 handler、service、DTO、单元测试或集成测试。

## Harness 开发规则

1. Harness 不应直接依赖具体 Eino 实现。
2. Harness 只能依赖 `TravelPlanner` 接口。
3. 当前默认使用 `MockPlanner`。
4. `EinoTravelPlanner` 必须放在 `internal/agent/eino`。
5. 评估结果必须输出到 `reports/eval_report.json`。
6. 每次新增评估指标，都必须同步更新 `internal/harness/evaluator.go`、`internal/harness/metrics.go`、`docs/evaluation-harness.md` 和 `README.md`。

## Eino 开发规则

1. Eino 相关代码只能放在 `internal/agent/eino` 或其子目录。
2. 不要让 `internal/harness` 直接依赖 Eino。
3. 后续接入真实 LLM 或外部 API 时，不能破坏当前 Mock Tools。
4. 新增 Eino Graph 节点时，必须更新 `docs/agent-flow.md`。
5. Eino 节点应保持输入输出结构清晰，内部状态不要污染 `internal/domain`。
6. 不要硬编码 API Key，真实模型和外部 API 配置必须来自环境变量或配置文件。

## 文档更新规则

1. 新增 Harness 功能时，必须说明测试用例、评估指标、报告字段和运行方式。
2. 新增外部 API 时，必须更新 `docs/external-apis.md`。
3. 新增数据库表或持久化模型时，必须更新 `docs/database.md`。
4. 不要在文档中编造尚未实现的复杂功能细节。

## 测试要求

完成任务前尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness
go run ./cmd/harness -planner eino
make harness
make harness-eino
```

如果项目后续包含前端，相关前端修改还应尽量运行：

```bash
npm run typecheck
npm run lint
```

## 最终回复要求

每次完成任务后，请说明：

1. 修改了哪些文件
2. 做了哪些设计决策
3. 跑了哪些测试
4. 还有哪些风险或未完成事项

## 前端 Skills

本仓库使用安装在 `.agents/skills` 下的 GitHub skills。

可用前端 skills：

- `$frontend-design`：在构建或重新设计前端 UI 前使用。它可以帮助避免通用的 AI 感界面，并改进字体、布局、间距、色彩和视觉识别。
- `$playwright-skill`：在前端改动后使用，用 Playwright 测试本地页面、截图、检查交互、查看 console 输出、验证链接和移动端响应式表现。

推荐工作流：

1. UI 实现前调用 `$frontend-design`。
2. UI 实现后调用 `$playwright-skill`。
3. 未经明确批准，不要引入新的 UI 库。
4. 优先使用现有项目组件和设计 token。
5. 浏览器测试优先使用本地开发服务器 URL。
