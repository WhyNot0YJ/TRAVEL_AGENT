# Stage 19: 真实价格数据与可信预算改造

## 任务目标

本阶段目标是把路线规划结果中的价格、预算和来源说明从“本地估算为主”改造成“真实数据优先、缺失明确展示”的产品能力。

核心原则：

* 能从高德或其他已接入真实数据源获取的信息，必须使用真实数据。
* 不能获取真实数据的信息，不允许编造、默认估算或用固定比例补齐。
* 缺失的信息在接口和页面上都必须明确展示为“暂无信息”。
* 缺失金额不得计入完整预算或总价。
* 不新增 `RealBudgetTool`；直接改造当前已有的 `BudgetTool` 实现和调用链。

## 当前上下文

当前 Eino Graph 的路线生成链路为：

```text
parse_request
-> search_pois
-> get_weather
-> compute_route
-> estimate_budget
-> optimize_itinerary
-> validate_route_feasibility
-> generate_plan
-> validate_plan
```

当前存在的问题：

* `RealPOITool` 调用高德 POI 搜索时使用 `extensions=base`，只解析 `name`、`type`、`address`、`location`。
* 高德 POI `extensions=all` 可返回更多真实字段，例如 `id`、`typecode`、`tel`、`business_area`、`tag`、`photos`、`biz_ext.rating`、`biz_ext.cost`，当前没有接入。
* `RealRouteTool` 当前只解析路线 `distance` 和 `duration`，没有解析交通费用。
* `MockBudgetTool` 当前按固定比例拆分用户预算，生成 `transport`、`food`、`hotel`、`ticket`、`total`，这些不是实时价格。
* `domain.TravelItem.EstimatedCost` 和 `domain.TravelBudget` 都是裸数字，无法区分“0 元”“免费”“未知”“估算值”。
* 前端预算面板当前直接展示数字，无法展示“暂无信息”，也无法说明预算是否完整。

## 本阶段不做什么

* 不新增 `RealBudgetTool` 类型。
* 不接入酒店实时房价 API。
* 不接入 12306、航司、OTA 或票务平台。
* 不把高铁、机票、酒店价格用固定默认值补齐。
* 不用 LLM 猜测、补全或改写价格。
* 不把缺失价格计入 `known_total` 或完整预算。
* 不破坏现有 SSE、任务状态、缓存、harness 和 mock/test mode 基本可运行能力。

## 需要阅读的文件

后端：

* `internal/domain/travel.go`
* `internal/agent/eino/types.go`
* `internal/agent/eino/real_poi_tool.go`
* `internal/agent/eino/real_route_tool.go`
* `internal/agent/eino/tools.go`
* `internal/agent/eino/tool_mode.go`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/schema.go`
* `internal/agent/eino/prompt.go`
* `internal/agent/eino/json_parser.go`
* `internal/travel/dto.go`
* `internal/travel/service.go`

前端：

* `web/src/api/types.ts`
* `web/src/components/PlanDetail.tsx`
* `web/src/styles.css`

文档：

* `docs/api.md`
* `docs/agent-flow.md`
* `docs/external-apis.md`
* `README.md`

## 需要新增或修改的文件

预计修改：

* `internal/domain/travel.go`
* `internal/agent/eino/types.go`
* `internal/agent/eino/real_poi_tool.go`
* `internal/agent/eino/real_route_tool.go`
* `internal/agent/eino/tools.go`
* `internal/agent/eino/tool_mode.go`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/schema.go`
* `internal/agent/eino/prompt.go`
* `internal/agent/eino/json_parser.go`
* `internal/agent/eino/*_test.go`
* `web/src/api/types.ts`
* `web/src/components/PlanDetail.tsx`
* `web/src/styles.css`
* `docs/api.md`
* `docs/agent-flow.md`
* `docs/external-apis.md`
* `README.md`

禁止新增：

* `RealBudgetTool`
* 独立的真实预算工具实现类型

允许改造：

* `MockBudgetTool` 的名称可以保留，但其 `Run` 逻辑必须从“固定比例估算”改为“真实数据汇总 + 缺失项标记”。
* 如果后续重命名 `MockBudgetTool`，必须同步更新所有调用点和测试；但本阶段推荐最小改动，优先保留类型名。

