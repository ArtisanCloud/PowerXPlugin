export interface PowerXAdminConfigOptions {
  pluginId: string
  starterPages?: boolean
}

export function definePowerXAdminConfig(options: PowerXAdminConfigOptions) {
  return {
    extends: ['@powerx-plugin/framework-admin/layer'],
    appConfig: {
      powerx: {
        pluginId: options.pluginId,
        starterPages: options.starterPages ?? true
      }
    }
  }
}
