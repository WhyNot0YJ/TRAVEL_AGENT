import { useState } from "react";
import { createTravelPlanTask } from "./api/client";
import type { TravelPlanRequest } from "./api/types";
import PlanDetail from "./components/PlanDetail";
import PlanProgress from "./components/PlanProgress";
import StateView from "./components/StateView";
import TravelPlanForm from "./components/TravelPlanForm";
import { initialFormState } from "./formState";
import { useTravelPlanStream } from "./hooks/useTravelPlanStream";

export default function App() {
  const [form, setForm] = useState(initialFormState);
  const [taskId, setTaskId] = useState<string | null>(null);
  const [createError, setCreateError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const stream = useTravelPlanStream(taskId);

  const handleSubmit = async (request: TravelPlanRequest) => {
    setCreateError("");
    setIsSubmitting(true);
    setTaskId(null);
    try {
      const response = await createTravelPlanTask(request);
      setTaskId(response.task_id);
    } catch (error) {
      setCreateError(error instanceof Error ? error.message : "创建任务失败");
    } finally {
      setIsSubmitting(false);
    }
  };

  const isBusy =
    isSubmitting ||
    (!!taskId && !stream.plan && stream.status !== "failed");

  return (
    <main className="app-shell">
      <section className="tool-panel">
        <div className="title-block">
          <p>Travel Agent</p>
          <h1>旅行路线规划</h1>
        </div>
        <TravelPlanForm value={form} disabled={isBusy} onChange={setForm} onSubmit={handleSubmit} />
      </section>

      <section className="result-panel">
        {createError ? <StateView title="创建失败" message={createError} /> : null}
        <PlanProgress
          taskId={taskId}
          status={stream.status}
          events={stream.events}
          connected={stream.connected}
          polling={stream.polling}
          error={stream.error}
        />
        <PlanDetail plan={stream.plan} />
      </section>
    </main>
  );
}
