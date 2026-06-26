import { useEffect, useRef, type FormEvent, type ReactNode } from "react";
import type { AgentMode, TravelPlanRequest } from "../api/types";

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
  testMode: boolean;
  agentMode: AgentMode;
  onInputChange: (value: string) => void;
  onSend: (value: string) => void;
  onQuickReply: (value: string) => void;
  onGenerate: (request: TravelPlanRequest) => void;
  onTestModeChange: (enabled: boolean) => void;
  onAgentModeChange: (mode: AgentMode) => void;
  briefPanel: ReactNode;
  progressPanel: ReactNode;
  planPanel: ReactNode;
  errorPanel?: ReactNode;
  activityKey: string;
  planReady: boolean;
  planningActive: boolean;
  planningText: string;
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

function modeLabel(testMode: boolean, agentMode: AgentMode): string {
  if (testMode) {
    return "测试模式";
  }
  return agentMode === "expert" ? "真实 LLM · 专家" : "真实 LLM · 快速";
}

export default function AgentConversation({
  messages,
  brief,
  input,
  disabled,
  canGenerate,
  testMode,
  agentMode,
  onInputChange,
  onSend,
  onQuickReply,
  onGenerate,
  onTestModeChange,
  onAgentModeChange,
  briefPanel,
  progressPanel,
  planPanel,
  errorPanel,
  activityKey,
  planReady,
  planningActive,
  planningText,
}: AgentConversationProps) {
  const latestRef = useRef<HTMLDivElement>(null);
  const planCardRef = useRef<HTMLElement>(null);
  const showPlanningMessage = planningActive || planReady || Boolean(errorPanel);

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
      <header className="console-head">
        <div className="console-title">
          <span>Travel Agent</span>
          <strong>{canGenerate ? "需求已完整" : "正在确认需求"}</strong>
        </div>

        <div className="mode-bar" aria-label="运行模式">
          <label className="mode-toggle">
            <input
              type="checkbox"
              checked={testMode}
              disabled={disabled}
              onChange={(event) => onTestModeChange(event.target.checked)}
            />
            <span aria-hidden="true" />
            <strong>{testMode ? "测试" : "真实"}</strong>
          </label>
          <div className="agent-mode-tabs" aria-label="DeepSeek 模式">
            <button
              type="button"
              className={agentMode === "quick" ? "active" : ""}
              disabled={disabled}
              onClick={() => onAgentModeChange("quick")}
              title={testMode ? "测试模式不调用真实 LLM；切到真实模式后此选择生效" : "使用 deepseek-v4-flash，速度优先"}
            >
              快速
            </button>
            <button
              type="button"
              className={agentMode === "expert" ? "active" : ""}
              disabled={disabled}
              onClick={() => onAgentModeChange("expert")}
              title={testMode ? "测试模式不调用真实 LLM；切到真实模式后此选择生效" : "使用 deepseek-v4-pro，更强推理（成本更高）"}
            >
              专家
            </button>
          </div>
        </div>
      </header>

      <div className="message-rail" data-testid="message-rail">
        <div className="mode-strip">{modeLabel(testMode, agentMode)}</div>
        {messages.map((message) => (
          <article className={`message ${message.role}`} data-testid={`message-${message.role}`} key={message.id}>
            <p>{message.text || "正在输入..."}</p>
          </article>
        ))}

        {canGenerate && !showPlanningMessage ? (
          <article className="message assistant assistant-result-message" data-testid="planning-message" ref={planCardRef}>
            <p>信息齐了，可以开始生成行程。</p>
            <div className="assistant-result-content">
              <div className="result-block result-brief">{briefPanel}</div>
              <div className="confirm-actions">
                <button type="button" data-testid="generate-plan" onClick={() => onGenerate(buildRequest(brief))}>
                  生成行程
                </button>
                <button type="button" onClick={() => onInputChange("我想修改一下：")}>
                  修改需求
                </button>
              </div>
            </div>
          </article>
        ) : null}

        {showPlanningMessage ? (
          <article className="message assistant assistant-result-message" data-testid="planning-message" ref={planCardRef}>
            {planningText ? (
              <p className="streaming-answer" data-testid="planning-stream-text">
                {planningText}
              </p>
            ) : null}
            <div className="assistant-result-content">
              {errorPanel ? <div className="result-block result-error">{errorPanel}</div> : null}
              {planReady ? (
                <div className="result-block result-plan">{planPanel}</div>
              ) : (
                <div className="result-block result-progress">{progressPanel}</div>
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
        <button className="composer-tool" type="button" disabled aria-label="更多">
          +
        </button>
        <label className="sr-only" htmlFor="travel-chat-input">
          输入旅行需求
        </label>
        <textarea
          id="travel-chat-input"
          data-testid="chat-input"
          value={input}
          onChange={(event) => onInputChange(event.target.value)}
          placeholder="说说你想去哪里、玩几天、预算和偏好"
          rows={1}
          disabled={disabled}
        />
        <button className="send-action" type="submit" data-testid="send-message" disabled={disabled || input.trim().length === 0}>
          发送
        </button>
      </form>
    </section>
  );
}
