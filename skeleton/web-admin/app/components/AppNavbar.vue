<template>
  <div
    class="border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900"
  >
    <div class="flex h-16 items-center justify-between px-6">
      <!-- 左侧品牌 -->
      <div class="flex items-center">
        <UIcon
          name="i-heroicons-rocket-launch"
          class="w-6 h-6 mr-3 text-primary-600"
        />
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
          {{ $t("common.appName") }}
        </h1>
      </div>

      <!-- 中间快捷操作 -->
      <div class="flex items-center space-x-4"></div>

      <!-- 右侧控制区 -->
      <div class="flex items-center space-x-4">
        <!-- 通知 -->
        <UButton
          variant="ghost"
          color="neutral"
          size="sm"
          square
          @click="toggleNotifications"
        >
          <UIcon name="i-heroicons-bell" class="w-5 h-5" />
        </UButton>

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
  </div>
</template>

<script setup lang="ts">
const { t } = useI18n();

// 用户菜单项
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
      click: () => logout(),
    },
  ],
];

// 切换通知
const toggleNotifications = () => {
  // TODO: 实现通知面板切换
  console.log("切换通知");
};

// 退出登录
const logout = () => {
  // TODO: 实现退出登录功能
  console.log("退出登录");
};
</script>
