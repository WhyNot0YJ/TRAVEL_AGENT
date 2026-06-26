import { useMemo, useState } from "react";
import { createTravelPlanTask, streamChatTravelInfo } from "./api/client";
import type { AgentMode, ChatRequest, TravelPlanRequest } from "./api/types";
import AgentConversation, { type ChatMessage, type TravelBrief } from "./components/AgentConversation";
import PlanDetail from "./components/PlanDetail";
import PlanProgress from "./components/PlanProgress";
import StateView from "./components/StateView";
import TravelBriefPanel from "./components/TravelBriefPanel";
import { useTravelPlanStream } from "./hooks/useTravelPlanStream";

const initialBrief: TravelBrief = {
  departureCity: "",
  destinationCity: "",
  days: "",
  budget: "",
  interests: [],
  transportMode: "train_taxi",
  pace: "balanced",
};

const initialMessages: ChatMessage[] = [
  {
    id: "welcome",
    role: "assistant",
    text: "你好，我会先把出发地、目的地、天数、预算和偏好收集完整。信息齐了以后，我会在这里给你一个生成行程按钮。",
  },
];

function nextId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function missingFields(brief: TravelBrief): string[] {
  const missing: string[] = [];
  if (!brief.departureCity) {
    missing.push("出发城市");
  }
  if (!brief.destinationCity) {
    missing.push("目的地");
  }
  if (!brief.days) {
    missing.push("天数");
  }
  if (!brief.budget) {
    missing.push("预算");
  }
  if (brief.interests.length === 0) {
    missing.push("兴趣偏好");
  }
  return missing;
}

function chatPayloadFromBrief(message: string, brief: TravelBrief, testMode: boolean, agentMode: AgentMode): ChatRequest {
  return {
    message,
    departure_city: brief.departureCity || undefined,
    destination_city: brief.destinationCity || undefined,
    days: Number(brief.days) || undefined,
    budget: Number(brief.budget) || undefined,
    interests: brief.interests.length > 0 ? brief.interests : undefined,
    transport_mode: brief.transportMode || undefined,
    pace: brief.pace || undefined,
    test_mode: testMode,
    agent_mode: agentMode,
  };
}

function applyBriefResponse(response: {
  departure_city?: string;
  destination_city?: string;
  days?: number;
  budget?: number;
  interests?: string[];
  transport_mode?: string;
  pace?: string;
}): TravelBrief {
  return {
    departureCity: response.departure_city || "",
    destinationCity: response.destination_city || "",
    days: response.days || "",
    budget: response.budget || "",
    interests: response.interests || [],
    transportMode: response.transport_mode || "train_taxi",
    pace: response.pace || "balanced",
  };
}

export default function App() {
  const [brief, setBrief] = useState<TravelBrief>(initialBrief);
  const [messages, setMessages] = useState<ChatMessage[]>(initialMessages);
  const [input, setInput] = useState("");
  const [taskId, setTaskId] = useState<string | null>(null);
  const [createError, setCreateError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isChatting, setIsChatting] = useState(false);
  const [testMode, setTestMode] = useState(true);
  const [agentMode, setAgentMode] = useState<AgentMode>("quick");
  const stream = useTravelPlanStream(taskId);

  const missing = useMemo(() => missingFields(brief), [brief]);
  const canGenerate = missing.length === 0;
  const isBusy = isSubmitting || isChatting || (!!taskId && !stream.plan && stream.status !== "failed");
  const planningActive = isSubmitting || Boolean(taskId) || Boolean(stream.plan) || Boolean(createError);
  const activityKey = `${taskId ?? "no-task"}-${stream.status}-${stream.events.length}-${stream.plan?.title ?? ""}-${isSubmitting}-${stream.assistantText.length}`;

  const acceptText = async (text: string) => {
    const normalized = text.trim();
    if (!normalized) {
      return;
    }

    setInput("");
    setCreateError("");
    setIsChatting(true);
    const assistantId = nextId("assistant");
    setMessages((current) => [
      ...current,
      { id: nextId("user"), role: "user", text: normalized },
      { id: assistantId, role: "assistant", text: "" },
    ]);

    try {
      const payload = chatPayloadFromBrief(normalized, brief, testMode, agentMode);
      const response = await streamChatTravelInfo(payload, (chunk) => {
        setMessages((current) =>
          current.map((message) =>
            message.id === assistantId ? { ...message, text: `${message.text}${chunk}` } : message,
          ),
        );
      });

      setBrief(applyBriefResponse(response));
      setMessages((current) =>
        current.map((message) => (message.id === assistantId ? { ...message, text: response.reply } : message)),
      );
    } catch (error) {
      setMessages((current) =>
        current.map((message) =>
          message.id === assistantId
            ? { ...message, text: error instanceof Error ? `处理消息失败：${error.message}` : "处理消息失败，请重试。" }
            : message,
        ),
      );
    } finally {
      setIsChatting(false);
    }
  };

  const handleGenerate = async (request: TravelPlanRequest) => {
    setCreateError("");
    setIsSubmitting(true);
    setTaskId(null);

    try {
      const response = await createTravelPlanTask({ ...request, test_mode: testMode, agent_mode: agentMode });
      setTaskId(response.task_id);
    } catch (error) {
      setCreateError(error instanceof Error ? error.message : "创建任务失败");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <main className="agent-shell">
      <div className="chat-frame">
        <AgentConversation
          messages={messages}
          brief={brief}
          input={input}
          disabled={isBusy}
          canGenerate={canGenerate}
          testMode={testMode}
          agentMode={agentMode}
          onInputChange={setInput}
          onSend={acceptText}
          onQuickReply={acceptText}
          onGenerate={handleGenerate}
          onTestModeChange={setTestMode}
          onAgentModeChange={setAgentMode}
          briefPanel={<TravelBriefPanel brief={brief} missing={missing} />}
          errorPanel={createError ? <StateView title="创建失败" message={createError} /> : undefined}
          activityKey={activityKey}
          planReady={Boolean(stream.plan)}
          planningActive={planningActive}
          planningText={stream.assistantText}
          progressPanel={
            <PlanProgress
              taskId={taskId}
              status={stream.status}
              events={stream.events}
              connected={stream.connected}
              polling={stream.polling}
              error={stream.error}
              creating={isSubmitting}
            />
          }
          planPanel={<PlanDetail plan={stream.plan} status={stream.status} onRefine={acceptText} />}
        />
      </div>
    </main>
  );
}
