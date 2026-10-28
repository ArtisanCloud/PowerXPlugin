import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { seedAuthStorage } from './_utils';
const zh = JSON.parse(readFileSync(new URL('../../i18n/locales/zh.json', import.meta.url), 'utf8'));
const fixture = process.env.REGRESSION_BROWSER_FIXTURE;
const message = (value: string, params: Record<string, string | number>) => value.replace(/\{(\w+)\}/g, (_, key) => String(params[key]));
async function identity(page: Page, readOnly = false) {
  await seedAuthStorage(page);
  const headers = {
    'access-control-allow-origin': new URL(process.env.PLAYWRIGHT_BASE_URL!).origin,
    'access-control-allow-credentials': 'true',
    'access-control-allow-headers': 'authorization,content-type,tenant_uuid,x-tenant-uuid,x-request-id,x-trace-id,accept-language',
    'access-control-allow-methods': 'GET,POST,OPTIONS',
  };
  await page.route('**/admin/user/auth/me/context', route => route.fulfill({ headers, json: { success: true, data: {
    is_root: !readOnly,
    user: { user_uuid: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb', display_name: zh.regression.fixtures.name, status: 1 },
    current_tenant_uuid: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa', current_member_id: 1,
    members: [], roles: [], permissions: readOnly ? ['regression.runs:read'] : ['regression.runs:read', 'regression.runs:invoke', 'regression.runs:manage'],
  } } }));
  return headers;
}
async function businessApi(page: Page) {
  const { api_base } = JSON.parse(readFileSync(fixture!, 'utf8'));
  await page.route('**/admin/regression/**', route => {
    const source = new URL(route.request().url());
    return route.continue({ url: api_base + source.pathname.slice(source.pathname.indexOf('/admin/regression')) + source.search });
  });
  return api_base as string;
}
const preparation = (page: Page, count = 1) => page.getByRole('dialog', { name: message(zh.regression.preparation.title, { count }), exact: true });
async function start(page: Page, count = 1) {
  const created = page.waitForResponse(response => response.request().method() === 'POST' && response.url().endsWith('/admin/regression/runs'));
  await preparation(page, count).getByRole('button', { name: zh.regression.actions.start, exact: true }).click();
  const response = await created;
  expect(response.status()).toBe(202);
  const data = (await response.json()).data;
  await expect(preparation(page, count)).not.toBeVisible();
  await expect(page.getByRole('tab', { name: zh.regression.tabs.runs, exact: true })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByTestId('regression-detail')).toHaveAttribute('data-run-uuid', data.run.uuid);
  return data;
}
async function screenshot(page: Page, testId: string, name: string) {
  const path = test.info().outputPath(name + '.png');
  await expect(page.getByTestId(testId)).toBeVisible();
  if (testId === 'regression-detail') await page.locator('main').evaluate(element => { element.scrollTop = 0; });
  await page.screenshot({ path, animations: 'disabled' });
  await test.info().attach(name, { path, contentType: 'image/png' });
}

test('unavailable catalog gives a clear error without a runnable action', async ({ page }) => {
  test.skip(!fixture, 'REGRESSION_BROWSER_FIXTURE');
  const headers = await identity(page);
  await page.route('**/admin/regression/**', route => route.fulfill({ headers, status: route.request().method() === 'OPTIONS' ? 204 : 404, body: route.request().method() === 'OPTIONS' ? '' : 'not_found' }));
  await page.goto('/admin/regression');
  await expect(page.getByTestId('regression-error')).toContainText(zh.regression.errors.REGRESSION_API_UNAVAILABLE);
  await expect(page.getByTestId('regression-error')).not.toContainText(zh.regression.errors.REGRESSION_INVALID_REQUEST);
  await expect(page.getByTestId('regression-case-count')).toHaveText(zh.regression.overview.unavailable);
  await expect(page.getByTestId('regression-run-case')).toHaveCount(0);
});

test('a user can understand a case, prepare business inputs, run it and inspect real results', async ({ page }) => {
  test.skip(!fixture, 'REGRESSION_BROWSER_FIXTURE');
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  await identity(page);
  const api = await businessApi(page);
  const posts: string[] = [];
  page.on('request', request => { if (request.method() === 'POST' && request.url().endsWith('/admin/regression/runs')) posts.push(request.url()); });
  await page.goto('/admin/regression');
  const row = page.getByTestId('regression-case-template.lifecycle');
  await expect(page.getByRole('tab', { name: zh.regression.tabs.cases, exact: true })).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByTestId('regression-case-count')).toHaveText(message(zh.regression.overview.available, { count: 8 }));
  await expect(row).toContainText(zh.regression.scenarios.template.description);
  await expect(page.getByTestId('regression-case-metadata.tag_bindings')).toHaveCount(0);
  await expect(page.getByTestId('regression-history')).toHaveCount(0);
  await expect(page.getByTestId('regression-detail')).toHaveCount(0);
  await expect(page.getByTestId('regression-batch-actions')).toHaveCount(0);
  await expect(page.getByTestId('regression-workbench').getByRole('textbox')).toHaveCount(1);
  await screenshot(page, 'regression-workbench', 'catalog-light');
  await page.emulateMedia({ colorScheme: 'dark' });
  await expect(page.locator('html')).toHaveClass(/dark/);
  await screenshot(page, 'regression-workbench', 'catalog-dark');

  await row.getByRole('button', { name: zh.regression.actions.viewCase, exact: true }).click();
  const drawer = page.getByRole('dialog', { name: zh.regression.scenarios.template.title, exact: true });
  await expect(drawer).toContainText(message(zh.regression.caseDetail.steps, { count: 3 }));
  for (const key of ['created', 'updated', 'readback']) await expect(drawer).toContainText(zh.regression.assertions[key]);
  await expect(drawer).toContainText(zh.regression.scenarios.template.impact);
  await screenshot(page, 'regression-case-detail', 'case-explanation');
  await drawer.getByRole('button', { name: zh.regression.actions.runCase, exact: true }).click();
  await expect(preparation(page)).toBeVisible();
  await expect(preparation(page).getByLabel(zh.regression.inputs.templateName, { exact: true })).toHaveValue(zh.regression.fixtures.name);
  await expect(preparation(page).getByLabel(zh.regression.inputs.updatedContent, { exact: true })).toHaveValue(zh.regression.fixtures.updatedContent);
  await expect(preparation(page)).toContainText(zh.regression.profiles.controlled);
  await expect(preparation(page)).toContainText(zh.regression.scenarios.template.impact);
  await expect(preparation(page).getByText(zh.regression.scenarios.template.impact, { exact: true })).toBeInViewport();
  expect(posts).toHaveLength(0);
  await preparation(page).getByLabel(zh.regression.inputs.templateName, { exact: true }).fill('');
  await preparation(page).getByRole('button', { name: zh.regression.actions.start, exact: true }).click();
  await expect(preparation(page)).toContainText(message(zh.regression.validation.required, { field: zh.regression.inputs.templateName }));
  expect(posts).toHaveLength(0);
  await preparation(page).getByLabel(zh.regression.inputs.templateName, { exact: true }).fill(zh.regression.fixtures.name);
  await expect(preparation(page)).not.toContainText(message(zh.regression.validation.required, { field: zh.regression.inputs.templateName }));
  await screenshot(page, 'regression-preparation', 'business-input-form');
  await preparation(page).getByRole('button', { name: zh.regression.actions.cancelPreparation, exact: true }).click();
  await expect(preparation(page)).not.toBeVisible();
  expect(posts).toHaveLength(0);

  for (let index = 0; index < 3; index++) {
    await page.getByRole('tab', { name: zh.regression.tabs.cases, exact: true }).click();
    if (index === 1) {
      await row.getByRole('checkbox').check();
      await expect(page.getByTestId('regression-run-selected')).toHaveText(message(zh.regression.actions.runSelected, { count: 1 }));
      await page.getByTestId('regression-run-selected').click();
    } else await row.getByTestId('regression-run-case').click();
    await expect(preparation(page)).toBeVisible();
    const data = await start(page);
    await expect(page.getByTestId('regression-verdict')).toHaveText(zh.regression.states.passed);
    await expect(page.getByTestId('regression-result-counts')).toContainText(message(zh.regression.results.count, { status: zh.regression.states.passed, count: 1 }));
    await expect(page.getByTestId('regression-cleanup')).toContainText(zh.regression.states.succeeded);
    await page.getByRole('tab', { name: zh.regression.resultSections.checks, exact: true }).click();
    await expect(page.getByTestId('regression-check-comparison')).toHaveCount(3);
    await page.getByRole('tab', { name: zh.regression.resultSections.diagnostics, exact: true }).click();
    await expect(page.getByTestId('regression-result-steps').locator(':scope > li')).toHaveCount(3);
    await expect(page.getByTestId('regression-catalog')).toHaveCount(0);
    if (index === 0) await screenshot(page, 'regression-detail', 'passed-result');
    await expect(page.getByTestId('regression-run-snapshot')).toContainText(zh.regression.fields.sourceRevision);
    await expect(page.getByTestId('regression-run-snapshot')).not.toContainText(zh.regression.timing.incomplete);
    await expect(page.getByTestId('regression-dependency-evidence')).toContainText(zh.regression.fields.actualMode);
    await page.getByTestId('regression-result-technical').click();
    const technical = page.getByRole('dialog', { name: zh.regression.actions.technical, exact: true });
    await expect(technical).toContainText('required_assertion_keys');
    await technical.getByRole('button', { name: zh.regression.actions.close, exact: true }).last().click();
    const report = await page.request.get(`${api}/admin/regression/runs/${data.run.uuid}/report`);
    expect(report.ok()).toBeTruthy();
    const result = await report.json();
    expect(result.run.verdict).toBe('passed'); expect(result.run.cleanup_state).toBe('succeeded');
    expect(result.case_executions[0].missing_assertion_keys).toEqual([]);
    await test.info().attach(`business-run-${index + 1}.json`, { body: JSON.stringify(result, null, 2), contentType: 'application/json' });
  }
  await page.request.post(new URL('/_fixture/readback_fault', api).href);
  await page.getByRole('tab', { name: zh.regression.tabs.cases, exact: true }).click();
  await row.getByTestId('regression-run-case').click();
  const failedInput = zh.regression.fixtures.updatedContent + ' #4';
  await preparation(page).getByLabel(zh.regression.inputs.updatedContent, { exact: true }).fill(failedInput);
  const failedData = await start(page);
  await expect(page.getByTestId('regression-verdict')).toHaveText(zh.regression.states.failed);
  await page.getByRole('button', { name: zh.regression.resultActions.locateFailure, exact: true }).click();
  const difference = page.getByTestId('regression-difference');
  await expect(difference).toBeVisible();
  await expect(difference).toContainText(zh.regression.fields.expected);
  await expect(difference).toContainText(zh.regression.fixtures.updatedContent);
  const failedReport = await (await page.request.get(`${api}/admin/regression/runs/${failedData.run.uuid}/report`)).json();
  const failedCheck = failedReport.case_executions.flatMap((item: any) => item.steps.flatMap((step: any) => step.assertions)).find((item: any) => item.status === 'failed');
  expect(failedCheck).toBeTruthy();
  const checkName = failedCheck.title_key.split('.').reduce((value: any, key: string) => value[key], zh);
  await expect(page.getByRole('dialog', { name: checkName, exact: true })).toBeVisible();
  const failedStep = failedReport.case_executions[0].steps.find((item: any) => item.status === 'failed');
  const stepName = failedStep.title_key.split('.').reduce((value: any, key: string) => value[key], zh);
  await expect(page.getByTestId('regression-failure-location')).toContainText(message(zh.regression.results.failureLocation, { name: zh.regression.scenarios.template.title, step: stepName }));
  await screenshot(page, 'regression-detail', 'failed-result');
  await page.getByRole('dialog', { name: checkName, exact: true }).getByRole('button', { name: zh.regression.actions.close, exact: true }).last().click();
  await page.getByRole('tab', { name: zh.regression.resultSections.resources, exact: true }).click();
  await expect(page.getByTestId('regression-failure-location')).toBeVisible();
  await page.getByRole('button', { name: zh.regression.actions.rerun, exact: true }).click();
  await expect(preparation(page)).toBeVisible();
  await preparation(page).getByRole('button', { name: zh.regression.actions.cancelPreparation, exact: true }).click();
  expect(posts).toHaveLength(4);

  await page.request.post(new URL('/_fixture/cancel_gate', api).href);
  await page.getByRole('tab', { name: zh.regression.tabs.cases, exact: true }).click();
  await row.getByTestId('regression-run-case').click(); await start(page);
  await expect(page.getByTestId('regression-detail')).toContainText(zh.regression.results.headlines.running);
  await page.getByRole('tab', { name: zh.regression.resultSections.diagnostics, exact: true }).click();
  await page.getByRole('button', { name: zh.regression.actions.cancel, exact: true }).click();
  await expect(page.getByTestId('regression-detail')).toContainText(zh.regression.results.headlines.cancelled);
  await expect(page.getByRole('tab', { name: zh.regression.resultSections.diagnostics, exact: true })).toHaveAttribute('aria-selected', 'true');
  await page.reload();
  await expect(page.getByTestId('regression-catalog')).toBeVisible();
  await page.getByRole('tab', { name: zh.regression.tabs.runs, exact: true }).click();
  await expect(page.getByTestId('regression-history')).toContainText(zh.regression.scenarios.template.title);
  await expect(page.getByTestId('regression-history')).toContainText(zh.regression.profiles.controlled);
  await screenshot(page, 'regression-history', 'execution-records');
  await page.getByTestId('regression-history').getByRole('button', { name: zh.regression.actions.viewResult, exact: true }).first().click();
  await expect(page.getByTestId('regression-detail')).toContainText(zh.regression.results.headlines.cancelled);
  await page.getByRole('button', { name: zh.regression.actions.backToRuns, exact: true }).click();
  await page.getByTestId('regression-history').getByRole('row').filter({ hasText: zh.regression.states.failed }).getByRole('button', { name: zh.regression.actions.viewResult, exact: true }).click();
  await page.getByRole('button', { name: zh.regression.actions.rerun, exact: true }).click();
  await expect(preparation(page).getByLabel(zh.regression.inputs.updatedContent, { exact: true })).toHaveValue(failedInput);
  await preparation(page).getByRole('button', { name: zh.regression.actions.cancelPreparation, exact: true }).click();
  expect(posts).toHaveLength(5);
  expect(errors).toEqual([]);
});

