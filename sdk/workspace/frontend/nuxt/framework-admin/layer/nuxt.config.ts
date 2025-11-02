import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'
import { defineNuxtConfig } from 'nuxt/config'

const layerDir = dirname(fileURLToPath(import.meta.url))
const appDir = resolve(layerDir, 'app')

export default defineNuxtConfig({
  components: {
    dirs: [
      { path: resolve(appDir, 'components'), pathPrefix: false }
    ]
  },
  imports: {
    dirs: [
      resolve(appDir, 'composables'),
      resolve(appDir, 'composables/api')
    ]
  },
  hooks: {
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
})
