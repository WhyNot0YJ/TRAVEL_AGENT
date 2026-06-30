import { expect, test } from "@playwright/test";

const sampleUser = {
  id: "user_test",
  email: "alice@example.com",
  display_name: "Alice",
  status: "active",
  created_at: "2026-06-30T12:00:00Z",
};

const samplePlan = {
  plan_id: "plan_alpha",
  user_id: "user_test",
  task_id: "task_ui_save",
  title: "杭州 3 日旅行",
  summary: "西湖与灵隐",
  tags: ["杭州", "美食", "3日"],
  destination_city: "杭州",
  days: 3,
  visibility: "private",
  publish_status: "draft",
  created_at: "2026-06-30T12:00:00Z",
  updated_at: "2026-06-30T12:00:00Z",
  plan: {
    title: "杭州 3 日旅行",
    summary: "西湖与灵隐",
    days: [
      {
        day: 1,
        theme: "西湖漫步",
        items: [
          {
            time: "10:00",
            type: "poi",
            name: "西湖断桥",
            address: "杭州市西湖区",
            reason: "经典起点",
            estimated_cost: 0,
            duration_minutes: 90,
          },
        ],
      },
    ],
    budget: { transport: 0, food: 200, hotel: 600, ticket: 0, total: 800 },
    warnings: [],
  },
};

const mockTaskPlan = samplePlan.plan;

test.describe("save and publish flow", () => {
  test("anonymous user is bounced to login when saving and the action resumes after sign-in", async ({ page }) => {
    let authState: typeof sampleUser | null = null;
    await page.route("**/api/v1/auth/me", async (route) => {
      if (authState) {
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ user: authState }) });
      } else {
        await route.fulfill({ status: 401, contentType: "application/json", body: JSON.stringify({ message: "unauthenticated" }) });
      }
    });
    await page.route("**/api/v1/auth/login", async (route) => {
      authState = sampleUser;
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ user: sampleUser }) });
    });
    await page.route("**/api/v1/public/plans?**", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [], total: 0, page: 1, page_size: 6 }) });
    });
    await page.route("**/api/v1/me/current", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
    });
    await page.route("**/api/v1/me/plans?**", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [], total: 0, page: 1, page_size: 20 }) });
    });
    let saveCalled = false;
    await page.route("**/api/v1/me/plans", async (route) => {
      if (route.request().method() !== "POST") {
        await route.fallback();
        return;
      }
      saveCalled = true;
      await route.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify({ plan: samplePlan }) });
    });

    // Visit a planner page that already has a completed task via task_id query.
    await page.route("**/api/v1/travel/plans/task_ui_save/stream", async (route) => {
      const body = [
        "event: progress",
        `data: ${JSON.stringify({ type: "progress", task_id: "task_ui_save", status: "running", message: "loading" })}`,
        "",
        "event: done",
        `data: ${JSON.stringify({ type: "done", task_id: "task_ui_save", status: "succeeded", message: "ok", plan: mockTaskPlan })}`,
        "",
        "",
      ].join("\n");
      await route.fulfill({ status: 200, headers: { "Content-Type": "text/event-stream" }, body });
    });
    await page.route("**/api/v1/travel/plans/task_ui_save", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ task_id: "task_ui_save", request_hash: "h", status: "succeeded", plan: mockTaskPlan, created_at: "", updated_at: "" }),
      });
    });

    await page.goto("/planner?task_id=task_ui_save");
    await expect(page.getByTestId("planner-save-plan")).toBeVisible();
    await page.getByTestId("planner-save-plan").click();
    // Should navigate to /login
    await expect(page).toHaveURL(/\/login/);

    await page.getByTestId("auth-email").fill("alice@example.com");
    await page.getByTestId("auth-password").fill("longenough!");
    await page.getByTestId("auth-submit").click();

    // After login the planner remounts and the pending save resumes via sessionStorage.
    await expect(page).toHaveURL(/\/planner\?task_id=task_ui_save/);
    await expect.poll(() => saveCalled, { timeout: 5_000 }).toBe(true);
    await expect(page.getByTestId("planner-view-saved")).toBeVisible();
  });
});
