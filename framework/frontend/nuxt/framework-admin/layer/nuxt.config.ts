import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'
import type { NuxtConfig } from 'nuxt/schema'

const layerDir = dirname(fileURLToPath(import.meta.url))
const appDir = resolve(layerDir, 'app')

const config: NuxtConfig = {
  runtimeConfig: {
    public: {
      powerx: {
        apiBase: process.env.NUXT_PUBLIC_POWERX_API_BASE ?? '',
        capabilityEndpoint: process.env.NUXT_PUBLIC_POWERX_CAPABILITY_ENDPOINT ?? '/integration/capabilities/invoke'
      }
    }
  },
  imports: {
    dirs: [
      resolve(appDir, 'composables'),
      resolve(appDir, 'composables/api')
    ]
  },
  hooks: {
    'components:dirs'(dirs) {
      dirs.push({
        path: resolve(appDir, 'components'),
        pathPrefix: false
      })
    },
    'pages:extend'(pages, nuxt) {
      const starterEnabled = (nuxt?.options?.appConfig?.powerx?.starterPages ?? true) as boolean
      if (starterEnabled) {
        return
      }
      for (const page of [...pages]) {
        if (typeof page.file === 'string' &&
          (page.file.includes('app/pages/intro.vue') || page.file.includes('app/pages/templates/'))) {
          pages.splice(pages.indexOf(page), 1)
        }
      }
    }
  }
}

export default config

declare module 'nuxt/schema' {
  interface PublicRuntimeConfig {
    powerx: {
      apiBase: string
      capabilityEndpoint: string
    }
  }
}
