<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { storeToRefs } from "pinia";
import { usePermissionStore } from "~/stores/permission";

const permissionStore = usePermissionStore();
const { listData, isLoading, error } = storeToRefs(permissionStore);
const searchQuery = ref("");

const permissions = computed(() => {
  const query = searchQuery.value.trim().toLowerCase();
  return (listData.value.items || []).filter((permission) => {
    if (!query) return true;
    const label = permission.meta?.label || "";
    return [permission.resource, permission.action, permission.description, label]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(query));
  });
});

onMounted(() => permissionStore.fetchList());
</script>

<template>
  <section class="space-y-4">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h2 class="text-xl font-semibold text-gray-800 dark:text-white">
          {{ $t("organization.permission.title") }}
        </h2>
        <p class="text-sm text-gray-500 dark:text-gray-300">
          {{ $t("organization.permission.description") }}
        </p>
      </div>
      <UButton color="neutral" variant="outline" icon="i-heroicons-arrow-path" :loading="isLoading" @click="permissionStore.fetchList()">
        {{ $t("common.refresh") }}
      </UButton>
    </div>

    <UInput v-model="searchQuery" icon="i-heroicons-magnifying-glass" :placeholder="$t('organization.permission.search')" />

    <div v-if="error" class="text-sm text-red-600 dark:text-red-300">{{ error }}</div>
    <div v-else-if="isLoading" class="py-8 text-center text-gray-500">{{ $t("common.loading") }}</div>
    <div v-else-if="permissions.length === 0" class="py-8 text-center text-gray-500">{{ $t("organization.permission.empty.title") }}</div>
    <div v-else class="divide-y rounded-lg border border-gray-200 dark:border-slate-700">
      <div v-for="permission in permissions" :key="permission.permission_uuid" class="p-4">
        <div class="font-medium text-gray-900 dark:text-white">{{ permission.meta?.label || `${permission.resource}:${permission.action}` }}</div>
        <div class="mt-1 text-sm text-gray-500 dark:text-gray-300">{{ permission.description || "-" }}</div>
      </div>
    </div>
  </section>
</template>