test('all eight real template cases can be selected and executed as one batch', async ({ page }) => {
  test.skip(!fixture, 'REGRESSION_BROWSER_FIXTURE');
  await page.setViewportSize({ width: 1600, height: 1100 });
  await identity(page);
  const api = await businessApi(page);
  await page.request.post(new URL('/_fixture/reset', api).href);
  await page.goto('/admin/regression');
  await expect(page.getByTestId('regression-case-count')).toHaveText(message(zh.regression.overview.available, { count: 8 }));
  await page.getByRole('checkbox', { name: zh.regression.actions.selectVisible, exact: true }).check();
  await expect(page.getByTestId('regression-run-selected')).toHaveText(message(zh.regression.actions.runSelected, { count: 8 }));
  await screenshot(page, 'regression-catalog', 'eight-case-catalog');
  await page.getByTestId('regression-run-selected').click();
  const dialog = preparation(page, 8);
  await expect(dialog).toBeVisible();
  await expect(dialog.locator('details[open]')).toHaveCount(0);
  const clone = dialog.getByTestId('regression-inputs-template.clone');
  await clone.locator('summary').click();
  await expect(clone.getByLabel(zh.regression.inputs.updatedContent, { exact: true })).toHaveValue(zh.regression.fixtures.updatedContent);
  await clone.locator('summary').click();
  const validation = dialog.getByTestId('regression-inputs-template.validation');
  await validation.locator('summary').click();
  await validation.getByLabel(zh.regression.inputs.repairedContent, { exact: true }).fill('');
  await validation.locator('summary').click();
  await dialog.getByRole('button', { name: zh.regression.actions.start, exact: true }).click();
  await expect(validation).toHaveAttribute('open', '');
  await expect(validation).toContainText(message(zh.regression.validation.required, { field: zh.regression.inputs.repairedContent }));
  await validation.getByLabel(zh.regression.inputs.repairedContent, { exact: true }).fill(zh.regression.fixtures.validationContent);
  await validation.getByLabel(zh.regression.inputs.repairedContent, { exact: true }).blur();
  await expect(validation).not.toContainText(message(zh.regression.validation.required, { field: zh.regression.inputs.repairedContent }));
  await screenshot(page, 'regression-preparation', 'batch-preparation');
  const created = await start(page, 8);
  await expect(page.getByTestId('regression-verdict')).toHaveText(zh.regression.states.passed, { timeout: 20000 });
  await expect(page.getByTestId('regression-result-counts')).toContainText(message(zh.regression.results.count, { status: zh.regression.states.passed, count: 8 }));
  await expect(page.getByTestId('regression-cleanup')).toContainText(zh.regression.states.succeeded);
  const report = await (await page.request.get(`${api}/admin/regression/runs/${created.run.uuid}/report`)).json();
  expect(report.case_executions).toHaveLength(8);
  for (const result of report.case_executions) {
    expect(result.verdict).toBe('passed'); expect(result.cleanup_state).toBe('succeeded'); expect(result.missing_assertion_keys).toEqual([]);
    const name = result.definition_snapshot.title_key.split('.').reduce((value: any, key: string) => value[key], zh);
    await expect(page.getByTestId('regression-detail').getByRole('heading', { name, exact: true })).toBeVisible();
  }
  await screenshot(page, 'regression-detail', 'eight-case-results');
  await test.info().attach('eight-case-business-report.json', { body: JSON.stringify(report, null, 2), contentType: 'application/json' });
});