## 数据模型要求

### 价格字段

新增统一价格结构，建议命名为 `PriceInfo` 或 `CostInfo`：

```go
type CostStatus string

const (
	CostAvailable     CostStatus = "available"
	CostUnavailable   CostStatus = "unavailable"
	CostNotApplicable CostStatus = "not_applicable"
)

type CostInfo struct {
	Amount   *float64   `json:"amount"`
	Currency string     `json:"currency"`
	Unit     string     `json:"unit"`
	Status   CostStatus `json:"status"`
	Source   string     `json:"source,omitempty"`
	Display  string     `json:"display,omitempty"`
	Included bool       `json:"included"`
}
```

规则：

* `status=available` 时，`amount` 必须非空。
* `status=unavailable` 时，`amount=null`，`display=暂无信息`，`included=false`。
* `status=not_applicable` 用于步行、骑行等天然无费用项目，`amount=0` 或 `null` 均可，但不得误导为接口报价。
* `source` 必须说明来源，例如 `amap.poi.biz_ext.cost`、`amap.route.taxi_cost`、`amap.route.transits.cost`。
* `included=true` 只允许出现在真实可用金额上。

### POI 扩展信息

在 `MockPOI` 或替代结构中新增真实 POI 元数据：

```go
type POIMetadata struct {
	Provider     string      `json:"provider,omitempty"`
	ID           string      `json:"id,omitempty"`
	Parent       string      `json:"parent,omitempty"`
	TypeCode     string      `json:"typecode,omitempty"`
	BizType      string      `json:"biz_type,omitempty"`
	Tel          string      `json:"tel,omitempty"`
	Postcode     string      `json:"postcode,omitempty"`
	Website      string      `json:"website,omitempty"`
	Email        string      `json:"email,omitempty"`
	PCode        string      `json:"pcode,omitempty"`
	PName        string      `json:"pname,omitempty"`
	CityCode     string      `json:"citycode,omitempty"`
	CityName     string      `json:"cityname,omitempty"`
	ADCode       string      `json:"adcode,omitempty"`
	ADName       string      `json:"adname,omitempty"`
	EntrLocation string      `json:"entr_location,omitempty"`
	ExitLocation string      `json:"exit_location,omitempty"`
	BusinessArea string      `json:"business_area,omitempty"`
	Tag          string      `json:"tag,omitempty"`
	Rating       *float64    `json:"rating,omitempty"`
	Photos       []POIPhoto  `json:"photos,omitempty"`
	Cost         CostInfo    `json:"cost"`
}
```

高德 `address`、`photos` 等字段可能出现空数组、字符串或对象列表；解析时必须兼容空值和类型差异。

### Route 扩展信息

在路线段中新增费用信息：

```go
type RouteCostInfo struct {
	CostInfo
	DistanceMeters  int `json:"distance_meters,omitempty"`
	DurationMinutes int `json:"duration_minutes,omitempty"`
}
```

`MockRoute` 或替代结构中应包含：

* `From`
* `To`
* `DurationMinutes`
* `DistanceMeters`
* `Mode`
* `Cost`

### TravelItem

`TravelItem` 需要支持展示真实费用和缺失费用：

```go
type TravelItem struct {
	Time            string   `json:"time"`
	Type            string   `json:"type"`
	Name            string   `json:"name"`
	Address         string   `json:"address"`
	Reason          string   `json:"reason"`
	EstimatedCost   float64  `json:"estimated_cost"`
	Cost            CostInfo `json:"cost"`
	DurationMinutes int      `json:"duration_minutes"`
	POI             *POIMetadata `json:"poi,omitempty"`
}
```

兼容要求：

* `estimated_cost` 可以暂时保留，避免一次性破坏前端和测试。
* 新前端必须优先使用 `cost`。
* 当 `cost.status=unavailable` 时，`estimated_cost` 即使为 0，也不得被页面展示为 0 元预算。

### TravelBudget

新增预算明细结构：

