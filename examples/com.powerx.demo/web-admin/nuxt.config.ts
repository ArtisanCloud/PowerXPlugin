import { defineNuxtConfig } from 'nuxt/config'
import { definePowerXAdminConfig } from '@powerx-plugin/framework-admin'

const powerx = definePowerXAdminConfig({
  pluginId: 'com.powerx.demo',
  starterPages: true
})

export default defineNuxtConfig({
  extends: powerx.extends,
  appConfig: powerx.appConfig
})
