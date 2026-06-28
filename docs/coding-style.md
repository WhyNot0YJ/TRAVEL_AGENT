# 编码规范

本文定义 Travel Agent 项目的日常开发规范。目标是让代码在多人协作、长期演进和线上运行时保持清晰、稳定、可测试、可观测、可回滚。

## 1. 基本原则

1. 以可读性优先。代码首先写给后续维护者看，其次才是写给机器执行。
2. 保持改动聚焦。一次变更只解决一个清晰问题，避免把无关重构、格式化和功能修改混在一起。
3. 优先使用项目已有模式。新增代码应贴近现有目录结构、命名方式、错误处理方式和测试组织方式。
4. 避免过度抽象。只有当抽象能明确减少重复、隔离变化或表达稳定边界时才引入。
5. 让失败可解释。错误、日志、warning、报告字段应能帮助定位原因，而不是只暴露“失败了”。
6. 让行为可验证。新增逻辑应配套测试、示例、文档或可重复运行的验证命令。
7. 默认安全。任何 API Key、token、密码、用户敏感信息和外部原始响应都不得硬编码或直接落日志。
8. 保持向后兼容。接口、报告字段、SSE 事件、环境变量和数据结构变更应尽量兼容旧调用方；破坏性变更必须在文档中明确说明。

## 2. 目录与边界

1. `internal/domain` 只放稳定业务实体，不依赖 Gin、Redis、MySQL、Eino、LLM provider 或外部 API 原始结构。
2. `internal/agent` 放 `TravelPlanner` 接口、MockPlanner 和 agent 通用抽象。
3. `internal/agent/eino` 放 Eino Graph、节点、工具、LLM 客户端和 Eino 内部状态。
4. `internal/harness` 只能依赖 `TravelPlanner` 接口，不得 import `internal/agent/eino`。
5. `internal/travel` 放 HTTP DTO、handler、service、任务 store、EventBus、SSE 事件和应用层编排。
6. `internal/server` 放 router、中间件、CORS、recovery、request id 和 HTTP server 组装。
7. `docs` 放设计与接口文档。行为、接口、流程、数据库、外部 API 或评估指标变化时必须同步更新对应文档。
8. `testdata` 放稳定评估数据；测试数据应可复现，不依赖真实外部服务。
9. `reports` 放生成报告。提交报告文件前确认是否确实需要纳入版本控制。
10. 前端代码放在 `web` 下，优先复用已有组件、样式变量、API client 和类型定义。

## 3. Go 编码规范

### 3.1 命名

1. 包名使用小写短名，避免下划线和泛化名称。
2. 导出类型、函数、常量使用清晰业务名，避免 `Manager`、`Helper`、`Util` 这类含义过宽的命名。
3. 私有变量用短而明确的名字；作用域越大，名字越具体。
4. 接口命名应表达调用方需要的能力，例如 `TravelPlanner`、`TaskStore`，不要为了测试而提前拆出没有业务含义的接口。
5. 错误分类、状态值、事件类型、报告字段应集中定义，避免散落字符串。

### 3.2 结构

1. handler 只做请求解析、参数校验、调用 service、返回响应，不承载业务流程。
2. service 负责应用层编排、事务边界、缓存和异步任务生命周期。
3. DTO 与 domain 分离。HTTP JSON 字段变化不要直接污染 `internal/domain`。
4. 外部 provider 响应结构只存在于 client/tool 内部，转换后再进入内部状态或 domain。
5. 长函数应按业务步骤拆分，但不要把顺序流程拆成难以追踪的碎片。
6. 构造函数应保证对象进入可用状态，必要依赖通过参数传入，可选配置使用明确默认值。

### 3.3 错误处理

1. Go 代码必须显式处理 `error`，不得静默丢弃，除非有注释解释为什么安全。
2. 对外返回的错误信息应稳定、可读、脱敏；内部日志可包含更多上下文，但不能包含密钥或敏感原始响应。
3. 使用 `%w` 包装底层错误，保留错误链，便于上层分类。
4. provider、tool、LLM、JSON 解析、业务校验等错误应尽量分类，例如 `configuration`、`timeout`、`provider_error`、`invalid_json`。
5. fallback 必须可观测。发生 mock fallback、LLM fallback、polling fallback 时，应有 warning、trace 或日志说明原因。
6. 不要用 panic 表示普通业务失败。panic 只用于不可恢复的程序员错误，并应由 recovery 兜底。

### 3.4 Context 与并发

1. 所有可能阻塞的 I/O、外部 API、LLM 调用、数据库操作都应接收 `context.Context`。
2. 不要把 context 存入结构体作为长期字段；调用链传递即可。
3. goroutine 必须有清晰退出条件，避免泄漏。
4. 后台任务应捕获 panic，记录 request id/task id，并把任务状态更新为失败。
5. 并发写共享状态必须使用锁、channel 或并发安全结构；不要依赖“当前看起来不会并发”。
6. 外部 API 调用必须有限流、超时和 fallback 策略。