test('batch selection and environment changes keep the execution scope clear', async ({ page }) => {
  test.skip(!fixture, 'REGRESSION_BROWSER_FIXTURE');
  const { api_base } = JSON.parse(readFileSync(fixture!, 'utf8'));
  const headers = await identity(page);
  const cases = (await (await page.request.get(`${api_base}/admin/regression/cases`)).json()).data.items;
  const profiles = (await (await page.request.get(`${api_base}/admin/regression/profiles`)).json()).data.items;
  // Catalog-only data exercises two configured business forms; no Core or synthetic case is executed.
  const metadata = cases.find((item: any) => item.key === 'metadata.tag_bindings');
  metadata.supported_profile_kinds = ['controlled', 'live'];
  const items = [...profiles, { ...profiles[0], uuid: '55555555-5555-4555-8555-555555555555', key: 'live', kind: 'live', title_key: 'regression.profiles.live' }];
  await page.route('**/admin/regression/**', route => {
    const source = new URL(route.request().url());
    if (route.request().method() === 'OPTIONS') return route.fulfill({ headers, status: 204 });
    if (source.pathname.endsWith('/cases')) return route.fulfill({ headers, json: { success: true, data: { items: cases } } });
    if (source.pathname.endsWith('/profiles')) return route.fulfill({ headers, json: { success: true, data: { items } } });
    return route.continue({ url: api_base + source.pathname.slice(source.pathname.indexOf('/admin/regression')) + source.search });
  });
  await page.goto('/admin/regression');
  const count = page.getByTestId('regression-case-count');
  await expect(count).toHaveText(message(zh.regression.overview.available, { count: 9 }));
  const template = page.getByTestId('regression-case-template.lifecycle');
  await template.getByRole('checkbox').check();
  await expect(page.getByTestId('regression-run-selected')).toHaveText(message(zh.regression.actions.runSelected, { count: 1 }));
  await page.getByLabel(zh.regression.fields.filter, { exact: true }).fill('metadata');
  await expect(template).toHaveCount(0);
  await expect(page.getByTestId('regression-batch-actions')).toHaveCount(0);
  await page.getByTestId('regression-case-metadata.tag_bindings').getByRole('checkbox').check();
  await page.getByRole('combobox').click();
  await page.getByRole('option', { name: zh.regression.profiles.live, exact: true }).click();
  await expect(count).toHaveText(message(zh.regression.overview.available, { count: 1 }));
  await expect(page.getByTestId('regression-batch-actions')).toHaveCount(0);
  await expect(page.getByLabel(zh.regression.fields.filter, { exact: true })).toHaveCount(0);
  await expect(page.getByTestId('regression-case-metadata.tag_bindings')).toBeVisible();
  await page.getByRole('combobox').click();
  await page.getByRole('option', { name: zh.regression.profiles.controlled, exact: true }).click();
  await expect(count).toHaveText(message(zh.regression.overview.available, { count: 9 }));
  await template.getByRole('checkbox').check();
  await page.getByTestId('regression-case-metadata.tag_bindings').getByRole('checkbox').check();
  await page.getByTestId('regression-run-selected').click();
  await expect(preparation(page, 2)).toContainText(zh.regression.scenarios.template.title);
  await expect(preparation(page, 2)).toContainText(zh.regression.scenarios.metadata.title);
  await preparation(page, 2).getByRole('button', { name: zh.regression.actions.start, exact: true }).click();
  await expect(preparation(page, 2)).toContainText(message(zh.regression.validation.required, { field: zh.regression.inputs.resourceType }));
  await expect(preparation(page, 2)).toContainText(message(zh.regression.validation.required, { field: zh.regression.inputs.testTag }));
  await preparation(page, 2).getByRole('button', { name: zh.regression.actions.cancelPreparation, exact: true }).click();
});

