# Stage 8：React + TypeScript 前端

## 任务目标

新增 React + TypeScript H5 前端：

* 接入 Gin API
* 支持创建旅游路线规划请求
* 支持 SSE 流式展示生成过程
* 支持路线详情展示
* 支持 loading / error / empty 状态
* 支持 typed API client
* 先做 H5，后续再套壳 App

## 当前前置条件

第 1 到 7 阶段已完成：后端 Gin API、POST 创建任务、GET 查询任务、SSE stream 均可用，`docs/api.md` 已描述接口。

## 本阶段不做什么

* 不改后端核心 Planner 逻辑，除非发现前端必须依赖的小型 API bug
* 不接登录注册
* 不做支付、订单、收藏、分享
* 不接移动端原生壳
* 不引入大型 UI 框架，除非项目已有明确选择
* 不做复杂地图渲染
* 不硬编码后端地址，配置来自环境变量

## 需要阅读的文件

* `AGENTS.md`
* `README.md`
* `docs/api.md`
* `docs/architecture.md`
* `cmd/server/main.go`
* `internal/travel/dto.go`
* `internal/travel/handler.go`
* 现有前端目录：如果不存在，需要创建
* `skills/agent-feature-dev.md`：如果不存在，说明该文件当前不存在，后续阶段需要创建

## 需要新增或修改的文件

预计新增或修改：

* `web/package.json`
* `web/tsconfig.json`
* `web/vite.config.ts`
* `web/index.html`
* `web/src/main.tsx`
* `web/src/App.tsx`
* `web/src/api/client.ts`
* `web/src/api/types.ts`
* `web/src/hooks/useTravelPlanStream.ts`
* `web/src/components/TravelPlanForm.tsx`
* `web/src/components/PlanProgress.tsx`
* `web/src/components/PlanDetail.tsx`
* `web/src/components/StateView.tsx`
* `web/src/styles.css`
* `README.md`
* `docs/api.md`：如发现前后端契约需要补充
* `docs/architecture.md`

## 实现要求

* 使用 React + TypeScript。
* 建议使用 Vite，除非项目已有前端构建方案。
* API base URL 来自环境变量，例如 `VITE_API_BASE_URL`。
* typed API client 必须覆盖创建任务、查询任务、连接 SSE。
* 表单至少包含出发城市、目的地城市、天数、预算、兴趣标签、交通方式、节奏。
* 提交后调用 POST 创建任务，展示 loading/progress，连接 SSE stream，收到 done 后展示路线详情。
* 必须实现 loading、error、empty 状态，以及 SSE 断开或失败降级提示。
* UI 以移动端 H5 为主，布局紧凑、清晰、适合反复使用。
* 不要做营销 landing page，第一屏就是可用的路线规划工具。
* 不要用大段说明文字描述功能。
* 表单、按钮、状态提示在移动端不能溢出或重叠。
* 路线详情展示 days、items、budget、warnings。
* TypeScript 类型不要用大量 `any`。
* 如后端跨域未配置，可最小化补充 CORS 配置，并同步更新 docs。

## 文档更新要求

* 更新 `README.md`：说明前端启动、环境变量、后端依赖。
* 更新 `docs/architecture.md`：说明 React -> Gin API -> SSE 的关系。
* 如 API 契约有补充，更新 `docs/api.md`。
* 不更新 `docs/database.md`，除非新增持久化结构。
* 不更新 `docs/agent-flow.md`，除非改变 Agent 流程。

## 测试要求

后端尽量运行：

```bash
go test ./...
go vet ./...
```

前端尽量运行：

```bash
cd web
npm install
npm run typecheck
npm run lint
npm run build
```

如果项目未配置 lint，需要在最终回复中说明未运行原因，或补充合理 lint 脚本。

本地联调：

```bash
go run ./cmd/server
cd web
npm run dev
```

## 验收标准

* 前端可启动。
* 可以提交旅行规划请求。
* 可以看到 SSE progress。
* done 后展示路线详情。
* loading / error / empty 状态完整。
* typed API client 存在。
* 移动端 H5 布局可用。
* 后端 Harness 和 API 不被破坏。

## 最终回复要求

完成后请输出：

1. 修改了哪些文件
2. 新增了哪些文件
3. 做了哪些设计决策
4. 跑了哪些测试
5. 测试是否通过
6. 前端和接口如何验证
7. 风险和未完成事项
