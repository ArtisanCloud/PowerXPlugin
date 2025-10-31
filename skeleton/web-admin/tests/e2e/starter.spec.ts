import { test, expect } from '@playwright/test';

test.describe('Starter admin page', () => {
  test('shows welcome content', async ({ page }) => {
    await page.goto('/_p/com.powerx.sample/admin');
    await expect(page.getByRole('heading', { name: 'PowerX Admin' })).toBeVisible();
    await expect(page.getByText('默认启动页')).toBeVisible();
  });
});
