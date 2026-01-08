import { test, expect } from '@playwright/test';
import { gotoWithFallback } from './_utils';

test.describe('Starter admin page', () => {
  test('shows welcome content', async ({ page }) => {
    await gotoWithFallback(page, '/', page.getByRole('heading', { name: 'PowerX 基础插件' }));
    await expect(page.getByRole('heading', { name: 'PowerX 基础插件' })).toBeVisible();
    await expect(page.getByText('欢迎使用 PowerX 插件脚手架示例页面').first()).toBeVisible();
  });
});
