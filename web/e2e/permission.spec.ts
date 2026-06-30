import { expect, test } from "@playwright/test";

test.describe("permission guards", () => {
  test("unauthenticated /me/* redirects to login with return_to", async ({ page }) => {
    await page.route("**/api/v1/auth/me", async (route) => {
      await route.fulfill({ status: 401, contentType: "application/json", body: JSON.stringify({ message: "unauthenticated" }) });
    });
    await page.route("**/api/v1/public/plans?**", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [], total: 0, page: 1, page_size: 6 }) });
    });
    await page.goto("/me/plans/plan_unknown");
    await expect(page).toHaveURL(/\/login\?return_to=/);
    await expect(page.getByTestId("auth-view")).toBeVisible();
  });

  test("private plan owned by another user surfaces a not-found message", async ({ page }) => {
    await page.route("**/api/v1/auth/me", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          user: {
            id: "user_test",
            email: "alice@example.com",
            display_name: "Alice",
            status: "active",
            created_at: "2026-06-30T12:00:00Z",
          },
        }),
      });
    });
    await page.route("**/api/v1/me/plans/plan_other", async (route) => {
      await route.fulfill({
        status: 404,
        contentType: "application/json",
        body: JSON.stringify({ code: "not_found", message: "plan not found" }),
      });
    });

    await page.goto("/me/plans/plan_other");
    await expect(page.getByTestId("private-detail-error")).toContainText("不存在或已不可见");
  });
});
