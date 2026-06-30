# Stage 21：账户、首页与旅行计划社区

## 任务目标

本阶段目标是把 Travel Agent 从匿名路线生成工具升级为可长期使用的个人旅行计划产品：用户可以注册登录、生成路线、保存计划、归档对话、在用户中心管理历史计划，并选择把计划发布到首页社区。首页不做营销落地页，而是作为登录后的产品工作台和计划发现页，承接“继续生成”“查看我的计划”“发现高质量公开计划”和“搜索目的地 / 主题”的核心路径。

本阶段同时完善 Stage 16 的“用户与业务闭环”粗粒度方向，把它拆成一份可实现、可验收的 PRD。酒店、票务、支付、商业订单仍然不进入本阶段。

核心体验：

```text
注册 / 登录
-> 进入首页
-> 查看热门 / 推荐旅行计划
-> 搜索目的地或主题
-> 发起自己的旅行计划生成
-> 生成完成后保存计划
-> 保存后对话归档
-> 用户中心查看历史记录
-> 对已保存计划执行查看、重命名、编辑、删除、发布 / 取消发布
-> 发布后的计划进入首页公开流和排行榜
```

## 当前上下文

当前系统已经具备：

* React + TypeScript H5 前端。
* 聊天式 Travel Brief 收集和确认卡。
* 异步路线生成任务。
* SSE 业务事件级路线生成过程。
* `GET /api/v1/travel/plans/:task_id` 查询任务结果。
* 可选 MySQL 持久化 `travel_tasks` / `travel_plans`。
* Redis / memory fallback。
* Eino / Mock Planner。
* 后端高可用专项中已规划结构化日志、错误追踪和更强持久化能力。

当前问题：

* 当前仍以匿名任务为核心，生成结果没有稳定用户归属。
* 用户刷新、换设备或隔天回来后，很难从产品内找到历史计划。
* 生成完成后的计划缺少“保存为我的计划”的明确动作。
* 对话和计划没有归档关系，无法回看某个计划是如何生成的。
* 计划没有标题重命名、编辑、删除、发布、取消发布等基础资产管理能力。
* 首页还不是产品化入口，缺少热门计划、搜索、推荐和“当前生成中的计划”入口。
* 已生成的优秀计划无法被其他用户发现，缺少公开发布和排行机制。

## 产品定位

Stage 21 后，产品应形成三个清晰空间：

1. **首页**：用户打开应用后的主入口。展示继续生成入口、我的最近计划、热门公开计划、搜索和排行榜。
2. **计划详情 / 生成页**：继续承接当前聊天收集、确认卡、SSE 生成和最终路线展示；生成完成后可以保存为用户计划。
3. **用户中心**：管理自己的计划、归档对话和账号信息，支持基础 CRUD、重命名、发布状态管理。

首页是产品工作台，不是营销 landing page。第一屏必须让用户能立刻继续旅行计划相关操作。

## 用户故事

### 新用户

* 作为新用户，我可以注册账号并登录，这样我的旅行计划不会丢失。
* 作为新用户，我进入首页后能看到热门旅行计划，理解这个产品能产出什么样的路线。
* 作为新用户，我可以从首页直接开始生成自己的旅行计划。

### 已登录用户

* 作为已登录用户，我可以在首页看到当前生成中或最近生成完成的计划入口。
* 作为已登录用户，我可以保存生成好的计划，并在用户中心历史记录中再次查看。
* 作为已登录用户，我可以重命名计划，让标题更符合我的使用习惯。
* 作为已登录用户，我可以删除不需要的计划，删除前应有明确确认。
* 作为已登录用户，我可以编辑已保存计划的标题、备注、可见性和部分结构化内容。
* 作为已登录用户，我可以发布计划，让其他用户在首页刷到。
* 作为已登录用户，我可以取消发布，公开页不再展示该计划。

### 浏览用户

* 作为浏览用户，我可以按热门排行看到公开计划。
* 作为浏览用户，我可以搜索目的地、天数、主题、兴趣标签和标题。
* 作为浏览用户，我可以打开公开计划详情查看路线内容。
* 作为浏览用户，我不能修改或删除别人的计划。

## 设计原则

* 登录注册要轻量、安全、可本地运行，不引入复杂身份平台作为第一阶段依赖。
* 首页优先服务高频动作：继续生成、查看我的计划、发现公开计划、搜索。
* 计划是用户资产，生成任务只是生产过程。保存后才进入用户的长期计划库。
* 对话归档跟随保存的计划，不要求独立聊天产品化。
* 发布是显式动作，默认保存的计划为私有。
* 公开计划展示要尊重隐私，不展示用户原始对话、完整 request payload 或敏感偏好。
* 排行和推荐先用可解释的轻量规则，不接复杂推荐系统。
* 不为了社区功能破坏当前 Travel Agent 核心生成链路。
* 前端 UI 要克制、清晰、移动端友好；页面不是卡片堆叠式营销页。

## 本期 MVP 边界

本阶段覆盖“可上线的基础闭环”，不追求一次做完整社区产品。推荐把交付拆成以下最小闭环：

### 必须交付

* 邮箱 / 密码注册、登录、登出和登录态恢复。
* 已登录用户生成完成后保存计划。
* 保存后的计划出现在用户中心历史记录。
* 保存计划时归档最终 Travel Brief 和基础对话记录。
* 用户可以查看、重命名、编辑基础元信息、软删除自己的计划。
* 用户可以发布、取消发布自己的计划。
* 首页展示公开计划列表、热门排行、搜索入口和当前 / 最近计划入口。
* 其他用户可以查看公开计划，并保存为自己的私有副本。
* 私有计划和对话归档有严格权限校验。

### 本期可以简化

* 首页推荐先使用热门、最新、目的地匹配，不做个性化画像。
* 搜索先使用 MySQL `LIKE` 和结构化筛选，不做全文索引和向量检索。
* 对话归档可以先保存完整 JSON，不做逐条消息表。
* 编辑计划先编辑标题、备注、标签、摘要和可见性，不做路线拖拽重排。
* 热度统计可以异步失败可忽略，不阻塞公开计划详情。
* 内容审核先保留 `removed` 状态和下架能力，不做审核后台。

### 明确延后

* 第三方登录、短信验证码、找回密码。
* 评论、点赞、关注、收藏夹分组。
* 地图编辑、路线拖拽、多人协作。
* 个性化推荐、机器学习排序、向量召回。
* 举报审核后台、运营配置后台。
* 支付、订单、酒店 / 票务真实交易链路。

## 本阶段不做什么

