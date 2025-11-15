import { test, expect } from '@playwright/test';

const shouldRun = process.env.PLAYWRIGHT_LOCAL_IAM === '1';
const loginEmail = process.env.PLAYWRIGHT_LOCAL_EMAIL || 'admin@local.test';
const loginPassword = process.env.PLAYWRIGHT_LOCAL_PASSWORD || 'S3cret!!';

if (!shouldRun) {
  test.skip(true, 'Set PLAYWRIGHT_LOCAL_IAM=1 to run local IAM tests.');
}

test.describe('Local IAM Login Flow', () => {
  test('logs in via local backend and persists tokens', async ({ page }) => {
    await page.goto('/users/login');
    await page.locator('input[name="identifier"]').fill(loginEmail);
    await page.locator('input[name="password"]').fill(loginPassword);
    await page.getByRole('button', { name: /登录|sign/i }).click();

    await expect.poll(() => page.evaluate(() => window.localStorage.getItem('access_token'))).not.toBeNull();
    await expect(page).toHaveURL(/\/(agent|templates)/);
  });
});
