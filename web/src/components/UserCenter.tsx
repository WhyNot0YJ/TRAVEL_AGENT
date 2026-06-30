import { useState } from "react";
import { useAuth } from "../hooks/useAuth";
import PlanLibrary from "./PlanLibrary";

type Tab = "plans" | "published" | "account";

export default function UserCenter() {
  const auth = useAuth();
  const [tab, setTab] = useState<Tab>("plans");

  return (
    <section data-testid="user-center">
      <div className="ta-section-head">
        <h2>用户中心</h2>
      </div>
      <div role="tablist" style={{ display: "flex", gap: 8, marginBottom: 14 }}>
        <button
          type="button"
          role="tab"
          aria-selected={tab === "plans"}
          className={`ta-action-button${tab === "plans" ? "" : " secondary"}`}
          onClick={() => setTab("plans")}
          data-testid="tab-plans"
        >
          我的计划
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={tab === "published"}
          className={`ta-action-button${tab === "published" ? "" : " secondary"}`}
          onClick={() => setTab("published")}
          data-testid="tab-published"
        >
          已发布
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={tab === "account"}
          className={`ta-action-button${tab === "account" ? "" : " secondary"}`}
          onClick={() => setTab("account")}
          data-testid="tab-account"
        >
          账号信息
        </button>
      </div>

      {tab === "plans" ? <PlanLibrary /> : null}
      {tab === "published" ? <PublishedView /> : null}
      {tab === "account" ? (
        <div className="ta-card" data-testid="account-panel">
          <p style={{ margin: 0, color: "var(--ta-slate)" }}>邮箱:{auth.user?.email ?? "—"}</p>
          <p style={{ margin: "6px 0 0", color: "var(--ta-slate)" }}>昵称:{auth.user?.display_name ?? "—"}</p>
          <p style={{ margin: "6px 0 0", color: "var(--ta-slate)" }}>状态:{auth.user?.status ?? "active"}</p>
        </div>
      ) : null}
    </section>
  );
}

function PublishedView() {
  return (
    <div data-testid="published-tab">
      <PlanLibrary />
      <p style={{ marginTop: 10, color: "var(--ta-slate)", fontSize: 13 }}>
        在我的计划列表中可以看到「已发布」标签。在详情页可以取消发布。
      </p>
    </div>
  );
}
