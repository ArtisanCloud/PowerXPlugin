import { createResolver, defineNuxtModule, extendPages } from 'nuxt/kit'
import { publisherAPIPath, publisherPagePath } from './types'
import { createPublisherDevHandler } from './runtime/handler'
import { createPublisherManager } from './manager'
import type { NitroConfig } from 'nitropack'

export default defineNuxtModule<{ workspace?: string; executable?: string }>({
  meta: { name: 'powerx-publisher-development' },
  setup(_options, nuxt) {
    if (!nuxt.options.dev) return
    const resolver = createResolver(import.meta.url)
    const manager = createPublisherManager({ rootDir: nuxt.options.rootDir, ..._options })
    nuxt.hook('close', () => manager.close())
    nuxt.options.appConfig.powerx = {
      ...(nuxt.options.appConfig.powerx as Record<string, unknown>),
      publisherDevelopment: true
    }
    extendPages((pages) => {
      pages.push({
        name: 'powerx-publisher-development',
        path: publisherPagePath,
        file: resolver.resolve('./runtime/PublisherPage.vue'),
        meta: { public: true, layout: 'default', publisherDevelopment: true }
      })
    })
    // Nitro worker requests do not retain the actual peer socket. Use the
    // outer development handler so LAN peers cannot forge loopback headers.
    const options = nuxt.options as typeof nuxt.options & { nitro?: NitroConfig }
    const nitro = options.nitro
    options.nitro = {
      ...nitro,
      devHandlers: [
        ...(nitro?.devHandlers || []).filter(handler => handler?.route !== publisherAPIPath),
        { route: publisherAPIPath, handler: createPublisherDevHandler(manager) }
      ]
    }
  }
})
