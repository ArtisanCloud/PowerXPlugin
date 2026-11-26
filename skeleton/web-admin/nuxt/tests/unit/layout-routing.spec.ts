import { afterEach, expect, it, vi } from 'vitest'

afterEach(() => vi.unstubAllGlobals())

it.each([
  { insidePowerX: false, path: '/powerx/publisher', meta: { publisherDevelopment: true }, layout: 'default' },
  { insidePowerX: true, path: '/powerx/publisher', meta: { publisherDevelopment: true }, layout: 'default' },
  { insidePowerX: false, path: '/_p/com.powerx.plugins.base/admin/settings', meta: {}, layout: 'default' },
  { insidePowerX: '0', path: '/_p/com.powerx.plugins.base/admin/settings', meta: {}, layout: 'default' },
  { insidePowerX: true, path: '/_p/com.powerx.plugins.base/admin/settings', meta: {}, layout: 'embedded' },
  { insidePowerX: true, path: '/intro', meta: {}, layout: 'default' }
])('layout_route_$insidePowerX_$path_$layout', async ({ insidePowerX, path, meta, layout }) => {
  const setLayout = vi.fn()
  vi.stubGlobal('defineNuxtRouteMiddleware', (guard: unknown) => guard)
  vi.stubGlobal('useRuntimeConfig', () => ({ public: { insidePowerX } }))
  vi.stubGlobal('setPageLayout', setLayout)
  const { default: guard } = await import('../../app/middleware/layout.global')
  ;(guard as unknown as (to: { path: string; meta: object }) => void)({ path, meta })
  expect(setLayout).toHaveBeenCalledExactlyOnceWith(layout)
})
