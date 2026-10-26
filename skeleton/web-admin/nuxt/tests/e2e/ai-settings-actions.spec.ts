import { test, expect, type Page } from '@playwright/test';
import { seedAuthStorage } from './_utils';
import { readFileSync } from 'node:fs';
const locale = JSON.parse(readFileSync(new URL('../../i18n/locales/zh.json', import.meta.url), 'utf8'));
const fixture = JSON.parse(readFileSync(new URL('../../../../../docs/contracts/fixtures/ai-media-v2.json', import.meta.url), 'utf8'));
const settingsCatalog = JSON.parse(readFileSync(new URL('../../../../../docs/contracts/fixtures/ai-settings-catalog.v1.json', import.meta.url), 'utf8'));

async function setup(page: Page, mode: 'local' | 'delegated' = 'delegated', failFirst = false) {
  const calls: any[] = [], saves: any[] = [], errors: string[] = [];
  let refreshed = 0;
  page.on('pageerror', error => errors.push(error.message));
  await seedAuthStorage(page);
  await page.context().addCookies([{ name: 'px_lang', value: 'zh', url: process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3147' }]);
  await page.route('**/api/v1/**', async route => {
    const request = route.request(), url = new URL(request.url()), path = url.pathname;
    const reply = (data: unknown, status = 200) => route.fulfill({ status, json: { success: true, data } });
    if (request.method() === 'OPTIONS') { await route.fulfill({ status: 204 }); return; }
    if (path.endsWith('/admin/user/auth/me/context')) { await reply({ is_root: true, user: { user_uuid: '00000000-0000-4000-8000-000000000001', display_name: 'fixture.user', status: 1 }, current_tenant_uuid: '40000000-0000-4000-8000-000000000001', tenants: [{ tenant_uuid: '40000000-0000-4000-8000-000000000001', name: 'fixture.tenant' }], members: [], roles: [], permissions: [] }); return; }
    if (path.endsWith('/ai-settings/mode')) { await reply({ mode, local_available: mode === 'local', delegated_available: mode === 'delegated' }); return; }
    if (path.endsWith('/ai-settings/model-profiles')) { await reply({ items: [] }); return; }
    if (path.endsWith('/ai-settings/local-setting')) {
      if (request.method() === 'PUT') saves.push(request.postDataJSON());
      await reply({}); return;
    }
    if (path.endsWith('/ai-settings/catalog/providers')) {
      await reply(url.searchParams.get('source') === 'powerx' && url.searchParams.get('modality') === 'llm' ? settingsCatalog.providers : { items: [{ id: 'qwen-cn', name: 'fixture.provider' }] }); return;
    }
    if (path.endsWith('/ai-settings/catalog/models')) {
      if (url.searchParams.get('source') === 'powerx' && url.searchParams.get('modality') === 'llm') {
        const provider = url.searchParams.get('provider');
        await reply(provider === 'ollama' ? settingsCatalog.models : { ...settingsCatalog.models, provider, items: [] }); return;
      }
      const capability = ({ vlm: 'vision_understanding', image: 'image_generation', video: 'video_generation', audio_tts: 'speech_synthesis' } as any)[url.searchParams.get('modality') || ''];
      if (!capability || url.searchParams.get('source') !== 'powerx') { await reply({ items: [{ id: 'fixture.local', label: 'fixture.local' }] }); return; }
      const model = structuredClone(fixture.catalog_result.items[0]);
      model.capabilities = [capability];
      model.operations = { [capability]: { ...model.operations.image_generation, input_types: ['text', 'image_url'], defaults: {} } };
      await reply({ contract_version: 2, items: [model] }); return;
    }
    if (path.endsWith('/test-connection') || path.endsWith('/quick-call')) {
      const body = request.postDataJSON(); calls.push({ path, body });
      if (failFirst && calls.length === 1) { await route.fulfill({ status: 503, json: { error_code: 'integration.invoke_failed', reason_code: 'AI_ENV_NOT_CONFIGURED', trace_id: 'fixture.trace' } }); return; }
      if (body.modality === 'vlm') { await reply({ contract_version: 2, type: 'analysis', text: 'fixture.analysis', provider: 'fixture.provider', model: 'fixture.model', trace_id: 'fixture.trace' }); return; }
      const capability = ({ image: 'image_generation', video: 'video_generation', audio_tts: 'speech_synthesis' } as any)[body.modality];
      await reply({ ...fixture.task_result, model_key: body.model_key, capability, status: 'queued', phase: 'queued', revision: 1, assets: [] }, 202); return;
    }
    if (path.includes('/media/tasks/')) {
      if (path.endsWith('/cancel')) { await reply({ ...fixture.task_result, status: 'cancelled', revision: 6, cancel_status: 'confirmed' }); return; }
      refreshed++;
      await reply({ ...fixture.task_result, revision: refreshed === 1 ? 5 : 2 }); return;
    }
    await route.fulfill({ status: 503, json: { reason_code: 'E2E_UNMOCKED_ENDPOINT' } });
  });
  await page.goto('/admin/settings/ai-settings');
  await expect(page.getByTestId('ai-settings-catalog-source')).toBeVisible();
  return { calls, saves, errors };
}
async function showTests(page: Page) {
  const trigger = page.getByTestId('ai-settings-open-test');
  // The layout follows the available workspace width, including the app sidebar.
  await expect.poll(async () => await trigger.isVisible() || await page.getByTestId('ai-test-quick').isVisible()).toBe(true);
  if (await trigger.isVisible()) await trigger.click();
  await expect(page.getByTestId('ai-test-quick')).toBeVisible();
}

async function selectCore(page: Page) {
  await page.getByTestId('ai-settings-catalog-source').click();
  await page.getByRole('option', { name: locale.aiSettings.catalogSources.powerx, exact: true }).click();
}

test('generation retry retains its attempt, tasks stay typed and old refreshes cannot overwrite state', async ({ page }, info) => {
  const state = await setup(page, 'delegated', true);
  await selectCore(page);
  await page.getByTestId('ai-settings-modality-image').click();
  await showTests(page);
  await expect(page.getByTestId('ai-test-connection')).toBeEnabled();
  await page.getByTestId('ai-test-connection').click();
  await expect(page.getByTestId('ai-test-result')).toHaveText(locale.aiSettings.testing.failed);
  await page.getByTestId('ai-settings-section-parameters').click();
  await expect(page.getByTestId('ai-settings-status')).toHaveText(locale.aiSettings.workspace.testFailed);
  await showTests(page);
  await page.getByTestId('ai-test-quick').click();
  await expect(page.getByTestId('ai-test-task-status')).toHaveText(locale.aiSettings.testing.taskStatus.queued);
  expect(state.calls).toHaveLength(2);
  expect(state.calls[0].body).toEqual(state.calls[1].body);
  expect(state.calls[0].body.idempotency_key).toMatch(/^[a-z0-9-]{36}$/);
  await expect(page.getByTestId('ai-test-quick')).toBeDisabled();
  await page.getByTestId('ai-test-refresh').click();
  await expect(page.getByTestId('ai-test-task-status')).toHaveText(locale.aiSettings.testing.taskStatus.succeeded);
  await page.getByTestId('ai-test-refresh').click();
  await expect(page.getByTestId('ai-test-task-status')).toHaveText(locale.aiSettings.testing.taskStatus.succeeded);
  await page.getByTestId('ai-settings-modality-llm').click();
  await page.getByTestId('ai-settings-modality-image').click();
  await expect(page.getByTestId('ai-test-task-status')).toHaveText(locale.aiSettings.testing.taskStatus.succeeded);
  await page.getByRole('heading', { name: locale.aiSettings.title, exact: true }).scrollIntoViewIfNeeded();
  await page.screenshot({ path: info.outputPath('desktop.png'), fullPage: true });
  await page.setViewportSize({ width: 390, height: 844 });
  await showTests(page);
  await expect(page.getByTestId('ai-test-new')).toBeEnabled();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  await page.screenshot({ path: info.outputPath('narrow.png'), fullPage: true });
  await page.getByTestId('ai-test-new').click();
  await page.getByTestId('ai-test-quick').click();
  await expect(page.getByTestId('ai-test-task-status')).toHaveText(locale.aiSettings.testing.taskStatus.queued);
  expect(state.calls[2].body.idempotency_key).not.toBe(state.calls[1].body.idempotency_key);
  await page.getByTestId('ai-test-cancel').click();
  await expect(page.getByTestId('ai-test-task-status')).toHaveText(locale.aiSettings.testing.taskStatus.cancelled);
  expect(state.errors).toEqual([]);
});

test('catalog failures remain visible in every section and retry preserves the selected environment', async ({ page }) => {
  const state = await setup(page, 'local');
  await expect(page.getByTestId('ai-settings-model')).toContainText('fixture.local');
  let attempts = 0;
  let recovered = false;
  await page.route('**/ai-settings/catalog/providers*', async route => {
    attempts++;
    if (!recovered) await route.fulfill({ status: 503, json: { error_code: 'fixture.catalog_unavailable' } });
    else await route.fulfill({ json: { success: true, data: { items: [{ id: 'qwen-cn', name: 'fixture.provider' }] } } });
  });
  await page.getByTestId('ai-settings-environment').click();
  await page.getByRole('option', { name: locale.aiSettings.environments.production, exact: true }).click();
  const error = page.getByTestId('ai-settings-catalog-error');
  await expect(error).toBeVisible();
  await page.getByTestId('ai-settings-section-parameters').click();
  await expect(error).toBeVisible();
  const failures = attempts;
  recovered = true;
  await page.getByTestId('ai-settings-retry-catalog').click();
  await expect(error).toBeHidden();
  await expect(page.getByTestId('ai-settings-environment')).toContainText(locale.aiSettings.environments.production);
  expect(attempts).toBe(failures + 1);
  expect(state.errors).toEqual([]);
});

test('vision requires an image and keeps the analysis result distinct from a task', async ({ page }) => {
  const state = await setup(page);
  await selectCore(page);
  await page.getByTestId('ai-settings-modality-vlm').click();
  await showTests(page);
  await expect(page.getByTestId('ai-test-quick')).toBeDisabled();
  await page.locator('textarea[data-testid="ai-test-image"], [data-testid="ai-test-image"] textarea').fill('https://fixture.invalid/image.png');
  await expect(page.getByTestId('ai-test-quick')).toBeEnabled();
  await page.getByTestId('ai-test-quick').click();
  await expect(page.getByTestId('ai-test-result')).toHaveText('fixture.analysis');
  expect(state.calls[0].body.modality).toBe('vlm');
  expect(state.calls[0].body.inputs[1]).toEqual({ type: 'image_url', role: 'user', url: 'https://fixture.invalid/image.png' });
  await expect(page.getByTestId('ai-test-task-status')).toHaveCount(0);
  expect(state.errors).toEqual([]);
});

test('local generation and unsupported modalities are disabled without submitting requests', async ({ page }) => {
  const state = await setup(page, 'local');
  await showTests(page);
  for (const modality of ['image', 'video', 'audio_tts', 'embedding', 'audio_asr', 'model3d', 'rerank']) {
    await page.getByTestId(`ai-settings-modality-${modality}`).click();
    await expect(page.getByTestId('ai-test-connection')).toBeDisabled();
    await expect(page.getByTestId('ai-test-quick')).toBeDisabled();
    await expect(page.getByTestId('ai-test-blocked')).toBeVisible();
  }
  expect(state.calls).toHaveLength(0); expect(state.errors).toEqual([]);
});

test('configuration keeps its draft across sections, scopes and mobile navigation before saving the selected scope', async ({ page }) => {
  const state = await setup(page, 'local');
  const endpoint = page.locator('input[data-testid="ai-settings-endpoint"], [data-testid="ai-settings-endpoint"] input');
  await expect(page.getByTestId('ai-settings-model')).toContainText('fixture.local');
  await endpoint.fill('http://127.0.0.1:11435');
  await page.getByTestId('ai-settings-section-model').focus();
  await page.keyboard.press('ArrowRight');
  await expect(page.getByTestId('ai-settings-section-parameters')).toHaveAttribute('aria-selected', 'true');
  await page.getByLabel(locale.aiSettings.temperature, { exact: true }).fill('0.3');
  await page.getByTestId('ai-settings-section-model').click();
  await expect(endpoint).toHaveValue('http://127.0.0.1:11435');
  await page.getByTestId('ai-settings-environment').click();
  await page.getByRole('option', { name: locale.aiSettings.environments.production, exact: true }).click();
  await expect(endpoint).toHaveValue('');
  await endpoint.fill('http://127.0.0.1:11436');
  await page.getByTestId('ai-settings-environment').click();
  await page.getByRole('option', { name: locale.aiSettings.environments.development, exact: true }).click();
  await expect(endpoint).toHaveValue('http://127.0.0.1:11435');
  await selectCore(page);
  await expect(endpoint).toHaveCount(0);
  await expect(page.getByTestId('ai-settings-core-managed')).toBeVisible();
  await page.getByTestId('ai-settings-catalog-source').click();
  await page.getByRole('option', { name: locale.aiSettings.catalogSources.local, exact: true }).click();
  await expect(endpoint).toHaveValue('http://127.0.0.1:11435');
  await page.setViewportSize({ width: 390, height: 844 });
  const picker = page.getByTestId('ai-settings-modality-picker');
  await expect(picker).toBeVisible();
  await expect(page.getByTestId('ai-settings-modality-llm')).toBeHidden();
  await picker.click();
  await page.getByRole('option', { name: locale.aiSettings.modalities.vlm, exact: true }).click();
  await expect(endpoint).toHaveCount(0);
  await picker.click();
  await page.getByRole('option', { name: locale.aiSettings.modalities.llm, exact: true }).click();
  await expect(endpoint).toHaveValue('http://127.0.0.1:11435');
  await page.getByTestId('ai-settings-save').click();
  await expect.poll(() => state.saves.length).toBe(1);
  expect(state.saves[0]).toMatchObject({ source: 'local', environment: 'development', modality: 'llm', endpoint: 'http://127.0.0.1:11435', parameters: { temperature: 0.3 } });
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
  expect(state.errors).toEqual([]);
});

test('the test panel retains inputs when closed and when moving between a drawer and the desktop column', async ({ page }) => {
  const state = await setup(page, 'local');
  await expect(page.getByTestId('ai-settings-model')).toContainText('fixture.local');
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByTestId('ai-settings-open-test')).toBeVisible();
  await showTests(page);
  const prompt = page.locator('textarea[data-testid="ai-test-prompt"], [data-testid="ai-test-prompt"] textarea');
  await prompt.fill('fixture.retained-prompt');
  await page.getByRole('button', { name: locale.aiSettings.workspace.closeTests, exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.getByTestId('ai-settings-status')).toBeVisible();
  await showTests(page);
  await expect(prompt).toHaveValue('fixture.retained-prompt');
  await page.setViewportSize({ width: 1440, height: 1000 });
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.getByTestId('ai-settings-open-test')).toHaveCount(0);
  await expect(prompt).toHaveValue('fixture.retained-prompt');
  const config = await page.locator('#ai-settings-workspace').boundingBox();
  const tests = await page.locator('.ai-settings-test-panel').boundingBox();
  expect(tests!.x).toBeGreaterThanOrEqual(config!.x + config!.width);
  expect(Math.abs(tests!.y - config!.y)).toBeLessThan(2);
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(page.getByTestId('ai-settings-open-test')).toBeVisible();
  await expect(page.getByTestId('ai-test-quick')).toHaveCount(0);
  await showTests(page);
  await expect(prompt).toHaveValue('fixture.retained-prompt');
  expect(state.errors).toEqual([]);
});

test('PowerX text models and providers come only from the settings catalog and never carry a local connection', async ({ page }) => {
  const state = await setup(page);
  await expect(page.getByTestId('ai-settings-model')).toContainText('fixture.local');
  await page.route('**/ai-settings/local-setting*', async route => {
    if (route.request().method() !== 'GET') { await route.fallback(); return; }
    await route.fulfill({ json: { success: true, data: { provider: 'ollama', model_key: 'ollama/stale:8b', endpoint: 'https://api.deepseek.com', credential_ref: 'fixture.stale-secret-ref' } } });
  });
  await selectCore(page);
  await expect(page.getByTestId('ai-settings-provider')).toContainText(settingsCatalog.providers.providers[1].name);
  await expect(page.getByTestId('ai-settings-model')).toBeEnabled();
  await expect(page.getByTestId('ai-settings-save')).toBeDisabled();
  await expect(page.getByTestId('ai-settings-endpoint')).toHaveCount(0);
  await expect(page.getByTestId('ai-settings-core-managed')).toHaveText(locale.aiSettings.coreConnectionManaged);
  await page.getByTestId('ai-settings-model').click();
  await expect(page.getByRole('option', { name: 'ollama/stale:8b', exact: true })).toHaveCount(0);
  await expect(page.getByRole('option', { name: 'fixture.local', exact: true })).toHaveCount(0);
  await page.getByRole('option', { name: settingsCatalog.models.items[0].label, exact: true }).click();
  await expect(page.getByTestId('ai-settings-save')).toBeEnabled();
  await page.getByTestId('ai-settings-save').click();
  await expect.poll(() => state.saves.length).toBe(1);
  expect(state.saves[0]).toMatchObject({ source: 'powerx', provider: 'ollama', model_key: settingsCatalog.models.items[0].model_key, endpoint: '', credential_ref: '' });
  expect(state.errors).toEqual([]);
});

test('a legacy generative catalog cannot appear as the PowerX AI Settings directory', async ({ page }) => {
  const state = await setup(page);
  await expect(page.getByTestId('ai-settings-model')).toContainText('fixture.local');
  await page.route('**/ai-settings/catalog/providers*', async route => {
    await route.fulfill({ json: { success: true, data: { items: [{ id: 'ollama', name: 'fixture.legacy-provider' }] } } });
  });
  await selectCore(page);
  await expect(page.getByTestId('ai-settings-catalog-error')).toContainText(locale.aiSettings.coreCatalogUnavailableDescription);
  await expect(page.getByTestId('ai-settings-provider')).toContainText(locale.aiSettings.notSelected);
  await expect(page.getByTestId('ai-settings-model')).toContainText(locale.aiSettings.notSelected);
  await expect(page.getByTestId('ai-settings-save')).toBeDisabled();
  await expect(page.getByTestId('ai-settings-model')).toBeDisabled();
  await showTests(page);
  await expect(page.getByTestId('ai-test-connection')).toBeDisabled();
  expect(state.calls).toHaveLength(0);
  expect(state.errors).toEqual([]);
});
