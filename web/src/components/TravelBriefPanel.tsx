import type { TravelBrief } from "./AgentConversation";

interface TravelBriefPanelProps {
  brief: TravelBrief;
  missing: string[];
}

const transportLabels: Record<string, string> = {
  any: "任意",
  任意: "任意",
  train_taxi: "高铁 + 打车",
  "高铁 + 打车": "高铁 + 打车",
  train_walk: "高铁 + 步行",
  "高铁 + 步行": "高铁 + 步行",
  train_subway: "高铁 + 地铁",
  "高铁 + 地铁": "高铁 + 地铁",
  subway_walk: "地铁 + 步行",
  "地铁 + 步行": "地铁 + 步行",
  subway_taxi: "地铁 + 打车",
  "地铁 + 打车": "地铁 + 打车",
  flight_taxi: "飞机 + 打车",
  "飞机 + 打车": "飞机 + 打车",
  flight_subway: "飞机 + 地铁",
  "飞机 + 地铁": "飞机 + 地铁",
  walk_taxi: "步行 + 打车",
  "步行 + 打车": "步行 + 打车",
};

const paceLabels: Record<string, string> = {
  relaxed: "轻松",
  轻松: "轻松",
  balanced: "适中",
  适中: "适中",
  均衡: "适中",
  intensive: "紧凑",
  紧凑: "紧凑",
};

const walkingLabels: Record<string, string> = {
  any: "任意",
  任意: "任意",
  low: "低",
  低: "低",
  medium: "中",
  中: "中",
  high: "高",
  高: "高",
};

const budgetTypeLabels: Record<string, string> = {
  total: "总预算",
  总预算: "总预算",
  per_person: "人均预算",
  人均预算: "人均预算",
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
        <span>已整理</span>
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
          <dt>人数</dt>
          <dd>{brief.travelers === "" ? "待确认" : `${brief.travelers} 人`}</dd>
        </div>
        <div>
          <dt>兴趣</dt>
          <dd>{valueOrEmpty(brief.interests)}</dd>
        </div>
      </dl>

      <div className="brief-defaults" aria-label="默认偏好">
        <span>默认项与可补充项</span>
        <dl>
          <div>
            <dt>日期</dt>
            <dd>{brief.dateRange || "任意"}</dd>
          </div>
          <div>
            <dt>交通</dt>
            <dd>{transportLabels[brief.transportMode] ?? brief.transportMode}</dd>
          </div>
          <div>
            <dt>节奏</dt>
            <dd>{paceLabels[brief.pace] ?? brief.pace}</dd>
          </div>
          <div>
            <dt>步行</dt>
            <dd>{walkingLabels[brief.walkingTolerance] ?? brief.walkingTolerance}</dd>
          </div>
          <div>
            <dt>酒店</dt>
            <dd>{brief.hotelArea || "任意"}</dd>
          </div>
          <div>
            <dt>同行</dt>
            <dd>{brief.travelerType || "无要求"}</dd>
          </div>
          <div>
            <dt>预算口径</dt>
            <dd>{budgetTypeLabels[brief.budgetType] ?? brief.budgetType}</dd>
          </div>
          <div>
            <dt>包含</dt>
            <dd>{valueOrEmpty(brief.budgetIncludes)}；不含往返大交通</dd>
          </div>
          <div>
            <dt>必去</dt>
            <dd>{brief.mustVisit.length > 0 ? brief.mustVisit.join("、") : "无要求"}</dd>
          </div>
          <div>
            <dt>避开</dt>
            <dd>{brief.avoid.length > 0 ? brief.avoid.join("、") : "无要求"}</dd>
          </div>
        </dl>
      </div>

      {missing.length > 0 ? (
        <div className="missing-strip">
          <span>还需要</span>
          <p>{missing.join("、")}</p>
        </div>
      ) : null}
    </aside>
  );
}