test('results separate tasks and retain the chosen case across desktop and narrow navigation', async ({ page }) => {
  test.skip(!fixture, 'REGRESSION_BROWSER_FIXTURE');
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.setViewportSize({ width: 1600, height: 1000 });
  await identity(page); await businessApi(page);
  await page.goto('/admin/regression');
  await page.getByRole('tab', { name: zh.regression.tabs.runs, exact: true }).click();
  const batch = page.getByTestId('regression-history').getByRole('row').filter({ hasText: zh.regression.scenarios.clone.title }).filter({ hasText: zh.regression.scenarios.pagination.title });
  await batch.getByRole('button', { name: zh.regression.actions.viewResult, exact: true }).click();
  const desktop = page.getByTestId('regression-result-nav-desktop');
  const mobile = page.getByTestId('regression-result-nav-mobile');
  await expect(desktop.getByRole('tablist')).toHaveAttribute('aria-orientation', 'vertical');
  await expect(mobile).not.toBeVisible();
  await expect(page.getByTestId('regression-case-results')).toBeVisible();
  await expect(page.getByTestId('regression-result-steps')).toHaveCount(0);
  await expect(page.getByTestId('regression-resource-row')).toHaveCount(0);
  await screenshot(page, 'regression-detail', 'result-overview-desktop');
  await page.getByTestId('regression-case-result-template.clone').getByRole('button', { name: zh.regression.resultActions.inspectChecks, exact: true }).click();
  await expect(page.getByTestId('regression-selected-case')).toHaveText(zh.regression.scenarios.clone.title);
  await expect(page.getByTestId('regression-assertion-template.cloneCount').getByTestId('regression-check-comparison')).toHaveCount(3);
  const contents = page.getByRole('heading', { name: zh.regression.assertions.cloneContents, exact: true }).locator('..').locator('..').locator('..');
  await contents.getByRole('button', { name: zh.regression.resultActions.compare, exact: true }).click();
  const comparison = page.getByRole('dialog', { name: zh.regression.assertions.cloneContents, exact: true });
  await expect(page.getByTestId('regression-difference').locator(':scope > div')).toHaveCount(20);
  await expect(comparison).toContainText(zh.regression.fixtures.content);
  await screenshot(page, 'regression-difference', 'result-business-comparison');
  await comparison.getByRole('button', { name: zh.regression.actions.close, exact: true }).last().click();
  await expect(comparison).not.toBeVisible();
  await screenshot(page, 'regression-detail', 'result-checks-desktop');
  await desktop.getByRole('tab', { name: zh.regression.resultSections.resources, exact: true }).click();
  await expect(desktop.getByRole('tab', { name: zh.regression.resultSections.resources, exact: true })).toHaveAttribute('aria-selected', 'true');
  const resources = page.getByTestId('regression-resource-row');
  await expect(resources).toHaveCount(6);
  await expect(resources.filter({ hasText: zh.regression.resourceRoles.source })).toHaveCount(2);
  await expect(resources.filter({ hasText: zh.regression.resourceRoles.copy })).toHaveCount(4);
  for (const resource of await resources.all()) await expect(resource).toContainText(message(zh.regression.resultLabels.readbackRecorded, { count: 1 }));
  expect(await page.getByTestId('regression-result-content').innerText()).not.toMatch(/[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}/i);
  await screenshot(page, 'regression-detail', 'result-resources-desktop');
  await resources.first().getByRole('button', { name: zh.regression.resultActions.cleanupEvidence, exact: true }).click();
  const evidence = page.getByRole('dialog', { name: zh.regression.resultActions.evidence, exact: true });
  await expect(evidence).toContainText('template.read_after_delete');
  await expect(evidence).toContainText('"exists": false');
  await evidence.getByRole('button', { name: zh.regression.actions.close, exact: true }).last().click();
  await expect(evidence).not.toBeVisible();
  await desktop.getByRole('tab', { name: zh.regression.resultSections.checks, exact: true }).click();
  await expect(page.getByTestId('regression-selected-case')).toHaveText(zh.regression.scenarios.clone.title);
  await page.getByRole('combobox', { name: zh.regression.resultLabels.currentCase, exact: true }).click();
  await page.getByRole('option', { name: zh.regression.scenarios.pagination.title, exact: true }).click();
  await desktop.getByRole('tab', { name: zh.regression.resultSections.diagnostics, exact: true }).click();
  await expect(page.getByTestId('regression-selected-case')).toHaveText(zh.regression.scenarios.pagination.title);
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(desktop).not.toBeVisible(); await expect(mobile).toBeVisible();
  await expect(page.getByTestId('standalone-desktop-navigation')).not.toBeVisible();
  const menuButton = page.getByRole('button', { name: zh.navigation.openMenu, exact: true });
  await expect(menuButton).toBeVisible();
  await menuButton.click();
  const menu = page.getByRole('dialog', { name: zh.navigation.primaryMenu, exact: true });
  await expect(menu.getByRole('link', { name: zh.regression.title, exact: true })).toBeVisible();
  await menu.getByRole('link', { name: zh.regression.title, exact: true }).click();
  await expect(menu).not.toBeVisible();
  await expect(page.getByTestId('regression-selected-case')).toHaveText(zh.regression.scenarios.pagination.title);
  await expect(mobile.getByRole('tab', { name: zh.regression.resultSections.diagnostics, exact: true })).toHaveAttribute('aria-selected', 'true');
  await mobile.getByRole('tab', { name: zh.regression.resultSections.resources, exact: true }).click();
  await expect(page.getByTestId('regression-selected-case')).toHaveText(zh.regression.scenarios.pagination.title);
  await expect(resources).toHaveCount(5);
  await screenshot(page, 'regression-detail', 'result-resources-narrow');
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
  await mobile.getByRole('tab', { name: zh.regression.resultSections.checks, exact: true }).click();
  await expect(page.getByTestId('regression-result-steps')).toHaveCount(0);
  await screenshot(page, 'regression-detail', 'result-checks-narrow-light');
  await page.emulateMedia({ colorScheme: 'dark' });
  await expect(page.locator('html')).toHaveClass(/dark/);
  await screenshot(page, 'regression-detail', 'result-checks-narrow-dark');
  const firstCheck = page.getByTestId('regression-check-comparison').first();
  await firstCheck.scrollIntoViewIfNeeded();
  await screenshot(page, 'regression-result-content', 'result-comparison-narrow-dark');
  await page.getByTestId('regression-result-content').getByRole('button', { name: zh.regression.resultActions.compare, exact: true }).first().click();
  const narrowComparison = page.getByRole('dialog', { name: zh.regression.assertions.pagination, exact: true });
  await expect(narrowComparison).toBeVisible();
  await expect.poll(async () => {
    const bounds = await narrowComparison.boundingBox();
    return bounds ? Math.round(bounds.x + bounds.width) : Infinity;
  }).toBeLessThanOrEqual(390);
  await screenshot(page, 'regression-difference', 'result-drawer-narrow-dark');
  await narrowComparison.getByRole('button', { name: zh.regression.actions.close, exact: true }).last().click();
  await expect(narrowComparison).not.toBeVisible();
  await expect(page.getByTestId('regression-selected-case')).toHaveText(zh.regression.scenarios.pagination.title);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBeTruthy();
  expect(errors).toEqual([]);
});

