# External APIs

## 1. LLM Provider

阶段 3 已支持可选真实 LLM，用于在 Eino Graph 的 `GenerateTravelPlanNode` 中生成结构化 `TravelPlan`。

默认不开启 LLM。未启用、配置缺失、provider 调用失败或输出校验失败时，系统会 fallback 到 deterministic generator。

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

## 5. 尚未接入

地图 POI、路线规划、天气、酒店、票务等真实外部 API 尚未接入。后续阶段接入时，需要继续从环境变量读取 Key，并保留 Mock Tools fallback。
