import { Navigate, Route, Routes } from "react-router-dom";
import AppShell from "./components/AppShell";
import AuthView from "./components/AuthView";
import HomeView from "./components/HomeView";
import PlannerView from "./components/PlannerView";
import PrivatePlanDetail from "./components/PrivatePlanDetail";
import PublicPlanDetail from "./components/PublicPlanDetail";
import PublicPlanList from "./components/PublicPlanList";
import RequireAuth from "./components/RequireAuth";
import UserCenter from "./components/UserCenter";

export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<HomeView />} />
        <Route path="login" element={<AuthView />} />
        <Route path="planner" element={<PlannerView />} />
        <Route path="public" element={<PublicPlanList />} />
        <Route path="public/:publicPlanId" element={<PublicPlanDetail />} />
        <Route
          path="me"
          element={
            <RequireAuth>
              <UserCenter />
            </RequireAuth>
          }
        />
        <Route
          path="me/plans/:planId"
          element={
            <RequireAuth>
              <PrivatePlanDetail />
            </RequireAuth>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
