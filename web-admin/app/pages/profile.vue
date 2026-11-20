<template>
  <div class="min-h-screen bg-gray-50 p-6 text-gray-900 dark:bg-gray-950 dark:text-white">
    <div class="mx-auto max-w-4xl space-y-5">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold">个人中心</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">当前登录账号与租户身份。</p>
        </div>
        <UButton icon="i-heroicons-arrow-path" color="neutral" variant="soft" :loading="userStore.isLoading" @click="refresh">
          刷新
        </UButton>
      </div>

      <section class="rounded-lg border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
        <div class="flex flex-wrap items-center gap-4">
          <UAvatar :src="avatarUrl" :text="avatarText" size="xl" />
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="truncate text-xl font-semibold">{{ displayName }}</h2>
              <UBadge :color="userStore.isActive ? 'success' : 'error'" variant="soft">
                {{ userStore.isActive ? "启用" : "停用" }}
              </UBadge>
              <UBadge :color="userStore.isCurrentTenantAdmin ? 'primary' : 'neutral'" variant="soft">
                {{ userStore.isCurrentTenantAdmin ? "插件租户管理员" : "普通成员" }}
              </UBadge>
            </div>
            <p class="mt-1 truncate text-sm text-gray-500 dark:text-gray-400">{{ user?.email || "-" }}</p>
          </div>
        </div>
      </section>

      <section class="grid gap-4 md:grid-cols-2">
        <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-gray-100">账号信息</h3>
          <dl class="mt-4 space-y-3 text-sm">
            <div class="flex justify-between gap-4">
              <dt class="text-gray-500 dark:text-gray-400">用户 UUID</dt>
              <dd class="max-w-[65%] truncate font-mono text-xs">{{ userStore.currentUserUuid || "-" }}</dd>
            </div>
            <div class="flex justify-between gap-4">
              <dt class="text-gray-500 dark:text-gray-400">用户名</dt>
              <dd class="max-w-[65%] truncate">{{ user?.username || "-" }}</dd>
            </div>
            <div class="flex justify-between gap-4">
              <dt class="text-gray-500 dark:text-gray-400">手机</dt>
              <dd class="max-w-[65%] truncate">{{ user?.phone || "-" }}</dd>
            </div>
            <div class="flex justify-between gap-4">
              <dt class="text-gray-500 dark:text-gray-400">IAM 模式</dt>
              <dd>{{ providerModeLabel }}</dd>
            </div>
          </dl>
        </div>

        <div class="rounded-lg border border-gray-200 bg-white p-5 dark:border-gray-800 dark:bg-gray-900">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-gray-100">当前租户</h3>
          <dl class="mt-4 space-y-3 text-sm">
            <div class="flex justify-between gap-4">
              <dt class="text-gray-500 dark:text-gray-400">租户名称</dt>
              <dd class="max-w-[65%] truncate">{{ currentTenant?.tenant_name || "-" }}</dd>
            </div>
            <div class="flex justify-between gap-4">
              <dt class="text-gray-500 dark:text-gray-400">租户 UUID</dt>
              <dd class="max-w-[65%] truncate font-mono text-xs">{{ userStore.currentTenantUuid || "-" }}</dd>
            </div>
            <div class="flex justify-between gap-4">
              <dt class="text-gray-500 dark:text-gray-400">成员 UUID</dt>
              <dd class="max-w-[65%] truncate font-mono text-xs">{{ userStore.currentMemberUuid || "-" }}</dd>
            </div>
            <div class="flex justify-between gap-4">
              <dt class="text-gray-500 dark:text-gray-400">租户角色</dt>
              <dd>{{ userStore.isRoot ? "Root" : userStore.isCurrentTenantAdmin ? "Admin" : "Member" }}</dd>
            </div>
          </dl>
        </div>
      </section>

      <section class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-900/70 dark:bg-amber-950/30 dark:text-amber-100">
        这里显示的是插件管理端登录身份。PowerX Skill/Agent upsert 使用后端 PX_GATEWAY_* 出站凭证，是否具备 PowerX Admin 权限取决于那份 Gateway token/API key。
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useUserStore } from "~/stores/user";

const runtimeConfig = useRuntimeConfig();
const userStore = useUserStore();

const user = computed(() => userStore.user);
const currentTenant = computed(() => userStore.currentTenant);
const displayName = computed(() => userStore.displayName);
const avatarUrl = computed(() => userStore.avatarUrl);
const avatarText = computed(() => displayName.value.slice(0, 1).toUpperCase() || "U");
const providerModeLabel = computed(() => {
  const raw = String(runtimeConfig.public?.providerMode || "").trim();
  if (raw) return raw;
  return runtimeConfig.public?.delegatedMode || runtimeConfig.public?.insidePowerX ? "delegated" : "local";
});

async function refresh() {
  await userStore.fetchUserContext({ force: true });
}

onMounted(() => {
  if (!userStore.context) {
    void refresh();
  }
});
</script>