* 不接支付、订单、酒店预订、票务购买。
* 不接第三方 OAuth，除非后续另开阶段。
* 不做复杂社交关系：关注、私信、评论、多人协作暂不做。
* 不做复杂推荐算法、画像系统或向量搜索。
* 不做内容审核后台的完整运营系统；只预留发布状态和基础下架字段。
* 不保存明文密码。
* 不把匿名任务历史自动暴露给任意登录用户。
* 不把用户原始对话发布到公开首页。
* 不引入大型前端 UI 组件库或全局状态管理库，除非用户明确批准。客户端路由使用 `react-router-dom`。
* 不把首页做成纯介绍产品的 landing page。

## 信息架构

```text
App
  ├─ Auth
  │   ├─ 登录
  │   └─ 注册
  ├─ Home
  │   ├─ 当前生成 / 最近计划入口
  │   ├─ 开始新计划入口
  │   ├─ 搜索
  │   ├─ 热门排行
  │   └─ 推荐公开计划
  ├─ Planner
  │   ├─ 聊天收集需求
  │   ├─ Travel Brief 确认
  │   ├─ SSE 生成过程
  │   ├─ 计划结果
  │   └─ 保存计划
  ├─ Plan Detail
  │   ├─ 私有计划详情
  │   ├─ 公开计划详情
  │   ├─ 编辑 / 重命名
  │   ├─ 删除
  │   └─ 发布 / 取消发布
  └─ User Center
      ├─ 我的计划
      ├─ 对话归档
      ├─ 已发布计划
      └─ 账号信息
```

## UI 风格与体验规范

本阶段 UI 的关键词是“旅行计划工作台”：像一张清晰的路线台账，而不是泛用社区信息流或营销首页。用户应在 3 秒内知道自己可以继续哪个计划、开始哪个计划、查看哪些公开路线。

### 视觉方向

推荐使用“路线票夹 + 城市索引”的视觉语言：

* 首页结构像旅行计划索引，不做大幅 hero 营销区。
* 列表项像可扫描的路线条目，突出目的地、天数、预算状态、标签和操作。
* 热门排行可以用紧凑排名列，但不要把数字装饰化；排名要服务排序理解。
* 计划详情像时间轴和预算清单，不做过多装饰卡片。
* 用户中心像轻量资产管理台，优先可读性、状态和操作效率。

### 颜色与类型建议

第一版不引入新 UI 组件库，优先在现有 `web/src/styles.css` 中扩展设计 token。

建议 token：

```text
Ink / 主文字          #17201B
Slate / 次级文字       #5D6A64
Paper / 页面底色       #F7F8F4
Mist / 分隔底色        #E8EEE8
Route Green / 主行动   #18745A
Lake Blue / 信息状态   #2F6F9F
Sunset Coral / 警示    #D86B4A
Gold / 排行强调        #B8872B
```

要求：

* 不使用单一蓝紫渐变主题。
* 不使用大面积米黄色复古报纸风。
* 主按钮使用稳定绿色系，发布 / 保存等正向动作保持一致。
* 删除使用低饱和警示色，并且必须有二次确认。
* 预算不完整、价格缺失等状态用信息色或轻警示色表达，不用错误色吓用户。

字体建议：

* 中文正文使用系统字体栈：`Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif`。
* 数据、预算、天数使用 tabular number 或等宽数字风格。
* 标题不做过大的营销字号；首页标题应服务导航，计划详情标题应服务识别。

### 布局规范

桌面端：

```text
┌──────────────────────────────────────────┐
│ Top bar: 搜索 / 新建计划 / 用户入口      │
├───────────────┬──────────────────────────┤
│ Left rail      │ Main content             │
│ 我的计划入口  │ 当前计划 / 热门 / 搜索    │
│ 已发布入口    │                          │
└───────────────┴──────────────────────────┘
```

移动端：

```text
┌────────────────────────┐
│ Top bar: 搜索 / 用户   │
├────────────────────────┤
│ 当前计划               │
├────────────────────────┤
│ 开始规划               │
├────────────────────────┤
│ 热门排行               │
├────────────────────────┤
│ 推荐计划               │
└────────────────────────┘
```

要求：

* 移动端底部可以保留 3 个主入口：`首页`、`生成`、`我的`。
* 按钮文字不能溢出；长标题在列表中最多 2 行截断。
* 计划列表项高度稳定，加载、空状态、错误状态不能造成大幅布局跳动。
* 不使用卡片套卡片；列表项可以是卡片，页面区块不要再包浮动大卡。
* 所有可点击元素需要可见 hover / active / focus 状态。

### 交互动效

* 首页列表加载可以使用轻量 skeleton。
* 保存、发布、取消发布、删除完成后使用 toast 或 inline status，不弹过多模态。
* 删除、取消发布需要确认弹窗或确认面板。
* 搜索输入建议 300ms debounce。
* 页面切换保持直接，不做复杂动画。
* 尊重 `prefers-reduced-motion`。

## 首页产品要求

### 首页第一屏

登录后首页第一屏必须包含：

* 清晰的“开始规划”入口。
* 当前生成中任务或最近生成完成计划入口。
* 搜索框，支持搜索目的地、标题、主题和兴趣标签。
* 热门计划排行榜入口或列表。

推荐布局：

```text
┌──────────────────────────────┐
│ 顶部：品牌 / 搜索 / 用户头像 │
├──────────────────────────────┤
│ 当前计划：继续生成 / 查看结果 │
├──────────────────────────────┤
│ 开始新计划：目的地、天数、预算 │
├──────────────────────────────┤
│ 热门排行：公开计划列表         │
├──────────────────────────────┤
│ 为你推荐：目的地 / 主题计划    │
└──────────────────────────────┘
```

未登录用户可以浏览公开首页，但点击保存、生成长期计划、用户中心、发布等动作时必须引导登录。

### 首页计划卡

公开计划卡至少展示：

* 计划标题。
* 目的地。
* 天数。
* 预算摘要：优先展示“已知预算”，预算不完整时标注“部分费用暂无信息”。
* 兴趣标签。
* 发布者昵称。
* 热度信息：浏览数、保存数或发布热度分。
* 发布时间或更新时间。
* 进入详情按钮。

不得展示：

* 用户原始对话内容。
* 用户私密备注。
* API Key、request hash、内部 task id。
* 未脱敏联系方式或敏感字段。

### 排行规则

第一阶段推荐使用可解释的热度分：

```text
hot_score =
  view_count * 1
  + save_count * 5
  + copy_count * 3
  + recent_publish_bonus
  - report_penalty
```

要求：

* 排行必须只展示 `visibility=public` 且 `publish_status=published` 的计划。
* 默认按 `hot_score` 倒序。
* 同分时按 `published_at` 倒序。
* 首页接口必须支持分页。
* 需要预留按目的地、天数、兴趣标签过滤。

### 搜索规则

