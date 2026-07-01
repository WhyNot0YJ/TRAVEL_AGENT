import { expect, test } from "@playwright/test";

const sampleUser = {
  id: "user_viewer",
  email: "viewer@example.com",
  display_name: "Viewer",
  status: "active",
  created_at: "2026-06-30T12:00:00Z",
};

const publicPlan = {
  public_plan_id: "pub_xyz",
  title: "杭州 3 日 西湖路线",
  summary: "适合首次到杭州的旅行者。",
  tags: ["杭州", "美食"],
  destination_city: "杭州",
  days: 3,
  author: { display_name: "Alice" },
  hot_score: 50,
  view_count: 30,
  save_count: 4,
  published_at: "2026-06-29T12:00:00Z",
  plan: {
    title: "杭州 3 日 西湖路线",
    summary: "适合首次到杭州的旅行者。",
    days: [
      {
        day: 1,
        theme: "西湖",
        items: [
          {
            time: "10:00",
            type: "poi",
            name: "西湖断桥",
            address: "杭州市西湖区",
            reason: "起点",
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

test.describe("public detail", () => {
  test("authenticated user can save a public plan as a private copy", async ({ page }) => {
    let detailRequests = 0;
    await page.route("**/api/v1/auth/me", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ user: sampleUser }) });
    });
    await page.route("**/api/v1/public/plans/pub_xyz", async (route) => {
      detailRequests += 1;
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ public_plan: publicPlan }) });
    });
    await page.route("**/api/v1/public/plans/pub_xyz/save", async (route) => {
      await route.fulfill({
        status: 201,
        contentType: "application/json",
        body: JSON.stringify({
          plan: {
            plan_id: "plan_copied",
            user_id: sampleUser.id,
            source_public_plan_id: publicPlan.public_plan_id,
            title: publicPlan.title,
            summary: publicPlan.summary,
            tags: publicPlan.tags,
            destination_city: publicPlan.destination_city,
            days: publicPlan.days,
            visibility: "private",
            publish_status: "draft",
            created_at: "2026-06-30T12:00:00Z",
            updated_at: "2026-06-30T12:00:00Z",
            plan: publicPlan.plan,
          },
        }),
      });
    });
    await page.route("**/api/v1/me/plans/plan_copied", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          plan: {
            plan_id: "plan_copied",
            user_id: sampleUser.id,
            title: publicPlan.title,
            summary: publicPlan.summary,
            tags: publicPlan.tags,
            destination_city: publicPlan.destination_city,
            days: publicPlan.days,
            visibility: "private",
            publish_status: "draft",
            created_at: "2026-06-30T12:00:00Z",
            updated_at: "2026-06-30T12:00:00Z",
            plan: publicPlan.plan,
          },
        }),
      });
    });

    await page.goto("/public/pub_xyz");
    await expect(page.getByTestId("public-detail-page")).toContainText("杭州 3 日 西湖路线");
    expect(detailRequests).toBe(1);
    await page.getByTestId("public-save-copy").click();
    await expect(page).toHaveURL(/\/me\/plans\/plan_copied/);
    await expect(page.getByTestId("private-detail-title")).toContainText("杭州 3 日 西湖路线");
  });
});
