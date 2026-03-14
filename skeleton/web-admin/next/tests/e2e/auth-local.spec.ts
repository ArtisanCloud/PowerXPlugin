import { expect, test } from '@playwright/test'
import { gotoLogin, loginWithPassword } from './_utils'

const loginEmail = process.env.PLAYWRIGHT_LOCAL_EMAIL || 'admin@local.test'
const loginPassword = process.env.PLAYWRIGHT_LOCAL_PASSWORD || 'S3cret!!'
const iamMode = process.env.PLAYWRIGHT_LOCAL_IAM

if (iamMode !== '1' && iamMode !== '0') {
  test.skip(true, 'Set PLAYWRIGHT_LOCAL_IAM=1(local) or PLAYWRIGHT_LOCAL_IAM=0(delegated).')
}

const localSuite = iamMode !== '0' ? test.describe : test.describe.skip
const delegatedSuite = iamMode === '0' ? test.describe : test.describe.skip

localSuite('Local IAM Login Flow (Next)', () => {
  test('logs in, persists tokens, and enters intro page', async ({ page }) => {
    await page.route('**/api/v1/admin/user/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 0,
          message: 'ok',
          data: {
            token_type: 'Bearer',
            access_token: 'next-e2e-access',
            refresh_token: 'next-e2e-refresh',
            expires_at: Math.floor(Date.now() / 1000) + 3600,
          },
        }),
      })
    })

    await gotoLogin(page)
    await loginWithPassword(page, loginEmail, loginPassword)

    await expect
      .poll(() => page.evaluate(() => window.localStorage.getItem('access_token')))
      .toBe('next-e2e-access')
    await expect(page).toHaveURL(/\/intro/)
    await expect(page.getByTestId('intro-title')).toBeVisible()
  })
})

delegatedSuite('Delegated guard (Next)', () => {
  test('shows warning and disables submit', async ({ page }) => {
    await gotoLogin(page)
    await expect(page.getByRole('alert')).toBeVisible()
    await expect(page.getByTestId('login-submit')).toBeDisabled()
  })
})
