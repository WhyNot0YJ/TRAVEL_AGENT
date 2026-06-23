# API

## Current Stage

当前阶段提供 Gin HTTP API，同步调用 `TravelPlanner` 并使用内存 store 保存最近生成的计划。

尚未接入 Redis、数据库、异步任务或 SSE。

## Base URL

默认监听：

```text
http://localhost:8080
```

可通过环境变量配置：

```text
TRAVEL_AGENT_HTTP_ADDR=:8080
TRAVEL_AGENT_PLANNER=mock|eino
```

## Error Response

```json
{
  "request_id": "20260623193000.000000000",
  "code": "invalid_request",
  "message": "days is required"
}
```

## POST /api/v1/travel/plans

同步创建旅行计划。

Request:

```json
{
  "departure_city": "上海",
  "destination_city": "杭州",
  "days": 3,
  "budget": 3000,
  "interests": ["自然风光", "美食"],
  "transport_mode": "train_taxi",
  "pace": "relaxed"
}
```

Response:

```json
{
  "plan_id": "plan_xxx",
  "plan": {
    "title": "杭州3日旅行规划",
    "summary": "...",
    "days": [],
    "budget": {
      "transport": 0,
      "food": 0,
      "hotel": 0,
      "ticket": 0,
      "total": 0
    },
    "warnings": []
  }
}
```

Status:

* `200`：创建成功
* `400`：请求 JSON 或必填字段无效
* `500`：planner 或服务内部错误

## GET /api/v1/travel/plans/:id

根据 `plan_id` 查询内存 store 中的计划。

Response:

```json
{
  "plan_id": "plan_xxx",
  "plan": {}
}
```

Status:

* `200`：查询成功
* `404`：计划不存在
* `500`：服务内部错误
