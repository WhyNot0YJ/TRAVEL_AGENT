import type { FormEvent } from "react";
import type { TravelPlanRequest } from "../api/types";
import type { FormState } from "../formState";

interface TravelPlanFormProps {
  value: FormState;
  disabled: boolean;
  onChange: (value: FormState) => void;
  onSubmit: (request: TravelPlanRequest) => void;
}

const transportOptions = [
  { value: "train_taxi", label: "高铁+打车" },
  { value: "train_walk", label: "高铁+步行" },
  { value: "subway_walk", label: "地铁+步行" },
  { value: "flight_taxi", label: "飞机+打车" },
];

const paceOptions = [
  { value: "relaxed", label: "舒缓" },
  { value: "balanced", label: "均衡" },
  { value: "intensive", label: "紧凑" },
];

function splitInterests(value: string): string[] {
  return value
    .split(/[,，]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

export default function TravelPlanForm({ value, disabled, onChange, onSubmit }: TravelPlanFormProps) {
  const update = <Key extends keyof FormState>(key: Key, nextValue: FormState[Key]) => {
    onChange({ ...value, [key]: nextValue });
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onSubmit({
      departure_city: value.departureCity.trim(),
      destination_city: value.destinationCity.trim(),
      days: value.days,
      budget: value.budget,
      interests: splitInterests(value.interests),
      transport_mode: value.transportMode,
      pace: value.pace,
    });
  };

  return (
    <form className="planner-form" onSubmit={handleSubmit}>
      <div className="form-grid">
        <label>
          <span>出发城市</span>
          <input
            value={value.departureCity}
            onChange={(event) => update("departureCity", event.target.value)}
            required
            autoComplete="address-level2"
          />
        </label>
        <label>
          <span>目的地</span>
          <input
            value={value.destinationCity}
            onChange={(event) => update("destinationCity", event.target.value)}
            required
            autoComplete="address-level2"
          />
        </label>
        <label>
          <span>天数</span>
          <input
            type="number"
            min={1}
            max={14}
            value={value.days}
            onChange={(event) => update("days", Number(event.target.value))}
            required
          />
        </label>
        <label>
          <span>预算</span>
          <input
            type="number"
            min={1}
            step={100}
            value={value.budget}
            onChange={(event) => update("budget", Number(event.target.value))}
            required
          />
        </label>
      </div>

      <label>
        <span>兴趣</span>
        <input
          value={value.interests}
          onChange={(event) => update("interests", event.target.value)}
          placeholder="自然风光，美食，亲子"
        />
      </label>

      <label>
        <span>交通方式</span>
        <select value={value.transportMode} onChange={(event) => update("transportMode", event.target.value)}>
          {transportOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </label>

      <fieldset>
        <legend>节奏</legend>
        <div className="segmented">
          {paceOptions.map((option) => (
            <label key={option.value} className={value.pace === option.value ? "selected" : ""}>
              <input
                type="radio"
                name="pace"
                value={option.value}
                checked={value.pace === option.value}
                onChange={() => update("pace", option.value)}
              />
              <span>{option.label}</span>
            </label>
          ))}
        </div>
      </fieldset>

      <button className="primary-action" type="submit" disabled={disabled}>
        {disabled ? "生成中" : "生成路线"}
      </button>
    </form>
  );
}
