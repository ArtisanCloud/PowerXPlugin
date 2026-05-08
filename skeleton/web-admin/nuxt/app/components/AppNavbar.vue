<template>
  <div
    class="border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900"
  >
    <div class="flex h-16 items-center justify-between px-6">
      <!-- 左侧品牌 -->
      <div class="flex items-center">
        <img
          :src="logoSrc"
          alt="PowerX Plugin Logo"
          class="h-8 w-auto mr-3"
        />
        <div class="flex items-center gap-2">
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
            {{ $t("common.appName") }}
          </h1>
          <UBadge color="neutral" variant="outline" class="text-[11px] font-medium">
            v{{ pluginVersion }}
          </UBadge>
        </div>
      </div>

      <!-- 中间快捷操作 -->
      <div class="flex items-center space-x-3">
        <UBadge
          :color="iamModeBadge.color"
          variant="outline"
          class="text-xs font-semibold"
        >
          {{ iamModeBadge.label }}
        </UBadge>
        <p class="text-xs text-gray-600 dark:text-gray-300">
          {{ iamModeBadge.description }}
        </p>
      </div>

      <!-- 右侧控制区 -->
      <div class="flex items-center space-x-4">
        <!-- 通知 -->
        <div class="relative">
          <UButton
            data-test="navbar-notify-button"
            variant="ghost"
            :color="wsStateColor"
            size="sm"
            square
            @click="toggleNotifications"
          >
            <UIcon name="i-heroicons-bell" class="w-5 h-5" />
          </UButton>
          <span
            v-if="unreadCount > 0"
            data-test="navbar-notify-unread"
            class="absolute -right-1 -top-1 min-w-[18px] rounded-full bg-rose-500 px-1 text-center text-[10px] leading-[18px] font-semibold text-white"
          >
            {{ unreadCount > 99 ? "99+" : unreadCount }}
          </span>
        </div>

        <ThemeSelector />
        <LanguageSelector />

        <!-- 用户头像和下拉菜单 -->
        <UDropdownMenu
          :items="userMenuItems"
          :popper="{ placement: 'bottom-end' }"
        >
          <UAvatar
            src="https://avatars.githubusercontent.com/u/739984?v=4"
            alt="管理员"
            size="sm"
            class="cursor-pointer"
          />
        </UDropdownMenu>
      </div>
    </div>

    <USlideover v-model:open="notificationsOpen">
      <template #content>
        <div class="flex h-full flex-col">
          <div class="border-b border-gray-200 px-4 py-3 dark:border-gray-800">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-gray-100">通知中心</h3>
              <div class="flex items-center gap-2">
                <UBadge data-test="navbar-notify-ws-state" :color="wsStateColor" variant="soft" size="sm">{{ wsStateLabel }}</UBadge>
                <UButton size="xs" variant="ghost" color="neutral" @click="reconnectWS">重连</UButton>
              </div>
            </div>
            <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
              最近事件：
              <span v-if="lastEventAt">{{ formatTime(lastEventAt) }} / {{ lastEventTopic || "-" }}</span>
              <span v-else>暂无</span>
            </p>
            <p v-if="wsError" class="mt-1 text-xs text-rose-500">{{ wsError }}</p>
          </div>

          <div class="flex items-center gap-2 border-b border-gray-200 px-4 py-3 dark:border-gray-800">
            <UButton data-test="notify-send-test" size="xs" color="primary" @click="sendProbe">发送测试通知</UButton>
            <UButton size="xs" variant="ghost" color="neutral" @click="markAllRead">全部已读</UButton>
            <UButton size="xs" variant="ghost" color="neutral" @click="clearEvents">清空</UButton>
          </div>

          <div class="flex-1 overflow-auto p-4">
            <div v-if="events.length === 0" class="text-sm text-gray-500 dark:text-gray-400">暂无通知事件</div>
            <div v-else class="space-y-3">
              <div
                v-for="item in events"
                :key="item.id"
                data-test="navbar-notify-item"
                class="rounded-lg border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-900"
              >
                <div class="flex items-center justify-between gap-2">
                  <p class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ item.title }}</p>
                  <UBadge color="neutral" variant="soft" size="xs">{{ item.type }}</UBadge>
                </div>
                <p class="mt-1 text-xs text-gray-600 dark:text-gray-300">{{ item.message || "无消息体" }}</p>
                <p class="mt-2 text-[11px] text-gray-500 dark:text-gray-400">topic: {{ item.topic }}</p>
                <p class="text-[11px] text-gray-500 dark:text-gray-400">{{ formatTime(item.receivedAt) }}</p>
              </div>
            </div>
          </div>
        </div>
      </template>
    </USlideover>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useAuth } from "~/composables/useAuth";
import { getTenantUuid } from "~/composables/api/_base";
import { resolveTenantUUIDForRequest } from "~/utils/tenant-context";
import { useNotificationProbe } from "~/composables/useNotificationProbe";

const { t } = useI18n();
const runtimeConfig = useRuntimeConfig();
const auth = useAuth();
const notificationsOpen = ref(false);
const {
  wsStateLabel,
  wsStateColor,
  wsError,
  lastEventAt,
  lastEventTopic,
  unreadCount,
  events,
  connect,
  disconnect,
  subscribeTopic,
  markAllRead,
  clearEvents,
  sendTestNotification,
} = useNotificationProbe();

const logoSrc = computed(() => {
  const base = runtimeConfig.public.insidePowerX
    ? runtimeConfig.public.pluginAdminBase ?? "/"
    : "/";
  return `${base.replace(/\/$/, "")}/images/logo-s.png`;
});

const pluginVersion = computed(
  () => String(runtimeConfig.public?.powerxPluginVersion || "dev")
);

// 用户菜单项
const handleLogout = async () => {
  await auth.logout();
};

const iamModeBadge = computed(() => {
  const standalone = auth.localIAMEnabled?.value ?? false;
  return {
    label: standalone ? "本地 IAM" : "Delegated IAM",
    description: standalone
      ? "当前使用本地目录鉴权"
      : "通过宿主 PowerX 鉴权",
    color: standalone ? "green" : "yellow",
  };
});

const userMenuItems = [
  [
    {
      label: t("navigation.profile"),
      avatar: {
        src: "https://avatars.githubusercontent.com/u/739984?v=4",
      },
      click: () => navigateTo("/profile"),
    },
  ],
  [
    {
      label: t("navigation.help"),
      icon: "i-heroicons-question-mark-circle",
      click: () => navigateTo("/help"),
    },
  ],
  [
    {
      label: t("navigation.logout"),
      icon: "i-heroicons-arrow-right-on-rectangle",
      onSelect: () => handleLogout(),
    },
  ],
];

// 切换通知
const toggleNotifications = () => {
  notificationsOpen.value = !notificationsOpen.value;
  if (notificationsOpen.value) {
    markAllRead();
  }
};

const reconnectWS = () => {
  disconnect();
  connect();
  subscribeNotificationTopics();
};

const sendProbe = async () => {
  try {
    await sendTestNotification();
  } catch (error) {
    console.error("[AppNavbar] failed to send WS probe", error);
  }
};

const formatTime = (value?: string) => {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
};

const subscribeNotificationTopics = () => {
  subscribeTopic("_topic.system.notification");
  const tenantUUID = String(getTenantUuid() || resolveTenantUUIDForRequest() || "").trim();
  if (tenantUUID) {
    subscribeTopic(`plugin.notify.tenant.${tenantUUID}`);
  }
};

onMounted(() => {
  connect();
  subscribeNotificationTopics();
});

onUnmounted(() => {
  disconnect();
});
</script>
