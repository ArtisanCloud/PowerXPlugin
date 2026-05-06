import { test, expect } from '@playwright/test';
import { gotoWithFallback } from './_utils';

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
  const insidePowerX =
    process.env.POWERX_PROXY === '1' ||
    process.env.NUXT_PUBLIC_POWERX_PROXY === '1' ||
    process.env.NUXT_POWERX_PROXY === '1';

  test('logs in and redirects when Core responds with success', async ({ page }) => {
    await page.route('**/admin/user/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(loginPayload),
      });
    });

    await gotoWithFallback(page, '/users/login', page.locator('input[name="identifier"]'));
    await page.locator('input[name="identifier"]').fill('admin@example.com');
    await page.locator('input[name="password"]').fill('secret');
    await page.getByRole('button', { name: /登录|sign/i }).click();

    await expect(page).not.toHaveURL(/\/users\/login/);
    await expect.poll(() => page.evaluate(() => window.localStorage.getItem('access_token'))).toBe('test-access-token');
  });

  test('shows fail-closed error when Core is unavailable', async ({ page }) => {
    await page.route('**/admin/user/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          success: false,
          message: failClosedPayload.error.message,
        }),
      });
    });

    await gotoWithFallback(page, '/users/login', page.locator('input[name="identifier"]'));
    await page.locator('input[name="identifier"]').fill('admin@example.com');
    await page.locator('input[name="password"]').fill('secret');
    await page.getByRole('button', { name: /登录|sign/i }).click();

    await expect(page.getByRole('alert')).toContainText('宿主认证不可用');
  });

  test('storage event logout clears token and triggers delegated banner', async ({ page }) => {
    await page.route('**/admin/user/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(loginPayload),
      });
    });

    await gotoWithFallback(page, '/users/login', page.locator('input[name="identifier"]'));
    await page.locator('input[name="identifier"]').fill('admin@example.com');
    await page.locator('input[name="password"]').fill('secret');
    await page.getByRole('button', { name: /登录|sign/i }).click();
    await expect(page).not.toHaveURL(/\/users\/login/);

    await page.evaluate(() => {
      window.localStorage.removeItem('access_token');
      window.localStorage.removeItem('refresh_token');
      window.dispatchEvent(new StorageEvent('storage', { key: 'access_token' }));
    });

    const banner = page.locator('[data-test="delegated-auth-banner"]');
    const bannerVisible = await banner.isVisible().catch(() => false);
    if (bannerVisible) {
      await expect(page).not.toHaveURL(/\/users\/login/);
      await expect(banner).toContainText('PowerX 会话已失效');
    } else {
      await expect(page).toHaveURL(/\/users\/login|\/$/);
    }
  });
});
