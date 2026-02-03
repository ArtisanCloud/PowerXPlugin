import { test, expect } from '@playwright/test';

const loginEmail = process.env.PLAYWRIGHT_LOCAL_EMAIL || 'admin@local.test';
const loginPassword = process.env.PLAYWRIGHT_LOCAL_PASSWORD || 'S3cret!!';
const iamMode = process.env.PLAYWRIGHT_LOCAL_IAM;

if (iamMode !== '1' && iamMode !== '0') {
  test.skip(true, 'Set PLAYWRIGHT_LOCAL_IAM=1 for local mode or 0 for delegated guard tests.');
}

const localSuite = iamMode !== '0' ? test.describe : test.describe.skip;
const delegatedSuite = iamMode === '0' ? test.describe : test.describe.skip;

localSuite('Local IAM Login Flow', () => {
  test('logs in via local backend, persists tokens, and shows IAM menu', async ({ page }) => {
    await page.goto('/users/login');
    await page.locator('input[name="identifier"]').fill(loginEmail);
    await page.locator('input[name="password"]').fill(loginPassword);
    await page.getByRole('button', { name: /登录|sign/i }).click();

    await expect
      .poll(() => page.evaluate(() => window.localStorage.getItem('access_token')))
      .not.toBeNull();
    await expect(page).toHaveURL(/\/(agent|templates)/);
    await expect(
      page.getByRole('link', { name: /组织与权限|IAM & Access/i })
    ).toBeVisible();
  });
});

delegatedSuite('Delegated IAM guard', () => {
  test('disables local login flow and surfaces warning', async ({ page }) => {
    await page.goto('/users/login');
    await expect(page.getByRole('alert')).toContainText(/PowerX|登录/i);
    const submitButton = page.getByRole('button', { name: /登录|sign/i });
    await expect(submitButton).toBeDisabled();
  });
});
