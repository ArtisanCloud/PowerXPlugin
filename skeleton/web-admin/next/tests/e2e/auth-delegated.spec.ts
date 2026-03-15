import { expect, test } from '@playwright/test'
import { gotoLogin } from './_utils'

const delegated = process.env.NEXT_PUBLIC_DELEGATED_IAM === '1'

if (!delegated) {
  test.skip(true, 'Set NEXT_PUBLIC_DELEGATED_IAM=1 to run delegated auth assertions.')
}

test.describe('Delegated auth parity (Next)', () => {
  test('login form is disabled and warning is visible', async ({ page }) => {
    await gotoLogin(page)
    await expect(page.getByRole('alert')).toContainText('委托鉴权模式')
    await expect(page.getByTestId('login-submit')).toBeDisabled()
  })

  test('protected page redirects to login with redirect parameter', async ({ page }) => {
    await page.goto('/templates/crud')
    await expect(page).toHaveURL(/\/users\/login\?redirect=/)
  })
})