```go
type BudgetLine struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Amount   *float64 `json:"amount"`
	Currency string   `json:"currency"`
	Status   string   `json:"status"`
	Source   string   `json:"source,omitempty"`
	Display  string   `json:"display,omitempty"`
	Included bool     `json:"included"`
}

type TravelBudget struct {
	Transport  float64      `json:"transport"`
	Food       float64      `json:"food"`
	Hotel      float64      `json:"hotel"`
	Ticket     float64      `json:"ticket"`
	Total      float64      `json:"total"`
	KnownTotal float64      `json:"known_total"`
	Complete   bool         `json:"complete"`
	Currency   string       `json:"currency"`
	Items      []BudgetLine `json:"items"`
	Missing    []string     `json:"missing"`
}
```

兼容要求：

* 旧字段 `transport`、`food`、`hotel`、`ticket`、`total` 暂时保留。
* 旧字段只填真实可得且已计入预算的金额合计，不得包含缺失项估算。
* `total` 与 `known_total` 可以保持一致，前端文案必须称为“已知预算”。
* `complete=false` 时，页面不得展示“完整总预算”。

## 高德 POI 接入要求

`RealPOITool` 必须把请求参数从：

```go
"extensions": "base"
```

改为：

```go
"extensions": "all"
```

需要解析并保存：

* `id`
* `parent`
* `name`
* `type`
* `typecode`
* `biz_type`
* `address`
* `location`
* `tel`
* `postcode`
* `website`
* `email`
* `pcode`
* `pname`
* `citycode`
* `cityname`
* `adcode`
* `adname`
* `entr_location`
* `exit_location`
* `navi_poiid`
* `business_area`
* `tag`
* `indoor_map`
* `indoor_data`
* `biz_ext.rating`
* `biz_ext.cost`
* `photos.title`
* `photos.url`

`biz_ext.cost` 规则：

* 餐饮类 POI：解释为人均消费，`unit=per_person`。
* 景点类 POI：如果存在，可解释为人均门票或参考消费，`unit=per_person`。
* 酒店类 POI：如果存在，只能作为参考价格，`unit=per_night_reference`，不得当作实时房价。
* 字段缺失、为空、无法解析为数字时，`status=unavailable`。

## 高德路线费用接入要求

`RealRouteTool` 必须继续保留距离和时长，同时接入可得费用。

### 驾车 / 打车

调用高德驾车路径规划时必须传：

```go
"extensions": "all"
```

需要解析：

* `route.taxi_cost`
* `paths[0].tolls`

规则：

* 用户交通偏好包含“打车”或 `taxi` 时，优先使用 `route.taxi_cost` 作为该路线段费用。
* 用户交通偏好包含“自驾”或 `driving` 时，`paths[0].tolls` 只能作为过路费，不代表完整油费。
* 如果 `taxi_cost` 或 `tolls` 缺失，不得使用距离乘固定单价补齐。

### 公交 / 地铁

需要新增或扩展路线模式，支持调用高德公交路径规划：

```text
/direction/transit/integrated
```

需要解析：

* `transits[0].cost`
* `duration`
* `walking_distance`
* `segments`

规则：

* 用户交通偏好包含“地铁”“公交”“公共交通”时，应优先走公交路径规划。
* `transits[0].cost` 可作为该路线段费用。
* 缺失时显示“暂无信息”，不计入预算。

### 步行 / 骑行

规则：

* 步行和骑行没有接口报价。
* 可展示为 `not_applicable` 或“无需费用”。
* 不得伪造交通费用。

### 跨城大交通

规则：

* 高铁、火车、飞机票价本阶段不接入真实票价。
* 如果用户行程涉及跨城大交通，但没有真实票价来源，应展示“暂无信息”。
* 不允许用固定高铁票价或机票价格计入完整预算。

## BudgetTool 改造要求

必须直接改造现有 `MockBudgetTool.Run`。

改造后的职责：

* 汇总真实可得的 POI 餐饮费用。
* 汇总真实可得的景点门票或参考消费。
* 汇总真实可得的路线交通费用。
* 标记住宿、跨城大交通等不可得费用为“暂无信息”。
* 输出预算完整性 `complete=false`，除非所有用户勾选的预算项都有真实可得金额。

禁止行为：

* 不得再按用户预算固定比例拆分预算。
* 不得为了凑满用户预算而回填费用。
* 不得把缺失项设置为 0 后计入总价。
* 不得让 LLM 自行推算缺失价格。