test('rendering an incomplete saved result keeps cleanup failures and presentation gaps discoverable', async ({ page }) => {
  test.skip(!fixture, 'REGRESSION_BROWSER_FIXTURE');
  const { api_base } = JSON.parse(readFileSync(fixture!, 'utf8'));
  const headers = await identity(page, true);
  const runs = (await (await page.request.get(`${api_base}/admin/regression/runs`)).json()).data.items;
  const run = runs.find((item: any) => item.selection_snapshot.length === 8);
  const snapshot = (await (await page.request.get(`${api_base}/admin/regression/runs/${run.uuid}`)).json()).data;
  // UI contract fixture only: render a saved failure without changing business data or issuing cleanup.
  snapshot.run.verdict = 'failed'; snapshot.run.cleanup_state = 'failed';
  const cleanupCase = snapshot.case_executions.find((item: any) => item.definition_snapshot.key === 'template.pagination');
  cleanupCase.verdict = 'failed'; cleanupCase.cleanup_state = 'failed';
  cleanupCase.fixture_resources[0].cleanup_state = 'failed'; cleanupCase.fixture_resources[0].cleanup_evidence = []; cleanupCase.fixture_resources[0].cleaned_at = null;
  snapshot.case_executions[0].definition_snapshot.key = 'unconfigured.case';
  const posts: string[] = [];
  await page.route('**/admin/regression/**', route => {
    const source = new URL(route.request().url());
    if (route.request().method() === 'OPTIONS') return route.fulfill({ headers, status: 204 });
    if (route.request().method() === 'POST') posts.push(source.pathname);
    if (source.pathname.endsWith(`/runs/${run.uuid}`)) return route.fulfill({ headers, json: { success: true, data: snapshot } });
    if (source.pathname.endsWith('/runs')) return route.fulfill({ headers, json: { success: true, data: { items: [snapshot.run] } } });
    if (source.pathname.endsWith('/events')) return route.fulfill({ headers: { ...headers, 'content-type': 'text/event-stream' }, body: '' });
    return route.continue({ url: api_base + source.pathname.slice(source.pathname.indexOf('/admin/regression')) + source.search });
  });
  await page.goto('/admin/regression');
  await page.getByRole('tab', { name: zh.regression.tabs.runs, exact: true }).click();
  await page.getByRole('button', { name: zh.regression.actions.viewResult, exact: true }).click();
  await page.getByRole('button', { name: zh.regression.resultActions.inspectResources, exact: true }).click();
  await expect(page.getByTestId('regression-selected-case')).toHaveText(zh.regression.scenarios.pagination.title);
  await expect(page.getByTestId('regression-resource-row').first()).toContainText(zh.regression.resultLabels.readbackMissing);
  for (const section of ['overview', 'checks', 'diagnostics']) {
    await page.getByRole('tab', { name: zh.regression.resultSections[section], exact: true }).click();
    await expect(page.getByText(zh.regression.errors.REGRESSION_CLEANUP_FAILED, { exact: true })).toBeVisible();
  }
  await page.getByRole('tab', { name: zh.regression.resultSections.overview, exact: true }).click();
  await page.getByTestId('regression-case-result-unconfigured.case').getByRole('button', { name: zh.regression.resultActions.inspectChecks, exact: true }).click();
  await expect(page.getByText(zh.regression.resultHelp.presentationMissing, { exact: true }).first()).toBeVisible();
  await expect(page.getByTestId('regression-check-comparison')).toHaveCount(0);
  await expect(page.getByText(zh.regression.errors.REGRESSION_CLEANUP_FAILED, { exact: true })).toBeVisible();
  expect(posts).toEqual([]);
});

