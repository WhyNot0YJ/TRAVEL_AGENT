# Agent Flow

## Current Stage

The project uses a Go `TravelPlanner` interface with an Eino implementation in `internal/agent/eino`.
The frontend is a React H5 chat interface. The backend exposes asynchronous travel-plan tasks, SSE progress, a chat-based requirement collection API, deterministic test mode, and optional real LLM / real external tools.

## Chat-First Flow

```text
User message
  -> POST /api/v1/travel/chat/stream
  -> TravelInfoExtractor
     -> test_mode: local rule extractor
     -> real mode: LLM extractor with rule fallback
  -> ChatResponse(is_complete, missing, collected fields)
  -> Assistant confirmation card in chat
  -> POST /api/v1/travel/plans
  -> Eino TravelPlanning Graph
  -> GET /api/v1/travel/plans/:task_id/stream
  -> TravelPlan
```

The frontend no longer renders a fixed outer "Generate itinerary" form button. When `is_complete=true`, the assistant message renders a requirement summary and a `生成行程` action. Clicking it creates a plan task from the collected structured brief.

## Runtime Options

Runtime behavior is carried by `agent.PlannerOptions`:

* `TestMode=true`: collection, tools, and final generation use local deterministic rules, mock tools, and deterministic fallback generation.
* `TestMode=false`: collection and final generation prefer the configured LLM; tools use `TRAVEL_AGENT_TOOL_MODE`.
* `AgentMode=quick`: routed to the model in `TRAVEL_AGENT_LLM_MODEL_QUICK` (default `deepseek-v4-flash`)—lower latency, lower cost.
* `AgentMode=expert`: routed to the model in `TRAVEL_AGENT_LLM_MODEL_EXPERT` (default `deepseek-v4-pro`)—stronger reasoning, higher cost. The agent_mode is part of `request_hash`, so quick/expert results never collide in the cache.

DeepSeek strict tool calling still sends `thinking.type=disabled` because DeepSeek thinking mode does not support forced `tool_choice`. When the configured model name contains `reasoner`, the client automatically falls back to `response_format=json_schema` and accepts JSON via `message.content`, since reasoner-class models do not emit tool calls.

## Eino Planning Graph

```text
TravelRequest
  -> ParseTravelRequestNode
  -> SearchPOIsToolNode
  -> GetWeatherToolNode
  -> ComputeRouteToolNode
  -> EstimateBudgetToolNode
  -> OptimizeItineraryNode
  -> ValidateRouteFeasibilityNode
  -> GenerateTravelPlanNode
  -> ValidatePlanNode
  -> TravelPlan
```

## LLM Generation

`GenerateTravelPlanNode` uses `travel-plan-v1`.

When LLM is enabled and configured, the OpenAI-compatible client requests provider-native structured output:

* DeepSeek: strict function tool `submit_travel_plan`.
* Other compatible providers: JSON schema response format when supported.
* Reasoner-class DeepSeek models: automatically downgraded to `response_format=json_schema` (no tool calls).

If test mode is enabled, the LLM is disabled, the API key is missing, the provider returns invalid output, retries are exhausted, or business validation fails, the node falls back to deterministic generation and records an `LLM fallback` warning.

### Streaming LLM output

When `TRAVEL_AGENT_LLM_STREAM_ENABLED=true` (default), DeepSeek chat completions are called with `stream=true` for requirement extraction, final plan generation, and assistant replies. Structured calls still use strict tool schemas; the server accumulates streamed tool arguments and parses the final JSON after `[DONE]`.

For plan generation, the same streamed `submit_travel_plan` tool arguments are scanned incrementally for the top-level `summary` field. If an `LLMDeltaReporter` is attached, newly observed summary text is forwarded as `assistant_delta`; after the stream completes, the accumulated arguments are parsed into the final `TravelPlan`. If no reporter is attached (for example in `cmd/harness`), the provider call still uses `stream=true`, but the deltas are only accumulated internally.

`LLMDeltaReporter` is defined in `internal/agent/metadata.go` and follows the same context-key pattern as `PlannerEventReporter`. The `internal/travel` package supplies the implementation that translates deltas into `EventAssistantDelta` events on the `EventBus`. `internal/agent/eino` depends only on the abstraction—it never imports `internal/travel`.

## Requirement Collection

`TravelInfoExtractor` uses `chat-info-v1`.

* `POST /api/v1/travel/chat` returns a single JSON response.
* `POST /api/v1/travel/chat/stream` returns SSE `assistant_delta` chunks followed by final `done` with the same structured `ChatResponse`.

Required fields are:

* `departure_city`
* `destination_city`
* `days`
* `budget`
* `interests`

Test mode always uses the local rule extractor. Real mode tries streamed LLM extraction first and falls back to rules if unavailable.

## Tools

Tools support `mock` and `real` modes:

```text
TRAVEL_AGENT_TOOL_MODE=mock
TRAVEL_AGENT_TOOL_MODE=real
```

Real AMap/weather/route requests share a backend limiter. Defaults:

* `TRAVEL_AGENT_EXTERNAL_API_CONCURRENCY=2`
* `TRAVEL_AGENT_EXTERNAL_API_QPS=2`

When `test_mode=true`, tools are forced to mock even if the server is configured for real tools.

## SSE Events

Planning task stream events:

* `progress`
* `node`
* `warning`
* `assistant_delta`
* `assistant_done`
* `done`
* `error`
* `heartbeat`

`done.plan` is the final source of truth for structured itinerary data. `assistant_delta` is user-facing generation text only.

## Boundaries

* `internal/harness` depends only on `TravelPlanner`, never on the Eino implementation.
* Eino code stays under `internal/agent/eino`.
* HTTP/SSE stays under `internal/travel` and `internal/server`.
* API keys are read from environment variables or `.env`, never hardcoded.