建议计算规则：

```text
food = sum(餐饮 POI cost.amount * travelers)
ticket = sum(景点 POI cost.amount * travelers)
transport = sum(route cost.amount)
hotel = unavailable，除非未来接入真实酒店价格
known_total = food + ticket + transport + hotel_available_part
complete = len(missing) == 0
```

如果 `budget_type=人均预算`：

* 用户输入预算仍用于约束或提醒。
* 真实餐饮 `per_person` 应乘以 `travelers` 后进入总预算。
* 前端可额外展示人均已知预算：`known_total / travelers`。

## Graph 顺序要求

当前 `estimate_budget` 在 `optimize_itinerary` 之前，无法准确知道最终选中的行程项。

本阶段应调整为：

```text
parse_request
-> search_pois
-> get_weather
-> compute_route
-> optimize_itinerary
-> estimate_budget
-> validate_route_feasibility
-> generate_plan
-> validate_plan
```

`BudgetToolInput` 需要包含 `Itinerary`：

```go
type BudgetToolInput struct {
	Request   domain.TravelRequest
	Days      int
	POIs      []MockPOI
	Routes    []MockRoute
	Itinerary []domain.TravelDay
}
```

预算应尽量基于“实际进入初版 itinerary 的 POI”和对应路线段，而不是所有候选 POI。

## LLM Prompt 要求

路线生成 prompt 必须新增约束：

```text
价格和预算只能使用上下文中 status=available 且 included=true 的真实金额。
status=unavailable 的费用必须展示为“暂无信息”。
不要猜测、补全、估算或按比例分摊缺失费用。
预算合计只能统计真实可得金额。
如果预算不完整，必须在 summary 或 warnings 中说明哪些项目暂无信息。
```

LLM 不得改变真实价格来源字段。

如果 LLM 输出的价格与上下文真实价格不一致，后端应以后端真实数据为准，或直接校验失败并 fallback。

## JSON Schema 要求

`travelPlanJSONSchema` 必须支持新费用结构。

要求：

* `cost.amount` 支持 `number` 或 `null`。
* `cost.status` 必须为 `available`、`unavailable`、`not_applicable` 之一。
* `budget.items[].amount` 支持 `number` 或 `null`。
* `budget.complete` 必须为 boolean。
* `budget.known_total` 必须为 number。
* `additionalProperties=false` 继续保留。

如果当前模型或 provider 对 `type: ["number", "null"]` 支持不好，可以改用兼容 schema 表达，但必须保留语义约束。

## 前端展示要求

### 路线 item

页面展示单个行程项时：

* `cost.status=available`：展示金额，例如 `¥172/人`。
* `cost.status=unavailable`：展示 `暂无信息`。
* `cost.status=not_applicable`：展示 `无需费用` 或不展示金额。
* 不再直接展示 `estimated_cost` 作为主价格。

### 预算面板

预算面板文案改为：

```text
已知预算
```

而不是：

```text
总预算
```

当 `budget.complete=false` 时，需要展示：

```text
部分费用暂无真实数据，未计入已知预算。
```

预算项示例：

```text
餐饮 ¥900
市内交通 ¥132
住宿 暂无信息
往返大交通 暂无信息
门票 暂无信息
```

前端不得把 `null`、`undefined` 或缺失字段格式化为 `¥0`。

## API 兼容策略

短期兼容：

* 保留旧字段 `estimated_cost`。
* 保留旧字段 `budget.transport`、`budget.food`、`budget.hotel`、`budget.ticket`、`budget.total`。
* 新前端优先使用新字段。
* 旧字段只代表真实已知金额的合计，不代表完整预算。

长期迁移：

* 在 docs 中标记旧字段为 deprecated。
* 后续版本可移除旧字段，但必须先完成前端和 API 文档迁移。

## 错误与降级要求

当高德 POI 可用但 `biz_ext.cost` 缺失：

* POI 仍然可用。
* 价格为 `unavailable`。
* 不触发整个 planner 失败。

当高德路线可用但费用缺失：

* 距离和时长仍然可用。
* 费用为 `unavailable`。
* 不计入预算。

