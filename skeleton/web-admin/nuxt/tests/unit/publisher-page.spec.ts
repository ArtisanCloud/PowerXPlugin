import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'
import { compileTemplate, parse } from '@vue/compiler-sfc'
import { beforeEach, expect, it, vi } from 'vitest'
import { publisherAPIPath, publisherPagePath } from '../../../../../framework/frontend/nuxt/framework-admin/publisher/types'

const hooks = { pages: vi.fn() }
const frameworkRequire = createRequire(new URL('../../../../../framework/frontend/nuxt/framework-admin/publisher/module.ts', import.meta.url))
vi.doMock(join(dirname(frameworkRequire.resolve('nuxt/package.json')), 'kit.js'), () => ({
  createResolver: () => ({ resolve: (path: string) => path }),
  defineNuxtModule: (definition: unknown) => definition,
  extendPages: (callback: unknown) => hooks.pages(callback)
}))
const { default: publisherModule } = await import('../../../../../framework/frontend/nuxt/framework-admin/publisher/module')

beforeEach(() => hooks.pages.mockClear())

it('registers the developer page in the admin shell and keeps its private API separate', () => {
  const nuxt = { hook: vi.fn(), options: { dev: true, rootDir: process.cwd(), appConfig: {}, nitro: { devHandlers: [] } } }
  const module = publisherModule as unknown as { setup: (options: unknown, nuxt: unknown) => void }
  module.setup({}, nuxt)
  const pages: Array<{ path: string; meta: Record<string, unknown> }> = []
  hooks.pages.mock.calls[0]![0](pages)
  expect(pages).toHaveLength(1)
  expect(pages[0]).toMatchObject({ path: '/powerx/publisher', meta: { layout: 'default', public: true, publisherDevelopment: true } })
  expect(pages[0]!.path).toBe(publisherPagePath)
  expect(nuxt.options.nitro.devHandlers).toMatchObject([{ route: publisherAPIPath }])
  expect(publisherAPIPath).toBe('/__powerx/publisher-api')
})

it('registers neither the publisher page nor its execution proxy in production', () => {
  const nuxt = { options: { dev: false, appConfig: {}, nitro: { devHandlers: [] } } }
  const module = publisherModule as unknown as { setup: (options: unknown, nuxt: unknown) => void }
  module.setup({}, nuxt)
  expect(hooks.pages).not.toHaveBeenCalled()
  expect(nuxt.options.nitro.devHandlers).toEqual([])
  expect(nuxt.options.appConfig).toEqual({})
})

it.each(['PublisherPage','MarketPanel','ReviewPanel','RequestHistory'])('compiles the development-only publisher template, which production excludes (%s)', (component) => {
  const filename = fileURLToPath(new URL(`../../../../../framework/frontend/nuxt/framework-admin/publisher/runtime/${component}.vue`, import.meta.url))
  const { descriptor, errors } = parse(readFileSync(filename, 'utf8'), { filename })
  expect(errors).toEqual([])
  expect(descriptor.template).not.toBeNull()
  const compiled = compileTemplate({ filename, id: 'publisher-page', source: descriptor.template!.content })
  expect(compiled.errors).toEqual([])
})
