import { defineNuxtConfig } from 'nuxt/config'

export default defineNuxtConfig({
  components: {
    dirs: [
      { path: 'app/components', pathPrefix: false },
      { path: 'app/components/templates', pathPrefix: false }
    ]
  },
  imports: {
    dirs: ['app/composables', 'app/composables/api']
  },
  hooks: {
    'pages:extend'(pages, nuxt) {
      const starterEnabled = (nuxt.options.appConfig?.powerx?.starterPages ?? true) as boolean
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
