import { defineNuxtModule } from 'nuxt/kit'

export interface ModuleOptions {
  pluginId: string
}

export default defineNuxtModule<ModuleOptions>({
  meta: {
    name: '@artisan-cloud/plugin-framework-admin'
  },
  setup() {
    // 占位：后续可在此扩展 nuxt runtime 功能
  }
})
