import { test, expect } from "@playwright/test";

test("chat interface displays response from agent", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByText("Palm Agent")).toBeVisible();
  await expect(page.getByText("Start a conversation with Palm")).toBeVisible();

  const input = page.getByPlaceholder("Ask anything...");
  await expect(input).toBeVisible();

  const sendButton = page.locator('button[type="submit"]');
  await expect(sendButton).toBeVisible();

  await input.fill("Hello, what is 2+2?");
  await sendButton.click();

  await expect(page.getByText("Hello, what is 2+2?")).toBeVisible();

  await page.waitForSelector(".bg-muted:has-text('2')", { timeout: 30000 });

  const assistantMessage = page.locator(".bg-muted").filter({ hasText: /\w+/ }).first();
  const responseText = await assistantMessage.textContent();

  expect(responseText).toBeTruthy();
  expect(responseText!.length).toBeGreaterThan(0);
});

test("loading state is shown while waiting for response", async ({ page }) => {
  await page.goto("/");

  const input = page.getByPlaceholder("Ask anything...");
  await expect(input).toBeVisible();

  const sendButton = page.locator('button[type="submit"]');
  await expect(sendButton).toBeVisible();

  await input.fill("test message");
  await sendButton.click();

  const loadingDots = page.locator(".animate-bounce").first();
  await expect(loadingDots).toBeVisible({ timeout: 1000 });
});

test("send button is enabled/disabled based on input", async ({ page }) => {
  await page.goto("/");

  const input = page.getByPlaceholder("Ask anything...");
  const sendButton = page.locator('button[type="submit"]');

  await expect(sendButton).toBeDisabled();

  await input.fill("Hello");
  await expect(sendButton).toBeEnabled();

  await input.clear();
  await expect(sendButton).toBeDisabled();

  await input.fill("   ");
  await expect(sendButton).toBeDisabled();

  await input.fill("  Hello  ");
  await expect(sendButton).toBeEnabled();
});

test("send button is disabled during loading", async ({ page }) => {
  await page.goto("/");

  const input = page.getByPlaceholder("Ask anything...");
  const sendButton = page.locator('button[type="submit"]');

  await input.fill("What is the weather?");
  await expect(sendButton).toBeEnabled();

  await sendButton.click();

  await expect(sendButton).toBeDisabled();

  const assistantMessage = page.locator(".bg-muted").first();
  await expect(assistantMessage).toBeVisible({ timeout: 30000 });

  await expect(sendButton).toBeDisabled();

  await input.fill("Another question");
  await expect(sendButton).toBeEnabled();
});
