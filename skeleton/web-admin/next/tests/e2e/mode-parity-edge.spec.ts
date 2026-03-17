import { expect, test } from '@playwright/test'
import { assertHostMode, assertStandaloneMode, seedAuthStorage } from './_utils'

const pluginId = process.env.POWERX_PLUGIN_ID || 'com.powerx.plugins.base'

test.describe('Mode parity edge cases (Next)', () => {
  test('standalone and host paths are both reachable', async ({ page }) => {
    await seedAuthStorage(page)

    await page.goto('/integration')
    await assertStandaloneMode(page)

    await page.goto(`/_p/${pluginId}/admin/integration`)
    await assertHostMode(page, pluginId)
    await expect(page.getByTestId('host-proxy-page')).toBeVisible()
  })

  test('unauthenticated access redirects with encoded path', async ({ page }) => {
    await page.goto('/admin/iam/overview')
    await expect(page).toHaveURL(/\/users\/login\?redirect=%2Fadmin%2Fiam%2Foverview/)
  })
})
