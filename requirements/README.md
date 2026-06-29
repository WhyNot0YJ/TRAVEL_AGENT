# Travel Agent 需求文档目录

本目录是 Travel Agent 项目的**需求文档主目录**，按交付阶段（stage）拆分管理。每一份文档既是 **Codex 执行提示词**，也是该阶段的 **需求规约**——必须先读对应文档，再动代码。

> 本目录是从 [docs/development-phase-prompts.md](../docs/development-phase-prompts.md) 拆分而来。原文件保留作为历史归档；今后所有阶段调整请只修改本目录下的对应文件。

## 阅读顺序

| Stage | 主题 | 文件 |
| :---: | :--- | :--- |
| 1 | Harness + MockPlanner | [stage-01-harness-mock-planner.md](./stage-01-harness-mock-planner.md) |
| 2 | EinoTravelPlanner + Mock Tools | [stage-02-eino-mock-tools.md](./stage-02-eino-mock-tools.md) |
| 3 | 接入真实 LLM | [stage-03-real-llm.md](./stage-03-real-llm.md) |
| 4 | 接入真实 高德 / 天气 / POI API | [stage-04-real-external-apis.md](./stage-04-real-external-apis.md) |
| 5 | Gin API | [stage-05-gin-api.md](./stage-05-gin-api.md) |
| 6 | Redis 缓存、限流、任务状态 | [stage-06-redis-cache-rate-limit.md](./stage-06-redis-cache-rate-limit.md) |
| 7 | SSE 流式接口 | [stage-07-sse.md](./stage-07-sse.md) |
| 8 | React + TypeScript 前端 | [stage-08-react-frontend.md](./stage-08-react-frontend.md) |
| 9 | 真实外部 Tool 稳定化 | [stage-09-external-tool-hardening.md](./stage-09-external-tool-hardening.md) |
| 10 | 真实 LLM 规划链路硬化 | [stage-10-llm-pipeline-hardening.md](./stage-10-llm-pipeline-hardening.md) |
| 11 | SQL 持久化与 Repository 层 | [stage-11-sql-persistence.md](./stage-11-sql-persistence.md) |
| 12 | 路线真实性校验 | [stage-12-route-feasibility.md](./stage-12-route-feasibility.md) |
| 13 | Evaluation Harness 升级 | [stage-13-harness-upgrade.md](./stage-13-harness-upgrade.md) |
| 14 | 观测性与可靠性 | [stage-14-observability.md](./stage-14-observability.md) |
| 15 | 前端产品化 | [stage-15-frontend-productization.md](./stage-15-frontend-productization.md) |
| 16 | 用户与业务闭环 | [stage-16-user-business-loop.md](./stage-16-user-business-loop.md) |
| 17 | DeepSeek 真·流式 SSE 透传 | [stage-17-deepseek-true-streaming.md](./stage-17-deepseek-true-streaming.md) |
| 18 | Travel Brief 确认与可选偏好默认值 | [stage-18-travel-brief-confirmation.md](./stage-18-travel-brief-confirmation.md) |

## 专项需求

| 类型 | 主题 | 文件 |
| :--- | :--- | :--- |
| Backend | 后端高可用与性能优化专项 | [backend-performance-optimization.md](./backend-performance-optimization.md) |

## 使用约定

1. 每个 stage 文档都是**自包含**的：可以直接复制全文给 Codex 作为单次 PR 的需求规约。
2. 每个 stage 文档结构遵循统一模板：
   - `任务目标` —— 本阶段要交付什么
   - `当前前置条件 / 当前上下文` —— 已经存在的东西
   - `本阶段不做什么` —— 显式排除项，避免范围蔓延
   - `需要阅读的文件` —— Codex 必读清单
   - `需要新增或修改的文件` —— 预期改动范围
   - `实现要求` —— 具体技术约束
   - `文档更新要求` —— 同步要更新哪些 docs
   - `测试要求` —— 必跑的命令
   - `验收标准` —— 通过条件
3. 如果阶段已经实现完毕，文档不要删除；用作历史规约。后续阶段在此基础上叠加。
4. 新增阶段一律按 `stage-NN-<kebab-case>.md` 命名，并在本 README 表格末尾追加。

## 当前活跃阶段

> 当前优先推进：**Stage 18 — Travel Brief 确认与可选偏好默认值**。
> 详见 [stage-18-travel-brief-confirmation.md](./stage-18-travel-brief-confirmation.md)。

