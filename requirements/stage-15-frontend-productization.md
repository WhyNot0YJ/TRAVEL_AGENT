# Stage 15：前端产品化

## 任务目标

把当前 React H5 对话式 demo 打磨成可反复使用的旅行规划工具。重点是路线查看、局部调整、错误恢复、移动端体验和真实 API 联调，而不是做营销落地页。

## 当前上下文

当前前端已有：

* Vite + React + TypeScript
* 对话式 AgentConversation
* TravelBriefPanel
* PlanProgress
* PlanDetail
* typed API client
* SSE + polling fallback hook
* Playwright UI harness

## 不做什么

* 不引入大型 UI 框架，除非用户明确批准。
* 不做营销 landing page。
* 不做复杂地图渲染，除非后端已提供稳定坐标和路线数据。
* 不接登录、支付、订单、收藏、分享，除非进入阶段 16。
* 不在前端硬编码后端地址。

## 必须使用的技能

* 前端 UI 改造前使用 `$frontend-design`。
* 前端实现后使用 `$playwright-skill` 或项目已有 Playwright harness 进行验证。

## 需要阅读的文件

* `docs/frontend-skills.md`
* `docs/api.md`
* `docs/architecture.md`
* `web/src/App.tsx`
* `web/src/styles.css`
* `web/src/api/client.ts`
* `web/src/api/types.ts`
* `web/src/hooks/useTravelPlanStream.ts`
* `web/src/components/AgentConversation.tsx`
* `web/src/components/PlanDetail.tsx`
* `web/src/components/PlanProgress.tsx`
* `web/e2e/chat-agent.spec.ts`

## 实现方向

优先考虑以下能力：

* 路线详情从纯展示升级为可扫描的时间轴。
* 支持"重新生成某一天"或"调整预算/节奏/兴趣"的入口。
* 展示 warning/fallback 的温和解释。
* 预算拆分更清晰。
* 移动端输入、按钮、进度、路线详情不能溢出或重叠。
* SSE 断开、任务失败、后端限流要有明确恢复路径。
* 增加真实后端联调模式文档。

## 文档更新要求

* 更新 `README.md`：前端运行、构建、UI harness、联调方式。
* 更新 `docs/architecture.md`：如前端状态流变化，需要同步。
* 更新 `docs/api.md`：如前端推动 API 契约变化，必须同步。

## 测试要求

尽量运行：

```bash
cd web
npm run typecheck
npm run lint
npm run build
npm run harness:ui
```

如涉及后端联调：

```bash
go test ./...
go run ./cmd/server
cd web
npm run dev
```

## 验收标准

* 第一屏仍是可用工具，不是 landing page。
* 移动端和桌面端核心流程可用。
* Playwright 覆盖生成、进度、done 渲染和错误/断线至少一种路径。
* UI 文案不夸大真实 LLM 或真实 API 能力。
