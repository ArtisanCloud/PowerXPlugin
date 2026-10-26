import { test, expect } from '@playwright/test';
import { seedAuthStorage } from './_utils';
import { readFileSync } from 'node:fs';
const locale = JSON.parse(readFileSync(new URL('../../i18n/locales/zh.json', import.meta.url), 'utf8'));
const fixture = JSON.parse(readFileSync(new URL('../../../../../docs/contracts/fixtures/ai-media-v2.json', import.meta.url), 'utf8'));
const settingsCatalog = JSON.parse(readFileSync(new URL('../../../../../docs/contracts/fixtures/ai-settings-catalog.v1.json', import.meta.url), 'utf8'));

test('media catalog preserves defaults and selection on desktop and narrow screens', async ({ page }, info) => {
  const errors: string[] = [], writes: any[] = [];
  page.on('pageerror', error => errors.push(error.message));
  await seedAuthStorage(page);
  await page.context().addCookies([{ name: 'px_lang', value: 'zh', url: process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3131' }]);
  await page.route('**/api/v1/**', async route => {
    const request = route.request(), url = new URL(request.url()), path = url.pathname;
    const reply = (data: unknown) => route.fulfill({ json: { success: true, data } });
    if (request.method() === 'OPTIONS') { await route.fulfill({ status: 204 }); return; }
    if (path.endsWith('/admin/user/auth/me/context')) { await reply({ is_root: true, user: { user_uuid: '00000000-0000-4000-8000-000000000001', display_name: 'fixture.user', status: 1 }, current_tenant_uuid: '40000000-0000-4000-8000-000000000001', tenants: [{ tenant_uuid: '40000000-0000-4000-8000-000000000001', name: 'fixture.tenant' }], members: [], roles: [], permissions: [] }); return; }
    if (path.endsWith('/ai-settings/mode')) { await reply({ mode: 'local', local_available: true, delegated_available: false }); return; }
    if (path.endsWith('/ai-settings/model-profiles')) { await reply({ items: [] }); return; }
    if (path.endsWith('/ai-settings/local-setting')) {
      if (request.method() === 'PUT') { writes.push(request.postDataJSON()); await reply({}); }
      else await reply({});
      return;
    }
    if (path.endsWith('/ai-settings/catalog/providers')) { await reply(url.searchParams.get('source') === 'powerx' && url.searchParams.get('modality') === 'llm' ? settingsCatalog.providers : { items: [{ id: 'qwen-cn', name: 'fixture.provider' }] }); return; }
    if (path.endsWith('/ai-settings/catalog/models')) {
      if (url.searchParams.get('source') === 'powerx' && url.searchParams.get('modality') === 'llm') { await reply({ ...settingsCatalog.models, provider: url.searchParams.get('provider'), items: [] }); return; }
      await reply(url.searchParams.get('source') === 'powerx' && url.searchParams.get('modality') === 'image' ? fixture.catalog_result : { items: [] }); return;
    }
    await route.fulfill({ status: 503, json: { reason_code: 'E2E_UNMOCKED_ENDPOINT' } });
  });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto('/admin/settings/ai-settings');
  await expect(page.getByTestId('ai-settings-catalog-source')).toBeVisible();
  await page.getByTestId('ai-settings-catalog-source').click();
  await page.getByRole('option', { name: locale.aiSettings.catalogSources.powerx, exact: true }).click();
  await page.getByTestId('ai-settings-modality-image').click();
  await expect(page.getByTestId('ai-settings-model')).toContainText(fixture.catalog_result.items[0]!.label);
  await page.getByTestId('ai-settings-section-parameters').click();
  await expect(page.locator('[data-testid="ai-media-param-n"] input, input[data-testid="ai-media-param-n"]')).toHaveValue('1');
  await page.locator('[data-testid="ai-media-param-n"] input, input[data-testid="ai-media-param-n"]').fill('2');
  await page.locator('[data-testid="ai-media-param-seed"] input, input[data-testid="ai-media-param-seed"]').fill('0');
  await page.screenshot({ path: info.outputPath('desktop.png'), fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByTestId('ai-settings-modality-picker')).toContainText(locale.aiSettings.modalities.image);
  await page.getByTestId('ai-settings-section-model').click();
  await expect(page.getByTestId('ai-settings-model')).toContainText(fixture.catalog_result.items[0]!.label);
  await page.getByTestId('ai-settings-section-parameters').click();
  await page.getByTestId('ai-settings-modality-picker').click();
  await page.getByRole('option', { name: locale.aiSettings.modalities.llm, exact: true }).click();
  await page.getByTestId('ai-settings-modality-picker').click();
  await page.getByRole('option', { name: locale.aiSettings.modalities.image, exact: true }).click();
  await expect(page.locator('[data-testid="ai-media-param-n"] input, input[data-testid="ai-media-param-n"]')).toHaveValue('2');
  await page.getByTestId('ai-settings-save').click();
  await expect.poll(() => writes.length).toBe(1);
  expect(writes[0].model_key).toBe(fixture.catalog_result.items[0]!.model_key);
  expect(writes[0].parameters).toEqual({ ...fixture.catalog_result.items[0]!.operations.image_generation.defaults, n: 2, seed: 0 });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  await page.screenshot({ path: info.outputPath('narrow.png'), fullPage: true });
  expect(errors).toEqual([]);
});
