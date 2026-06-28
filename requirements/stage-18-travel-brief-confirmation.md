# Stage 18：Travel Brief 确认与可选偏好默认值

## 任务目标

把当前“字段齐全即可生成”的聊天需求收集升级为 **Travel Brief 确认机制**：用户在生成行程前应看到一张更完整、更可信的需求确认卡。必填项不完整时阻止生成；可选项缺失时使用稳定默认值，并清楚展示为“已默认 / 可继续补充”的内容。

本阶段交付应覆盖后端 DTO、`domain.TravelRequest`、Eino extractor、本地 rule fallback、request hash、Planner、前端确认卡、文档和测试，确保 `POST /api/v1/travel/chat/stream` 的 `done` payload 是确认卡的数据源，`POST /api/v1/travel/plans` 是确认后创建任务的数据源。

## 当前上下文

当前系统已经具备：

* React + TypeScript H5 对话式前端。
* `POST /api/v1/travel/chat/stream` 聊天式需求收集。
* `POST /api/v1/travel/plans` 创建异步规划任务。
* `GET /api/v1/travel/plans/:task_id/stream` 订阅规划 SSE。
* `GET /api/v1/travel/plans/:task_id` 作为 SSE 断线后的查询兜底。
* `TravelBriefPanel` / `AgentConversation` 等前端组件雏形。
* `MockPlanner`、`EinoTravelPlanner`、Eino Graph + Mock Tools、LLM / rule fallback extractor。
* Stage 17 已把聊天和规划链路的自然语言反馈升级为更真实的 SSE 增量体验。

当前问题：

* 需求收集更偏“字段凑齐”，用户无法系统性确认模型理解是否正确。
* 必填项和可选偏好的边界不够产品化，容易出现前端、后端、LLM 各自猜默认值。
* 出行人数、步行强度、必去地点、避开内容等 richer brief 字段如果只在 UI 展示，不能真正影响 Planner 输出。
* request hash 必须覆盖 richer brief，否则不同偏好的用户可能命中同一缓存结果。

## 需求检查结论

本阶段需求整体合理，但实现时必须明确以下细节，避免歧义：

1. `budget_includes` 在代码和 API 中应保持数组字段，默认值为 `["住宿", "餐饮", "门票", "市内交通"]`；“不含往返大交通”是预算口径说明，不应混入 includes 数组作为包含项。前端展示文案可以渲染为“住宿、餐饮、门票、市内交通；不含往返大交通”。
2. `pace`、`transport_mode`、`walking_tolerance`、`budget_type` 不能在同一层 API 中混用 `balanced/any/total` 和 `适中/任意/总预算`。本阶段应定义一种 canonical wire format，并提供兼容映射。推荐后端对旧英文枚举做输入兼容，对 `ChatResponse` / `CreatePlanRequest` 文档中的默认值使用产品中文值。
3. 出行人数作为必填项是合理的，不应默认 1 人；否则预算、人均/总预算和住宿安排都会误判。
4. `POST /api/v1/travel/chat` 可以标记为兼容 / 可下线接口，但不能影响现有核心体验链路：`chat/stream -> plans -> plans/:task_id/stream`。
5. 不应删除 `GET /api/v1/travel/plans/:task_id`，它仍是 SSE 断线后的任务查询兜底。

## 本阶段不做什么

* 不接入新的外部 API。
* 不引入新的大型 UI 框架或状态管理框架。
* 不做登录、订单、支付、收藏、多人协作。
* 不把 `GET /api/v1/travel/plans/:task_id` 下线。
* 不让 `internal/harness` 直接依赖 `internal/agent/eino`。
* 不硬编码 API Key；所有真实模型和外部 API 配置仍来自环境变量或配置文件。
* 不在文档中宣称尚未实现的真实地图级排程、酒店库存或实时价格能力。

## 字段规则

### 必填项

以下字段缺失时，`is_complete=false`，`missing` 必须包含对应中文字段名，前端不得允许生成：

