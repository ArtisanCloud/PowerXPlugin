import type { RouterConfig } from '@nuxt/schema'

export default <RouterConfig>{
  scrollBehavior(to, _from, savedPosition) {
    if (savedPosition) {
      return savedPosition
    }

    if (to.hash) {
      const rawHash = String(to.hash || '').trim()
      const normalized = rawHash.replace(/^#/, '')
      const looksLikeOAuthFragment =
        normalized.startsWith('access_token=') ||
        normalized.startsWith('fed_access_token=') ||
        normalized.includes('access_token=')

      if (looksLikeOAuthFragment) {
        return { left: 0, top: 0 }
      }

      if (typeof document !== 'undefined') {
        const escaped =
          typeof CSS !== 'undefined' && typeof CSS.escape === 'function'
            ? `#${CSS.escape(normalized)}`
            : rawHash

        try {
          if (document.querySelector(escaped)) {
            return { el: escaped, behavior: 'smooth', top: 0 }
          }
        } catch {
          return { left: 0, top: 0 }
        }
      }
    }

    return { left: 0, top: 0 }
  },
}
