import { test, expect } from '@playwright/test';

const loginEmail = process.env.PLAYWRIGHT_LOCAL_EMAIL || 'admin@local.test';
const loginPassword = process.env.PLAYWRIGHT_LOCAL_PASSWORD || 'S3cret!!';
const localIAMFlag = process.env.PLAYWRIGHT_LOCAL_IAM;

if (localIAMFlag !== '1') {
  test.skip(true, 'Set PLAYWRIGHT_LOCAL_IAM=1 to run IAM organization e2e tests.');
}

async function loginIfNeeded(page) {
  await page.goto('/users/login');
  if (await page.getByRole('button', { name: /退出|登出|logout/i }).isVisible().catch(() => false)) {
    return;
  }
  await page.locator('input[name="identifier"]').fill(loginEmail);
  await page.locator('input[name="password"]').fill(loginPassword);
  await page.getByRole('button', { name: /登录|sign/i }).click();
  await expect(page).toHaveURL(/admin|templates|intro|agent/);
}

test.describe('IAM organization flows', () => {
  test.beforeEach(async ({ page }) => {
    await loginIfNeeded(page);
  });

  test('invites member in local IAM', async ({ page }) => {
    const suffix = Date.now();
    const memberEmail = `iam-e2e-${suffix}@local.test`;

    await page.goto('/admin/iam/members');
    await page.getByTestId('create-member').click();
    await page.getByPlaceholder('user@example.com').fill(memberEmail);
    await page.getByRole('button', { name: /保存|save/i }).last().click();
    await expect(page.getByTestId('member-table')).toContainText(memberEmail);
  });
});