| 字段 | JSON | 规则 |
| :--- | :--- | :--- |
| 出发地 | `departure_city` | 非空字符串 |
| 目的地 | `destination_city` | 非空字符串 |
| 天数 | `days` | 大于 0 的整数 |
| 预算 | `budget` | 大于 0 的数字 |
| 兴趣偏好 | `interests` | 至少 1 项 |
| 出行人数 | `travelers` | 大于 0 的整数，不默认 |

### 可选项与默认值

以下字段缺失时不阻塞生成，后端 extractor 和前端确认卡都必须使用同一套默认值：

| 字段 | JSON | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| 出行日期 | `date_range` | `任意` | 不要求用户必须给具体日期 |
| 节奏 | `pace` | `适中` | 可兼容旧值 `balanced` |
| 交通偏好 | `transport_mode` | `任意` | 可兼容旧值 `any` |
| 步行强度 | `walking_tolerance` | `任意` | 可兼容旧值 `any` |
| 酒店区域 | `hotel_area` | `任意` | 用户未指定时不强行推荐固定商圈 |
| 必去地点 | `must_visit` | `[]` | 空数组 |
| 避开内容 | `avoid` | `[]` | 空数组 |
| 同行人群 | `traveler_type` | `无要求` | 如亲子、情侣、老人同行等 |
| 预算口径 | `budget_type` | `总预算` | 可兼容旧值 `total` |
| 预算包含项 | `budget_includes` | `["住宿", "餐饮", "门票", "市内交通"]` | 展示时追加“不含往返大交通”说明 |

### Canonical 与兼容映射

实现时必须有明确 normalization 层：

| 旧值 | 新展示 / 输出值 |
| :--- | :--- |
| `any` | `任意` |
| `balanced` | `适中` |
| `relaxed` | `轻松` |
| `intensive` | `紧凑` |
| `low` | `低` |
| `medium` | `中` |
| `high` | `高` |
| `total` | `总预算` |
| `per_person` | `人均预算` |

如果保留内部英文枚举，必须保证 API 文档、前端展示、LLM prompt 和 tests 对边界有一致说明；不能让用户在确认卡看到 `balanced`、`any`、`total` 这类内部值。

## Public Interfaces

### TravelRequest

`domain.TravelRequest` 需要包含并持续传递以下 brief 字段：

* `travelers`
* `date_range`
* `walking_tolerance`
* `hotel_area`
* `must_visit`
* `avoid`
* `traveler_type`
* `budget_type`
* `budget_includes`

已有的 `departure_city`、`destination_city`、`days`、`budget`、`interests`、`transport_mode`、`pace` 继续保留。

### ChatRequest

`ChatRequest` 需要接受同一组 brief 字段，用于多轮补充时携带当前确认卡状态。新消息中显式提到的值可以覆盖旧值；未提到的字段应沿用当前状态或默认值。

### ChatResponse

`ChatResponse` 必须继续返回：

* `missing`
* `is_complete`
* `reply`
* `agent_mode`

并新增 / 保留完整 brief 字段，作为前端确认卡的数据源。`POST /api/v1/travel/chat/stream` 的 `done` payload 必须和非流式 chat response 使用同一结构。

### CreatePlanRequest

`CreatePlanRequest` 必须接受同一组 brief 字段，确保用户点击“确认生成”后传给规划任务时不丢信息。后端校验必须阻止缺失必填项的任务创建。

### 核心链路

保留并强化以下核心体验链路：

1. `POST /api/v1/travel/chat/stream`
2. 前端展示 Travel Brief 确认卡
3. `POST /api/v1/travel/plans`
4. `GET /api/v1/travel/plans/:task_id/stream`
5. SSE 断线时使用 `GET /api/v1/travel/plans/:task_id` 查询兜底

`POST /api/v1/travel/chat` 在本阶段文档中标记为兼容 / 可下线接口，不作为前端主链路。

## 需要阅读的文件

动手前必须阅读：

