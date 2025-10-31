import { defineNuxtConfig } from 'nuxt/config'
import { definePowerXAdminConfig } from '@powerx-plugin/framework-admin'

const powerx = definePowerXAdminConfig({
  pluginId: 'com.powerx.sample',
  starterPages: true
})

export default defineNuxtConfig({
  extends: powerx.extends,
  appConfig: powerx.appConfig,
  devServer: {
    host: '0.0.0.0',
    port: 3031
  },
  vite: {
    server: {
      hmr: {
        protocol: 'ws',
        host: 'localhost',
        port: 24731
      }
    }
  }
})
