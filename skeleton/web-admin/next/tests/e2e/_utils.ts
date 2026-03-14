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

export async function assertStandaloneMode(page: Page): Promise<void> {
  await expect(page).not.toHaveURL(/\/_p\//)
}

export async function assertHostMode(page: Page, pluginId: string): Promise<void> {
  await expect(page).toHaveURL(new RegExp(`^.*\/_p\/${pluginId}\/`))
}
