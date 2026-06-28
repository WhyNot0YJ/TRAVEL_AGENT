# Backend Harness Case Catalog

本文档说明 `testdata/travel_cases.json` 中每个后端 Agent Harness case 的测试场景。新增、删除或重命名后端 case 时，必须同步更新本文档。

## Harness 位置

* 命令入口：`cmd/harness`
* 核心实现：`internal/harness`
* 数据集：`testdata/travel_cases.json`
* 默认报告：`reports/eval_report.json`
* 常用命令：

```bash
go run ./cmd/harness
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
make harness
make harness-eino
```

## 维护规则

新增后端 case 时：

1. 在 `testdata/travel_cases.json` 增加唯一 `id`，并确保 `input.id` 与 `id` 一致。
2. `description` 要写清楚测试目的，不只写目的地。
3. 在本文档的 case 表格中追加一行，说明覆盖场景和主要验证点。
4. 如果新增了评分指标或报告字段，还要同步更新 `internal/harness/evaluator.go`、`internal/harness/metrics.go`、`docs/evaluation-harness.md` 和 `README.md`。

## Case 列表

所有后端 case 由 `testdata/travel_cases.json` 驱动，并通过统一 `TravelPlanner` 接口运行，因此同一份数据可用于 `MockPlanner` 和 `EinoTravelPlanner`。

