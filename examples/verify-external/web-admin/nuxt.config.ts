import { defineNuxtConfig } from 'nuxt/config'
import { definePowerXAdminConfig } from '@artisan-cloud/plugin-framework-admin'

const powerx = definePowerXAdminConfig({
  pluginId: 'com.powerx.verify',
  starterPages: false
})

export default defineNuxtConfig({
  extends: powerx.extends,
  appConfig: powerx.appConfig
})