搜索第一阶段使用 MySQL `LIKE` / 前缀匹配即可，不要求全文检索或向量搜索。

搜索范围：

* 标题。
* 目的地。
* 出发地。
* 兴趣标签。
* 计划摘要。

搜索结果要求：

* 只返回公开计划。
* 支持分页。
* 支持排序：综合、最新、最热。
* 空结果要给出“换个目的地或主题试试”的清晰空状态。

## 核心 UI 交互逻辑

### App Shell 与导航

前端使用 `react-router-dom` 管理客户端路由。建议路由结构：

```text
/
/login
/register
/planner
/plans/:plan_id
/public/plans/:public_plan_id
/me
/me/plans
/me/published
```

交互要求：

* 刷新页面后应通过 `GET /api/v1/auth/me` 恢复用户状态。
* 用户未登录访问 `/me`、`/me/plans`、私有计划详情时，跳转登录并记录 `return_to`。
* 登录成功后回到 `return_to`；如果没有 `return_to`，回到首页。
* 顶部搜索在首页和公开计划页可用；在生成页可折叠，避免干扰当前任务。
* 移动端底部导航只保留高频入口，避免超过 4 个入口。

### 登录 / 注册交互

登录表单：

* 字段：邮箱、密码。
* 操作：登录、去注册。
* loading 时禁用提交按钮。
* 失败文案统一为“邮箱或密码不正确”，避免泄漏账号存在性。

注册表单：

* 字段：邮箱、展示昵称、密码、确认密码。
* 前端先校验邮箱格式、密码长度、两次密码一致。
* 注册成功后可以直接登录，也可以返回登录页；推荐直接建立 session 并进入首页。
* 如果用户是从“保存计划”进入注册，注册成功后应回到保存动作。

退出登录：

* 点击头像菜单中的“退出登录”。
* 成功后清空用户状态，回到公开首页。
* 如果当前在私有页面，退出后跳转首页。

### 首页交互

首页进入时并行加载：

* 当前用户信息。
* 当前生成中 / 最近计划。
* 热门公开计划。
* 推荐公开计划。

加载策略：

* `auth/me` 失败不阻塞公开计划展示。
* 首页公开计划加载失败时展示局部错误，不影响开始规划入口。
* 当前计划加载失败时只隐藏当前计划区，不影响其他区块。

“当前计划”区规则：

* 有 running task：展示目的地、状态、最近更新时间和“继续查看”。
* 无 running task 但有最近保存计划：展示最近计划和“查看计划”。
* 两者都没有：展示“开始规划你的第一条路线”。

热门排行：

* 默认展示前 5 或前 10 条。
* 点击“查看更多”进入公开计划列表。
* 点击计划卡进入公开详情。

搜索：

* 输入 `q` 后 300ms debounce。
* 回车或点击搜索按钮进入搜索结果视图。
* 搜索结果页保留筛选：目的地、天数、兴趣、排序。
* 清空搜索后回到默认热门 / 推荐视图。

### 生成完成后的保存交互

计划生成页状态：

```text
collecting_brief -> ready_to_generate -> generating -> generated -> saved
```

规则：

* `generated` 前不允许保存。
* `generated` 后显示“保存计划”主按钮。
* 未登录点击“保存计划”时打开登录 / 注册流程，并保留当前 `task_id`。
* 登录成功后自动继续保存；如果自动保存失败，保留手动重试按钮。
* 保存成功后展示“已保存”，并给出“查看我的计划”和“发布计划”入口。
* 如果该用户已经保存过同一 `task_id`，后端返回已有 `plan_id`，前端展示“已保存”。

保存表单：

* 默认标题取 `plan.title`。
* 用户可以在保存前改标题。
* 可选填写私密备注。
* 标题为空时禁用保存。

### 用户中心交互

用户中心默认进入“我的计划”。

我的计划列表：

* 支持搜索标题 / 目的地。
* 支持筛选：全部、私有、已发布、已删除不展示。
* 每个计划项提供：查看、重命名、发布 / 取消发布、删除。
* 移动端把次要操作收进更多菜单。

重命名：

* 点击标题旁编辑按钮进入 inline edit 或弹窗。
* Enter 保存，Esc 取消。
* 保存中禁用输入。
* 成功后列表和详情标题同步。

编辑：

* 第一阶段编辑基础元信息：标题、备注、标签、摘要。
* 点击“保存更改”后 PATCH。
* 如果用户离开未保存表单，提示确认。

删除：

* 点击删除后出现确认文案，明确“计划将从我的历史中移除”。
* 已发布计划删除时文案补充“公开页面也会下架”。
* 删除成功后从列表移除。

### 发布交互

发布入口可以出现在：

* 保存成功后的生成结果页。
* 私有计划详情页。
* 用户中心计划列表项。

发布面板字段：

* 公开标题，默认当前计划标题。
* 公开摘要，默认 plan summary。
* 标签，默认目的地、兴趣、天数。
* 可见性，第一阶段固定 `public`。

发布前确认：

* 明确提示“公开后其他用户可以在首页看到这份计划”。
* 明确提示“不会公开你的原始对话和私密备注”。

发布成功：

* 状态变为“已发布”。
* 显示公开详情入口。
* 操作按钮变为“取消发布”。

取消发布：

* 需要确认。
* 成功后公开详情不可访问，首页不再展示。
* 私有计划仍保留。

### 公开计划详情交互

公开详情展示：

* 标题、摘要、作者昵称、发布时间、热度信息。
* 目的地、天数、兴趣标签、预算状态。
* 完整路线详情。
* “保存到我的计划”按钮。

保存公开计划：

* 未登录时引导登录。
* 登录后创建私有副本。
* 保存成功后展示“已保存到我的计划”。
* 保存按钮不应让用户误以为修改了原作者计划。

### 错误与空状态

必须有明确空状态：

* 首页无公开计划：展示“还没有公开计划，先生成并发布一条路线”。
* 我的计划为空：展示“保存生成结果后，会出现在这里”。
* 搜索无结果：展示“没有找到相关计划，换个目的地或主题试试”。
* 对话归档为空：展示“这条计划没有可回放的生成对话”。

常见错误：

* `401`：跳转登录。
* `403`：展示“你没有权限执行此操作”。
* `404`：展示“计划不存在或已不可见”。
* `409`：展示“计划状态已变化，请刷新后重试”。
* `429`：展示“操作太频繁，请稍后再试”。

## 认证与用户系统

### 登录注册方式

第一阶段支持邮箱 / 用户名 + 密码：

* 注册：`POST /api/v1/auth/register`
* 登录：`POST /api/v1/auth/login`
* 登出：`POST /api/v1/auth/logout`
* 当前用户：`GET /api/v1/auth/me`

注册字段：

