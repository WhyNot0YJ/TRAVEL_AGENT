import { useEffect, useMemo, useState } from "react";
import { createTravelPlanTask } from "./api/client";
import type { TravelPlanRequest } from "./api/types";
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
    text: "告诉我出发地、目的地、天数、预算和偏好。我会边聊边整理需求，确认完整后生成路线。",
  },
];

const cityCandidates = ["北京", "上海", "广州", "深圳", "杭州", "苏州", "南京", "成都", "重庆", "西安", "厦门", "青岛", "长沙", "武汉"];
const interestCandidates = ["自然风光", "美食", "亲子", "历史文化", "博物馆", "夜景", "徒步", "购物", "摄影", "温泉"];

function nextId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function unique(values: string[]): string[] {
  return Array.from(new Set(values.filter(Boolean)));
}

function mergeBrief(brief: TravelBrief, text: string): TravelBrief {
  const next = { ...brief, interests: [...brief.interests] };
  const compact = text.replace(/\s+/g, "");

  const fromMatch = compact.match(/(.{2,8})(?:出发|起程|启程)/);
  if (fromMatch) {
    next.departureCity = cleanupCity(fromMatch[1]);
  }

  const toMatch = compact.match(/(?:去|到|目的地|玩在)(.{2,8})(?:\d|[一二三四五六七八九十两]|，|,|。|预算|玩|天|$)/);
  if (toMatch) {
    next.destinationCity = cleanupCity(toMatch[1]);
  }

  const mentionedCities = cityCandidates.filter((city) => compact.includes(city));
  if (!next.departureCity && mentionedCities.length > 1) {
    next.departureCity = mentionedCities[0];
  }
  if (!next.destinationCity && mentionedCities.length > 0) {
    next.destinationCity = mentionedCities[mentionedCities.length - 1];
  }

  const daysMatch = compact.match(/(\d+|[一二三四五六七八九十两])天/);
  if (daysMatch) {
    next.days = chineseNumber(daysMatch[1]);
  }

  const budgetMatch = compact.match(/(?:预算|人均|总共|大概)?(\d{3,6})(?:元|块|预算)?/);
  if (budgetMatch) {
    next.budget = Number(budgetMatch[1]);
  }

  const foundInterests = interestCandidates.filter((interest) => compact.includes(interest.replace(/\s+/g, "")));
  if (foundInterests.length > 0) {
    next.interests = unique([...next.interests, ...foundInterests]);
  }

  if (compact.includes("高铁") || compact.includes("火车")) {
    next.transportMode = compact.includes("步行") ? "train_walk" : "train_taxi";
  } else if (compact.includes("地铁")) {
    next.transportMode = "subway_walk";
  } else if (compact.includes("飞机") || compact.includes("航班")) {
    next.transportMode = "flight_taxi";
  }

  if (compact.includes("轻松") || compact.includes("舒缓") || compact.includes("慢")) {
    next.pace = "relaxed";
  } else if (compact.includes("紧凑") || compact.includes("多安排") || compact.includes("充实")) {
    next.pace = "intensive";
  } else if (compact.includes("均衡")) {
    next.pace = "balanced";
  }

  return next;
}

function cleanupCity(value: string): string {
  return value.replace(/[，,。.!！?？想要计划安排去到]/g, "").slice(-4);
}

function chineseNumber(value: string): number {
  const map: Record<string, number> = {
    一: 1,
    二: 2,
    两: 2,
    三: 3,
    四: 4,
    五: 5,
    六: 6,
    七: 7,
    八: 8,
    九: 9,
    十: 10,
  };
  return Number(value) || map[value] || 3;
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

function assistantReply(missing: string[]): string {
  if (missing.length === 0) {
    return "信息够了。我已经整理好 brief，可以生成路线。";
  }
  return `我还需要确认：${missing.join("、")}。你可以直接补一句，比如“预算 3000，喜欢美食和夜景”。`;
}

export default function App() {
  const [brief, setBrief] = useState<TravelBrief>(initialBrief);
  const [messages, setMessages] = useState<ChatMessage[]>(initialMessages);
  const [input, setInput] = useState("");
  const [taskId, setTaskId] = useState<string | null>(null);
  const [createError, setCreateError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [announcedTaskId, setAnnouncedTaskId] = useState("");
  const stream = useTravelPlanStream(taskId);

  const missing = useMemo(() => missingFields(brief), [brief]);
  const canGenerate = missing.length === 0;
  const isBusy = isSubmitting || (!!taskId && !stream.plan && stream.status !== "failed");
  const activityKey = `${taskId ?? "no-task"}-${stream.status}-${stream.events.length}-${stream.plan?.title ?? ""}-${isSubmitting}`;

  useEffect(() => {
    if (!taskId || !stream.plan || announcedTaskId === taskId) {
      return;
    }
    setAnnouncedTaskId(taskId);
    setMessages((current) => [
      ...current,
      { id: nextId("assistant"), role: "assistant", text: "行程已经生成。我把完整路线放在下面，可以继续根据预算、节奏或兴趣再调整。" },
    ]);
  }, [announcedTaskId, stream.plan, taskId]);

  const acceptText = (text: string) => {
    const normalized = text.trim();
    if (!normalized) {
      return;
    }

    const nextBrief = mergeBrief(brief, normalized);
    const nextMissing = missingFields(nextBrief);
    setBrief(nextBrief);
    setInput("");
    setMessages((current) => [
      ...current,
      { id: nextId("user"), role: "user", text: normalized },
      { id: nextId("assistant"), role: "assistant", text: assistantReply(nextMissing) },
    ]);
  };

  const handleGenerate = async (request: TravelPlanRequest) => {
    setCreateError("");
    setIsSubmitting(true);
    setTaskId(null);
    setAnnouncedTaskId("");
    setMessages((current) => [
      ...current,
      { id: nextId("system"), role: "system", text: "已提交规划任务，正在接收实时进度。" },
    ]);

    try {
      const response = await createTravelPlanTask(request);
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
          onInputChange={setInput}
          onSend={acceptText}
          onQuickReply={acceptText}
          onGenerate={handleGenerate}
          briefPanel={<TravelBriefPanel brief={brief} missing={missing} />}
          errorPanel={createError ? <StateView title="创建失败" message={createError} /> : undefined}
          activityKey={activityKey}
          planReady={Boolean(stream.plan)}
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
          planPanel={<PlanDetail plan={stream.plan} status={stream.status} />}
        />
      </div>
    </main>
  );
}
