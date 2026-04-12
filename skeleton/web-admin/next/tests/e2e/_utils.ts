import { expect, type Page } from '@playwright/test'

export async function gotoLogin(page: Page): Promise<void> {
  await page.goto('/users/login')
  await expect(page).toHaveURL(/\/users\/login/)
}

export async function loginWithPassword(page: Page, username: string, password: string): Promise<void> {
  await page.getByTestId('login-username').fill(username)
  await page.getByTestId('login-password').fill(password)
  await page.getByTestId('login-submit').click()
}

export async function seedAuthStorage(page: Page): Promise<void> {
  const expiresAt = Math.floor(Date.now() / 1000) + 3600
  const baseURL = process.env.PLAYWRIGHT_BASE_URL || 'http://127.0.0.1:3231'
  const cookieURL = new URL(baseURL).origin
  await page.context().addCookies([
    {
      name: 'access_token',
      value: 'e2e-access-token',
      url: cookieURL,
    },
  ])

  await page.addInitScript(({ expiresAt }) => {
    window.localStorage.setItem('access_token', 'e2e-access-token')
    window.localStorage.setItem('refresh_token', 'e2e-refresh-token')
    window.localStorage.setItem('expires_at', String(expiresAt))
    document.cookie = 'access_token=e2e-access-token; Path=/; SameSite=Lax'
  }, { expiresAt })
}

export async function assertStandaloneMode(page: Page): Promise<void> {
  await expect(page).not.toHaveURL(/\/_p\//)
}

export async function assertHostMode(page: Page, pluginId: string): Promise<void> {
  await expect(page).toHaveURL(new RegExp(`^.*\/_p\/${pluginId}\/`))
}
