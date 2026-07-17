import { expect, test } from '@playwright/test'
import { seedAuthStorage } from './_utils'

const localIAMFlag = process.env.PLAYWRIGHT_LOCAL_IAM
if (localIAMFlag !== '1') {
  test.skip(true, 'Set PLAYWRIGHT_LOCAL_IAM=1 for local IAM flow tests.')
}

test.describe('IAM local flow parity (Next)', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuthStorage(page)

    await page.route('**/api/v1/admin/iam/overview', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, message: 'ok', data: { members: 2, roles: 3 } }),
      })
    })

    await page.route('**/api/v1/admin/iam/members', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          message: 'ok',
          data: { list: [{ id: 'm1', username: 'alice' }, { id: 'm2', username: 'bob' }], total: 2 },
        }),
      })
    })

    await page.route('**/api/v1/admin/iam/roles', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          message: 'ok',
          data: { list: [{ id: 'r1', code: 'admin', name: '管理员' }], total: 1 },
        }),
      })
    })

    await page.route('**/api/v1/admin/iam/settings', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, message: 'ok', data: { delegated: false } }),
      })
    })
  })

  test('overview/members/roles/settings are reachable and rendered', async ({ page }) => {
    await page.goto('/admin/iam/overview')
    await expect(page.getByTestId('iam-overview-page')).toBeVisible()

    await page.goto('/admin/iam/members')
    await expect(page.getByTestId('iam-members-list')).toContainText('alice')

    await page.goto('/admin/iam/roles')
    await expect(page.getByTestId('iam-roles-list')).toContainText('管理员')

    await page.goto('/admin/iam/settings')
    await expect(page.getByTestId('iam-settings-page')).toBeVisible()
  })
})
