import { test, expect } from '@playwright/test';

const loginPayload = {
  success: true,
  data: {
    token_type: 'Bearer',
    access_token: 'test-access-token',
    refresh_token: 'test-refresh-token',
    expires_in: 3600,
    expires_at: Date.now() + 3600 * 1000,
    scope: 'access',
  },
};

const failClosedPayload = {
  success: false,
  error: {
    code: 'SERVICE_UNAVAILABLE',
    message: '宿主认证不可用，请稍后重试',
  },
};

test.describe('Delegated Auth Flow', () => {
  test('logs in and redirects when Core responds with success', async ({ page }) => {
    await page.route('**/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(loginPayload),
      });
    });

    await page.goto('/users/login');
    await page.locator('input[name="identifier"]').fill('admin@example.com');
    await page.locator('input[name="password"]').fill('secret');
    await page.getByRole('button', { name: /登录|sign/i }).click();

    await expect(page).toHaveURL(/\/(agent|templates)/);
    await expect.poll(() => page.evaluate(() => window.localStorage.getItem('access_token'))).toBe('test-access-token');
  });

  test('shows fail-closed error when Core is unavailable', async ({ page }) => {
    await page.route('**/auth/login', async (route) => {
      await route.fulfill({
        status: 503,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(failClosedPayload),
      });
    });

    await page.goto('/users/login');
    await page.locator('input[name="identifier"]').fill('admin@example.com');
    await page.locator('input[name="password"]').fill('secret');
    await page.getByRole('button', { name: /登录|sign/i }).click();

    await expect(page.getByRole('alert')).toContainText('宿主认证不可用');
  });

  test('storage event logout clears token and redirects to login', async ({ page }) => {
    await page.route('**/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(loginPayload),
      });
    });

    await page.goto('/users/login');
    await page.locator('input[name="identifier"]').fill('admin@example.com');
    await page.locator('input[name="password"]').fill('secret');
    await page.getByRole('button', { name: /登录|sign/i }).click();
    await expect(page).toHaveURL(/\/(agent|templates)/);

    await page.evaluate(() => {
      window.localStorage.removeItem('access_token');
      window.localStorage.removeItem('refresh_token');
      window.dispatchEvent(new StorageEvent('storage', { key: 'access_token' }));
    });

    await expect(page).toHaveURL(/\/users\/login/);
  });
});
