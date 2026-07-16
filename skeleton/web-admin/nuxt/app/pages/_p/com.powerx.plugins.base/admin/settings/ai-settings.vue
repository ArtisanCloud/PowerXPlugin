<template>
  <UContainer class="py-8 space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-2">
        <div class="flex flex-wrap items-center gap-2">
          <UBadge color="info" variant="soft">{{ $t("aiSettings.badge") }}</UBadge>
          <UBadge color="neutral" variant="soft">{{ providerModeLabel }}</UBadge>
        </div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ $t("aiSettings.title") }}
        </h1>
        <p class="max-w-3xl text-sm text-gray-600 dark:text-gray-300">
          {{ $t("aiSettings.description") }}
        </p>
      </div>
      <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="loadAll">
        {{ $t("common.refresh") }}
      </UButton>
    </div>

    <UAlert
      v-if="errorMessage"
      color="warning"
      variant="soft"
      icon="i-heroicons-exclamation-triangle"
      :title="$t('aiSettings.providerNotConfigured')"
      :description="errorMessage"
    />

    <div class="grid gap-4 md:grid-cols-3">
      <UCard>
        <p class="text-sm text-gray-500 dark:text-gray-400">{{ $t("aiSettings.cards.provider") }}</p>
        <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ providerMode.provider }}</p>
      </UCard>
      <UCard>
        <p class="text-sm text-gray-500 dark:text-gray-400">{{ $t("aiSettings.cards.delegatedAvailable") }}</p>
        <UBadge class="mt-2" :color="providerMode.delegated_available ? 'success' : 'neutral'" variant="soft">
          {{ providerMode.delegated_available ? $t("common.enabled") : $t("common.disabled") }}
        </UBadge>
      </UCard>
      <UCard>
        <p class="text-sm text-gray-500 dark:text-gray-400">{{ $t("aiSettings.cards.health") }}</p>
        <p class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ healthLabel }}</p>
      </UCard>
    </div>

    <div class="grid gap-4 xl:grid-cols-2">
      <UCard>
        <template #header>
          <h2 class="text-base font-semibold">{{ $t("aiSettings.sections.summary") }}</h2>
        </template>
        <dl class="grid gap-3 sm:grid-cols-2">
          <div v-for="item in summaryEntries" :key="item.key" class="rounded-md border border-gray-200 p-3 dark:border-gray-800">
            <dt class="text-xs text-gray-500 dark:text-gray-400">{{ item.key }}</dt>
            <dd class="mt-1 break-all text-sm font-medium text-gray-900 dark:text-white">{{ item.value }}</dd>
          </div>
        </dl>
        <p v-if="!summaryEntries.length" class="text-sm text-gray-500 dark:text-gray-400">{{ $t("aiSettings.empty") }}</p>
      </UCard>

      <UCard>
        <template #header>
          <h2 class="text-base font-semibold">{{ $t("aiSettings.sections.routing") }}</h2>
        </template>
        <dl class="grid gap-3 sm:grid-cols-2">
          <div v-for="item in routingEntries" :key="item.key" class="rounded-md border border-gray-200 p-3 dark:border-gray-800">
            <dt class="text-xs text-gray-500 dark:text-gray-400">{{ item.key }}</dt>
            <dd class="mt-1 break-all text-sm font-medium text-gray-900 dark:text-white">{{ item.value }}</dd>
          </div>
        </dl>
        <p v-if="!routingEntries.length" class="text-sm text-gray-500 dark:text-gray-400">{{ $t("aiSettings.empty") }}</p>
      </UCard>
    </div>

    <div class="grid gap-4 xl:grid-cols-2">
      <UCard>
        <template #header>
          <h2 class="text-base font-semibold">{{ $t("aiSettings.sections.providers") }}</h2>
        </template>
        <UTable :data="providerProfiles" :columns="profileColumns" :loading="loading" />
      </UCard>

      <UCard>
        <template #header>
          <h2 class="text-base font-semibold">{{ $t("aiSettings.sections.models") }}</h2>
        </template>
        <UTable :data="modelProfiles" :columns="profileColumns" :loading="loading" />
      </UCard>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
import { useApiClient } from "~/composables/api/_client";
import { defaultProviderMode, normalizeProviderMode, type ProviderModeDiagnostics } from "~/composables/api/useProviderMode";

const { t } = useI18n();
const { client } = useApiClient();

const loading = ref(false);
const errorMessage = ref("");
const providerMode = ref<ProviderModeDiagnostics>(defaultProviderMode());
const summary = ref<Record<string, any>>({});
const routing = ref<Record<string, any>>({});
const health = ref<Record<string, any>>({});
const providerProfiles = ref<Record<string, any>[]>([]);
const modelProfiles = ref<Record<string, any>[]>([]);

const providerModeLabel = computed(() =>
  providerMode.value.mode === "delegated" ? t("providerMode.delegated") : t("providerMode.local")
);
const healthLabel = computed(() => String(health.value.status || health.value.state || "-"));
const summaryEntries = computed(() => objectEntries(summary.value));
const routingEntries = computed(() => objectEntries(routing.value));
const profileColumns = computed(() => [
  { accessorKey: "name", header: t("aiSettings.columns.name") },
  { accessorKey: "provider", header: t("aiSettings.columns.provider") },
  { accessorKey: "status", header: t("aiSettings.columns.status") },
]);

function objectEntries(input: Record<string, any>) {
  return Object.entries(input || {}).map(([key, value]) => ({
    key,
    value: typeof value === "object" && value !== null ? JSON.stringify(value) : String(value ?? "-"),
  }));
}

function unwrapData(response: any) {
  return response?.data ?? response ?? {};
}

function unwrapItems(response: any) {
  const data = unwrapData(response);
  return Array.isArray(data?.items) ? data.items : [];
}

async function loadMode() {
  const response = await client<any>("admin/ai-settings/mode", { method: "GET", silentAuthError: true });
  providerMode.value = normalizeProviderMode(response?.data);
}

async function loadAll() {
  loading.value = true;
  errorMessage.value = "";
  try {
    await loadMode();
    const [summaryRes, providersRes, modelsRes, routingRes, healthRes] = await Promise.all([
      client<any>("admin/ai-settings/summary", { method: "GET", silentAuthError: true }),
      client<any>("admin/ai-settings/provider-profiles", { method: "GET", silentAuthError: true }),
      client<any>("admin/ai-settings/model-profiles", { method: "GET", silentAuthError: true }),
      client<any>("admin/ai-settings/routing", { method: "GET", silentAuthError: true }),
      client<any>("admin/ai-settings/health", { method: "GET", silentAuthError: true }),
    ]);
    summary.value = unwrapData(summaryRes);
    providerProfiles.value = unwrapItems(providersRes);
    modelProfiles.value = unwrapItems(modelsRes);
    routing.value = unwrapData(routingRes);
    health.value = unwrapData(healthRes);
  } catch (error: any) {
    errorMessage.value = error?.data?.error?.message || error?.response?._data?.error?.message || error?.message || t("aiSettings.providerNotConfiguredDescription");
    summary.value = {};
    providerProfiles.value = [];
    modelProfiles.value = [];
    routing.value = {};
    health.value = {};
  } finally {
    loading.value = false;
  }
}

onMounted(loadAll);
</script>
