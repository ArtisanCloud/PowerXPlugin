import { test, expect } from '@playwright/test';

const capabilityEndpoint = '**/integration/capabilities/invoke';

test.describe('Capability invocation playground', () => {
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

    await page.goto('/tests/capability');
    await page.getByTestId('trigger-success').click();

    await expect(page.getByTestId('trace-output')).toHaveText(/trace-cli-success/);
    await expect(page.getByText('能力调用成功')).toBeVisible();
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

    await page.goto('/tests/capability');
    await page.getByTestId('trigger-fail').click();

    await expect(page.getByTestId('status-indicator')).toHaveText(/error/);
    await expect(page.getByTestId('error-indicator')).toHaveText('error');
    await expect(page.getByText('能力调用失败')).toBeVisible();
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

    await page.goto('/tests/capability');
    await page.getByTestId('trigger-mock').click();

    await expect(page.getByTestId('trace-output')).toHaveText(/trace-cli-mock/);
    await expect(page.getByText('已启用 Mock 模式')).toBeVisible();
  });
});