### 3.5 配置

1. 配置从环境变量或配置文件读取，不硬编码。
2. 所有环境变量要有默认值、含义说明和文档记录。
3. 配置解析失败要给出明确错误或使用安全默认值。
4. bool、duration、int 等类型不要在业务代码里重复解析，应集中在 config 层。
5. 不要在测试中依赖开发者本机环境变量，除非测试明确标记为集成或手动测试。

### 3.6 日志与可观测性

1. 日志字段保持稳定，优先使用 `request_id`、`task_id`、`planner`、`node`、`duration_ms`、`status`、`error`。
2. request id 应贯穿 handler、service、planner、tool、LLM 和 SSE 事件。
3. 日志要记录关键状态变化，不要在循环或高频路径中输出大量噪声。
4. 耗时统计使用毫秒级字段，便于报告聚合。
5. 不记录 API Key、Authorization header、cookie、用户完整隐私输入或 provider 原始大响应。

### 3.7 格式化

1. Go 文件使用 `gofmt` 或 `go fmt`。
2. import 顺序交给工具处理。
3. 不手动对齐大段变量声明来追求视觉效果，避免后续 diff 噪声。
4. 注释用于解释“为什么”或复杂边界，不重复描述代码已经表达的“做什么”。

## 4. TypeScript / React 编码规范

### 4.1 TypeScript

1. 开启并遵守严格类型检查，避免使用 `any`。确实需要时，应把范围压到最小并说明原因。
2. API 类型放在 `web/src/api/types.ts`，请求封装放在 `web/src/api/client.ts`。
3. 后端接口字段变化时，同步更新前端类型、调用逻辑和 `docs/api.md`。
4. 对外部输入、URL 参数、SSE payload 做运行时容错，不假设字段永远存在。
5. 复杂联合状态应使用明确的 discriminated union，而不是多个互相影响的 boolean。

### 4.2 React 组件

1. 组件职责单一。容器组件处理数据和状态，展示组件负责 UI 呈现。
2. 状态尽量靠近使用位置；跨页面或跨流程共享的状态才上提。
3. 避免在 render 中执行副作用。网络请求、订阅、计时器放入 `useEffect`，并正确清理。
4. SSE、轮询、定时器、事件监听必须在组件卸载或依赖变化时清理。
5. key 使用稳定业务 id，不使用数组下标表示可变列表。
6. 表单状态和提交状态要覆盖 loading、success、error、disabled、retry 等基本状态。
7. 用户可见错误应给出下一步动作，例如重试、修改输入、切换测试模式。

### 4.3 前端样式

1. 优先使用项目现有样式变量和组件模式，不随意引入 UI 库。
2. 移动端 H5 优先考虑单手操作、内容扫描、触控目标和弱网络反馈。
3. 文案、按钮、卡片、输入框在窄屏下不得溢出或互相遮挡。
4. 交互元素应有明确 hover/focus/disabled/loading 状态。
5. 不用纯装饰性复杂效果影响性能和可读性。
6. 前端改版前优先使用 `$frontend-design` 做设计审查；改版后使用 Playwright 做浏览器验证。

## 5. API 与协议规范

1. HTTP API 使用稳定 JSON 契约。字段新增优先向后兼容，避免删除或改变含义。
2. 错误响应统一包含 `request_id`、`code`、`message`。
3. 状态码语义清晰：请求问题用 `400`，未找到用 `404`，限流用 `429`，内部错误用 `500`。
4. SSE 事件类型必须文档化，新增事件要说明 payload、触发时机和前端处理建议。
5. 前端应以最终 `done.plan` 作为结构化行程可信来源，`assistant_delta` 只作为生成过程文本。
6. 对异步任务，创建、查询、订阅三类接口语义必须保持一致。
7. request hash 的组成变化会影响缓存命中，必须在文档中说明。
8. 任何 API 变更都必须更新 `docs/api.md`，必要时同步 README 示例。

## 6. Agent / Eino 规范

1. Eino 相关代码只放在 `internal/agent/eino` 或子目录。
2. 新增 Graph 节点必须有清晰输入、输出和错误语义。
3. 节点内部状态不要污染 `internal/domain`。
4. Mock Tools 必须稳定、可复现，不依赖真实网络。
5. 接入真实 LLM 或真实外部 API 时，不能破坏 Mock Tools 和默认 harness 路径。
6. LLM 输出必须结构化校验，不能只依赖 prompt 约束。
7. LLM 或 provider 失败时应 fallback 到 deterministic generator，并记录稳定 warning。
8. Tool 调用失败时应 fallback 到 mock tool，并记录稳定 warning。
9. 新增节点、工具、LLM 行为或流程顺序变化时，必须更新 `docs/agent-flow.md`。

## 7. 数据库与缓存规范

