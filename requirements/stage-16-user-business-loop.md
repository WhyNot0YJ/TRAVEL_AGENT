# Stage 16：用户与业务闭环

## 任务目标

在核心规划能力稳定后，补齐应用层闭环：用户身份、历史行程、收藏、分享、导出和权限边界。酒店、票务、支付等高风险外部 API 只作为后续可选扩展，不在本阶段默认接入。

## 当前上下文

当前系统没有：

* 用户系统
* 登录鉴权
* 历史行程归属
* 收藏/分享/导出
* 订单/支付
* 酒店/票务 API

当前 API 是匿名任务模式，适合 demo 和评估，不适合长期用户数据管理。

## 不做什么

* 不直接接支付。
* 不保存明文密码。
* 不把匿名历史任务直接暴露给任意用户。
* 不接酒店、票务、支付 API，除非另开安全和合规评估。
* 不在文档中编造未实现的商业闭环。

## 需要阅读的文件

* `docs/prd.md`
* `docs/api.md`
* `docs/database.md`
* `docs/architecture.md`
* `internal/server/router.go`
* `internal/server/middleware.go`
* `internal/travel/handler.go`
* `internal/travel/service.go`
* `web/src/api/client.ts`
* `web/src/App.tsx`

## 推荐能力拆分

先做低风险用户闭环：

1. 用户注册/登录或外部身份 provider 接入方案。
2. 用户与 travel task/plan 的归属关系。
3. 历史行程列表。
4. 行程详情复查。
5. 收藏或归档。
6. 导出 Markdown/JSON/PDF 的可选能力。
7. 分享链接，默认只读并可撤销。

酒店、票务、支付 API 应独立设计，不与基础用户系统混在一个阶段。

## 安全要求

* 所有鉴权配置来自环境变量或配置文件。
* 密码必须哈希存储；优先考虑成熟方案。
* API 必须校验当前用户是否有权访问 task/plan。
* 分享链接应使用不可预测 token，并支持过期或撤销。
* 文档需要说明隐私和数据保留策略。

## 文档更新要求

* 更新 `docs/prd.md`：补充用户故事和范围。
* 更新 `docs/api.md`：补充 auth、history、share/export 接口。
* 更新 `docs/database.md`：补充 users、sessions、plan ownership、share links 等表。
* 更新 `docs/architecture.md`：说明鉴权层和数据归属。
* 更新 `README.md`：补充本地运行所需环境变量。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
```

如涉及前端：

```bash
cd web
npm run typecheck
npm run lint
npm run build
npm run harness:ui
```

## 验收标准

* 匿名任务模式是否保留有明确决策。
* 用户只能访问自己的行程。
* 历史行程可查询。
* 分享链接权限边界清晰。
* 安全文档和数据库文档同步。