```json
{
  "email": "user@example.com",
  "password": "password",
  "display_name": "小林"
}
```

登录字段：

```json
{
  "email": "user@example.com",
  "password": "password"
}
```

安全要求：

* 密码必须哈希存储，推荐 `bcrypt`。
* 不保存明文密码。
* 登录失败返回稳定错误码，不区分邮箱不存在和密码错误。
* Session token 必须使用安全随机数生成，数据库只保存 token hash。
* Web 前端优先使用 HttpOnly Cookie 保存 session；如 H5 容器限制 Cookie，可增加 Bearer token 兼容模式，但必须说明风险。
* Cookie 建议 `HttpOnly`、`SameSite=Lax`；生产 HTTPS 下启用 `Secure`。
* Session 必须有过期时间。
* 用户只能访问自己的私有计划和对话归档。

### 匿名模式策略

匿名用户可以：

* 浏览公开首页。
* 搜索公开计划。
* 查看公开计划详情。
* 发起一次临时生成任务，具体是否允许由配置控制。

匿名用户不能：

* 保存计划。
* 查看用户中心。
* 发布计划。
* 编辑、删除、重命名计划。

如果匿名用户生成完成后点击保存：

* 前端应引导登录或注册。
* 登录成功后可以把当前 `task_id` 保存为该用户计划。
* 服务端必须校验该 task 是否存在、是否已完成、是否允许绑定到当前用户。

## 计划生命周期

### 生成任务与保存计划

当前 `travel_tasks` 代表生成任务，不应直接等同于用户长期计划。

建议新增用户计划实体：

```text
generated task -> saved user plan -> optional published public plan
```

生成完成后：

* 页面展示“保存计划”按钮。
* 保存成功后生成 `user_plan_id`。
* 对话归档与 `user_plan_id` 关联。
* 后续用户中心以 `user_plan_id` 为核心展示，不直接暴露 `task_id`。

保存接口：

```text
POST /api/v1/me/plans
```

请求：

```json
{
  "task_id": "task_x",
  "title": "杭州 3 日美食与西湖路线",
  "note": "适合周末出行"
}
```

响应：

```json
{
  "plan_id": "plan_x",
  "task_id": "task_x",
  "title": "杭州 3 日美食与西湖路线",
  "visibility": "private",
  "created_at": "2026-06-29T12:00:00Z"
}
```

### 私有计划状态

建议计划状态：

```text
draft
saved
archived
deleted
```

建议可见性：

```text
private
public
unlisted
```

第一阶段必须支持：

* `private`
* `public`

`unlisted` 可预留，不要求前端入口。

### 对话归档

保存计划后，需要归档生成过程中的对话和关键业务事件。

归档内容建议包括：

* 用户消息。
* 助手回复。
* Travel Brief 确认卡最终状态。
* 生成任务 id。
* 最终 plan id。
* 创建时间。

归档不发布到公开首页。公开计划详情只展示最终路线和用户选择公开的标题 / 摘要 / 标签。

## 我的计划 CRUD

用户中心必须支持：

* 查看我的计划列表。
* 查看计划详情。
* 重命名计划。
* 编辑计划备注、标签、可见性。
* 删除计划。
* 发布计划。
* 取消发布计划。

### 我的计划列表

接口：

```text
GET /api/v1/me/plans?page=1&page_size=20&status=saved&visibility=private
```

列表项字段：

```json
{
  "plan_id": "plan_x",
  "title": "杭州 3 日美食与西湖路线",
  "destination_city": "杭州",
  "days": 3,
  "cover_text": "西湖、灵隐寺、河坊街",
  "visibility": "private",
  "publish_status": "draft",
  "updated_at": "2026-06-29T12:00:00Z"
}
```

### 查看计划

接口：

```text
GET /api/v1/me/plans/:plan_id
```

要求：

* 只能查看自己的私有计划。
* 返回完整 `TravelPlan`。
* 返回归档对话入口信息，不一定默认返回完整对话，避免首屏过大。

### 重命名

接口：

```text
PATCH /api/v1/me/plans/:plan_id
```

请求：

```json
{
  "title": "杭州亲子 3 日轻松路线"
}
```

要求：

* 标题非空。
* 标题长度有限制，建议 2-80 个字符。
* 修改后更新 `updated_at`。
* 如果计划已发布，公开列表中的标题同步更新。

### 编辑计划

第一阶段编辑范围应保守，避免把结构化路线编辑做成复杂编辑器。

允许编辑：

* 标题。
* 备注。
* 标签。
* 可见性。
* 封面摘要或展示摘要。

可选编辑：

* 单日标题。
* 单个行程项备注。

暂不要求：

* 拖拽重排路线。
* 地图选点。
* 重新计算某段路线。
* 多人协作编辑。

如果编辑结构化计划内容，必须保留版本字段或 `updated_at`，避免并发覆盖。

### 删除

接口：

```text
DELETE /api/v1/me/plans/:plan_id
```

要求：

* 默认软删除，设置 `deleted_at`。
* 删除自己的计划后，公开首页不再展示。
* 删除操作需要前端二次确认。
* 不删除底层 `travel_tasks`，除非后续有数据保留策略。

## 发布计划

### 发布接口

```text
POST /api/v1/me/plans/:plan_id/publish
```

请求：

```json
{
  "title": "杭州 3 日美食与西湖路线",
  "summary": "适合第一次去杭州的轻松路线。",
  "tags": ["杭州", "美食", "自然风光", "3日"],
  "visibility": "public"
}
```

要求：

* 只能发布自己的计划。
* 计划必须已保存且未删除。
* 发布前必须有最终 `TravelPlan`。
* 发布后 `publish_status=published`，`visibility=public`，写入 `published_at`。
* 公开计划不能包含归档对话和私密备注。
* 发布内容需要保留 `source_plan_id`，方便用户后续取消发布或同步更新。

### 取消发布

接口：

```text
POST /api/v1/me/plans/:plan_id/unpublish
```

要求：

* 取消发布后公开首页和搜索不再返回该计划。
* 私有计划仍保留在用户中心。
* 历史浏览链接再次访问应返回 404 或不可见状态。

### 公开计划详情

接口：

```text
GET /api/v1/public/plans/:public_plan_id
```

要求：

* 只返回已发布计划。
* 返回公开字段和完整路线。
* 增加浏览计数时要考虑简单防刷策略，例如同 IP / session 短时间去重。
* 不返回用户私密备注、原始对话、内部 request hash。

## 首页公开计划 API

### 热门排行

```text
GET /api/v1/public/plans?sort=hot&page=1&page_size=20
```

可选参数：

```text
q
destination_city
days
interest
sort=hot|latest
```

响应：

