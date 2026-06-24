# External APIs

## 1. LLM Provider

阶段 3 已支持可选真实 LLM，用于在 Eino Graph 的 `GenerateTravelPlanNode` 中生成结构化 `TravelPlan`。

默认不开启 LLM。未启用、配置缺失、provider 调用失败或输出校验失败时，系统会 fallback 到 deterministic generator。

LLM 链路会记录 prompt version、调用耗时、fallback 分类和 provider 返回的 token usage。如果 provider 不返回 usage，系统记录为 `unknown`，不会伪造 token 数。

## 2. 环境变量

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `TRAVEL_AGENT_LLM_ENABLED` | 是否启用真实 LLM | `false` |
| `TRAVEL_AGENT_LLM_PROVIDER` | LLM provider，可用值：`deepseek`、`openai`、`compatible` | `deepseek` |
| `TRAVEL_AGENT_LLM_API_KEY` | LLM API Key | 空 |
| `DEEPSEEK_API_KEY` | DeepSeek provider 的备用 API Key 变量 | 空 |
| `TRAVEL_AGENT_LLM_BASE_URL` | OpenAI-compatible base URL | DeepSeek 为 `https://api.deepseek.com/beta` |
| `TRAVEL_AGENT_LLM_MODEL` | 模型名称 | `deepseek-v4-flash` |
| `TRAVEL_AGENT_LLM_TIMEOUT` | HTTP timeout，可写 `30s` 或秒数 | `30s` |
| `TRAVEL_AGENT_LLM_MAX_RETRIES` | LLM 输出解析或校验失败后的重试次数 | `1` |

不要把真实 API Key 写入代码、测试、文档或报告。

当前 prompt version：

```text
travel-plan-v1
```

LLM 成功路径会在计划 warning 中追加可解析 trace：

```text
LLM trace: prompt_version=travel-plan-v1 duration_ms=123 prompt_tokens=10 completion_tokens=20 total_tokens=30
```

如果 provider 未返回 usage：

```text
LLM trace: prompt_version=travel-plan-v1 duration_ms=123 prompt_tokens=unknown completion_tokens=unknown total_tokens=unknown
```

LLM fallback 使用稳定格式：

```text
LLM fallback: prompt_version=travel-plan-v1 category=invalid_json attempts=1 duration_ms=123 reason=...
```

fallback category 包括：

* `disabled`
* `missing_api_key`
* `provider_error`
* `timeout`
* `invalid_json`
* `business_validation_failed`
* `retry_exhausted`

## 3. DeepSeek 结构化输出

DeepSeek 默认使用 OpenAI-compatible Chat Completions beta endpoint：

```text
https://api.deepseek.com/beta/chat/completions
```

输出不依赖 prompt-only JSON 约束。请求会强制模型调用单个 tool：

```text
submit_travel_plan
```

该 tool 的 `parameters` 是 `domain.TravelPlan` 对应的 JSON Schema，并设置：

```json
{
  "strict": true,
  "additionalProperties": false
}
```

所有 object 字段都列入 `required`。本地解析时继续拒绝 unknown fields，并校验天数、预算、空字段、负数和目的地关键词。

## 4. 其他 Provider

`openai` 或 `compatible` provider 会优先使用 OpenAI Structured Outputs 风格：

```json
{
  "response_format": {
    "type": "json_schema"
  }
}
```

如果 provider 不支持 JSON Schema 输出或返回结构不符合本地校验，系统不会降级到 prompt-only JSON，而是 fallback 到 deterministic generator。

## 5. 高德 / 天气 / POI Tools

阶段 4 已为 Eino Tools 增加真实外部 API 能力。默认仍使用 Mock Tools；只有显式设置 real mode 且配置 API Key 时才调用真实外部接口。

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `TRAVEL_AGENT_TOOL_MODE` | Tool 模式：`mock` 或 `real` | `mock` |
| `TRAVEL_AGENT_AMAP_API_KEY` | 高德 Web 服务 Key | 空 |
| `TRAVEL_AGENT_AMAP_BASE_URL` | 高德 Web 服务 Base URL | `https://restapi.amap.com/v3` |
| `TRAVEL_AGENT_WEATHER_API_KEY` | 天气 API Key；为空时复用高德 Key | 空 |
| `TRAVEL_AGENT_WEATHER_BASE_URL` | 天气 API Base URL；为空时复用高德 Base URL | 空 |
| `TRAVEL_AGENT_EXTERNAL_API_TIMEOUT` | 外部 API timeout，可写 `10s` 或秒数 | `10s` |

当前 real tools：

* `RealPOITool`：调用高德 `place/text` 搜索 POI，并转换为内部 POI 类型。
* `RealWeatherTool`：先调用高德 `geocode/geo` 获取 adcode，再调用 `weather/weatherInfo` 查询预报。
* `RealRouteTool`：使用 POI 坐标调用高德路径规划接口，按交通模式选择 walking / bicycling / driving。

所有 real tool 都有 Mock fallback。以下情况会 fallback，并在最终 `TravelPlan.warnings` 中说明原因：

* real mode 下未配置 API Key
* HTTP 请求失败或超时
* 高德返回非成功状态
* 返回非 JSON 或 JSON 结构无效
* 响应缺少必要字段
* 路线计算缺少 POI 坐标

fallback warning 使用稳定的 key-value 文本格式，便于后续 Harness 统计：

```text
tool fallback: tool=poi provider=amap stage=request category=provider_error mock_fallback=true reason=...
```

字段语义：

* `tool`：`poi`、`weather` 或 `route`
* `provider`：当前为 `amap`
* `stage`：`configuration` 或 `request`
* `category`：`configuration`、`timeout`、`provider_error`、`invalid_json`、`missing_field`、`request_error` 或 `unknown`
* `mock_fallback`：是否已使用 mock tool 兜底
* `reason`：脱敏后的错误摘要，不包含 API Key 或原始敏感响应

外部 API 原始响应不会进入 `internal/domain`；只会转换为 Eino 内部状态使用的 POI、Weather、Route 数据。

当前稳定性边界：real tools 已通过 fake HTTP server 覆盖缺 key、timeout、非 2xx、provider 失败、无效 JSON、必填字段缺失、路线坐标缺失和天气城市编码失败。默认 `mock` mode 不会发起外部请求。真实生产数据的 POI 质量、限流策略和地图级排程准确性仍需要结合实际 Key 和 provider SLA 单独验证。

## 6. 尚未接入

酒店、票务、支付、用户系统等外部 API 尚未接入。后续阶段接入时，需要继续从环境变量读取 Key，并保留 Mock fallback。
