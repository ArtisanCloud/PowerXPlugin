import { isPowerXEmbedded } from "@artisan-cloud/plugin-framework-client";
import { isPluginAdminPath } from "~/utils/powerx-bridge";

export default defineNuxtRouteMiddleware((to) => {
  const config = useRuntimeConfig();
  if (to.meta.publisherDevelopment === true) {
    setPageLayout("default");
    return;
  }
  const inside = isPowerXEmbedded(config.public?.insidePowerX);
  // Host 内，并且真实路径位于插件管理前缀下时才切换 embedded 布局
  const useEmbedded = inside && isPluginAdminPath(to.path);
  setPageLayout(useEmbedded ? "embedded" : "default");
});
