<template>
  <div class="min-h-screen bg-gray-50 dark:bg-gray-900">
    <!-- 顶部导航栏 - 根据环境变量控制显示 -->
    <UContainer v-if="showNavigation" class="max-w-none">
      <AppNavbar />
    </UContainer>

    <div class="flex">
      <!-- 左侧边栏 - 根据环境变量控制显示 -->
      <AppSidebar v-if="showNavigation" />

      <!-- 主内容区 -->
      <main :class="mainContentClass">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup>
import { watch } from "vue";
import { setupHostBridgeAdapter } from "~/composables/useHostBridgeAdapter";
import { useTheme } from "~/composables/useTheme";
import { PLUGIN_ID, isPluginAdminPath } from "~/utils/powerx-bridge";

// 获取运行时配置
const runtimeConfig = useRuntimeConfig();
const route = useRoute();
const theme = useTheme();

// 是否处于 PowerX 宿主的插件嵌入路径下
const insidePowerX = computed(() => {
  const value = runtimeConfig.public.insidePowerX;
  return value === true || value === 'true';
});

const isEmbeddedInPowerX = computed(() => {
  if (insidePowerX.value) {
    return true;
  }
  return isPluginAdminPath(route.path);
});

// 控制导航显示的环境变量
const showNavigation = computed(() => {
  if (isEmbeddedInPowerX.value) {
    return false;
  }

  // 优先检查环境变量 NUXT_PUBLIC_SHOW_NAVIGATION
  const envShowNav = runtimeConfig.public.showNavigation;

  // 如果没有设置环境变量，开发环境默认显示，生产环境默认隐藏
  if (envShowNav !== undefined) {
    return envShowNav === "true" || envShowNav === true;
  }

  // 默认：开发环境显示，生产环境隐藏
  return import.meta.dev;
});

// 主内容区样式
const mainContentClass = computed(() => {
  return showNavigation.value ? "flex-1 p-6" : "w-full p-6";
});

const getAdapterRegistry = (win) => {
  if (!win.__PX_ADAPTERS__) {
    win.__PX_ADAPTERS__ = {};
  }
  return win.__PX_ADAPTERS__;
};

const mountBridgeIfNeeded = () => {
  if (!import.meta.client) {
    return;
  }

  const win = window;
  if (!isEmbeddedInPowerX.value) {
    return;
  }

  try {
    theme.initTheme?.();
  } catch {}

  const pluginId = typeof route.query.pluginId === "string" ? route.query.pluginId : PLUGIN_ID;
  const instanceId = typeof route.query.instanceId === "string" ? route.query.instanceId : route.fullPath;
  const registry = getAdapterRegistry(win);
  const adapterKey = `${pluginId}::${instanceId}`;

  if (registry[adapterKey]) {
    console.info("[Bridge][Plugin] adapter already mounted, reuse existing instance.", {
      pluginId,
      instanceId,
    });
    return;
  }

  const { bridge } = setupHostBridgeAdapter({
    pluginId,
    instanceId,
  });

  bridge.start?.();
  console.info("[Bridge][Plugin] adapter mounted.");
  registry[adapterKey] = { bridge };
  win.__PX_ADAPTER_BOUND__ = true;
  win.__PX_ADAPTER__ = registry[adapterKey];
  console.info("[embedded] Host bridge adapter mounted.", { pluginId, instanceId });
};

if (import.meta.client) {
  watch(
    () => isEmbeddedInPowerX.value,
    (value) => {
      if (value) {
        mountBridgeIfNeeded();
      }
    },
    { immediate: true }
  );
}
</script>
