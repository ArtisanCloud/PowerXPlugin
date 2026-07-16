import { useRuntimeConfig } from "#imports";

export default defineNuxtPlugin(() => {
  const auth = useAuth();
  const runtimeConfig = useRuntimeConfig();
  const delegatedMode =
    runtimeConfig.public?.delegatedMode === true ||
    runtimeConfig.public?.delegatedMode === "true";

  auth.setProviderModeFlags?.(delegatedMode);

  if (process.client) {
    auth.initAuth();
  }
});
