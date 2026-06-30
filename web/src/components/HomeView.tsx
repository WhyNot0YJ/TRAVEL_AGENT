import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { getCurrent } from "../api/client";
import type { CurrentResponse } from "../api/types";
import { useAuth } from "../hooks/useAuth";
import { usePublicPlans } from "../hooks/usePublicPlans";
import EmptyState from "./EmptyState";
import PublicPlanCard from "./PublicPlanCard";

export default function HomeView() {
  const auth = useAuth();
  const navigate = useNavigate();
  const [current, setCurrent] = useState<CurrentResponse | null>(null);
  const [searchInput, setSearchInput] = useState("");

  const hot = usePublicPlans({ sort: "hot", page: 1, pageSize: 6 });
  const recommended = usePublicPlans({ sort: "latest", page: 1, pageSize: 6 });

  useEffect(() => {
    if (auth.status !== "authenticated") {
      setCurrent(null);
      return;
    }
    let cancelled = false;
    getCurrent()
      .then((data) => {
        if (!cancelled) setCurrent(data);
      })
      .catch(() => {
        if (!cancelled) setCurrent(null);
      });
    return () => {
      cancelled = true;
    };
  }, [auth.status]);

  const submitSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmed = searchInput.trim();
    if (!trimmed) return;
    navigate(`/public?q=${encodeURIComponent(trimmed)}`);
  };

  return (
    <div data-testid="home-view">
      {auth.status === "authenticated" ? (
        <section className="ta-section" data-testid="home-current">
          <div className="ta-section-head">
            <h2>当前计划</h2>
          </div>
          {current?.running_task ? (
            <div className="ta-card ta-current-row">
              <div>
                <strong>{current.running_task.destination_city || "正在生成"}</strong>
                <span>状态：{current.running_task.status} · 更新于 {current.running_task.updated_at.slice(11, 16)}</span>
              </div>
              <Link to="/planner" className="ta-action-button" data-testid="home-resume">
                继续查看
              </Link>
            </div>
          ) : current?.latest_plan ? (
            <div className="ta-card ta-current-row">
              <div>
                <strong>{current.latest_plan.title}</strong>
                <span>
                  {current.latest_plan.destination_city} · {current.latest_plan.days} 日 · 最近更新{" "}
                  {current.latest_plan.updated_at.slice(0, 10)}
                </span>
              </div>
              <Link to={`/me/plans/${current.latest_plan.plan_id}`} className="ta-action-button" data-testid="home-latest">
                查看计划
              </Link>
            </div>
          ) : (
            <EmptyState
              title="开始规划你的第一条路线"
              description="只需描述出发地、目的地和兴趣，我们会生成一份可保存的旅行计划。"
              action={{ label: "开始规划", onClick: () => navigate("/planner") }}
              testId="home-start"
            />
          )}
        </section>
      ) : null}

      <section className="ta-section">
        <div className="ta-section-head">
          <h2>开始规划</h2>
        </div>
        <div className="ta-card" style={{ display: "grid", gap: 12 }}>
          <form role="search" onSubmit={submitSearch} style={{ display: "flex", gap: 8 }}>
            <input
              type="search"
              value={searchInput}
              onChange={(event) => setSearchInput(event.target.value)}
              placeholder="想去哪？输入目的地或主题"
              aria-label="搜索目的地或主题"
              data-testid="home-search-input"
              style={{
                flex: 1,
                height: 38,
                padding: "0 12px",
                border: "1px solid var(--ta-mist)",
                borderRadius: "var(--ta-radius)",
                background: "var(--ta-paper)",
                font: "inherit",
              }}
            />
            <button type="submit" className="ta-action-button" data-testid="home-search-submit">
              搜索
            </button>
          </form>
          <Link to="/planner" className="ta-action-button" data-testid="home-new-plan">
            新建对话式旅行计划
          </Link>
        </div>
      </section>

      <section className="ta-section" data-testid="home-hot">
        <div className="ta-section-head">
          <h2>热门排行</h2>
          <Link to="/public?sort=hot">查看更多</Link>
        </div>
        {hot.loading ? (
          <p style={{ color: "var(--ta-slate)" }}>正在加载热门计划…</p>
        ) : hot.items.length === 0 ? (
          <EmptyState
            title="还没有公开计划"
            description="先生成并发布一条路线，让大家看到你的旅行思路。"
            testId="home-hot-empty"
          />
        ) : (
          <div className="ta-grid">
            {hot.items.map((plan, index) => (
              <PublicPlanCard key={plan.public_plan_id} plan={plan} rank={index + 1} />
            ))}
          </div>
        )}
      </section>

      <section className="ta-section" data-testid="home-recommended">
        <div className="ta-section-head">
          <h2>为你推荐</h2>
          <Link to="/public?sort=latest">查看更多</Link>
        </div>
        {recommended.items.length === 0 ? (
          <EmptyState title="暂时没有推荐" description="先看看热门排行吧。" testId="home-recommended-empty" />
        ) : (
          <div className="ta-grid">
            {recommended.items.map((plan) => (
              <PublicPlanCard key={plan.public_plan_id} plan={plan} />
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
