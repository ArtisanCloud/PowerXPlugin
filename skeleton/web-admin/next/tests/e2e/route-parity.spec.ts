import { expect, test } from '@playwright/test'
import { seedAuthStorage } from './_utils'

const routes = [
  '/admin/iam/overview',
  '/admin/iam/members',
  '/admin/iam/roles',
  '/admin/iam/settings',
  '/capabilities/lifecycle',
  '/capabilities/register',
  '/powerx/capability-lab',
  '/tests/capability',
  '/integration',
  '/operations',
  '/security',
]

test.describe('Route parity and visibility (Next)', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuthStorage(page)

    await page.route('**/api/v1/admin/iam/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: { list: [], total: 0 } }) })
    })

    await page.route('**/api/v1/admin/capabilities/**', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: {} }) })
    })

    await page.route('**/api/v1/admin/integration/settings', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: {} }) })
    })

    await page.route('**/api/v1/admin/operations/overview', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: {} }) })
    })

    await page.route('**/api/v1/admin/security/overview', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, message: 'ok', data: {} }) })
    })
  })

  for (const route of routes) {
    test(`route reachable: ${route}`, async ({ page }) => {
      await page.goto(route)
      await expect(page).toHaveURL(new RegExp(`${route.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`))
    })
  }
})
