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
