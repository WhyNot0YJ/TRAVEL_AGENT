# Stage 12：路线真实性校验

## 任务目标

把当前"结构正确"的路线规划升级为"基本可信"的路线规划。校验应覆盖 POI 坐标、相邻点交通耗时、营业时间、天气影响、预算拆分和每日强度，并以 warning、score 或 validation error 的形式反馈。

## 当前上下文

当前 `ValidatePlanNode` 主要校验：

* title/summary/days/items/budget 结构完整
* 天数匹配
* 预算阈值
* 非负字段
* 目的地关键词

当前不校验真实路线距离、营业时间、实时天气或 POI 可达性。

## 不做什么

* 不要求一次性做到地图级精准排程。
* 不把验证规则写死到 Harness 里绕过 Planner。
* 不让真实 API 不可用时导致默认 mock harness 失败。
* 不引入复杂地图渲染。

## 需要阅读的文件

* `docs/agent-flow.md`
* `docs/evaluation-harness.md`
* `internal/agent/eino/nodes.go`
* `internal/agent/eino/types.go`
* `internal/agent/eino/tools.go`
* `internal/agent/eino/real_route_tool.go`
* `internal/domain/travel.go`
* `internal/harness/evaluator.go`
* `internal/harness/metrics.go`
* `testdata/travel_cases.json`

## 实现要求

* 为路线真实性定义清晰的内部校验结果，不污染稳定 domain 模型；如需进入报告，放在 Harness result 或 planner metadata。
* 校验项建议包括：
  * 每天 POI 数量与 pace 是否匹配
  * 相邻 POI route duration 是否过长
  * POI 缺坐标时是否明确降级
  * 雨天是否增加室内备选 warning
  * 预算拆分是否和天数、交通模式大致一致
  * 同一天是否出现明显重复 POI
* 对真实数据不可得的情况输出 warning，不要伪造精确信息。
* 保持 mock 数据可稳定通过基础 Harness。

## 文档更新要求

* 更新 `docs/agent-flow.md`：说明 ValidatePlanNode 新增真实性校验。
* 更新 `docs/evaluation-harness.md`：如新增真实性评分或检查项，必须说明计算方式。
* 更新 `README.md`：补充当前真实性校验的能力边界。

## 测试要求

尽量运行：

```bash
go test ./...
go vet ./...
go run ./cmd/harness -planner mock
go run ./cmd/harness -planner eino
```

## 验收标准

* 基础结构校验保持兼容。
* 明显不可达、重复、预算异常或天气冲突能被识别。
* 新增 warning 或 score 字段有测试和文档。
* Harness 报告能体现真实性校验结果。
