import { expect, test } from "@playwright/test";

const sampleUser = {
  id: "user_test",
  email: "alice@example.com",
  display_name: "Alice",
  status: "active",
  created_at: "2026-06-30T12:00:00Z",
};

const basePlan = {
  plan_id: "plan_alpha",
  user_id: "user_test",
  task_id: "task_a",
  title: "杭州 3 日旅行",
  summary: "西湖与灵隐",
  tags: ["杭州", "美食"],
  destination_city: "杭州",
  days: 3,
  visibility: "private",
  publish_status: "draft",
  created_at: "2026-06-30T12:00:00Z",
  updated_at: "2026-06-30T12:00:00Z",
};

test.describe("plan library", () => {
  test.beforeEach(async ({ page }) => {
    await page.route("**/api/v1/auth/me", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ user: sampleUser }) });
    });
    await page.route("**/api/v1/public/plans?**", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [], total: 0, page: 1, page_size: 6 }) });
    });
    await page.route("**/api/v1/me/current", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({}) });
    });
  });

  test("user can publish and then unpublish a plan from the library", async ({ page }) => {
    let plan = { ...basePlan };
    await page.route("**/api/v1/me/plans?**", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [plan], total: 1, page: 1, page_size: 20 }) });
    });
    await page.route("**/api/v1/me/plans/plan_alpha/publish", async (route) => {
      plan = { ...plan, publish_status: "published", visibility: "public" };
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          public_plan: {
            public_plan_id: "pub_alpha",
            title: plan.title,
            summary: plan.summary,
            tags: plan.tags,
            destination_city: plan.destination_city,
            days: plan.days,
            author: { display_name: sampleUser.display_name },
            hot_score: 0,
            view_count: 0,
            save_count: 0,
            published_at: "2026-06-30T12:00:00Z",
          },
        }),
      });
    });
    await page.route("**/api/v1/me/plans/plan_alpha/unpublish", async (route) => {
      plan = { ...plan, publish_status: "unpublished", visibility: "private" };
      await route.fulfill({ status: 204, body: "" });
    });

    await page.goto("/me");
    await expect(page.getByTestId("user-center")).toBeVisible();
    const row = page.getByTestId("library-row").first();
    await expect(row).toContainText("杭州 3 日旅行");
    await row.getByTestId("library-publish").click();
    await expect(page.getByTestId("toast")).toContainText("已发布");

    // After publish the row should expose 取消发布
    await expect(row.getByTestId("library-unpublish")).toBeVisible();
    await row.getByTestId("library-unpublish").click();
    await expect(page.getByTestId("confirm-dialog")).toBeVisible();
    await page.getByTestId("confirm-ok").click();
    await expect(page.getByTestId("toast")).toContainText("已取消发布");
  });

  test("delete confirmation removes the plan", async ({ page }) => {
    let plans = [{ ...basePlan }];
    await page.route("**/api/v1/me/plans?**", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: plans, total: plans.length, page: 1, page_size: 20 }) });
    });
    await page.route("**/api/v1/me/plans/plan_alpha", async (route) => {
      if (route.request().method() === "DELETE") {
        plans = [];
        await route.fulfill({ status: 204, body: "" });
        return;
      }
      await route.fallback();
    });

    await page.goto("/me");
    await page.getByTestId("library-delete").first().click();
    await expect(page.getByTestId("confirm-dialog")).toContainText("确认删除");
    await page.getByTestId("confirm-ok").click();
    await expect(page.getByTestId("library-empty")).toBeVisible();
  });
});