* `AGENTS.md`
* `requirements/README.md`
* `requirements/stage-15-frontend-productization.md`
* `requirements/stage-17-deepseek-true-streaming.md`
* `docs/api.md`
* `docs/agent-flow.md`
* `docs/architecture.md`
* `docs/evaluation-harness.md`
* `README.md`
* `internal/domain/travel.go`
* `internal/agent/planner.go`
* `internal/agent/mock_planner.go`
* `internal/agent/eino/chat.go`
* `internal/agent/eino/chat_test.go`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/prompt.go`
* `internal/travel/dto.go`
* `internal/travel/service.go`
* `internal/travel/cache.go`
* `internal/travel/service_test.go`
* `web/src/api/types.ts`
* `web/src/App.tsx`
* `web/src/components/AgentConversation.tsx`
* `web/src/components/TravelBriefPanel.tsx`
* `web/e2e/chat-agent.spec.ts`
* `testdata/travel_cases.json`

前端 UI 改造前必须使用 `$frontend-design`；前端实现后使用 `$playwright-skill` 或项目已有 Playwright harness 验证。

## 需要新增或修改的文件

预期修改：

* `internal/domain/travel.go`
* `internal/agent/planner.go`
* `internal/agent/mock_planner.go`
* `internal/agent/eino/chat.go`
* `internal/agent/eino/chat_test.go`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/prompt.go`
* `internal/travel/dto.go`
* `internal/travel/cache.go`
* `internal/travel/service.go`
* `internal/travel/service_test.go`
* `docs/api.md`
* `docs/agent-flow.md`
* `docs/evaluation-harness.md`（如新增 Harness case 或指标字段）
* `README.md`
* `web/src/api/types.ts`
* `web/src/App.tsx`
* `web/src/components/AgentConversation.tsx`
* `web/src/components/TravelBriefPanel.tsx`
* `web/src/styles.css`
* `web/e2e/chat-agent.spec.ts`
* `testdata/travel_cases.json`

按实际实现可新增：

* `internal/agent/eino/brief.go`：集中放默认值、canonical mapping 和 normalization。
* `internal/agent/eino/brief_test.go`：覆盖默认值、必填缺失、兼容映射。
* `internal/travel/cache_test.go`：覆盖新字段参与 request hash。

## 实现要求

### G1：扩展核心数据结构

* `domain.TravelRequest`、`ChatRequest`、`ChatResponse`、`CreatePlanRequest`、`TravelInfoResult` 必须携带完整 brief 字段。
* DTO 到 domain 的转换必须深拷贝 slice，避免请求对象被后续修改污染。
* `CreatePlanRequest` 必须校验 `travelers >= 1`，并继续校验目的地、天数、预算、兴趣偏好。
* 所有默认值应集中定义，避免前端、后端、Eino prompt 各自写一套。

### G2：升级需求抽取逻辑

* Eino extractor strict tool schema 必须包含新字段。
* 本地 rule fallback 必须能抽取：
  * 出行人数，如“2 人”“三个人”“亲子 3 人”。
  * 必去地点，如“必去西湖”“一定要安排灵隐寺”。
  * 避开内容，如“避开网红店”“不想太累”“不要爬山”。
  * 预算口径，如“人均 2000”“总预算 6000”。
  * 可选日期、节奏、交通、步行强度、酒店区域、同行人群。
* 只缺可选项时 `is_complete=true`，并填充默认值。
* 缺任意必填项时 `is_complete=false`，`reply` 只追问缺失必填项，不要反复要求用户补可选项。

### G3：升级确认卡体验

前端确认卡至少展示四类信息：

* 已理解内容：必填字段和用户明确给出的可选字段。
* 默认项：使用稳定默认值的可选字段，标注为默认而不是假装用户提供。
* 可继续补充项：酒店区域、必去地点、避开内容等不阻塞生成的偏好。
* 确认生成：只有必填完整时可点击。

交互要求：

* 必填缺失时，生成按钮 disabled，并展示缺失项。
* 必填完整但可选缺失时，可以生成，同时展示默认值。
* 用户继续发送补充消息后，确认卡应合并更新，不丢失已确认字段。
* 移动端不能出现按钮、字段值或标签溢出。

### G4：让 Planner 使用 richer brief

新字段必须影响最终路线，而不是只在 UI 展示：

