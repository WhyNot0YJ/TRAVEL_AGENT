# UI Harness Case Catalog

本文档说明 `web/e2e` 中每个前端 UI Harness case 的测试场景。新增、删除或重命名 UI Harness test 时，必须同步更新本文档。

## Harness 位置

* Playwright 用例：`web/e2e`
* 配置：`web/playwright.config.ts`
* 脚本：`web/package.json` 中的 `harness:ui`
* 默认报告：`reports/ui_eval_report.json`
* 常用命令：

```bash
cd web
npm run harness:ui
```

## 维护规则

新增前端 UI case 时：

1. 在 `web/e2e` 下增加或更新 Playwright test。
2. 在本文档的 case 表格中追加一行，说明覆盖场景、mock 契约和主要验证点。
3. 如果 mock API/SSE 契约发生变化，需要同步检查 `docs/api.md`。
4. UI case 默认会在 desktop 和 mobile 两个 Playwright project 下各跑一遍。

## Case 列表

当前 UI Harness 位于 `web/e2e/chat-agent.spec.ts`。每个 test 都会在 `chromium-desktop` 和 `chromium-mobile` project 下运行。

| Test | 覆盖场景 | Mock 契约 | 主要验证点 |
| :--- | :--- | :--- | :--- |
| `chat UI generates and displays a travel plan` | 完整聊天到生成路线的主链路 | Mock `chat/stream`、`POST /plans`、任务 SSE 和任务查询 | 输入需求后展示 brief，生成按钮可用，点击后展示路线详情、景点、调整入口和预算拆分 |
| `planning stream appends chunks inside one assistant result bubble` | 规划阶段的 `assistant_delta` 流式追加体验 | 任务 SSE 推送多段 `assistant_delta`，最后 `done.plan` | 多段生成文本追加在同一个助手结果气泡中，不额外制造重复消息 |
| `chat UI blocks generation when required brief fields are missing` | 必填项缺失时阻止生成 | 覆盖 `chat/stream`，返回 `travelers=0`、`missing=["出行人数"]`、`is_complete=false` | brief review 显示缺失字段，`生成行程` 按钮禁用 |
| `chat UI recovers with polling when SSE disconnects` | 规划 SSE 断线后的查询兜底 | 任务 SSE 返回 503，任务查询接口返回 succeeded plan | 前端能切换到 polling，并最终展示路线详情 |

## 新增 Case 检查清单

新增 case 前先确认它补的是新的 UI 风险，而不是重复覆盖已有路径。建议至少写清楚：

* 用户交互：输入内容、点击路径、模式开关、生成按钮或后续调整入口。
* Mock 契约：`chat/stream`、`POST /plans`、任务 SSE、任务查询或失败响应。
* 预期信号：按钮状态、brief 内容、进度面板、流式文本、最终计划、fallback 文案或移动端布局。
* 是否需要同步文档：本文档、`docs/evaluation-harness.md`、`README.md`、`docs/api.md`。
