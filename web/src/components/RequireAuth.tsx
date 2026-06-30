import { useEffect } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

interface RequireAuthProps {
  children: React.ReactElement;
}

// RequireAuth blocks /me/* and other authenticated routes when the visitor is
// anonymous and bounces them through /login with a return_to query parameter
// so we resume the original action after sign-in.
export default function RequireAuth({ children }: RequireAuthProps) {
  const auth = useAuth();
  const location = useLocation();

  useEffect(() => {
    // Avoid surfacing the gate while we're still resolving session state — it
    // would briefly render a redirect that immediately reverses.
  }, [auth.status]);

  if (auth.status === "loading") {
    return <div className="ta-empty" data-testid="auth-loading">正在恢复登录状态…</div>;
  }
  if (auth.status === "anonymous") {
    const returnTo = encodeURIComponent(location.pathname + location.search);
    return <Navigate to={`/login?return_to=${returnTo}`} replace />;
  }
  return children;
}