```json
{
  "items": [
    {
      "public_plan_id": "pub_x",
      "title": "杭州 3 日美食与西湖路线",
      "summary": "适合第一次去杭州的轻松路线。",
      "destination_city": "杭州",
      "days": 3,
      "tags": ["杭州", "美食", "自然风光"],
      "author": {
        "display_name": "小林"
      },
      "hot_score": 128,
      "view_count": 90,
      "save_count": 6,
      "published_at": "2026-06-29T12:00:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 120
}
```

### 保存公开计划到我的计划

用户可以把别人发布的公开计划保存为自己的副本。

接口：

```text
POST /api/v1/public/plans/:public_plan_id/save
```

要求：

* 必须登录。
* 创建当前用户自己的私有计划副本。
* 记录 `source_public_plan_id`。
* 增加公开计划 `save_count`。
* 不复制发布者的私密备注和对话归档。

## 产品指标与埋点

本期至少需要定义可观测的产品指标，先通过服务端结构化日志或 MySQL 事件表记录，不要求接入第三方埋点平台。

### 核心漏斗

```text
首页访问
-> 开始规划
-> brief 完成
-> 生成完成
-> 保存计划
-> 发布计划
-> 被其他用户查看
-> 被其他用户保存
```

指标：

* 注册转化率。
* 登录成功率。
* 计划生成完成率。
* 生成后保存率。
* 保存后发布率。
* 公开计划点击率。
* 公开计划保存率。
* 搜索无结果率。

### 事件建议

```text
auth.registered
auth.login_succeeded
auth.login_failed
home.viewed
home.search_submitted
plan.generated
plan.saved
plan.renamed
plan.deleted
plan.published
plan.unpublished
public_plan.viewed
public_plan.saved
```

事件字段：

* `request_id`
* `user_id`，匿名时为空
* `plan_id`
* `public_plan_id`
* `task_id`
* `destination_city`
* `days`
* `source`
* `created_at`

要求：

* 不记录密码、session token、Cookie、Authorization。
* 不记录用户原始对话全文作为埋点字段。
* 搜索词可以记录，但需要限制长度并去除明显敏感内容。
* 产品指标记录失败不能影响主流程。

## 数据模型要求

建议新增表：

### `users`

| 字段 | 说明 |
| --- | --- |
| `id` | 用户 id，建议 `user_` 前缀或 UUID |
| `email` | 邮箱，唯一 |
| `display_name` | 展示昵称 |
| `password_hash` | 密码哈希 |
| `status` | `active`、`disabled` |
| `created_at` / `updated_at` | UTC 时间 |

### `user_sessions`

| 字段 | 说明 |
| --- | --- |
| `id` | session id |
| `user_id` | 用户 id |
| `token_hash` | session token hash |
| `expires_at` | 过期时间 |
| `created_at` | 创建时间 |
| `revoked_at` | 登出或吊销时间 |

索引：

* `token_hash`
* `user_id + expires_at`

### `user_plans`

| 字段 | 说明 |
| --- | --- |
| `id` | 用户计划 id |
| `user_id` | 所属用户 |
| `task_id` | 来源生成任务，可为空 |
| `source_public_plan_id` | 来源公开计划，可为空 |
| `title` | 计划标题 |
| `note` | 私密备注 |
| `tags_json` | 标签数组 |
| `plan_json` | 保存时的完整 TravelPlan 快照 |
| `destination_city` | 冗余字段，用于列表和搜索 |
| `days` | 冗余字段，用于列表和过滤 |
| `visibility` | `private` / `public` |
| `publish_status` | `draft` / `published` / `unpublished` |
| `created_at` / `updated_at` / `deleted_at` | 时间 |

索引：

* `user_id + updated_at`
* `user_id + deleted_at`
* `destination_city`
* `visibility + publish_status + updated_at`

### `plan_conversation_archives`

| 字段 | 说明 |
| --- | --- |
| `id` | 归档 id |
| `plan_id` | 用户计划 id |
| `user_id` | 用户 id |
| `task_id` | 生成任务 id |
| `brief_json` | 最终 Travel Brief |
| `messages_json` | 用户与助手对话 |
| `events_json` | 可选，关键生成事件摘要 |
| `created_at` | 归档时间 |

### `public_plans`

| 字段 | 说明 |
| --- | --- |
| `id` | 公开计划 id |
| `plan_id` | 来源用户计划 id |
| `user_id` | 发布者 |
| `title` | 公开标题 |
| `summary` | 公开摘要 |
| `tags_json` | 公开标签 |
| `plan_json` | 公开计划快照 |
| `destination_city` | 目的地 |
| `days` | 天数 |
| `status` | `published` / `unpublished` / `removed` |
| `view_count` | 浏览数 |
| `save_count` | 保存数 |
| `copy_count` | 复制数，预留 |
| `hot_score` | 热度分 |
| `published_at` / `updated_at` | 时间 |

索引：

* `status + hot_score + published_at`
* `status + published_at`
* `destination_city + status`
* `user_id + published_at`

### `public_plan_events`

可选，用于统计热度事件：

| 字段 | 说明 |
| --- | --- |
| `id` | 事件 id |
| `public_plan_id` | 公开计划 id |
| `user_id` | 可为空 |
| `event_type` | `view` / `save` / `copy` |
| `client_hash` | 简单去重字段 |
| `created_at` | 时间 |

## API 设计

### Auth

```text
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout
GET  /api/v1/auth/me
```

### 我的计划

```text
POST   /api/v1/me/plans
GET    /api/v1/me/plans
GET    /api/v1/me/plans/:plan_id
PATCH  /api/v1/me/plans/:plan_id
DELETE /api/v1/me/plans/:plan_id
GET    /api/v1/me/plans/:plan_id/conversation
POST   /api/v1/me/plans/:plan_id/publish
POST   /api/v1/me/plans/:plan_id/unpublish
```

### 公开计划

```text
GET  /api/v1/public/plans
GET  /api/v1/public/plans/:public_plan_id
POST /api/v1/public/plans/:public_plan_id/save
```

### 当前计划入口

首页需要展示当前生成中或最近生成完成的计划。可以通过已有任务接口组合，也可以新增轻量接口：

```text
GET /api/v1/me/current
```

响应建议：

```json
{
  "running_task": {
    "task_id": "task_x",
    "status": "running",
    "destination_city": "杭州",
    "updated_at": "2026-06-29T12:00:00Z"
  },
  "latest_plan": {
    "plan_id": "plan_x",
    "title": "杭州 3 日路线",
    "updated_at": "2026-06-29T12:00:00Z"
  }
}
```

## 本期实现技术方案

### 总体技术路线

本阶段继续采用当前单仓库、模块化单体架构，不拆微服务：