* `travelers` 影响预算拆分、餐饮 / 门票 / 住宿估算和文案。
* `walking_tolerance` 影响每日步行强度、景点密度和交通方式。
* `must_visit` 应尽量进入路线或 warning 说明无法安排。
* `avoid` 应约束 POI 选择和文案，不安排明确避开的内容。
* `hotel_area` 影响每日起终点或住宿建议。
* `traveler_type` 影响路线强度和活动选择。
* `budget_type` 影响预算解释：总预算和人均预算不能混淆。
* `budget_includes` 影响预算 breakdown 说明。

MockPlanner 和 Eino deterministic fallback 也必须读取这些字段，确保 harness 不依赖真实 LLM。

### G5：缓存与 hash

`RequestHash` 必须包含所有会影响路线结果的 brief 字段：

* `travelers`
* `date_range`
* `transport_mode`
* `pace`
* `walking_tolerance`
* `hotel_area`
* `must_visit`
* `avoid`
* `traveler_type`
* `budget_type`
* `budget_includes`
* `test_mode`
* `agent_mode`

slice 字段应在 hash 前有稳定顺序策略。如果业务语义认为 `interests`、`must_visit`、`avoid` 的顺序不重要，应排序后 hash；如果顺序代表优先级，应在文档和测试中说明。

### G6：API 兼容

* `POST /api/v1/travel/chat/stream` 继续作为前端主入口。
* `POST /api/v1/travel/chat` 仅作为兼容 / 可下线接口，前端不应依赖它作为主链路。
* `GET /api/v1/travel/plans/:task_id` 保留为 SSE 断线后的查询兜底。
* 旧客户端如果只传 Stage 17 字段，后端应返回 `travelers` 缺失，而不是默认 1 人。
* 对旧英文枚举值做输入兼容，但最终确认卡展示不能出现内部枚举。

## 文档更新要求

实现本阶段时必须同步更新：

* `docs/api.md`：新增 request / response 字段、默认值、`POST /api/v1/travel/chat` 兼容说明、SSE fallback 说明。
* `docs/agent-flow.md`：更新需求抽取、Travel Brief 确认、Planner 使用 richer brief 的流程。
* `README.md`：更新前端主流程、测试命令、当前能力边界。
* `docs/evaluation-harness.md`：如新增 harness case 或报告字段，说明测试用例、评估指标、报告字段和运行方式。

如新增外部 API，必须更新 `docs/external-apis.md`。本阶段默认不新增外部 API。

## 测试要求

后端尽量运行：

```bash
go test ./...
go vet ./...
```

Extractor 新增 / 更新测试：

* 人数必填：缺人数时 `is_complete=false`。
* 可选项默认：只给必填项时 `is_complete=true` 且默认值稳定。
* 必去 / 避开提取：`must_visit`、`avoid` 正确合并。
* 预算口径提取：总预算 / 人均预算不混淆。
* 旧英文枚举输入兼容并输出产品可读值。

DTO / cache 测试：

* `CreatePlanRequest` 缺 `travelers` 返回 400。
* 新 brief 字段参与 request hash，避免不同 brief 命中同一缓存。
* slice 字段 hash 策略稳定。

Harness 尽量运行：

```bash
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
make harness
make harness-eino
```

Harness 至少新增 3 个 case：

* 带出行人数。
* 带避开内容。
* 只给必填项，走默认值。

前端尽量运行：

```bash
cd web
npm run typecheck
npm run lint
npm run build
npm run harness:ui
```

UI 验证：

* 必填缺失不可生成。
* 必填完整显示确认生成。
* 默认值展示为“任意 / 适中 / 无要求 / 总预算”等产品可读文案。
* SSE 断开后仍可通过任务查询兜底查看结果。

## 验收标准

* `requirements/README.md` 已把当前活跃阶段切到 Stage 18。
* `ChatResponse.done` payload 能完整驱动 Travel Brief 确认卡。
* 必填项缺失时，前后端都阻止生成。
* 可选项缺失时，系统使用稳定默认值，不阻塞生成。
* `CreatePlanRequest` 不丢失确认卡中的 richer brief 字段。
* Planner、MockPlanner 和 deterministic fallback 至少部分读取新字段并影响路线 / 预算 / warning。
* request hash 覆盖新字段。
* 文档、测试和 README 与实现一致。