| Case | 输入摘要 | 覆盖场景 | 主要验证点 |
| :--- | :--- | :--- | :--- |
| `case_001` | 上海 -> 杭州，3 天，3000 元，2 人，自然风光 + 美食，`train_taxi`，`relaxed` | 常规多兴趣、多日、预算充足的基线路径 | 能生成杭州关键词、3 天结构、预算不超过阈值，作为基础回归样本 |
| `case_002` | 上海 -> 南京，2 天，1500 元，2 人，历史文化 + 美食，`train_walk`，`balanced` | 短途周末游和均衡节奏 | 验证短行程结构完整、南京关键词、预算合规 |
| `case_003` | 北京 -> 北京，4 天，5000 元，3 人，亲子 + 博物馆，`subway_taxi`，`balanced` | 同城出发目的地 + 亲子主题 | 验证同城游不被误判、人数参与预算、亲子/博物馆路线结构稳定 |
| `case_004` | 上海 -> 成都，3 天，2500 元，2 人，美食 + 休闲，`flight_taxi`，`relaxed` | 飞行到达后的轻松本地体验 | 验证跨城飞行交通偏好、成都关键词和轻松节奏 |
| `case_005` | 深圳 -> 广州，2 天，1800 元，2 人，citywalk + 美食，`train_walk`，`balanced` | 近距离高铁 + 步行偏好 | 验证 citywalk 主题、短途高铁场景和预算合规 |
| `case_006` | 上海 -> 西安，3 天，2200 元，2 人，历史文化，`flight_subway`，`balanced` | 单一强主题 + 紧预算跨城场景 | 验证单兴趣输入仍能生成完整多日路线，预算约束有效 |
| `case_007` | 上海 -> 苏州，1 天，800 元，2 人，园林 + 轻松，`train_walk`，`relaxed` | 最短有效天数 | 验证单日计划结构、苏州关键词和预算非负 |
| `case_008` | 上海 -> 未知城市，2 天，1000 元，2 人，探索 + 小众，`train_walk`，`balanced` | 未知目的地兜底 | 验证 POI 不存在时仍能生成完整结构，并保留目的地关键词 |
| `case_009` | 南京 -> 杭州，2 天，600 元，2 人，自然风光 + 城市漫步，`train_walk`，`relaxed` | 低预算多日路线 | 验证预算拆分在低预算下不超阈值、不出现负数 |
| `case_010` | 广州 -> 北京，5 天，12000 元，2 人，历史文化 + 博物馆 + 亲子，`flight_subway`，`balanced` | 高预算长一点行程 | 验证 5 天结构、高预算拆分稳定和北京关键词 |
| `case_011` | 重庆 -> 成都，2 天，1600 元，2 人，美食 + 夜生活，`train_taxi`，`intensive` | 紧凑节奏短途美食路线 | 验证 intensive 节奏下的活动密度和预算合规 |
| `case_012` | 佛山 -> 广州，1 天，500 元，2 人，城市探索，`subway_walk`，`balanced` | 单日城市探索 + 默认偏好路径 | 验证必填兴趣已提供时能走默认可选项，并生成单日结构 |
| `case_013` | 杭州 -> 苏州，6 天，4800 元，2 人，园林 + 博物馆 + 美食，`train_walk`，`relaxed` | 长天数低强度路线 | 验证 POI 循环分配、多日结构连续性和预算合规 |
| `case_014` | 郑州 -> 西安，1 天，900 元，2 人，历史文化 + 夜游，`train_subway`，`intensive` | 单日高强度历史路线 | 验证短天数和 intensive 组合下结构完整 |
| `case_015` | 合肥 -> 南京，2 天，900 元，3 人，亲子 + 博物馆，`train_subway`，`relaxed` | 亲子 + 低预算双约束 | 验证 3 人预算口径、亲子主题和低预算约束 |
| `case_016` | 杭州 -> 杭州，2 天，1200 元，2 人，本地生活 + 咖啡，`subway_walk`，`balanced` | 本地同城周末路线 | 验证出发城市与目的地相同仍能生成完整计划 |
| `case_017` | 天津 -> 北京，3 天，3600 元，2 人，老人友好 + 公园 + 博物馆，`train_taxi`，`relaxed` | 老人慢节奏路线 | 验证 relaxed 节奏、活动密度和老人友好主题 |
| `case_018` | 西安 -> 成都，4 天，4200 元，2 人，美食 + 潮流街区 + 夜生活，`flight_taxi`，`intensive` | 年轻人高强度多兴趣路线 | 验证多兴趣轮转、紧凑节奏和预算合规 |
| `case_019` | 上海 -> 广州，3 天，3500 元，2 人，商务间隙 + 美食 + citywalk，`flight_taxi`，`relaxed` | 商务间隙轻量安排 | 验证工作日短时体验、轻松节奏和广州关键词 |
| `case_020` | 上海 -> 苏州，1 天，300 元，2 人，园林 + 步行，`train_walk`，`balanced` | 极低预算单日游 | 验证预算下限附近不出现负预算，结构仍完整 |
| `case_021` | 上海 -> 未知城市，5 天，2500 元，2 人，小众 + 探索 + 本地生活，`train_walk`，`balanced` | 未知城市长天数兜底 | 验证未知目的地、多日 POI 复用和稳定输出 |
| `case_022` | 深圳 -> 西安，5 天，9000 元，2 人，历史文化 + 博物馆 + 夜游，`flight_taxi`，`balanced` | 高预算深度文化路线 | 验证高预算、多日历史主题和预算拆分 |
| `case_023` | 上海 -> 南京，3 天，2400 元，2 人，城市漫步 + 历史文化，交通方式为空，`balanced` | 交通方式缺省兼容 | 验证 Eino 节点和 planner 对空 `transport_mode` 的默认值处理 |
| `case_024` | 上海 -> 杭州，4 天，4200 元，2 人，自然风光 + 茶文化，`train_taxi`，节奏为空 | 节奏缺省兼容 | 验证空 `pace` 输入会走默认节奏，并保持结构完整 |
| `case_025` | 上海 -> 杭州，3 天，3000 元，2 人，美食 + 自然风光，`train_taxi`，`relaxed` | 聊天式快速模式常见输入 | 验证高频用户请求的基础回归，与前端 mock 场景保持接近 |
| `case_026` | 广州 -> 北京，5 天，12000 元，2 人，亲子 + 历史文化 + 博物馆，`flight_taxi`，`balanced` | 专家模式高约束长天数输入 | 验证长行程、高预算、多主题结构稳定性 |
| `case_027` | 上海 -> 杭州，2 天，2600 元，2 人，美食 + 自然风光，低步行，西湖附近，必去西湖，情侣 | Stage 18 richer brief：人数、步行强度、酒店区域、必去点 | 验证 richer brief 进入 planner，并影响预算、强度和 POI 排序 |
| `case_028` | 上海 -> 南京，3 天，3600 元，2 人，历史文化 + 美食，中步行，夫子庙附近，必去南京博物院，避开夜游 | Stage 18 richer brief：avoid + must_visit | 验证 `avoid` 参与 POI 选择和文案，`must_visit` 尽量进入路线 |
| `case_029` | 上海 -> 苏州，2 天，1800 元，2 人，园林 + 美食，只给必填项 | Stage 18 默认值路径 | 验证可选偏好缺失时统一使用默认值，不阻塞生成 |

## 新增 Case 检查清单

新增 case 前先确认它补的是新的风险，而不是重复覆盖已有路径。建议至少写清楚：

* 用户输入形态：城市、天数、预算、人数、兴趣、交通、节奏和 richer brief 字段。
* 触发的风险：预算边界、默认值、未知城市、真实工具 fallback、路线可行性等。
* 预期信号：关键词、天数、预算阈值、warning、fallback 文案或报告字段。
* 是否需要同步文档：本文档、`docs/evaluation-harness.md`、`README.md`、`docs/api.md`。