当高德 API 调用失败并 fallback 到 mock：

* 必须保留现有 `tool fallback` warning。
* fallback 数据不得标记为真实来源。
* fallback 价格不得计入完整预算，除非明确标记为非生产测试数据。

## 文档更新要求

必须更新：

* `docs/api.md`
  * 新增 `CostInfo`、`BudgetLine`、`TravelBudget` 字段说明。
  * 标明 `estimated_cost` 和旧预算字段的兼容语义。
* `docs/agent-flow.md`
  * 更新 Eino Graph 顺序。
  * 说明预算节点基于真实可得数据汇总。
* `docs/external-apis.md`
  * 说明高德 POI `extensions=all` 字段使用。
  * 说明高德路径规划费用字段使用与限制。
* `README.md`
  * 更新预算展示和运行说明。

## 测试要求

后端单元测试：

* `RealPOITool` fake AMap server 返回 `extensions=all` 字段，验证 `biz_ext.cost`、`rating`、`tag`、`photos` 被解析。
* `RealPOITool` 在 `biz_ext.cost` 缺失时，费用为 `unavailable`。
* `RealRouteTool` fake AMap driving 返回 `taxi_cost`，验证 route cost 可用。
* `RealRouteTool` fake transit 返回 `transits[].cost`，验证公交费用可用。
* `RealRouteTool` 在费用缺失时不失败，费用为 `unavailable`。
* `MockBudgetTool.Run` 不再按固定比例拆预算。
* `MockBudgetTool.Run` 只汇总 `available` 金额。
* `MockBudgetTool.Run` 对住宿和跨城大交通输出 missing。
* `travelPlanJSONSchema` 覆盖新费用结构。
* `parseTravelPlanArguments` 拒绝非法 status、未知字段和错误金额类型。

前端测试：

* `cost.amount=null` 时展示“暂无信息”。
* `budget.complete=false` 时展示“已知预算”和缺失提示。
* 不把 `null` 显示为 `¥0`。
* 旧接口字段存在时页面仍可渲染。

建议运行：

```bash
go test ./...
go vet ./...
npm run typecheck
npm run lint
```

如果修改前端交互，还应使用 Playwright 验证：

```bash
npm run dev
```

并检查桌面和移动端预算面板不会溢出、重叠或误导展示。

## 验收标准

本阶段完成后必须满足：

* 高德 POI 能拿到的真实字段被接入接口。
* 高德 POI `biz_ext.cost` 能被用于餐饮、景点等可识别费用。
* 高德路线费用字段能被用于市内交通预算。
* 住宿、跨城高铁、飞机等未接入真实数据的费用展示为“暂无信息”。
* 缺失费用不计入 `known_total`。
* 页面不再把缺失费用展示为 `¥0` 或固定估算值。
* LLM 不再生成、猜测或覆盖真实价格。
* 预算面板明确区分“已知预算”和“完整预算”。
* 不新增 `RealBudgetTool`。
* `go test ./...` 通过。
* 相关 API 文档、Agent Flow 文档和外部 API 文档同步更新。

## 示例返回

示例：

```json
{
  "budget": {
    "transport": 132,
    "food": 900,
    "hotel": 0,
    "ticket": 0,
    "total": 1032,
    "known_total": 1032,
    "complete": false,
    "currency": "CNY",
    "items": [
      {
        "key": "food",
        "label": "餐饮",
        "amount": 900,
        "currency": "CNY",
        "status": "available",
        "source": "amap.poi.biz_ext.cost",
        "included": true
      },
      {
        "key": "hotel",
        "label": "住宿",
        "amount": null,
        "currency": "CNY",
        "status": "unavailable",
        "display": "暂无信息",
        "included": false
      },
      {
        "key": "intercity_transport",
        "label": "往返大交通",
        "amount": null,
        "currency": "CNY",
        "status": "unavailable",
        "display": "暂无信息",
        "included": false
      }
    ],
    "missing": ["hotel", "intercity_transport"]
  }
}
```

前端应展示：

```text
已知预算：¥1,032
餐饮：¥900
市内交通：¥132
住宿：暂无信息
往返大交通：暂无信息
```

不得展示：

```text
总预算：¥1,032
住宿：¥0
```
