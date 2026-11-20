import { onScopeDispose, watch } from 'vue';
import { createRegressionClient, createRealtimeClient, isPowerXEmbedded } from '@artisan-cloud/plugin-framework-client';
import { useApiClient } from './_client';
import { resolveApiBase, getAuthToken, getTenantUuid } from './_base';
import { useAuth } from '~/composables/useAuth';
import { useUserStore } from '~/stores/user';
export function useRegressionApi(){
 const config=useRuntimeConfig(),api=useApiClient(),auth=useAuth(),user=useUserStore();
 const pluginId=String(config.public.powerxPluginId);
 const realtime=createRealtimeClient({pluginId,apiBaseURL:resolveApiBase(),insidePowerX:isPowerXEmbedded(config.public.insidePowerX),token:getAuthToken(),tenantUuid:getTenantUuid()});
 watch(()=>[auth.token.value,user.context?.current_tenant_uuid,user.userUuid],()=>realtime.setContext({token:getAuthToken(),tenantUuid:getTenantUuid()}));
 onScopeDispose(()=>realtime.close());
 return createRegressionClient({pluginId,request:api.client,realtime});
}
