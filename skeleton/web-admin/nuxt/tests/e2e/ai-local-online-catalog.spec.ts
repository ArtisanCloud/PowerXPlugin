import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { seedAuthStorage } from './_utils';
const locale = JSON.parse(readFileSync(new URL('../../i18n/locales/zh.json', import.meta.url), 'utf8'));
const fixture = JSON.parse(readFileSync(new URL('../../../../../docs/contracts/fixtures/ai-local-model-catalog.v1.json', import.meta.url), 'utf8'));
const settings = JSON.parse(readFileSync(new URL('../../../../../docs/contracts/fixtures/ai-settings-catalog.v1.json', import.meta.url), 'utf8'));

async function localPage(page: Page, theme = { manual: 'dark', system: 'light' }, admin = true) {
  const state = { syncs: [] as any[], saves: [] as any[], forbidden: [] as string[], modelReads: [] as any[], errors: [] as string[], fail: false, coreExtra: false, catalog: undefined as any };
  page.on('pageerror', error => state.errors.push(error.message));
  await seedAuthStorage(page); await page.emulateMedia({ colorScheme: theme.system as 'light' | 'dark' });
  await page.addInitScript(manual => localStorage.setItem('powerx-color-mode', manual), theme.manual);
  await page.context().addCookies([{ name: 'px_lang', value: 'zh', url: process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3158' }]);
  await page.route('**/api/v1/**', async route => {
    const request = route.request(), url = new URL(request.url()), path = url.pathname;
    const reply = (data: unknown, status = 200) => route.fulfill({ status, json: { success: true, data } });
    if (request.method() === 'OPTIONS') { await route.fulfill({ status: 204 }); return; }
    if (path.endsWith('/admin/user/auth/me/context')) { await reply({ is_root: admin, user: { user_uuid: '00000000-0000-4000-8000-000000000001', display_name: 'fixture.user', status: 1 }, current_tenant_uuid: '40000000-0000-4000-8000-000000000001', tenants: [{ tenant_uuid: '40000000-0000-4000-8000-000000000001', name: 'fixture.tenant' }], members: [], roles: [], permissions: [] }); return; }
    if (path.endsWith('/ai-settings/mode')) { await reply({ mode: 'local', local_available: true, delegated_available: false, connection_probes: { local: true, powerx: true } }); return; }
    if (path.endsWith('/ai-settings/model-profiles')) { await reply({ items: [] }); return; }
    if (path.endsWith('/ai-settings/local-setting')) {
      if (request.method() === 'PUT') { state.saves.push(request.postDataJSON()); await reply({ uuid: '99999999-9999-4999-8999-999999999999' }); return; }
      await reply(url.searchParams.get('source') === 'local' && url.searchParams.get('environment') === 'development' ? { model_key: 'old:8b', provider: 'ollama', endpoint: fixture.input.endpoint, parameters: { temperature: 0.35 } } : {}); return;
    }
    if (path.endsWith('/ai-settings/catalog/providers')) {
      await reply(url.searchParams.get('source') === 'powerx' ? settings.providers : { items: [
        { id: 'ollama', name: 'fixture.ollama', auth: { scheme: 'none', fields: ['base_url'], defaults: { base_url: fixture.input.endpoint } } },
        { id: 'openai', name: 'fixture.openai', auth: { scheme: 'bearer', fields: ['api_key', 'base_url'], defaults: { base_url: 'https://api.openai.com/v1' } } },
        { id: 'coze', name: 'fixture.unsupported', auth: { scheme: 'bearer', fields: ['api_key'], defaults: { base_url: 'https://fixture.invalid' } } },
      ] }); return;
    }
    if (path.endsWith('/ai-settings/catalog/models')) {
      const query = Object.fromEntries(url.searchParams); state.modelReads.push(query);
      if (query.source === 'powerx') { await reply({ ...settings.models, provider: query.provider, app: query.app || '', items: [...settings.models.items.map((m: any) => ({ ...m, provider: query.provider, model_key: `${query.provider}/${m.model}` })), ...(state.coreExtra ? [{ model_key: `${query.provider}/private-core:new`, provider: query.provider, model: 'private-core:new', label: 'private-core:new' }] : [])] }); return; }
      const snapshot = state.catalog?.provider === query.provider && state.catalog.environment === query.environment ? state.catalog : undefined;
      const items = snapshot ? snapshot.models.filter((m: any) => m.modalities.includes(query.modality) || m.manual_modalities.includes(query.modality)).map((m: any) => ({ id: m.id, label: m.label })) : [{ id: query.environment === 'production' ? 'production:tag' : 'old:8b', label: query.environment === 'production' ? 'production:tag' : 'old:8b' }];
      await reply({ items }); return;
    }
    if (path.endsWith('/ai-settings/catalog/local-online')) {
      const body = request.postDataJSON(); state.syncs.push(body);
      if (state.fail) { await route.fulfill({ status: 422, json: { reason_code: 'AI_MODEL_SYNC_AUTH_FAILED', details: { upstream_status: 403 }, trace_id: 'fixture.trace', request_id: 'fixture.request' } }); return; }
      if (body.action === 'assign') {
        expect(body.revision).toBe(state.catalog.revision);
        state.catalog.models = state.catalog.models.map((m: any) => body.models.includes(m.id) ? { ...m, match: 'manual', manual_modalities: [body.modality] } : m);
      } else {
        state.catalog = { ...structuredClone(fixture.result), environment: body.environment, provider: body.provider, source: `${body.endpoint}${body.provider === 'ollama' ? '/api/tags' : '/models'}`, protocol: body.provider === 'ollama' ? 'ollama' : 'openai' };
        state.catalog.models.push({ id: 'old:8b', label: 'old:8b', modalities: ['llm'], manual_modalities: [], match: 'provider_declared' });
        if (body.provider === 'openai') state.catalog.models.push({ id: 'opaque/custom:tag', label: 'opaque/custom:tag', modalities: [], manual_modalities: [], match: 'unmatched' });
      }
      await reply(state.catalog); return;
    }
    if (/test-connection|quick-call|media\/tasks|agents\/models\/sync/.test(path)) state.forbidden.push(path);
    await route.fulfill({ status: 503, json: { reason_code: 'E2E_UNMOCKED_ENDPOINT' } });
  });
  await page.goto('/_p/com.powerx.plugins.base/admin/settings/ai-settings');
  await expect(page.getByTestId('ai-settings-model')).toContainText('old:8b');
  return state;
}

test('local model discovery preserves the draft, saves a native tag and makes one request per explicit open', async ({ page }) => {
  const state = await localPage(page);
  expect(state.syncs).toHaveLength(0);
  await page.getByTestId('ai-settings-online-models').click();
  await expect(page.getByTestId('ai-online-summary')).toBeVisible();
  expect(state.syncs).toEqual([{ ...fixture.input, action: 'refresh' }]);
  expect(state.saves).toHaveLength(0);
  await page.getByRole('button', { name: locale.aiSettings.online.close, exact: true }).click();
  await expect(page.getByTestId('ai-settings-model')).toContainText('old:8b');
  await page.getByTestId('ai-settings-model').click(); await page.getByRole('option', { name: 'private-chat:14b', exact: true }).click();
  await page.getByTestId('ai-settings-save').click();
  await expect.poll(() => state.saves.length).toBe(1);
  expect(state.saves[0]).toMatchObject({ source: 'local', provider: 'ollama', model_key: 'private-chat:14b', endpoint: fixture.input.endpoint, parameters: { temperature: 0.35 } });
  await page.getByTestId('ai-settings-online-models').click(); await expect(page.getByTestId('ai-online-summary')).toBeVisible();
  expect(state.syncs).toHaveLength(2); expect(state.forbidden).toEqual([]); expect(state.errors).toEqual([]);
});

test('a failed local refresh retains the selection and parameters and supports only explicit retry', async ({ page }) => {
  const state = await localPage(page); state.fail = true;
  await page.getByTestId('ai-settings-online-models').click();
  await expect(page.getByTestId('ai-online-error')).toContainText(locale.aiSettings.online.errors.AI_MODEL_SYNC_AUTH_FAILED);
  expect(state.syncs).toHaveLength(1);
  await page.getByRole('button', { name: locale.aiSettings.online.close, exact: true }).click();
  await expect(page.getByTestId('ai-online-page-error')).toBeVisible(); await expect(page.getByTestId('ai-settings-model')).toContainText('old:8b');
  await page.getByTestId('ai-settings-section-parameters').click(); await expect(page.getByTestId('ai-online-page-error')).toBeVisible();
  await page.getByTestId('ai-settings-section-model').click(); state.fail = false;
  await page.getByTestId('ai-settings-online-models').click(); await expect(page.getByTestId('ai-online-summary')).toBeVisible();
  expect(state.syncs).toHaveLength(2); expect(state.saves).toHaveLength(0); expect(state.forbidden).toEqual([]); expect(state.errors).toEqual([]);
});

test('cloud candidates require an explicit versioned classification and keep credentials as references', async ({ page }) => {
  const state = await localPage(page);
  await page.getByTestId('ai-settings-provider').click(); await page.getByRole('option', { name: 'fixture.openai', exact: true }).click();
  await page.getByTestId('ai-settings-credential-ref').fill('22222222-2222-4222-8222-222222222222');
  await page.getByTestId('ai-settings-online-models').click(); await expect(page.getByTestId('ai-online-summary')).toBeVisible();
  await page.getByTestId('ai-online-tab-pending').click();
  await page.getByRole('checkbox', { name: 'opaque/custom:tag', exact: true }).check();
  await page.getByTestId('ai-online-assign').click();
  await expect.poll(() => state.syncs.length).toBe(2);
  expect(state.syncs[1]).toMatchObject({ action: 'assign', revision: fixture.result.revision, models: ['opaque/custom:tag'], credential_ref: '22222222-2222-4222-8222-222222222222' });
  await page.getByRole('button', { name: locale.aiSettings.online.close, exact: true }).click();
  await page.getByTestId('ai-settings-model').click(); await page.getByRole('option', { name: 'opaque/custom:tag', exact: true }).click();
  await page.getByTestId('ai-settings-save').click(); await expect.poll(() => state.saves.length).toBe(1);
  expect(state.saves[0].model_key).toBe('opaque/custom:tag'); expect(JSON.stringify(state.syncs)).not.toContain('api_key'); expect(state.forbidden).toEqual([]); expect(state.errors).toEqual([]);
});

test('PowerX refresh rereads its typed candidates without invoking the administrative synchronization route', async ({ page }) => {
  const state = await localPage(page);
  await page.getByTestId('ai-settings-catalog-source').click(); await page.getByRole('option', { name: locale.aiSettings.catalogSources.powerx, exact: true }).click();
  await expect(page.getByTestId('ai-settings-refresh-models')).toBeEnabled();
  const before = state.modelReads.length; state.coreExtra = true;
  await page.getByTestId('ai-settings-refresh-models').click(); await expect(page.getByTestId('ai-settings-refresh-models')).toBeEnabled();
  expect(state.modelReads.length).toBe(before + 1);
  await page.getByTestId('ai-settings-model').click(); await expect(page.getByRole('option', { name: 'private-core:new', exact: true })).toBeVisible();
  expect(state.syncs).toHaveLength(0); expect(state.saves).toHaveLength(0); expect(state.forbidden).toEqual([]); expect(state.errors).toEqual([]);
});

test('non-administrators cannot submit catalog updates', async ({ page }) => {
  const state = await localPage(page, undefined, false);
  await expect(page.getByTestId('ai-settings-online-models')).toBeDisabled();
  await expect(page.getByText(locale.aiSettings.online.adminRequired, { exact: true })).toBeVisible();
  expect(state.syncs).toHaveLength(0);
});

test('unsupported local providers remain explicit and do not submit discovery requests', async ({ page }) => {
  const state = await localPage(page);
  await page.getByTestId('ai-settings-provider').click(); await page.getByRole('option', { name: 'fixture.unsupported', exact: true }).click();
  await expect(page.getByTestId('ai-settings-online-models')).toBeDisabled(); await expect(page.getByText(locale.aiSettings.online.unsupported, { exact: true })).toBeVisible();
  expect(state.syncs).toHaveLength(0); expect(state.forbidden).toEqual([]);
});

test('a late local catalog read cannot replace another environment after the dialog closes', async ({ page }) => {
  const state = await localPage(page);
  let release: (() => void) | undefined, held = false;
  const gate = new Promise<void>(resolve => { release = resolve; });
  await page.route('**/ai-settings/catalog/models*', async route => {
    const url = new URL(route.request().url());
    if (state.syncs.length && url.searchParams.get('environment') === 'development') { held = true; await gate; await route.fulfill({ json: { success: true, data: { items: [{ id: 'stale:tag', label: 'stale:tag' }] } } }); return; }
    await route.fallback();
  });
  await page.getByTestId('ai-settings-online-models').click(); await expect(page.getByTestId('ai-online-summary')).toBeVisible(); await expect.poll(() => held).toBe(true);
  await page.getByRole('button', { name: locale.aiSettings.online.close, exact: true }).click();
  await page.getByTestId('ai-settings-environment').click(); await page.getByRole('option', { name: locale.aiSettings.environments.production, exact: true }).click();
  await expect(page.getByTestId('ai-settings-model')).toContainText('production:tag'); release!();
  await page.getByTestId('ai-settings-model').click(); await expect(page.getByRole('option', { name: 'production:tag', exact: true })).toBeVisible(); await expect(page.getByRole('option', { name: 'stale:tag', exact: true })).toHaveCount(0);
  expect(state.saves).toHaveLength(0); expect(state.forbidden).toEqual([]); expect(state.errors).toEqual([]);
});

test('a missing local catalog migration is visible and leaves the independent connection probe usable', async ({ page }) => {
  const state = await localPage(page);
  await page.route('**/ai-settings/catalog/models*', route => route.fulfill({ status: 503, json: { reason_code: 'AI_LOCAL_CATALOG_MIGRATION_REQUIRED' } }));
  await page.getByTestId('ai-settings-endpoint').fill('http://127.0.0.1:11435');
  await page.getByTestId('ai-settings-section-parameters').click();
  await expect(page.getByTestId('ai-settings-catalog-error')).toContainText(locale.aiSettings.online.errors.AI_LOCAL_CATALOG_MIGRATION_REQUIRED);
  await page.getByTestId('ai-settings-section-model').click();
  await expect(page.getByTestId('ai-settings-online-models')).toBeDisabled(); await expect(page.getByTestId('ai-settings-save')).toBeDisabled();
  const trigger = page.getByTestId('ai-settings-open-test'); if (await trigger.isVisible()) await trigger.click();
  await expect(page.getByTestId('ai-test-connection')).toBeEnabled(); expect(state.syncs).toHaveLength(0); expect(state.forbidden).toEqual([]); expect(state.errors).toEqual([]);
});

for (const manual of ['light', 'dark']) for (const system of ['light', 'dark']) for (const narrow of [false, true]) {
  test(`local catalog modal remains readable: ${manual}, ${narrow ? 'narrow' : 'desktop'}, system ${system}`, async ({ page }, info) => {
    await page.setViewportSize(narrow ? { width: 390, height: 844 } : { width: 1440, height: 1000 });
    const state = await localPage(page, { manual, system });
    await page.getByTestId('ai-settings-provider').click(); await page.getByRole('option', { name: 'fixture.openai', exact: true }).click();
    await page.getByTestId('ai-settings-credential-ref').fill('22222222-2222-4222-8222-222222222222');
    await page.getByTestId('ai-settings-online-models').click(); await expect(page.getByTestId('ai-online-summary')).toBeVisible();
    await expect(page.locator('html')).toHaveClass(new RegExp(manual));
    await page.getByTestId('ai-online-tab-pending').click(); await expect(page.getByTestId('ai-online-assign')).toBeDisabled();
    async function contrast(minimum = 5) {
      const values = await page.evaluate(() => {
        const canvas = document.createElement('canvas'); canvas.width = canvas.height = 1; const ctx = canvas.getContext('2d', { willReadFrequently: true })!;
        const rgba = (color: string) => { ctx.clearRect(0,0,1,1); ctx.fillStyle = color; ctx.fillRect(0,0,1,1); const c = ctx.getImageData(0,0,1,1).data; return [c[0]!,c[1]!,c[2]!,c[3]!/255]; };
        const background = (el: Element | null): number[] => { if (!el) return [255,255,255]; const c=rgba(getComputedStyle(el).backgroundColor), p=background(el.parentElement); return c.slice(0,3).map((v,i)=>v*c[3]!+p[i]!*(1-c[3]!)); };
        const luminance = (c: number[]) => c.slice(0,3).map(v=>v/255).map(v=>v<=.04045?v/12.92:((v+.055)/1.055)**2.4).reduce((n,v,i)=>n+v*[.2126,.7152,.0722][i]!,0);
        return Array.from(document.querySelectorAll('.ai-settings-online-dialog h2, .ai-settings-online-dialog p, .ai-settings-online-dialog [role="tab"], .ai-settings-online-dialog input, [data-testid="ai-online-refresh"], [data-testid="ai-online-assign"], [data-testid="ai-online-page-error"], [data-testid="ai-settings-online-models"]')).filter(el=>el.getBoundingClientRect().width>0).map(el=>{
          const style = getComputedStyle(el, el instanceof HTMLInputElement && !el.value ? '::placeholder' : null), bg=background(el), color=rgba(style.color), fg=color.slice(0,3).map((v,i)=>v*color[3]!+bg[i]!*(1-color[3]!)), a=luminance(fg), b=luminance(bg);
          return { text: el.getAttribute('data-testid') || el.tagName, ratio:(Math.max(a,b)+.05)/(Math.min(a,b)+.05) };
        });
      });
      expect(values.length).toBeGreaterThan(minimum); for(const value of values) expect(value.ratio,value.text).toBeGreaterThanOrEqual(4.5);
    }
    await contrast(); await page.getByTestId('ai-online-refresh').hover(); await contrast(); await page.getByTestId('ai-online-refresh').focus(); await contrast();
    await page.getByTestId('ai-online-tab-matched').click(); await page.getByTestId('ai-online-tab-matched').focus(); await contrast();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)).toBe(true);
    await page.screenshot({ path: info.outputPath('online-models.png'), fullPage: true });
    await page.getByRole('button', { name: locale.aiSettings.online.close, exact: true }).click();
    state.fail = true; await page.getByTestId('ai-settings-online-models').click(); await expect(page.getByTestId('ai-online-error')).toBeVisible();
    await page.getByRole('button', { name: locale.aiSettings.online.close, exact: true }).click(); await expect(page.getByTestId('ai-online-page-error')).toBeVisible(); await contrast(1);
    await page.screenshot({ path: info.outputPath('online-models-error.png'), fullPage: true });
    expect(state.forbidden).toEqual([]); expect(state.errors).toEqual([]);
  });
}
