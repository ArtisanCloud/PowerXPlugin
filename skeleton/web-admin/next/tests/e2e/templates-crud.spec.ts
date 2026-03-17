import { expect, test } from '@playwright/test'
import { seedAuthStorage } from './_utils'

type Template = {
  id: number
  name: string
  description: string
  content: string
}

test.describe('Templates CRUD parity (Next)', () => {
  test('list/create/update/delete full flow', async ({ page }) => {
    const templates: Template[] = [
      { id: 1, name: '欢迎模板', description: '默认欢迎语', content: 'hello world' },
    ]

    await page.route('**/api/v1/templates**', async (route) => {
      const request = route.request()
      const url = new URL(request.url())
      const method = request.method()

      if (method === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 0,
            message: 'ok',
            data: {
              list: templates,
              total: templates.length,
            },
          }),
        })
        return
      }

      if (method === 'POST' && url.pathname.endsWith('/templates')) {
        const payload = request.postDataJSON() as Partial<Template>
        const created: Template = {
          id: templates.length + 10,
          name: payload.name || 'unnamed',
          description: payload.description || '',
          content: payload.content || '',
        }
        templates.unshift(created)
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: created }),
        })
        return
      }

      if (method === 'PUT') {
        const payload = request.postDataJSON() as Partial<Template>
        const id = Number(url.pathname.split('/').pop())
        const index = templates.findIndex((item) => item.id === id)
        if (index >= 0) {
          templates[index] = { ...templates[index], ...payload }
        }
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: templates[index] }),
        })
        return
      }

      if (method === 'DELETE') {
        const id = Number(url.pathname.split('/').pop())
        const index = templates.findIndex((item) => item.id === id)
        if (index >= 0) {
          templates.splice(index, 1)
        }
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 0, message: 'ok', data: { ok: true } }),
        })
        return
      }

      await route.continue()
    })

    page.on('dialog', async (dialog) => {
      await dialog.accept()
    })

    await seedAuthStorage(page)
    await page.goto('/templates/crud')

    await expect(page.getByTestId('templates-crud-title')).toBeVisible()
    await expect(page.getByTestId('template-row-1')).toBeVisible()

    await page.getByTestId('templates-create-btn').click()
    await expect(page.getByTestId('template-form-modal')).toBeVisible()
    await page.getByTestId('template-form-name').fill('营销模板')
    await page.getByTestId('template-form-description').fill('用于营销活动')
    await page.getByTestId('template-form-content').fill('campaign-content')
    await page.getByTestId('template-form-submit').click()

    await expect(page.getByText('营销模板')).toBeVisible()

    const createdId = await page
      .locator('tr[data-testid^="template-row-"]')
      .first()
      .getAttribute('data-testid')

    expect(createdId).toBeTruthy()
    const id = Number((createdId || '').replace('template-row-', ''))

    await page.getByTestId(`template-edit-${id}`).click()
    await page.getByTestId('template-form-name').fill('营销模板-更新')
    await page.getByTestId('template-form-submit').click()
    await expect(page.getByText('营销模板-更新')).toBeVisible()

    await page.getByTestId(`template-delete-${id}`).click()
    await expect(page.getByText('营销模板-更新')).not.toBeVisible()
  })
})
