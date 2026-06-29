import { useEffect, useMemo, useRef, useState } from "react";
import { createTravelPlanEventSource, getTravelPlanTask } from "../api/client";
import type { POIInfo, RouteInfo, TaskEvent, TravelBudget, TravelDay, TravelPlan, WeatherInfo } from "../api/types";

interface StreamState {
  events: TaskEvent[];
  connected: boolean;
  polling: boolean;
  error: string;
  plan?: TravelPlan;
  draftPlan?: TravelPlan;
  draftDays: TravelDay[];
  pois: POIInfo[];
  weather: WeatherInfo[];
  routes: RouteInfo[];
  budget?: TravelBudget;
  status: string;
  assistantText: string;
}

const initialState: StreamState = {
  events: [],
  connected: false,
  polling: false,
  error: "",
  status: "empty",
  assistantText: "",
  draftDays: [],
  pois: [],
  weather: [],
  routes: [],
};

function parseEvent(raw: MessageEvent<string>, fallbackType: TaskEvent["type"]): TaskEvent {
  try {
    return JSON.parse(raw.data) as TaskEvent;
  } catch {
    return { type: fallbackType, message: raw.data };
  }
}

const emptyBudget: TravelBudget = {
  transport: 0,
  food: 0,
  hotel: 0,
  ticket: 0,
  total: 0,
  known_total: 0,
  complete: false,
  currency: "CNY",
  items: [],
  missing: [],
};

function upsertDay(days: TravelDay[], day: TravelDay): TravelDay[] {
  const next = days.filter((item) => item.day !== day.day);
  next.push(day);
  return next.sort((a, b) => a.day - b.day);
}

function buildDraftPlan(days: TravelDay[], budget?: TravelBudget): TravelPlan | undefined {
  if (days.length === 0) {
    return undefined;
  }
  return {
    title: "路线草稿生成中",
    summary: "已生成的天数会先作为草稿展示，完整路线完成后会自动替换。",
    days,
    budget: budget ?? emptyBudget,
    warnings: [],
  };
}

function isDuplicateEvent(events: TaskEvent[], event: TaskEvent): boolean {
  return typeof event.sequence === "number" && events.some((item) => item.task_id === event.task_id && item.sequence === event.sequence);
}

export function useTravelPlanStream(taskId: string | null) {
  const [state, setState] = useState<StreamState>(initialState);
  const shouldPoll = useRef(false);

  useEffect(() => {
    shouldPoll.current = false;
    setState(initialState);

    if (!taskId) {
      return;
    }

    let eventSource: EventSource | null = createTravelPlanEventSource(taskId);
    let pollTimer: number | undefined;

    const appendEvent = (event: TaskEvent) => {
      const shouldKeepEvent =
        event.type !== "heartbeat" && event.type !== "assistant_delta" && event.type !== "assistant_done";
      setState((current) => {
        if (shouldKeepEvent && isDuplicateEvent(current.events, event)) {
          return current;
        }
        const draftDays = event.type === "day_delta" && event.day ? upsertDay(current.draftDays, event.day) : current.draftDays;
        const budget = event.budget ?? current.budget;
        const plan = event.plan ?? current.plan;
        return {
          ...current,
          events: shouldKeepEvent ? [...current.events, event].slice(-30) : current.events,
          plan,
          draftDays,
          draftPlan: plan ? current.draftPlan : buildDraftPlan(draftDays, budget),
          pois: event.pois ?? current.pois,
          weather: event.weather ?? current.weather,
          routes: event.routes ?? current.routes,
          budget,
          status: event.status ?? current.status,
          error: event.type === "error" ? event.message || "任务失败" : current.error,
          assistantText:
            event.type === "assistant_delta"
              ? current.assistantText + (event.message || "")
              : event.type === "assistant_done"
                ? event.message || current.assistantText
                : current.assistantText,
        };
      });
    };

    const stopStream = () => {
      if (eventSource) {
        eventSource.close();
        eventSource = null;
      }
    };

    const poll = async () => {
      try {
        const task = await getTravelPlanTask(taskId);
        setState((current) => ({
          ...current,
          polling: task.status !== "succeeded" && task.status !== "failed",
          status: task.status,
          plan: task.plan ?? current.plan,
          error: task.error ?? current.error,
        }));

        if (task.status === "succeeded" || task.status === "failed") {
          if (pollTimer !== undefined) {
            window.clearInterval(pollTimer);
          }
          shouldPoll.current = false;
        }
      } catch (error) {
        setState((current) => ({
          ...current,
          polling: false,
          error: error instanceof Error ? error.message : "查询任务失败",
        }));
      }
    };

    const startPolling = () => {
      if (shouldPoll.current) {
        return;
      }
      shouldPoll.current = true;
      setState((current) => ({
        ...current,
        connected: false,
        polling: true,
        error: current.error || "实时连接已断开，正在切换到轮询",
      }));
      void poll();
      pollTimer = window.setInterval(poll, 1600);
    };

    eventSource.onopen = () => {
      setState((current) => ({ ...current, connected: true, polling: false, error: "" }));
    };

    eventSource.addEventListener("progress", (raw) => appendEvent(parseEvent(raw as MessageEvent<string>, "progress")));
    eventSource.addEventListener("node", (raw) => appendEvent(parseEvent(raw as MessageEvent<string>, "node")));
    eventSource.addEventListener("warning", (raw) => appendEvent(parseEvent(raw as MessageEvent<string>, "warning")));
    eventSource.addEventListener("assistant_delta", (raw) =>
      appendEvent(parseEvent(raw as MessageEvent<string>, "assistant_delta")),
    );
    eventSource.addEventListener("assistant_done", (raw) =>
      appendEvent(parseEvent(raw as MessageEvent<string>, "assistant_done")),
    );
    eventSource.addEventListener("brief_delta", (raw) =>
      appendEvent(parseEvent(raw as MessageEvent<string>, "brief_delta")),
    );
    eventSource.addEventListener("poi_batch", (raw) => appendEvent(parseEvent(raw as MessageEvent<string>, "poi_batch")));
    eventSource.addEventListener("weather_delta", (raw) =>
      appendEvent(parseEvent(raw as MessageEvent<string>, "weather_delta")),
    );
    eventSource.addEventListener("route_delta", (raw) =>
      appendEvent(parseEvent(raw as MessageEvent<string>, "route_delta")),
    );
    eventSource.addEventListener("budget_delta", (raw) =>
      appendEvent(parseEvent(raw as MessageEvent<string>, "budget_delta")),
    );
    eventSource.addEventListener("day_delta", (raw) => appendEvent(parseEvent(raw as MessageEvent<string>, "day_delta")));
    eventSource.addEventListener("plan_draft", (raw) =>
      appendEvent(parseEvent(raw as MessageEvent<string>, "plan_draft")),
    );
    eventSource.addEventListener("done", (raw) => {
      appendEvent(parseEvent(raw as MessageEvent<string>, "done"));
      stopStream();
    });
    eventSource.addEventListener("error", (raw) => {
      const parsed = parseEvent(raw as MessageEvent<string>, "error");
      if (parsed.message || parsed.status === "failed") {
        appendEvent(parsed);
        stopStream();
        return;
      }
      startPolling();
    });
    eventSource.addEventListener("heartbeat", (raw) => appendEvent(parseEvent(raw as MessageEvent<string>, "heartbeat")));

    eventSource.onerror = () => {
      startPolling();
    };

    return () => {
      stopStream();
      if (pollTimer !== undefined) {
        window.clearInterval(pollTimer);
      }
    };
  }, [taskId]);

  return useMemo(() => state, [state]);
}
