import { defineNuxtConfig } from 'nuxt/config'
import { definePowerXAdminConfig } from '@powerx-plugin/framework-admin'

const pluginId = 'com.powerx.plugin.base'
const pluginAdminBase = `/_p/${pluginId}/admin/`
const pluginApiBase = `/_p/${pluginId}/api/v1`
const localApiBase = process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8078/api/v1'

const INSIDE_POWERX = process.env.POWERX_PROXY === '1'
const rawBridgeDebug = process.env.NUXT_PUBLIC_BRIDGE_DEBUG ?? process.env.BRIDGE_DEBUG
const BRIDGE_DEBUG = rawBridgeDebug !== undefined
  ? /^(1|true)$/i.test(String(rawBridgeDebug))
  : !INSIDE_POWERX

const imgSources = ["'self'", "data:", "https://avatars.githubusercontent.com"]
const connectSources = ["'self'"]

if (!INSIDE_POWERX) {
  const apiCandidates = new Set<string>()
  const registerCandidate = (value?: string) => {
    if (!value) return
    apiCandidates.add(value)
  }

  registerCandidate(localApiBase)
  try {
    const apiOrigin = new URL(localApiBase).origin
    registerCandidate(apiOrigin)
    if (apiOrigin.includes("localhost")) {
      registerCandidate(apiOrigin.replace("localhost", "127.0.0.1"))
    }
  } catch {
    // Swallow URL parse errors; fall back to raw string
  }

  registerCandidate("ws:")
  registerCandidate("wss:")

  connectSources.push(...apiCandidates)
}

connectSources.push("https://api.iconify.design")

const powerx = definePowerXAdminConfig({
  pluginId,
  starterPages: true
})

export default defineNuxtConfig({
  extends: powerx.extends,
  appConfig: powerx.appConfig,
  compatibilityDate: '2025-11-02',
  ssr: false,
  srcDir: 'app',
  devtools: {
    enabled: !INSIDE_POWERX
  },
  app: {
    baseURL: INSIDE_POWERX ? pluginAdminBase : '/',
    buildAssetsDir: '/assets/',
    head: {
      meta: [
        { name: 'referrer', content: 'no-referrer' },
        { httpEquiv: 'X-Content-Type-Options', content: 'nosniff' },
        { name: 'permissions-policy', content: 'camera=(), microphone=(), geolocation=()' }
      ]
    }
  },
  css: [
    '~/assets/css/main.css',
    '@/assets/scss/main.scss'
  ],
  postcss: {
    plugins: {
      '@tailwindcss/postcss': {},
      autoprefixer: {}
    }
  },
  modules: [
    '@nuxt/ui',
    '@nuxt/icon',
    '@pinia/nuxt',
    '@nuxtjs/color-mode',
    '@nuxtjs/i18n'
  ],
  colorMode: {
    preference: 'system',
    fallback: 'light',
    storageKey: 'powerx-color-mode'
  },
  i18n: {
    defaultLocale: 'zh',
    strategy: 'no_prefix',
    locales: [
      { code: 'zh', name: '简体中文', file: 'zh.json' },
      { code: 'en', name: 'English', file: 'en.json' }
    ],
    langDir: '../i18n/locales',
    detectBrowserLanguage: INSIDE_POWERX ? false : {
      useCookie: true,
      cookieKey: 'px_lang',
      alwaysRedirect: true,
      fallbackLocale: 'zh',
      redirectOn: 'root'
    }
  },
  runtimeConfig: {
    public: {
      apiBaseUrl: INSIDE_POWERX ? pluginApiBase : localApiBase,
      insidePowerX: INSIDE_POWERX,
      pluginAdminBase,
      bridgeDebug: BRIDGE_DEBUG
    }
  },
  nitro: {
    preset: 'node-server',
    serveStatic: true,
    experimental: {
      websocket: true
    },
    routeRules: {
      '/**': {
        headers: {
          'X-Frame-Options': 'SAMEORIGIN',
          'Content-Security-Policy': [
            "default-src 'self'",
            `img-src ${imgSources.join(' ')}`,
            "style-src 'self' 'unsafe-inline'",
            INSIDE_POWERX
              ? "script-src 'self' 'unsafe-inline'"
              : "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
            `connect-src ${connectSources.join(' ')}`,
            "font-src 'self' data:",
            "frame-ancestors 'self'"
          ].join('; ') + ';',
          'Strict-Transport-Security': 'max-age=31536000; includeSubDomains',
          'Referrer-Policy': 'no-referrer'
        }
      }
    },
    output: {
      dir: '.output',
      publicDir: '.output/public'
    }
  },
  vite: {
    server: {
      hmr: {
        protocol: 'ws',
        host: 'localhost',
        port: 24731
      },
      proxy: INSIDE_POWERX
        ? {}
        : {
            '/api': {
              target: 'http://localhost:8086',
              changeOrigin: true,
              ws: true
            },
            '/ws': {
              target: 'ws://127.0.0.1:4000',
              changeOrigin: true,
              ws: true
            }
          }
    }
  },
  devServer: {
    host: '0.0.0.0',
    port: 3031
  },
  ui: {
    fonts: false
  }
})
