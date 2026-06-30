import { expect, test, type Route } from "@playwright/test";

const sampleUser = {
  id: "user_test",
  email: "alice@example.com",
  display_name: "Alice",
  status: "active",
  created_at: "2026-06-30T12:00:00Z",
};

async function emptyHomeRoutes(page: Parameters<Parameters<typeof test.beforeEach>[0]>[0]["page"]) {
  await page.route("**/api/v1/public/plans?**", async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ items: [], total: 0, page: 1, page_size: 6 }),
    });
  });
  await page.route("**/api/v1/me/current", async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({}),
    });
  });
}

test.describe("auth flow", () => {
  test("anonymous visitor cannot reach user center and is bounced to login", async ({ page }) => {
    await page.route("**/api/v1/auth/me", async (route) => {
      await route.fulfill({ status: 401, contentType: "application/json", body: JSON.stringify({ message: "unauthenticated" }) });
    });
    await emptyHomeRoutes(page);

    await page.goto("/me");
    await expect(page.getByTestId("auth-view")).toBeVisible();
    await expect(page).toHaveURL(/\/login/);
  });

  test("registration auto-signs in and lands the visitor on the saved return path", async ({ page }) => {
    let authState: typeof sampleUser | null = null;
    await page.route("**/api/v1/auth/me", async (route) => {
      if (authState) {
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ user: authState }) });
      } else {
        await route.fulfill({ status: 401, contentType: "application/json", body: JSON.stringify({ message: "unauthenticated" }) });
      }
    });
    await page.route("**/api/v1/auth/register", async (route) => {
      authState = sampleUser;
      await route.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify({ user: sampleUser }) });
    });
    await emptyHomeRoutes(page);
    await page.route("**/api/v1/me/plans?**", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ items: [], total: 0, page: 1, page_size: 20 }) });
    });

    await page.goto("/login?mode=register&return_to=%2Fme");
    await page.getByTestId("auth-email").fill("alice@example.com");
    await page.getByTestId("auth-display-name").fill("Alice");
    await page.getByTestId("auth-password").fill("longenough!");
    await page.getByTestId("auth-confirm-password").fill("longenough!");
    await page.getByTestId("auth-submit").click();

    await expect(page).toHaveURL(/\/me$/);
    await expect(page.getByTestId("user-center")).toBeVisible();
  });

  test("login error shows the stable invalid credentials message", async ({ page }) => {
    await page.route("**/api/v1/auth/me", async (route) => {
      await route.fulfill({ status: 401, contentType: "application/json", body: JSON.stringify({ message: "unauthenticated" }) });
    });
    await page.route("**/api/v1/auth/login", async (route) => {
      await route.fulfill({
        status: 401,
        contentType: "application/json",
        body: JSON.stringify({ code: "invalid_credentials", message: "invalid email or password" }),
      });
    });
    await emptyHomeRoutes(page);

    await page.goto("/login");
    await page.getByTestId("auth-email").fill("ghost@example.com");
    await page.getByTestId("auth-password").fill("wrongpassword");
    await page.getByTestId("auth-submit").click();
    await expect(page.getByTestId("auth-error")).toContainText("邮箱或密码不正确");
  });
});
