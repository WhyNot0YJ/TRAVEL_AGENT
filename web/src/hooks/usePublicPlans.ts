import { useCallback, useEffect, useState } from "react";
import { getPublicPlan, listPublicPlans, savePublicPlanCopy } from "../api/client";
import type { PublicPlan } from "../api/types";

interface PublicPlansFilter {
  query?: string;
  destinationCity?: string;
  days?: number;
  interest?: string;
  sort?: "hot" | "latest";
  page?: number;
  pageSize?: number;
}

interface PublicPlansState {
  items: PublicPlan[];
  total: number;
  loading: boolean;
  error: string;
}

export function usePublicPlans(filter: PublicPlansFilter) {
  const [state, setState] = useState<PublicPlansState>({ items: [], total: 0, loading: true, error: "" });

  const reload = useCallback(async () => {
    setState((prev) => ({ ...prev, loading: true, error: "" }));
    try {
      const data = await listPublicPlans({
        q: filter.query,
        destination_city: filter.destinationCity,
        days: filter.days,
        interest: filter.interest,
        sort: filter.sort,
        page: filter.page,
        page_size: filter.pageSize,
      });
      setState({ items: data.items, total: data.total, loading: false, error: "" });
    } catch (err) {
      setState({ items: [], total: 0, loading: false, error: err instanceof Error ? err.message : "加载失败" });
    }
  }, [filter.query, filter.destinationCity, filter.days, filter.interest, filter.sort, filter.page, filter.pageSize]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { ...state, reload };
}

export function usePublicPlanDetail(publicPlanId: string | undefined) {
  const [plan, setPlan] = useState<PublicPlan | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!publicPlanId) {
      setPlan(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError("");
    getPublicPlan(publicPlanId)
      .then((data) => {
        if (!cancelled) setPlan(data);
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : "加载失败");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [publicPlanId]);

  return { plan, loading, error };
}

export async function savePublicCopy(publicPlanId: string) {
  return savePublicPlanCopy(publicPlanId);
}