```text
Gin HTTP API
  -> auth middleware
  -> auth / plans / travel handlers
  -> service layer
  -> repository layer
  -> MySQL authoritative storage
  -> Redis optional cache / rate limit
  -> React H5 frontend
```

选择原因：

* 当前项目已有 Gin、MySQL migration、Redis fallback 和 React H5。
* 用户计划、发布和首页搜索都需要强一致的权限边界，优先放在同一后端内实现。
* 本期目标是打通产品闭环，不引入 OAuth、搜索引擎、推荐系统等额外运维组件。

### 后端技术方案

推荐新增包：

```text
internal/auth
internal/plans
```

职责：

* `internal/auth`：用户、密码哈希、session、认证 middleware、auth handler。
* `internal/plans`：用户计划、对话归档、公开计划、搜索排行、发布状态。
* `internal/travel`：继续负责生成任务和 SSE，不直接承担用户计划资产管理。

认证方案：

* 本期使用 opaque session token，不使用 JWT。
* session token 使用安全随机数生成。
* 数据库只保存 `token_hash`。
* 前端优先使用 HttpOnly Cookie。
* `GET /api/v1/auth/me` 用于刷新后恢复用户状态。

密码方案：

* 使用 `golang.org/x/crypto/bcrypt`。
* `password_hash` 不返回给任何 API。
* 登录失败统一返回 `invalid_credentials`。

数据库方案：

* MySQL 是用户、session、用户计划和公开计划的权威存储。
* Redis 只用于限流、热点公开计划缓存和浏览计数去重，Redis 不可用不应造成计划数据丢失。
* 迁移文件建议新增 `migrations/mysql/003_users_and_plan_library.sql`。
* 所有 JSON 快照字段必须控制大小，不保存无上限原始响应。

事务边界：

* 保存计划：读取已完成 task、写 `user_plans`、写归档，应在同一事务或具备补偿策略。
* 发布计划：更新 `user_plans` 发布状态、upsert `public_plans`，应保持一致。
* 删除计划：软删除 `user_plans`，同时下架 `public_plans`。
* 保存公开计划副本：写 `user_plans` 副本、增加 `save_count`，计数失败不影响副本保存。

### 前端技术方案

本期不新增大型 UI 组件库和全局状态管理库；客户端路由使用 `react-router-dom`。

推荐实现方式：

* 继续使用 React + TypeScript + Vite。
* 使用现有 `web/src/api/client.ts` 扩展 auth、plans、public plans API。
* 使用轻量 `useAuth` hook 管理用户状态。
* 使用 `react-router-dom`（v6）管理页面切换；保持路由扁平，仅在确有需要时引入嵌套路由或加载器。
* 使用现有 CSS 文件扩展 token、布局、组件样式。
* 表单状态使用 React local state，避免引入表单库。
* 列表查询使用简单 hook，支持 loading、error、empty、pagination。

前端数据流：

```text
App mount
  -> useAuth calls /auth/me
  -> Home loads current + public plans
  -> Planner generates task through existing travel APIs
  -> generated plan calls /me/plans save
  -> UserCenter lists /me/plans
  -> Publish calls /me/plans/:id/publish
  -> Home public list calls /public/plans
```

### API 兼容策略

* 保留现有匿名生成接口。
* 登录态不改变 `POST /api/v1/travel/plans` 的基本契约。
* 保存计划通过新增 `/api/v1/me/plans` 完成，不强行改造生成接口。
* 公开计划 API 不复用私有计划 API，避免权限分支复杂。
* 如果未来需要“生成即自动保存”，应作为配置或后续阶段，不作为本期默认行为。

### 安全与隐私技术方案

* 所有私有接口必须通过 auth middleware 获取 `user_id`。
* repository 查询必须带 `user_id` 条件，不只在 service 层判断。
* 公开计划使用独立 `public_plan_id`，不暴露私有 `plan_id`。
* session cookie 配置来自环境变量。
* CORS 如果允许携带 cookie，必须明确 allowed origin，不能继续生产环境 `*`。
* 日志中不记录密码、session token、Cookie、Authorization。

### 本期不引入的技术

* 不引入 Elasticsearch / Meilisearch / Typesense。
* 不引入 Redis Streams 作为发布计划主链路。
* 不引入 OAuth provider。
* 不引入 JWT refresh token 体系。
* 不引入大型前端 UI 组件库。
* 不引入复杂推荐模型或向量数据库。

## 后端实现要求

### G1：用户与 Session

* 新增 `internal/auth` 包，封装注册、登录、登出、当前用户查询。
* 新增认证 middleware，把当前用户注入 `context.Context`。
* 密码哈希使用成熟算法，不能自写 hash。
* Session token 只在创建时返回明文，数据库保存 hash。
* 所有需要登录的接口必须统一校验。
* 所有用户输入字段必须 trim 和长度校验。

### G2：计划归属

* `user_plans` 是用户中心的权威计划资产。
* 保存计划时必须校验 `task_id` 已完成且有最终 plan。
* 同一个用户对同一个 task 重复保存时，应复用或返回已有 `plan_id`，避免重复资产。
* 私有计划查询必须带 `user_id` 条件。
* 删除、编辑、发布都必须校验计划归属。

### G3：对话归档

* 保存计划时归档最终 Travel Brief 和对话消息。
* 如果当前前端还没有完整消息历史结构，至少归档最终 brief、task id、plan id 和生成事件摘要。
* 对话归档默认只对本人可见。
* 公开计划不得包含归档对话。

### G4：发布与公开计划

* 发布时写入 `public_plans` 快照，避免用户后续私密编辑影响已发布内容边界。
* 已发布计划重命名时，可以同步公开标题，但要保持操作可控。
* 取消发布必须从公开列表和搜索中移除。
* 公开计划详情只允许访问 `status=published`。
* 公开接口不得泄漏 `user_id` 以外的内部字段，作者信息只返回 display name。

### G5：首页排行与搜索

* 首页公开计划接口必须分页。
* 排行只基于公开计划。
* 搜索必须限制最大 `page_size`，避免全表扫描放大。
* 第一阶段可以使用 SQL `LIKE`，但字段和索引要为后续全文检索预留空间。
* 浏览数和保存数更新失败不应影响计划详情主流程。

### G6：权限与错误

* 未登录访问私有接口返回 `401 unauthorized`。
* 访问他人私有计划返回 `404 not_found`，避免暴露资源存在性。
* 无权发布 / 删除返回稳定错误码。
* 删除已发布计划时必须同时取消公开展示。
* 所有 auth / plan / publish 操作应记录结构化日志字段：`request_id`、`user_id`、`plan_id`、`public_plan_id`、`operation`。

## 前端实现要求

