# Frontend Skills

This project installs two external GitHub skills for Codex.

## Installed Skills

### frontend-design

Source:

https://github.com/anthropics/skills/tree/main/skills/frontend-design

Use it before creating or redesigning frontend UI.

Example:

```txt
$frontend-design 请优化这个页面，让它更像真实产品，不要 AI 味太重。
```

### playwright-skill

Source:

https://github.com/lackeyjb/playwright-skill

Use it after frontend changes to test pages with Playwright.

Example:

```txt
$playwright-skill 请启动本地前端页面，测试首页、核心交互、移动端截图，并检查 console 和 network 错误。
```

## Recommended Workflow

```txt
$frontend-design 先给这个页面做设计审查和改版方案。
```

Then implement the UI.

```txt
$playwright-skill 启动项目并验收刚刚修改的页面，截图、点击关键交互、检查报错。
```