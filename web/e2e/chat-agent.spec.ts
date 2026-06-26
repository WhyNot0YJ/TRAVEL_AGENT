import { expect, test } from "@playwright/test";

const mockPlan = {
  title: "杭州 3 日旅行规划",
  summary: "从上海出发，围绕西湖、自然风光和美食安排轻松路线。",
  days: [
    {
      day: 1,
      theme: "抵达杭州与西湖散步",
      items: [
        {
          time: "09:00",
          type: "transport",
          name: "上海到杭州高铁",
          address: "上海虹桥站",
          reason: "高铁优先，减少路上时间。",
          estimated_cost: 150,
          duration_minutes: 70,
        },
        {
          time: "11:00",
          type: "poi",
          name: "西湖断桥",
          address: "杭州市西湖区",
          reason: "适合轻松开始杭州行程。",
          estimated_cost: 0,
          duration_minutes: 90,
        },
      ],
    },
    {
      day: 2,
      theme: "自然风光与本地美食",
      items: [
        {
          time: "10:00",
          type: "poi",
          name: "灵隐寺",
          address: "杭州市西湖区灵隐路",
          reason: "覆盖自然和文化体验。",
          estimated_cost: 75,
          duration_minutes: 150,
        },
      ],
    },
    {
      day: 3,
      theme: "运河街区与返程",
      items: [
        {
          time: "10:00",
          type: "food",
          name: "胜利河美食街",
          address: "杭州市拱墅区",
          reason: "满足美食偏好。",
          estimated_cost: 120,
          duration_minutes: 90,
        },
      ],
    },
  ],
  budget: {
    transport: 360,
    food: 600,
    hotel: 1200,
    ticket: 180,
    total: 2340,
  },
  warnings: ["预算为估算值，请以现场价格为准。"],
};

const chatResponse = {
  departure_city: "上海",
  destination_city: "杭州",
  days: 3,
  budget: 3000,
  interests: ["美食", "自然风光"],
  transport_mode: "train_taxi",
  pace: "relaxed",
  reply: "信息已经齐全，可以生成行程了。",
  missing: [],
  is_complete: true,
  agent_mode: "quick",
};

function sseBlock(event: string, data: unknown): string {
  return [`event: ${event}`, `data: ${JSON.stringify(data)}`, "", ""].join("\n");
}

test.beforeEach(async ({ page }) => {
  await page.route("**/api/v1/travel/chat/stream", async (route) => {
    await route.fulfill({
      status: 200,
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
      },
      body:
        sseBlock("assistant_delta", { type: "assistant_delta", message: "信息已经齐全，" }) +
        sseBlock("assistant_delta", { type: "assistant_delta", message: "可以生成行程了。" }) +
        sseBlock("assistant_done", { type: "assistant_done", message: chatResponse.reply }) +
        sseBlock("done", chatResponse),
    });
  });

  await page.route("**/api/v1/travel/plans", async (route) => {
    if (route.request().method() !== "POST") {
      await route.fallback();
      return;
    }

    await route.fulfill({
      status: 202,
      contentType: "application/json",
      body: JSON.stringify({
        task_id: "task_ui_mock",
        request_hash: "ui_hash",
        status: "pending",
        cached: false,
      }),
    });
  });

  await page.route("**/api/v1/travel/plans/task_ui_mock/stream", async (route) => {
    await route.fulfill({
      status: 200,
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
      },
      body:
        sseBlock("progress", {
          type: "progress",
          task_id: "task_ui_mock",
          status: "running",
          message: "planner started",
          created_at: new Date().toISOString(),
        }) +
        sseBlock("assistant_delta", {
          type: "assistant_delta",
          task_id: "task_ui_mock",
          status: "running",
          message: "Planning ",
          created_at: new Date().toISOString(),
        }) +
        sseBlock("assistant_delta", {
          type: "assistant_delta",
          task_id: "task_ui_mock",
          status: "running",
          message: "Hangzhou ",
          created_at: new Date().toISOString(),
        }) +
        sseBlock("assistant_delta", {
          type: "assistant_delta",
          task_id: "task_ui_mock",
          status: "running",
          message: "route",
          created_at: new Date().toISOString(),
        }) +
        sseBlock("node", {
          type: "node",
          task_id: "task_ui_mock",
          status: "running",
          node_name: "SearchPOIsToolNode",
          node_status: "success",
          duration_ms: 8,
          created_at: new Date().toISOString(),
        }) +
        sseBlock("done", {
          type: "done",
          task_id: "task_ui_mock",
          status: "succeeded",
          message: "task finished",
          plan: mockPlan,
          created_at: new Date().toISOString(),
        }),
    });
  });

  await page.route("**/api/v1/travel/plans/task_ui_mock", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        task_id: "task_ui_mock",
        request_hash: "ui_hash",
        status: "succeeded",
        plan: mockPlan,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }),
    });
  });
});

test("chat UI generates and displays a travel plan", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByTestId("chat-input")).toBeVisible();
  await expect(page.getByTestId("generate-plan")).toHaveCount(0);

  await page.getByTestId("chat-input").fill("上海出发，杭州 3 天，预算 3000，喜欢美食和自然风光，高铁优先");
  await page.getByTestId("send-message").click();

  await expect(page.getByTestId("brief-panel")).toContainText("杭州");
  await expect(page.getByTestId("generate-plan")).toBeEnabled();

  await page.getByTestId("generate-plan").click();

  await expect(page.getByTestId("plan-detail")).toContainText("杭州 3 日旅行规划");
  await expect(page.getByTestId("plan-detail")).toContainText("西湖断桥");
  await expect(page.getByTestId("plan-detail")).toContainText("调整预算");
  await expect(page.getByTestId("plan-detail")).toContainText("预算拆分");
});

test("planning stream appends chunks inside one assistant result bubble", async ({ page }) => {
  await page.goto("/");

  await page.getByTestId("chat-input").fill("上海出发，杭州 3 天，预算 3000，喜欢美食和自然风光，高铁优先");
  await page.getByTestId("send-message").click();
  await expect(page.getByTestId("generate-plan")).toBeEnabled();

  await expect(page.getByTestId("message-assistant")).toHaveCount(2);
  await page.getByTestId("generate-plan").click();

  await expect(page.getByTestId("planning-message")).toHaveCount(1);
  await expect(page.getByTestId("planning-stream-text")).toHaveText("Planning Hangzhou route");
  await expect(page.getByTestId("message-assistant")).toHaveCount(2);
});

test("chat UI recovers with polling when SSE disconnects", async ({ page }) => {
  let streamAttempts = 0;

  await page.route("**/api/v1/travel/plans/task_ui_mock/stream", async (route) => {
    streamAttempts += 1;
    await route.fulfill({ status: 503, body: "stream unavailable" });
  });

  await page.goto("/");
  await page.getByTestId("chat-input").fill("上海出发，杭州 3 天，预算 3000，喜欢美食和自然风光，高铁优先");
  await page.getByTestId("send-message").click();
  await page.getByTestId("generate-plan").click();

  await expect(page.getByTestId("plan-detail")).toContainText("杭州 3 日旅行规划");
  expect(streamAttempts).toBeGreaterThan(0);
});
