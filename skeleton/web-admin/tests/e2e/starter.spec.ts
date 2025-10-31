import { test, expect } from '@playwright/test';

test.describe('Starter admin page', () => {
  test('shows welcome content', async ({ page }) => {
    await page.goto('/_p/com.powerx.sample/admin');
    await expect(page.getByRole('heading', { name: 'PowerX Sample' })).toBeVisible();
    await expect(page.getByText('欢迎使用 PowerX 插件脚手架示例页面。')).toBeVisible();
  });
});
