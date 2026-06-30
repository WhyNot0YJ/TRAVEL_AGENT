import { useEffect, useState } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

type Mode = "login" | "register";

export default function AuthView() {
  const auth = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const initialMode: Mode = searchParams.get("mode") === "register" ? "register" : "login";
  const returnTo = searchParams.get("return_to") || "/";

  const [mode, setMode] = useState<Mode>(initialMode);
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (auth.status === "authenticated") {
      navigate(returnTo, { replace: true });
    }
  }, [auth.status, navigate, returnTo]);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError("");
    if (mode === "register") {
      if (password.length < 8) {
        setError("密码至少 8 位");
        return;
      }
      if (password !== confirmPassword) {
        setError("两次输入的密码不一致");
        return;
      }
      if (!displayName.trim()) {
        setError("请填写昵称");
        return;
      }
    }
    setSubmitting(true);
    try {
      if (mode === "login") {
        await auth.login({ email, password });
      } else {
        await auth.register({ email, password, display_name: displayName });
      }
      navigate(returnTo, { replace: true });
    } catch (err) {
      // Stable error wording per Stage 21 spec — never leak account existence.
      if (mode === "login") {
        setError("邮箱或密码不正确");
      } else {
        const message = err instanceof Error ? err.message : "注册失败";
        if (message.includes("email_exists") || message.includes("registered")) {
          setError("该邮箱已被注册");
        } else {
          setError(message);
        }
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="ta-auth" data-testid="auth-view">
      <div className="ta-auth-card">
        <h1>{mode === "login" ? "登录" : "注册"}</h1>
        <p>
          {mode === "login"
            ? "登录后即可保存路线、查看历史和发布公开计划。"
            : "注册一个 Travel Agent 账号，长期保存你的旅行路线。"}
        </p>
        <form className="ta-form" onSubmit={submit}>
          <label>
            邮箱
            <input
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              data-testid="auth-email"
            />
          </label>
          {mode === "register" ? (
            <label>
              展示昵称
              <input
                type="text"
                value={displayName}
                onChange={(event) => setDisplayName(event.target.value)}
                data-testid="auth-display-name"
              />
            </label>
          ) : null}
          <label>
            密码
            <input
              type="password"
              autoComplete={mode === "login" ? "current-password" : "new-password"}
              required
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              data-testid="auth-password"
            />
          </label>
          {mode === "register" ? (
            <label>
              确认密码
              <input
                type="password"
                autoComplete="new-password"
                required
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                data-testid="auth-confirm-password"
              />
            </label>
          ) : null}
          {error ? (
            <span className="ta-form-error" data-testid="auth-error" role="alert">
              {error}
            </span>
          ) : null}
          <button
            type="submit"
            className="ta-action-button"
            disabled={submitting}
            data-testid="auth-submit"
          >
            {submitting ? "处理中…" : mode === "login" ? "登录" : "注册并登录"}
          </button>
        </form>
        <div className="ta-form-switch">
          {mode === "login" ? (
            <>
              还没有账号？
              <button type="button" onClick={() => setMode("register")} data-testid="auth-switch-register">
                注册新账号
              </button>
            </>
          ) : (
            <>
              已有账号？
              <button type="button" onClick={() => setMode("login")} data-testid="auth-switch-login">
                去登录
              </button>
            </>
          )}
        </div>
        {location.state ? null : null}
      </div>
    </section>
  );
}
