export interface PowerXAdminConfigOptions {
  pluginId: string
  starterPages?: boolean
}

export function definePowerXAdminConfig(options: PowerXAdminConfigOptions) {
  const pluginId = options.pluginId
  const inferredApiBase =
    process.env.NUXT_PUBLIC_POWERX_API_BASE?.trim() || `/_p/${pluginId}/api/v1`
  const inferredCapabilityEndpoint =
    process.env.NUXT_PUBLIC_POWERX_CAPABILITY_ENDPOINT?.trim() || '/integration/capabilities/invoke'

  return {
    extends: ['@artisan-cloud/plugin-framework-admin/layer'],
    appConfig: {
      powerx: {
        pluginId,
        starterPages: options.starterPages ?? true
      }
    },
    runtimeConfig: {
      public: {
        powerx: {
          apiBase: inferredApiBase,
          capabilityEndpoint: inferredCapabilityEndpoint
        }
      }
    }
  }
}
