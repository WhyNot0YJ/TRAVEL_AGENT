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

当前 UI Harness 位于 `web/e2e/`。Stage 18-20 链路位于 `chat-agent.spec.ts`，Stage 21 在 `auth-flow.spec.ts`、`home.spec.ts`、`save-publish.spec.ts`、`plan-library.spec.ts`、`public-detail.spec.ts`、`permission.spec.ts`。每个 test 都会在 `chromium-desktop` 和 `chromium-mobile` project 下运行。

| Test | 覆盖场景 | Mock 契约 | 主要验证点 |
| :--- | :--- | :--- | :--- |
| `chat UI generates and displays a travel plan` | 完整聊天到生成路线的主链路（/planner） | Mock `chat/stream`、`POST /plans`、任务 SSE、任务查询、`/auth/me=401` | 输入需求后展示 brief，生成按钮可用，点击后展示路线详情、景点、调整入口和预算拆分 |
| `planning stream appends chunks inside one assistant result bubble` | 规划阶段 `assistant_delta` 流式追加体验 | 任务 SSE 推送多段 `assistant_delta`，最后 `done.plan` | 多段生成文本追加在同一个助手结果气泡中，不额外制造重复消息 |
| `chat UI blocks generation when required brief fields are missing` | 必填项缺失时阻止生成 | 覆盖 `chat/stream`，返回 `travelers=0`、`missing=["出行人数"]`、`is_complete=false` | brief review 显示缺失字段，`生成行程` 按钮禁用 |
| `chat UI recovers with polling when SSE disconnects` | 规划 SSE 断线后的查询兜底 | 任务 SSE 返回 503，任务查询接口返回 succeeded plan | 前端能切换到 polling，并最终展示路线详情 |
| `auth flow › anonymous visitor cannot reach user center and is bounced to login` | RequireAuth 守卫 | `/auth/me=401`，`/public/plans=200 []` | 直接访问 `/me` 跳转 `/login`，展示 AuthView |
| `auth flow › registration auto-signs in and lands the visitor on the saved return path` | 注册后自动登录并跳回 return_to | `/auth/register`，`/auth/me` 注册后改为已登录 | 提交注册表单后到达 `/me`，UserCenter 渲染 |
| `auth flow › login error shows the stable invalid credentials message` | 错误密码统一文案 | `/auth/login=401 invalid_credentials` | `auth-error` 显示 "邮箱或密码不正确" |
| `home view › home renders hot ranking and recommended sections` | 首页核心区块加载 | `/auth/me=401`，`/public/plans?sort=hot|latest` | 排行榜与推荐区都展示公开计划，排行榜首位有排名 |
| `home view › typing in search submits to the public list` | 首页搜索 | `/public/plans` 返回 mock 数据 | 输入查询并点击搜索后跳转 `/public?q=`，公开列表渲染 |
| `home view › mobile bottom navigation exposes home / planner / me without overflow` | 移动端底栏 | 与上述路由相同 | mobile project 下底栏可见，3 个入口存在 |
| `save and publish flow › anonymous user is bounced to login when saving and the action resumes after sign-in` | 生成完成后保存计划的全链路 | 任务 SSE/查询，`/auth/me`，`/auth/login`，`/me/plans` POST | 未登录点保存跳登录；登录成功后回到 `/planner?task_id=`，自动调用 POST `/me/plans`，显示 "查看我的计划" |
| `plan library › user can publish and then unpublish a plan from the library` | 用户中心发布/取消发布 | `/auth/me=200`，`/me/plans?` 列表，`publish`/`unpublish` 接口 | 发布后行内出现 "取消发布" 按钮，toast 显示对应文案，确认弹窗后取消发布 |
| `plan library › delete confirmation removes the plan` | 软删除二次确认 | `DELETE /me/plans/:id`，列表第二次返回 [] | 弹窗显示 "确认删除"，确认后空状态出现 |
| `public detail › authenticated user can save a public plan as a private copy` | 公开详情保存为私有副本 | `/public/plans/:id`，`/public/plans/:id/save`，`/me/plans/:plan_id` | 保存成功跳转 `/me/plans/plan_copied`，私有详情展示来源标题 |
| `permission guards › unauthenticated /me/* redirects to login with return_to` | 未登录访问私有路径 | `/auth/me=401` | URL 含 `return_to=`，AuthView 展示 |
| `permission guards › private plan owned by another user surfaces a not-found message` | 他人私有计划 | `/auth/me=200`，`/me/plans/:id=404` | 私有详情错误状态展示 "不存在或已不可见" |

## 新增 Case 检查清单

新增 case 前先确认它补的是新的 UI 风险，而不是重复覆盖已有路径。建议至少写清楚：

* 用户交互：输入内容、点击路径、模式开关、生成按钮或后续调整入口。
* Mock 契约：`chat/stream`、`POST /plans`、任务 SSE、任务查询或失败响应。
* 预期信号：按钮状态、brief 内容、进度面板、流式文本、最终计划、fallback 文案或移动端布局。
* 是否需要同步文档：本文档、`docs/evaluation-harness.md`、`README.md`、`docs/api.md`。
