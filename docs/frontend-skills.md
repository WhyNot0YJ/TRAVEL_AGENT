# 前端 Skills

本项目为 Codex 准备了外部 GitHub skills，用于前端设计和浏览器测试。

## 已安装 Skills

### frontend-design

来源：

https://github.com/am-will/codex-skills/tree/main/skills/frontend-design

在创建或重新设计前端 UI 之前使用。

示例：

```txt
$frontend-design 请优化这个页面，让它更像真实产品，不要 AI 味太重。
```

### playwright

来源：

https://github.com/openai/skills/tree/main/skills/.curated/playwright

在前端改动后使用，用 Playwright 测试页面、交互、截图、console 和网络错误。

示例：

```txt
$playwright 请启动本地前端页面，测试首页、核心交互、移动端截图，并检查 console 和 network 错误。
```

### andrej-karpathy-skill

来源：

https://github.com/duolahypercho/andrej-karpathy-skills/tree/main/skills/andrej-karpathy-skill

在写代码、审查、调试或重构时使用，用于提醒 Codex 先明确假设、保持改动克制、避免过度工程化，并定义可验证的完成标准。

示例：

```txt
$andrej-karpathy-skill 请按这个风格审查这次改动，重点看假设、边界和验证方式。
```

## 推荐工作流

```txt
$frontend-design 先给这个页面做设计审查和改版方案。
```

然后实现 UI。

```txt
$playwright 启动项目并验收刚刚修改的页面，截图、点击关键交互、检查报错。
```

## 工作台 / 用户中心专项

工作台路由层使用 AppShell + react-router-dom，把首页、生成页、用户中心、计划详情、公开列表、公开详情拆分成独立路由。在改这块 UI 时建议：

1. 设计阶段使用 `$frontend-design`，重点确认 token (`--ta-route-green`、`--ta-lake-blue`、`--ta-sunset-coral`、`--ta-gold`) 是否覆盖目标视觉、是否避免大面积渐变与营销卡片堆叠。
2. 实现完后跑 `$playwright-skill`，覆盖以下回归点（已在 `web/e2e/` 提供 spec，只需扩展即可）:
   - 注册 / 登录 / 登出 / refresh 后 `/auth/me` 恢复
   - 首页 4 区块 + 移动底栏（mobile project）
   - 生成完成 → 未登录保存 → 登录引导 → 自动续保存
   - 我的计划：重命名、编辑、删除二次确认、发布 / 取消发布
   - 公开详情：保存为副本
   - 私有计划权限：他人 plan_id 显示 "不存在或已不可见"

3. 每次新增 Playwright case，需要把它写入 `web/e2e/harness-cases.md`，并保证与 `docs/api.md` 的 mock 契约对齐。