前端 UI 改造前必须使用 `$frontend-design`；实现后必须使用 `$playwright-skill` 或项目已有 Playwright harness 验证。

### G1：首页

首页必须是实际产品入口，包含：

* 顶部导航：搜索、用户入口。
* 当前计划入口：生成中 / 最近保存。
* 新建计划入口。
* 热门排行。
* 推荐公开计划列表。

视觉方向：

* 使用旅行计划产品的“路线、时间、地点、预算”作为视觉结构，而不是泛用 SaaS 卡片堆叠。
* 计划列表应适合快速扫描，优先展示目的地、天数、预算状态和标签。
* 移动端第一屏不能被大标题和说明文案占满，必须保留可操作入口。

### G2：登录注册

* 提供登录 / 注册表单。
* 表单错误必须明确，例如“邮箱格式不正确”“密码至少 8 位”。
* 登录后返回原先想执行的动作，例如保存计划或进入用户中心。
* 登出后清理本地用户状态。

### G3：生成完成后的保存

在计划生成详情页：

* 生成完成前不展示保存按钮，或展示 disabled 状态。
* 生成完成后展示“保存计划”。
* 未登录点击保存时进入登录引导，登录后继续保存。
* 保存成功后按钮状态变为“已保存”，并提供“查看我的计划”入口。
* 保存后把对话归档关联到计划。

### G4：用户中心

用户中心至少包含：

* 我的计划。
* 已发布。
* 对话归档入口。
* 账号信息。

我的计划列表必须支持：

* 查看。
* 重命名。
* 删除。
* 发布 / 取消发布。
* 搜索或筛选，第一阶段可只做本地输入触发接口搜索。

### G5：计划详情与编辑

私有计划详情：

* 展示完整路线。
* 展示标题、标签、备注。
* 提供重命名和编辑入口。
* 提供发布入口。
* 提供删除入口。

公开计划详情：

* 展示公开标题、摘要、作者、标签、路线。
* 提供“保存到我的计划”。
* 不展示编辑、删除、私密备注和对话归档。

### G6：状态设计

必须覆盖：

* 未登录。
* 登录中。
* 注册失败。
* 首页无公开计划。
* 搜索无结果。
* 我的计划为空。
* 保存成功 / 失败。
* 发布成功 / 失败。
* 删除确认。
* 网络错误。

## 需要阅读的文件

动手前必须阅读：

* `AGENTS.md`
* `requirements/README.md`
* `requirements/stage-15-frontend-productization.md`
* `requirements/stage-16-user-business-loop.md`
* `requirements/stage-18-travel-brief-confirmation.md`
* `requirements/stage-20-business-event-streaming.md`
* `requirements/backend-performance-optimization.md`
* `docs/api.md`
* `docs/database.md`
* `docs/architecture.md`
* `docs/frontend-skills.md`
* `README.md`
* `cmd/server/main.go`
* `internal/config/config.go`
* `internal/server/router.go`
* `internal/server/middleware.go`
* `internal/travel/handler.go`
* `internal/travel/service.go`
* `internal/travel/task_store.go`
* `internal/travel/mysql_task_store.go`
* `web/src/App.tsx`
* `web/src/api/client.ts`
* `web/src/api/types.ts`
* `web/src/components/AgentConversation.tsx`
* `web/src/components/PlanDetail.tsx`
* `web/src/components/TravelBriefPanel.tsx`
* `web/src/hooks/useTravelPlanStream.ts`
* `web/src/styles.css`
* `migrations/mysql/*`

## 需要新增或修改的文件

预期修改：

* `internal/config/config.go`
* `internal/server/router.go`
* `internal/server/middleware.go`
* `internal/travel/handler.go`
* `internal/travel/service.go`
* `internal/travel/task.go`
* `internal/travel/mysql_task_store.go`
* `web/src/App.tsx`
* `web/src/api/client.ts`
* `web/src/api/types.ts`
* `web/src/components/AgentConversation.tsx`
* `web/src/components/PlanDetail.tsx`
* `web/src/styles.css`
* `docs/api.md`
* `docs/database.md`
* `docs/architecture.md`
* `README.md`
* `.env.example`

按实际实现可新增：

* `internal/auth/user.go`
* `internal/auth/session.go`
* `internal/auth/password.go`
* `internal/auth/service.go`
* `internal/auth/handler.go`
* `internal/auth/middleware.go`
* `internal/auth/*_test.go`
* `internal/plans/plan.go`
* `internal/plans/service.go`
* `internal/plans/store.go`
* `internal/plans/mysql_store.go`
* `internal/plans/handler.go`
* `internal/plans/search.go`
* `internal/plans/*_test.go`
* `internal/analytics/events.go`
* `internal/analytics/store.go`
* `migrations/mysql/003_users_and_plan_library.sql`
* `web/src/components/AppShell.tsx`
* `web/src/components/AuthView.tsx`
* `web/src/components/HomeView.tsx`
* `web/src/components/UserCenter.tsx`
* `web/src/components/PlanLibrary.tsx`
* `web/src/components/PublicPlanList.tsx`
* `web/src/components/PublicPlanDetail.tsx`
* `web/src/components/PlanEditor.tsx`
* `web/src/components/ConfirmDialog.tsx`
* `web/src/components/Toast.tsx`
* `web/src/hooks/useAuth.ts`
* `web/src/hooks/usePlanLibrary.ts`
* `web/src/hooks/usePublicPlans.ts`
* `web/src/hooks/useDebouncedValue.ts`
* `web/e2e/auth-plan-library.spec.ts`

## 配置要求

新增配置必须来自环境变量或配置文件：

```text
TRAVEL_AGENT_AUTH_ENABLED=true
TRAVEL_AGENT_SESSION_COOKIE_NAME=travel_agent_session
TRAVEL_AGENT_SESSION_TTL_HOURS=168
TRAVEL_AGENT_PASSWORD_MIN_LENGTH=8
TRAVEL_AGENT_PUBLIC_PLAN_PAGE_SIZE=20
TRAVEL_AGENT_ALLOW_ANONYMOUS_PLAN_GENERATION=true
```

要求：

* 本地开发应有合理默认值。
* 生产环境如果启用 auth，必须配置 session secret 或等价安全随机材料。
* `.env.example` 必须同步说明。

## 文档更新要求

实现本阶段时必须同步更新：

* `docs/api.md`：新增 auth、我的计划、公开计划、发布、搜索接口。
* `docs/database.md`：新增用户、session、用户计划、对话归档、公开计划表。
* `docs/architecture.md`：说明匿名任务、用户计划、公开计划之间的关系。
* `docs/frontend-skills.md`：如首页和用户中心形成新的前端工作流，需要说明验证方式。
* `README.md`：补充登录注册、环境变量、本地运行和测试方式。