test('read-only users can understand cases and inspect records without execution controls', async ({ page }) => {
  test.skip(!fixture, 'REGRESSION_BROWSER_FIXTURE');
  await identity(page, true); await businessApi(page);
  await page.goto('/admin/regression');
  const row = page.getByTestId('regression-case-template.lifecycle');
  await expect(row).toBeVisible();
  await expect(row.getByRole('checkbox')).toHaveCount(0);
  await expect(row.getByTestId('regression-run-case')).toHaveCount(0);
  await row.getByRole('button', { name: zh.regression.actions.viewCase, exact: true }).click();
  const drawer = page.getByRole('dialog', { name: zh.regression.scenarios.template.title, exact: true });
  await expect(drawer).toContainText(zh.regression.scenarios.template.impact);
  await expect(drawer.getByRole('button', { name: zh.regression.actions.runCase, exact: true })).toHaveCount(0);
  await drawer.getByRole('button', { name: zh.regression.actions.close, exact: true }).last().click();
  await page.getByRole('tab', { name: zh.regression.tabs.runs, exact: true }).click();
  await page.getByTestId('regression-history').getByRole('button', { name: zh.regression.actions.viewResult, exact: true }).first().click();
  await expect(page.getByTestId('regression-detail')).toBeVisible();
  await expect(page.getByRole('button', { name: zh.regression.actions.rerun, exact: true })).toHaveCount(0);
});

