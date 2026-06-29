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
  travelers: "",
  dateRange: "任意",
  transportMode: "任意",
  pace: "适中",
  walkingTolerance: "任意",
  hotelArea: "任意",
  mustVisit: [],
  avoid: [],
  travelerType: "无要求",
  budgetType: "总预算",
  budgetIncludes: ["住宿", "餐饮", "门票", "市内交通"],
};

const initialMessages: ChatMessage[] = [
  {
    id: "welcome",
    role: "assistant",
    text: "你好，我会先确认出发地、目的地、天数、预算、兴趣和出行人数。其他偏好没有要求时会用默认值，信息齐了以后我会给你一张确认卡。",
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
  if (!brief.travelers) {
    missing.push("出行人数");
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
    travelers: Number(brief.travelers) || undefined,
    date_range: brief.dateRange || undefined,
    transport_mode: brief.transportMode || undefined,
    pace: brief.pace || undefined,
    walking_tolerance: brief.walkingTolerance || undefined,
    hotel_area: brief.hotelArea || undefined,
    must_visit: brief.mustVisit.length > 0 ? brief.mustVisit : undefined,
    avoid: brief.avoid.length > 0 ? brief.avoid : undefined,
    traveler_type: brief.travelerType || undefined,
    budget_type: brief.budgetType || undefined,
    budget_includes: brief.budgetIncludes.length > 0 ? brief.budgetIncludes : undefined,
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
  travelers?: number;
  date_range?: string;
  transport_mode?: string;
  pace?: string;
  walking_tolerance?: string;
  hotel_area?: string;
  must_visit?: string[];
  avoid?: string[];
  traveler_type?: string;
  budget_type?: string;
  budget_includes?: string[];
}): TravelBrief {
  return {
    departureCity: response.departure_city || "",
    destinationCity: response.destination_city || "",
    days: response.days || "",
    budget: response.budget || "",
    interests: response.interests || [],
    travelers: response.travelers || "",
    dateRange: response.date_range || "任意",
    transportMode: response.transport_mode || "任意",
    pace: response.pace || "适中",
    walkingTolerance: response.walking_tolerance || "任意",
    hotelArea: response.hotel_area || "任意",
    mustVisit: response.must_visit || [],
    avoid: response.avoid || [],
    travelerType: response.traveler_type || "无要求",
    budgetType: response.budget_type || "总预算",
    budgetIncludes: response.budget_includes || ["住宿", "餐饮", "门票", "市内交通"],
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
  const visiblePlan = stream.plan ?? stream.draftPlan;
  const activityKey = `${taskId ?? "no-task"}-${stream.status}-${stream.events.length}-${visiblePlan?.title ?? ""}-${isSubmitting}-${stream.assistantText.length}`;

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
              pois={stream.pois}
              weather={stream.weather}
              routes={stream.routes}
              budget={stream.budget}
              draftDays={stream.draftDays}
            />
          }
          planPanel={
            <PlanDetail
              plan={visiblePlan}
              status={stream.status}
              draft={!stream.plan && Boolean(stream.draftPlan)}
              onRefine={acceptText}
            />
          }
        />
      </div>
    </main>
  );
}
