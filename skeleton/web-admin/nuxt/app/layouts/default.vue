<template>
  <div v-if="disableShell" class="min-h-screen">
    <div v-if="showDelegatedBanner" class="px-4 pt-4">
      <DelegatedAuthBanner
        :message="delegatedBannerMessage"
        @retry="handleDelegatedRetry"
        @dismiss="handleDelegatedDismiss"
      />
    </div>
    <slot v-if="!insidePowerX || auth.isAuthenticated.value" />
    <p v-else-if="!showDelegatedBanner" class="p-4 text-gray-600 dark:text-slate-200">{{ t('hostSession.loading') }}</p>
  </div>
  <div v-else class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- 顶部导航栏 - 根据环境变量控制显示 -->
    <UContainer v-if="showNavigation" class="max-w-none">
      <AppNavbar />
    </UContainer>

    <div :class="shellClass">
      <!-- 左侧边栏 - 根据环境变量控制显示 -->
      <AppSidebar v-if="showNavigation" />

      <!-- 主内容区 -->
      <main :class="mainContentClass">
        <DelegatedAuthBanner
          v-if="showDelegatedBanner"
          :message="delegatedBannerMessage"
          @retry="handleDelegatedRetry"
          @dismiss="handleDelegatedDismiss"
        />
        <slot v-if="!insidePowerX || auth.isAuthenticated.value" />
    <p v-else-if="!showDelegatedBanner" class="p-4 text-gray-600 dark:text-slate-200">{{ t('hostSession.loading') }}</p>
      </main>
    </div>
  </div>
</template>

<script setup>

import { isPowerXEmbedded } from '@artisan-cloud/plugin-framework-client';
import { useTheme } from "~/composables/useTheme";
import { PLUGIN_ID, isPluginAdminPath } from "~/utils/powerx-bridge";
import { useAuth } from "~/composables/useAuth";
import DelegatedAuthBanner from "~/components/DelegatedAuthBanner.vue";
import { createFrameworkLogger } from "@artisan-cloud/plugin-framework-client";

// 获取运行时配置
const { t } = useI18n();
const runtimeConfig = useRuntimeConfig();
const route = useRoute();
const theme = useTheme();
const auth = useAuth();
const logger = createFrameworkLogger("layout.default");

const disableShell = computed(() => {
  if (route.meta?.fullBleed === true) {
    return true;
  }
  return route.meta?.layout === false;
});

// 是否处于 PowerX 宿主的插件嵌入路径下
const insidePowerX = computed(() => {
  const value = runtimeConfig.public.insidePowerX;
  return isPowerXEmbedded(value);
});

const isEmbeddedInPowerX = computed(() => {
  // Only mount host bridge when runtime explicitly declares insidePowerX.
  // Path-based `_p/...` detection is not enough in standalone/local mode.
  return insidePowerX.value;
});

// 控制导航显示的环境变量
const showNavigation = computed(() => {
  if (disableShell.value || isEmbeddedInPowerX.value) {
    return false;
  }

  // 优先检查环境变量 NUXT_PUBLIC_SHOW_NAVIGATION
  const envShowNav = runtimeConfig.public.showNavigation;

  // 如果没有设置环境变量，开发环境默认显示，生产环境默认隐藏
  if (envShowNav !== undefined) {
    return envShowNav === "true" || envShowNav === true;
  }

  // 默认：非嵌入模式显示导航。若需隐藏，显式配置 NUXT_PUBLIC_SHOW_NAVIGATION=false。
  return true;
});

// 主内容区样式
const mainContentClass = computed(() => {
  if (disableShell.value) {
    return "w-full";
  }
  return showNavigation.value ? "min-h-0 flex-1 overflow-y-auto p-6" : "min-h-0 w-full overflow-y-auto p-6";
});

const shellClass = computed(() => {
  const base = "flex min-h-0 overflow-hidden";
  return showNavigation.value ? `${base} h-[calc(100vh-4rem)]` : `${base} h-dvh`;
});

const delegatedBannerMessage = computed(() => auth.delegatedError?.value || "");
const showDelegatedBanner = computed(
  () => insidePowerX.value && Boolean(delegatedBannerMessage.value)
);

const handleDelegatedRetry = async () => {
  try { await auth.restoreFromStorage(); } catch { /* Auth composable exposes the error banner. */ }
};
const handleDelegatedDismiss = () => auth.clearDelegatedError?.();
</script>