test('a case missing its business form fails explicitly without a JSON fallback', async ({ page }) => {
  test.skip(!fixture, 'REGRESSION_BROWSER_FIXTURE');
  const { api_base } = JSON.parse(readFileSync(fixture!, 'utf8'));
  const headers = await identity(page);
  const cases = (await (await page.request.get(`${api_base}/admin/regression/cases`)).json()).data.items;
  const unknown = { ...cases.find((item: any) => item.key === 'template.lifecycle'), key: 'unconfigured.case' };
  await page.route('**/admin/regression/**', route => {
    const source = new URL(route.request().url());
    if (route.request().method() === 'OPTIONS') return route.fulfill({ headers, status: 204 });
    if (source.pathname.endsWith('/cases')) return route.fulfill({ headers, json: { success: true, data: { items: [unknown] } } });
    return route.continue({ url: api_base + source.pathname.slice(source.pathname.indexOf('/admin/regression')) + source.search });
  });
  await page.goto('/admin/regression');
  await page.getByTestId('regression-case-unconfigured.case').getByTestId('regression-run-case').click();
  await expect(preparation(page)).toContainText(zh.regression.errors.REGRESSION_PRESENTATION_MISSING);
  await expect(preparation(page).getByRole('textbox')).toHaveCount(0);
  await expect(preparation(page).getByRole('button', { name: zh.regression.actions.start, exact: true })).toBeDisabled();
});
