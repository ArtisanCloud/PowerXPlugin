import { expect, test } from '@playwright/test'
import { seedAuthStorage } from './_utils'

test.describe('Capability invocation parity (Next)', () => {
  test('success and failure invocation semantics', async ({ page }) => {
    await seedAuthStorage(page)

    await page.route('**/api/v1/admin/capabilities/invoke', async (route) => {
      const payload = route.request().postDataJSON() as { kind?: string }
      if (payload.kind === 'fail') {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ code: 50001, message: 'invoke failed', data: null }),
        })
        return
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, message: 'ok', data: { trace_id: 'trace-001', status: 'ok' } }),
      })
    })

    await page.goto('/tests/capability')
    await expect(page.getByTestId('capability-playground')).toBeVisible()

    await page.getByTestId('trigger-success').click()
    await expect(page.getByTestId('status-indicator')).toHaveText('succeeded')

    await page.getByTestId('trigger-fail').click()
    await expect(page.getByTestId('status-indicator')).toHaveText('failed')
    await expect(page.getByTestId('error-indicator')).toHaveText('error')
  })
})