如新增外部身份 provider，必须更新 `docs/external-apis.md`。本阶段默认不新增。

## 测试要求

后端尽量运行：

```bash
go test ./...
go vet ./...
```

Auth 测试：

* 注册成功。
* 重复邮箱注册失败。
* 登录成功。
* 错误密码登录失败且不泄漏用户是否存在。
* 登出后 session 失效。
* 过期 session 不可访问私有接口。

计划库测试：

* 已登录用户可以保存已完成 task。
* 未完成 task 不能保存。
* 同一用户重复保存同一 task 不产生重复计划。
* 用户不能查看、编辑、删除他人的私有计划。
* 重命名更新标题和 `updated_at`。
* 删除后列表不返回。

发布与公开计划测试：

* 用户可以发布自己的计划。
* 用户不能发布他人的计划。
* 取消发布后公开接口不返回。
* 首页只返回公开已发布计划。
* 搜索只返回公开已发布计划。
* 保存公开计划会创建私有副本并增加保存数。

前端尽量运行：

```bash
cd web
npm run typecheck
npm run lint
npm run build
npm run harness:ui
```

Playwright 应覆盖：

* 注册 / 登录 / 登出。
* 未登录保存计划时跳转登录，登录后继续保存。
* 首页热门计划和搜索。
* 生成完成后保存计划。
* 用户中心重命名、删除、发布、取消发布。
* 移动端首页、计划列表、详情页文本不溢出、不重叠。

## 验收标准

本阶段完成后必须满足：

* 用户可以注册、登录、登出，并通过 `GET /api/v1/auth/me` 恢复登录态。
* 首页是可操作产品入口，包含开始规划、当前计划、搜索、热门排行和推荐公开计划。
* 生成完成的计划可以保存到当前用户。
* 保存后可以在用户中心历史记录中查看。
* 保存后对话归档可以从对应计划进入查看。
* 用户可以对自己的计划执行查看、重命名、基础编辑、删除。
* 用户可以发布和取消发布自己的计划。
* 发布后的计划可以在首页公开列表、排行和搜索中出现。
* 其他用户可以查看公开计划，并保存为自己的私有副本。
* 用户不能访问、编辑、删除他人的私有计划。
* 公开计划不泄漏原始对话、私密备注、request hash 或内部 task id。
* 登录注册、计划管理和公开浏览的 API 文档、数据库文档、架构文档同步更新。
* 后端测试和前端类型 / lint / build 尽量通过；如因外部环境无法运行，必须说明原因。

## 后续优化路线

Stage 21 完成后，后续优化应按风险和收益拆分，不建议在本期一次性做完。

### 21.1：账号安全与登录体验

目标：提高账号体系可用性和安全性。

可优化能力：

* 邮箱验证。
* 找回密码。
* 修改密码。
* 登录设备管理。
* Session 列表和远程登出。
* 登录失败次数限制和验证码。
* 第三方 OAuth，例如微信、GitHub、Google。

前置条件：

* 基础 session 表稳定。
* auth middleware 和权限测试完善。

### 21.2：计划编辑器增强

目标：让保存后的计划可持续维护，而不是只能重命名。

可优化能力：

* 单日路线标题编辑。
* 行程项备注编辑。
* 行程项顺序调整。
* 删除 / 新增自定义行程项。
* 基于修改后的路线重新计算预算和可行性。
* 计划版本历史和回滚。

风险：

* 结构化路线编辑会影响预算、路线可行性和公开计划一致性。
* 需要明确“用户编辑内容”和“Agent 生成内容”的边界。

### 21.3：搜索与发现升级

目标：提升首页发现效率。

可优化能力：

* MySQL FULLTEXT 或外部搜索服务。
* 按目的地、天数、预算、出行人群、季节筛选。
* 搜索高亮。
* 热门目的地聚合页。
* 标签页，例如亲子、情侣、周末、低预算。
* 搜索历史和热门搜索。

技术演进：

```text
LIKE 查询
-> MySQL FULLTEXT
-> 独立搜索服务
-> 向量召回 + 结构化过滤
```

### 21.4：推荐系统增强

目标：从“热门公开计划”升级为“更适合当前用户的路线推荐”。

可优化能力：

* 根据用户保存、浏览、搜索行为推荐。
* 根据用户常选目的地、预算、同行人群推荐。
* 冷启动使用热门目的地和高质量公开计划。
* 推荐原因展示，例如“与你收藏的杭州路线相似”。

约束：

* 推荐必须可解释。
* 用户行为数据要有隐私说明和清理策略。
* 不应影响用户主动搜索结果的确定性。

### 21.5：社区互动

目标：让公开计划有轻量社区反馈。

可优化能力：

* 收藏公开计划。
* 点赞。
* 评论。
* 举报。
* 作者主页。
* 公开计划合集。

风险：

* 评论和举报会引入内容审核需求。
* 作者主页会引入更多隐私边界。
* 需要防刷和反垃圾策略。

### 21.6：内容审核与运营后台

目标：保证公开首页内容质量。

可优化能力：

* 计划发布审核状态：`pending_review`、`approved`、`rejected`。
* 自动敏感词检测。
* 人工下架。
* 举报处理。
* 首页推荐位运营配置。
* 热门排行权重配置。

本期只预留 `removed` / `unpublished` 状态，不实现完整后台。

### 21.7：性能与高可用优化

目标：支撑更大公开计划流量。

可优化能力：

* 首页公开计划 Redis 缓存。
* hot_score 定时批量计算。
* 浏览数异步聚合。
* 公开计划分页游标。
* 用户计划列表索引优化。
* 对话归档冷热分层。
* 大计划 JSON 压缩或拆表。

需要与 `backend-performance-optimization.md` 中的 MQ、Worker、日志和容量指标协同推进。

### 21.8：分享与外部传播

目标：让公开计划可以优雅分享。

可优化能力：

* 公开分享链接。
* 分享海报。
* 导出 Markdown / PDF。
* 只读 unlisted 链接。
* 分享链接过期和撤销。

约束：

* 分享内容不能包含私密备注和归档对话。
* 导出预算必须保留“已知预算 / 暂无信息”的语义。

## 推荐实施顺序

1. 新增用户、session、user_plans、public_plans、conversation archives 迁移。
2. 实现 `internal/auth`：密码哈希、session、middleware、auth handler。
3. 实现用户计划库 store / service / handler。
4. 打通“生成完成后保存计划”和对话归档。
5. 实现发布 / 取消发布和公开计划查询。
6. 实现首页公开计划排行和搜索。
7. 实现前端登录注册、首页、保存按钮、用户中心、计划 CRUD。
8. 实现公开计划详情和保存公开计划副本。
9. 补充测试、文档和 README。
