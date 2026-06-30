import { Link, NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useState } from "react";
import { useAuth } from "../hooks/useAuth";
import { ToastProvider } from "./Toast";

export default function AppShell() {
  const auth = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [searchValue, setSearchValue] = useState("");

  const submitSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const query = searchValue.trim();
    if (!query) {
      return;
    }
    navigate(`/public?q=${encodeURIComponent(query)}`);
  };

  const onLogout = async () => {
    await auth.logout();
    if (location.pathname.startsWith("/me")) {
      navigate("/");
    }
  };

  return (
    <ToastProvider>
      <div className="ta-app">
        <header className="ta-topbar" data-testid="app-topbar">
          <Link to="/" className="ta-topbar-brand" aria-label="Travel Agent 首页">
            Travel Agent
          </Link>
          <form className="ta-topbar-search" role="search" onSubmit={submitSearch}>
            <input
              type="search"
              value={searchValue}
              onChange={(event) => setSearchValue(event.target.value)}
              placeholder="搜索目的地、主题或公开计划"
              aria-label="搜索公开计划"
              data-testid="topbar-search"
            />
          </form>
          <div className="ta-topbar-actions">
            {auth.status === "authenticated" ? (
              <>
                <NavLink to="/planner" className="ta-primary" data-testid="nav-planner">
                  开始规划
                </NavLink>
                <NavLink to="/me" data-testid="nav-me">
                  {auth.user?.display_name ?? "我的"}
                </NavLink>
                <button type="button" onClick={onLogout} data-testid="nav-logout">
                  退出
                </button>
              </>
            ) : auth.status === "anonymous" ? (
              <>
                <NavLink to="/login" data-testid="nav-login">
                  登录
                </NavLink>
                <NavLink to="/login?mode=register" className="ta-primary" data-testid="nav-register">
                  注册
                </NavLink>
              </>
            ) : null}
          </div>
        </header>
        <main className="ta-content" id="content">
          <Outlet />
        </main>
        <div className="ta-bottom-nav" data-testid="bottom-nav">
          <nav>
            <NavLink to="/" end className={({ isActive }) => (isActive ? "active" : undefined)}>
              首页
            </NavLink>
            <NavLink to="/planner" className={({ isActive }) => (isActive ? "active" : undefined)}>
              生成
            </NavLink>
            <NavLink to="/me" className={({ isActive }) => (isActive ? "active" : undefined)}>
              我的
            </NavLink>
          </nav>
        </div>
      </div>
    </ToastProvider>
  );
}
