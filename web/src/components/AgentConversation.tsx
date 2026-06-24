import { useEffect, useRef, type FormEvent, type ReactNode } from "react";
import type { TravelPlanRequest } from "../api/types";

export interface TravelBrief {
  departureCity: string;
  destinationCity: string;
  days: number | "";
  budget: number | "";
  interests: string[];
  transportMode: string;
  pace: string;
}

export interface ChatMessage {
  id: string;
  role: "assistant" | "user" | "system";
  text: string;
}

interface AgentConversationProps {
  messages: ChatMessage[];
  brief: TravelBrief;
  input: string;
  disabled: boolean;
  canGenerate: boolean;
  onInputChange: (value: string) => void;
  onSend: (value: string) => void;
  onQuickReply: (value: string) => void;
  onGenerate: (request: TravelPlanRequest) => void;
  briefPanel: ReactNode;
  progressPanel: ReactNode;
  planPanel: ReactNode;
  errorPanel?: ReactNode;
  activityKey: string;
  planReady: boolean;
}

const quickReplies = ["上海出发，杭州 3 天，预算 3000", "想轻松一点，喜欢美食和自然风光", "高铁优先，少走回头路"];

function buildRequest(brief: TravelBrief): TravelPlanRequest {
  return {
    departure_city: brief.departureCity,
    destination_city: brief.destinationCity,
    days: Number(brief.days),
    budget: Number(brief.budget),
    interests: brief.interests,
    transport_mode: brief.transportMode,
    pace: brief.pace,
  };
}

export default function AgentConversation({
  messages,
  brief,
  input,
  disabled,
  canGenerate,
  onInputChange,
  onSend,
  onQuickReply,
  onGenerate,
  briefPanel,
  progressPanel,
  planPanel,
  errorPanel,
  activityKey,
  planReady,
}: AgentConversationProps) {
  const latestRef = useRef<HTMLDivElement>(null);
  const planCardRef = useRef<HTMLElement>(null);
  const showResultMessage = canGenerate || disabled || planReady || Boolean(errorPanel);
  const resultTitle = planReady ? "路线已生成" : disabled ? "正在生成路线" : errorPanel ? "需要处理" : "需求已整理";

  useEffect(() => {
    if (planReady) {
      planCardRef.current?.scrollIntoView({ block: "start", behavior: "smooth" });
      return;
    }
    latestRef.current?.scrollIntoView({ block: "end", behavior: "smooth" });
  }, [messages.length, canGenerate, disabled, activityKey, planReady]);

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onSend(input);
  };

  return (
    <section className="conversation-console" aria-label="旅行助手对话">
      <div className="console-head">
        <div className="window-controls" aria-hidden="true">
          <span />
          <span />
          <span />
        </div>
        <div>
          <span>Travel Agent</span>
          <strong>{canGenerate ? "需求已完整" : "正在确认需求"}</strong>
        </div>
      </div>

      <div className="message-rail" data-testid="message-rail">
        {messages.map((message) => (
          <article className={`message ${message.role}`} key={message.id}>
            <span>{message.role === "assistant" ? "助手" : message.role === "user" ? "你" : "状态"}</span>
            <p>{message.text}</p>
          </article>
        ))}

        {showResultMessage ? (
          <article className="message assistant assistant-result-message" ref={planCardRef}>
            <span>助手 · {resultTitle}</span>
            <div className="assistant-result-content">
              {errorPanel ? <div className="result-block result-error">{errorPanel}</div> : null}
              {planReady ? (
                <div className="result-block result-plan">{planPanel}</div>
              ) : disabled ? (
                <div className="result-block result-progress">{progressPanel}</div>
              ) : (
                <div className="result-block result-brief">{briefPanel}</div>
              )}
            </div>
          </article>
        ) : null}

        <div ref={latestRef} aria-hidden="true" />
      </div>

      <div className="quick-replies" aria-label="快捷输入">
        {quickReplies.map((reply) => (
          <button type="button" key={reply} onClick={() => onQuickReply(reply)} disabled={disabled}>
            {reply}
          </button>
        ))}
      </div>

      <form className="chat-composer" onSubmit={handleSubmit}>
        <label className="sr-only" htmlFor="travel-chat-input">
          输入旅行需求
        </label>
        <textarea
          id="travel-chat-input"
          data-testid="chat-input"
          value={input}
          onChange={(event) => onInputChange(event.target.value)}
          placeholder="告诉我你从哪里出发、想去哪、玩几天、预算和偏好"
          rows={3}
          disabled={disabled}
        />
        <div className="composer-actions">
          <button type="submit" data-testid="send-message" disabled={disabled || input.trim().length === 0}>
            发送
          </button>
          <button
            type="button"
            className="generate-action"
            data-testid="generate-plan"
            disabled={disabled || !canGenerate}
            onClick={() => onGenerate(buildRequest(brief))}
          >
            生成行程
          </button>
        </div>
      </form>
    </section>
  );
}
