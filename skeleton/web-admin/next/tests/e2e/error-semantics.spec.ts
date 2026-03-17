import { expect, test } from '@playwright/test'
import { seedAuthStorage } from './_utils'

test.describe('Error semantics matrix (Next)', () => {
  test('IAM endpoint returns envelope code/message and UI keeps semantics', async ({ page }) => {
    await seedAuthStorage(page)

    await page.route('**/api/v1/admin/iam/overview', async (route) => {
      await route.fulfill({
        status: 403,
        contentType: 'application/json',
        body: JSON.stringify({ code: 40301, message: 'permission denied', data: null }),
      })
    })

    await page.goto('/admin/iam/overview')
    await expect(page.getByRole('alert')).toContainText('permission denied')
  })

  test('capability invoke error keeps failed status and error indicator', async ({ page }) => {
    await seedAuthStorage(page)

    await page.route('**/api/v1/admin/capabilities/invoke', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ code: 50001, message: 'invoke failed', data: null }),
      })
    })

    await page.goto('/tests/capability')
    await page.getByTestId('trigger-fail').click()
    await expect(page.getByTestId('status-indicator')).toHaveText('failed')
    await expect(page.getByTestId('error-indicator')).toHaveText('error')
  })
})