1. 新增表、字段、索引或持久化模型时，必须更新 `docs/database.md`。
2. 数据库 schema 变化必须通过 migration 管理，不手动要求用户改库。
3. 表字段命名使用 snake_case，语义稳定，避免缩写过度。
4. 关键查询路径必须有索引，例如 task id、request hash、status、updated_at。
5. JSON 快照字段要控制大小，不保存敏感原始响应。
6. Redis key 应有清晰命名空间、TTL 和用途说明。
7. Redis 不作为唯一长期持久化来源；服务重启后的行为要在文档中说明。
8. 数据库连接池配置来自环境变量，并有合理默认值。

## 8. 安全规范

1. 不硬编码 API Key、token、密码、DSN 中的敏感部分或真实用户凭据。
2. 示例中的 key 使用 `your-api-key`、`your-amap-key` 等占位符。
3. 日志、报告、warning、错误响应不得包含完整密钥、Authorization header 或 cookie。
4. 外部 API 原始响应默认不落库；如需保存，必须脱敏并限制大小。
5. 用户输入进入日志或报告前应考虑脱敏、截断和注入风险。
6. CORS、限流、超时、请求体大小和 recovery 中间件应作为 HTTP 服务基础能力。
7. 不把 `.env`、本地报告中的敏感内容或真实凭据提交到仓库。

## 9. 测试规范

### 9.1 后端测试

1. 纯函数、解析器、校验器和评分逻辑优先写单元测试。
2. handler/service/task store/SSE 流程可写集成测试或使用 fake store/fake planner。
3. 外部 API client 使用 fake HTTP server 覆盖成功、超时、非 2xx、无效 JSON、缺字段和限流。
4. LLM client 使用 mock/fake 响应覆盖 tool call、JSON schema、stream、fallback 和 retry。
5. 测试不得依赖真实 API Key 或真实外部服务。
6. 时间相关逻辑应可控，避免 flaky。
7. 并发逻辑应覆盖重复请求、缓存命中、任务复用和 race 风险。

### 9.2 前端测试

1. API client、状态转换、SSE 解析可写单元测试。
2. 关键页面和交互使用 Playwright 验证。
3. 至少覆盖桌面和移动端主要 viewport。
4. 浏览器测试应检查 console error、网络错误、加载态、失败态和重试路径。
5. 对话式流程要覆盖信息不完整、信息齐全、生成中、SSE 断开 fallback、最终计划展示。

### 9.3 推荐命令

完成任务前尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness
go run ./cmd/harness -planner eino
make harness
make harness-eino
```

前端相关修改尽量运行：

```bash
cd web
npm run typecheck
npm run lint
npm run harness:ui
```

如果某些命令因环境、依赖或外部服务不可用而无法运行，应在最终说明中写清楚原因。

## 10. 文档规范

1. 文档描述必须对应已实现行为，不编造尚未落地的复杂能力。
2. README 面向使用者，说明运行方式、核心能力和常用配置。
3. `docs/api.md` 面向接口调用方，说明请求、响应、错误、SSE 事件和兼容性。
4. `docs/agent-flow.md` 面向 Agent 开发，说明节点、工具、LLM、fallback 和流程边界。
5. `docs/external-apis.md` 面向外部服务接入，说明配置、限制、fallback 和稳定性边界。
6. `docs/evaluation-harness.md` 面向评估，说明测试集、指标、报告字段和运行方式。
7. `docs/database.md` 面向持久化，说明表、字段、索引、Redis key 和清理策略。
8. 文档示例应可运行或明确标注为示意。
9. 技术名词、环境变量、路径、字段名和命令保持原样，不做意译。
10. 文档大段改动后应检查标题层级、代码块闭合和链接路径。

## 11. 代码审查清单

提交或合并前至少自查：

1. 改动是否聚焦，是否混入无关重构。
2. 是否破坏 `TravelPlanner`、HTTP API、SSE、报告字段或数据结构兼容性。
3. 是否有必要的测试或可重复验证步骤。
4. 错误处理是否有分类、上下文和脱敏。
5. 是否有 goroutine、SSE、timer、subscription 泄漏风险。
6. 是否新增环境变量，且已更新文档和示例。
7. 是否新增外部 API、数据库表、Agent 节点或评估指标，且已更新对应文档。
8. 是否避免硬编码密钥和真实敏感数据。
9. 前端是否覆盖 loading、error、empty、mobile、disabled 等状态。
10. 是否运行了相关测试，未运行的原因是否记录清楚。

## 12. 提交与变更管理

1. 提交信息应简洁说明变更目的，例如 `docs: expand coding style guide`。
2. 功能、修复、重构、文档尽量拆成独立提交。
3. 生成文件、报告文件和格式化变更不要和核心逻辑混在一起。
4. 大改前先确认边界和回滚方案。
5. 发现已有未提交改动时，不要擅自回滚；先判断是否与当前任务相关。
