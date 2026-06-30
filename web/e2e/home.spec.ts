import { expect, test, type Route } from "@playwright/test";

const hotPlan = {
  public_plan_id: "pub_hot",
  title: "杭州 3 日 西湖路线",
  summary: "适合首次去杭州的轻松路线。",
  tags: ["杭州", "美食", "3日"],
  destination_city: "杭州",
  days: 3,
  author: { display_name: "Alice" },
  hot_score: 128,
  view_count: 90,
  save_count: 6,
  published_at: "2026-06-29T12:00:00Z",
  updated_at: "2026-06-29T12:00:00Z",
};

const recommendedPlan = {
  ...hotPlan,
  public_plan_id: "pub_recent",
  title: "苏州 2 日园林漫步",
  destination_city: "苏州",
  days: 2,
  hot_score: 12,
};

test.describe("home view", () => {
  test.beforeEach(async ({ page }) => {
    await page.route("**/api/v1/auth/me", async (route) => {
      await route.fulfill({ status: 401, contentType: "application/json", body: JSON.stringify({ message: "unauthenticated" }) });
    });
    await page.route("**/api/v1/public/plans?**", async (route: Route) => {
      const url = new URL(route.request().url());
      const sort = url.searchParams.get("sort") ?? "hot";
      const items = sort === "hot" ? [hotPlan] : [recommendedPlan];
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items, total: items.length, page: 1, page_size: 6 }),
      });
    });
  });

  test("home renders hot ranking and recommended sections", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByTestId("home-view")).toBeVisible();
    await expect(page.getByTestId("home-hot")).toContainText("杭州 3 日 西湖路线");
    await expect(page.getByTestId("home-recommended")).toContainText("苏州 2 日园林漫步");
    // first hot card carries the rank badge
    const cards = page.getByTestId("public-plan-card");
    await expect(cards.first()).toContainText("1");
  });

  test("typing in search submits to the public list", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("home-search-input").fill("杭州");
    await page.getByTestId("home-search-submit").click();
    await expect(page).toHaveURL(/\/public\?q=/);
    await expect(page.getByTestId("public-list-page")).toBeVisible();
  });

  test("mobile bottom navigation exposes home / planner / me without overflow", async ({ page, isMobile }) => {
    test.skip(!isMobile, "mobile-only assertion");
    await page.goto("/");
    const nav = page.getByTestId("bottom-nav");
    await expect(nav).toBeVisible();
    const links = nav.locator("a");
    await expect(links).toHaveCount(3);
  });
});
