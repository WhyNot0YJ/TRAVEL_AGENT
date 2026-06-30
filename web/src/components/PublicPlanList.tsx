import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import { usePublicPlans } from "../hooks/usePublicPlans";
import EmptyState from "./EmptyState";
import PublicPlanCard from "./PublicPlanCard";

export default function PublicPlanList() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialQuery = searchParams.get("q") ?? "";
  const [input, setInput] = useState(initialQuery);
  const [sort, setSort] = useState<"hot" | "latest">((searchParams.get("sort") as "hot" | "latest") ?? "hot");
  const debounced = useDebouncedValue(input, 300);

  const filter = useMemo(
    () => ({
      query: debounced.trim() || undefined,
      sort,
      page: 1,
      pageSize: 20,
    }),
    [debounced, sort],
  );

  const { items, total, loading, error } = usePublicPlans(filter);

  useEffect(() => {
    const params: Record<string, string> = {};
    if (filter.query) params.q = filter.query;
    if (filter.sort) params.sort = filter.sort;
    setSearchParams(params, { replace: true });
  }, [filter.query, filter.sort, setSearchParams]);

  return (
    <section data-testid="public-list-page">
      <div className="ta-section-head">
        <h2>公开旅行计划</h2>
        <span style={{ color: "var(--ta-slate)", fontSize: 13 }}>共 {total} 条</span>
      </div>
      <div className="ta-library-toolbar">
        <input
          type="search"
          value={input}
          onChange={(event) => setInput(event.target.value)}
          placeholder="搜索目的地、主题、标签"
          aria-label="搜索公开计划"
          data-testid="public-search-input"
        />
        <select
          value={sort}
          onChange={(event) => setSort(event.target.value as "hot" | "latest")}
          aria-label="排序方式"
          data-testid="public-sort"
        >
          <option value="hot">综合热度</option>
          <option value="latest">最新发布</option>
        </select>
      </div>
      {error ? <p style={{ color: "var(--ta-sunset-coral)" }}>{error}</p> : null}
      {loading ? (
        <p style={{ color: "var(--ta-slate)" }}>正在加载…</p>
      ) : items.length === 0 ? (
        <EmptyState title="没有找到相关计划" description="换个目的地或主题试试。" testId="public-empty" />
      ) : (
        <div className="ta-grid" data-testid="public-list">
          {items.map((plan, index) => (
            <PublicPlanCard key={plan.public_plan_id} plan={plan} rank={sort === "hot" ? index + 1 : undefined} />
          ))}
        </div>
      )}
    </section>
  );
}
