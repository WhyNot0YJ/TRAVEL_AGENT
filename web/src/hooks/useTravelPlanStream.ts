import { useEffect, useMemo, useRef, useState } from "react";
import { createTravelPlanEventSource, getTravelPlanTask } from "../api/client";
import type { TaskEvent, TravelPlan } from "../api/types";

interface StreamState {
  events: TaskEvent[];
  connected: boolean;
  polling: boolean;
  error: string;
  plan?: TravelPlan;
  status: string;
}

const initialState: StreamState = {
  events: [],
  connected: false,
  polling: false,
  error: "",
  status: "empty",
};

function parseEvent(raw: MessageEvent<string>, fallbackType: TaskEvent["type"]): TaskEvent {
  try {
    return JSON.parse(raw.data) as TaskEvent;
  } catch {
    return { type: fallbackType, message: raw.data };
  }
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
      setState((current) => ({
        ...current,
        events: event.type === "heartbeat" ? current.events : [...current.events, event].slice(-20),
        plan: event.plan ?? current.plan,
        status: event.status ?? current.status,
        error: event.type === "error" ? event.message || "任务失败" : current.error,
      }));
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
    eventSource.addEventListener("warning", (raw) => appendEvent(parseEvent(raw as MessageEvent<string>, "warning")));
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
