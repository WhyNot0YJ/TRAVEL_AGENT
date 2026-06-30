import { useCallback, useEffect, useState } from "react";
import {
  deleteMyPlan,
  getMyPlan,
  listMyPlans,
  patchMyPlan,
  publishMyPlan,
  unpublishMyPlan,
} from "../api/client";
import type { PatchPlanRequest, PublishPlanRequest, UserPlan } from "../api/types";

interface PlanLibraryState {
  items: UserPlan[];
  total: number;
  loading: boolean;
  error: string;
}

interface PlanLibraryFilter {
  query?: string;
  visibility?: string;
  publishStatus?: string;
  page?: number;
  pageSize?: number;
}

export function usePlanLibrary(filter: PlanLibraryFilter) {
  const [state, setState] = useState<PlanLibraryState>({ items: [], total: 0, loading: true, error: "" });

  const reload = useCallback(async () => {
    setState((prev) => ({ ...prev, loading: true, error: "" }));
    try {
      const data = await listMyPlans({
        q: filter.query,
        visibility: filter.visibility,
        publish_status: filter.publishStatus,
        page: filter.page,
        page_size: filter.pageSize,
      });
      setState({ items: data.items, total: data.total, loading: false, error: "" });
    } catch (error) {
      setState({ items: [], total: 0, loading: false, error: error instanceof Error ? error.message : "加载失败" });
    }
  }, [filter.query, filter.visibility, filter.publishStatus, filter.page, filter.pageSize]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { ...state, reload };
}

export function usePlanDetail(planId: string | undefined) {
  const [plan, setPlan] = useState<UserPlan | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const reload = useCallback(async () => {
    if (!planId) {
      setPlan(null);
      return;
    }
    setLoading(true);
    setError("");
    try {
      const data = await getMyPlan(planId);
      setPlan(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, [planId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { plan, loading, error, reload };
}

export async function patchPlan(planId: string, payload: PatchPlanRequest) {
  return patchMyPlan(planId, payload);
}

export async function deletePlan(planId: string) {
  await deleteMyPlan(planId);
}

export async function publishPlan(planId: string, payload: PublishPlanRequest = {}) {
  return publishMyPlan(planId, payload);
}

export async function unpublishPlan(planId: string) {
  await unpublishMyPlan(planId);
}
