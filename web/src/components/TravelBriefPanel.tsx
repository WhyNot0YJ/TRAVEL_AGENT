import type { TravelBrief } from "./AgentConversation";

interface TravelBriefPanelProps {
  brief: TravelBrief;
  missing: string[];
}

const transportLabels: Record<string, string> = {
  train_taxi: "高铁+打车",
  train_walk: "高铁+步行",
  subway_walk: "地铁+步行",
  flight_taxi: "飞机+打车",
};

const paceLabels: Record<string, string> = {
  relaxed: "舒缓",
  balanced: "均衡",
  intensive: "紧凑",
};

function valueOrEmpty(value: string | number | string[]): string {
  if (Array.isArray(value)) {
    return value.length > 0 ? value.join("、") : "待确认";
  }
  if (value === "") {
    return "待确认";
  }
  return String(value);
}

export default function TravelBriefPanel({ brief, missing }: TravelBriefPanelProps) {
  return (
    <aside className="brief-panel" aria-label="旅行需求摘要" data-testid="brief-panel">
      <div className="brief-head">
        <span>Live brief</span>
        <strong>{missing.length === 0 ? "完整" : `${missing.length} 项待确认`}</strong>
      </div>

      <dl className="brief-grid">
        <div>
          <dt>出发</dt>
          <dd>{valueOrEmpty(brief.departureCity)}</dd>
        </div>
        <div>
          <dt>目的地</dt>
          <dd>{valueOrEmpty(brief.destinationCity)}</dd>
        </div>
        <div>
          <dt>天数</dt>
          <dd>{brief.days === "" ? "待确认" : `${brief.days} 天`}</dd>
        </div>
        <div>
          <dt>预算</dt>
          <dd>{brief.budget === "" ? "待确认" : `¥${Number(brief.budget).toLocaleString("zh-CN")}`}</dd>
        </div>
        <div>
          <dt>兴趣</dt>
          <dd>{valueOrEmpty(brief.interests)}</dd>
        </div>
        <div>
          <dt>交通</dt>
          <dd>{transportLabels[brief.transportMode] ?? brief.transportMode}</dd>
        </div>
        <div>
          <dt>节奏</dt>
          <dd>{paceLabels[brief.pace] ?? brief.pace}</dd>
        </div>
      </dl>

      {missing.length > 0 ? (
        <div className="missing-strip">
          <span>还需要</span>
          <p>{missing.join("、")}</p>
        </div>
      ) : null}
    </aside>
  );
}
