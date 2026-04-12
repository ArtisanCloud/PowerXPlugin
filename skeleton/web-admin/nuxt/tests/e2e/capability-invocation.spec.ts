import { test, expect } from '@playwright/test';
import { gotoWithFallback, seedAuthStorage } from './_utils';

const capabilityEndpoint = '**/integration/capabilities/invoke';
const localEndpoint = '**/tests/local-api*';

test.describe('Capability invocation playground', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuthStorage(page);
  });

  test('surfaces success toast and trace id', async ({ page }) => {
    await page.route(capabilityEndpoint, async (route) => {
      const body = route.request().postDataJSON() as Record<string, any>;
      expect(body?.capabilityId).toBe('com.corex.media.assets.manage');
      expect(body?.action).toBe('TestAction');
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        headers: {
          'x-trace-id': 'trace-cli-success'
        },
        body: JSON.stringify({
          traceId: 'trace-cli-success',
          status: 'ok',
          data: {
            result: 'success'
          }
        })
      });
    });

    await gotoWithFallback(page, '/tests/capability', page.getByTestId('trigger-success'));
    await page.getByTestId('trigger-success').click();

    await expect(page.getByTestId('trace-output')).toHaveText(/trace-cli-success/);
    await expect(page.getByText('能力调用成功', { exact: true })).toBeVisible();
  });

  test('handles failure toast and error indicator', async ({ page }) => {
    await page.route(capabilityEndpoint, async (route) => {
      await route.fulfill({
        status: 429,
        contentType: 'application/json',
        headers: {
          'x-trace-id': 'trace-cli-failure'
        },
        body: JSON.stringify({
          traceId: 'trace-cli-failure',
          error: {
            code: 'RATE_LIMIT',
            message: 'rate limited'
          }
        })
      });
    });

    await gotoWithFallback(page, '/tests/capability', page.getByTestId('trigger-fail'));
    await page.getByTestId('trigger-fail').click();

    await expect(page.getByTestId('status-indicator')).toHaveText(/error/);
    await expect(page.getByTestId('error-indicator')).toHaveText('error');
    await expect(page.getByText('能力调用失败', { exact: true })).toBeVisible();
  });

  test('shows mock banner when mock response returned', async ({ page }) => {
    await page.route(capabilityEndpoint, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        headers: {
          'x-trace-id': 'trace-cli-mock'
        },
        body: JSON.stringify({
          traceId: 'trace-cli-mock',
          status: 'mock',
          data: {
            mock: true,
            module: 'media',
            message: 'Mock 模式生效'
          }
        })
      });
    });

    await gotoWithFallback(page, '/tests/capability', page.getByTestId('trigger-mock'));
    await page.getByTestId('trigger-mock').click();

    await expect(page.getByTestId('trace-output')).toHaveText(/trace-cli-mock/);
    await expect(page.getByText('已启用 Mock 模式', { exact: true })).toBeVisible();
  });
});

test('invokes local capability endpoint directly via Capability Lab', async ({ page }) => {
  await seedAuthStorage(page);
  await page.route(localEndpoint, async (route) => {
    const request = route.request();
    expect(request.method()).toBe('POST');
    const url = new URL(request.url());
    expect(url.searchParams.get('tag')).toBe('demo');
    expect(url.searchParams.get('page')).toBe('1');
    const headers = request.headers();
    expect(headers['x-test-header']).toBe('local-debug');
    const body = request.postDataJSON() as Record<string, any>;
    expect(body?.message).toBe('hello from local debug');
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      headers: {
        'x-trace-id': 'trace-local-debug'
      },
      body: JSON.stringify({
        traceId: 'trace-local-debug',
        status: 'completed',
        data: {
          echoed: true,
          input: body
        }
      })
    });
  });

  await gotoWithFallback(page, '/tests/capability', page.getByTestId('trigger-local-debug'));
  await page.getByTestId('trigger-local-debug').click();

  await expect(page.getByTestId('local-trace')).toHaveText(/trace-local-debug/);
  await expect(page.getByTestId('local-status')).toHaveText(/completed/);
  await expect(page.getByTestId('local-error')).toHaveText(/none/);
  await expect(page.getByTestId('local-response-preview')).toContainText('echoed');
});
