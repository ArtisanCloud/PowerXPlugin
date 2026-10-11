<template>
  <div class="mx-auto max-w-7xl space-y-6 p-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t("hostContract.title") }}</h1>
        <p class="mt-1 max-w-3xl text-sm text-gray-600 dark:text-gray-300">{{ t("hostContract.description") }}</p>
      </div>
      <UButton icon="i-heroicons-arrow-path" :loading="refreshing" @click="refreshAll">{{ t("hostContract.refresh") }}</UButton>
    </div>
    <UAlert color="info" icon="i-heroicons-information-circle" :title="t('hostContract.boundaryTitle')" :description="t('hostContract.boundaryDescription')" />
    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <UCard v-for="item in visibleModules" :key="item.module" class="bg-white/95 dark:bg-gray-900">
        <template #header>
          <div class="flex items-center justify-between gap-3">
            <div><h2 class="font-semibold text-gray-900 dark:text-white">{{ t(["hostContract", "modules", item.module].join(".")) }}</h2><p class="mt-1 text-xs text-gray-500 dark:text-gray-400">status</p></div>
            <UBadge :color="badgeColor(item)" variant="subtle">{{ statusLabel(item) }}</UBadge>
          </div>
        </template>
        <dl class="space-y-2 text-sm">
          <div><dt class="text-gray-500 dark:text-gray-400">{{ t("hostContract.capability") }}</dt><dd class="break-all font-mono text-xs">{{ item.capabilityID || t("hostContract.notDeclared") }}</dd></div>
          <div><dt class="text-gray-500 dark:text-gray-400">{{ t("hostContract.providerMode") }}</dt><dd>{{ item.providerMode || "-" }}</dd></div>
          <div><dt class="text-gray-500 dark:text-gray-400">{{ t("hostContract.trace") }}</dt><dd class="break-all font-mono text-xs">{{ item.traceID || "-" }}</dd></div>
          <div><dt class="text-gray-500 dark:text-gray-400">{{ t("hostContract.reason") }}</dt><dd class="break-all font-mono text-xs">{{ item.reasonCode || "-" }}</dd></div>
        </dl>
      </UCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute } from "#imports";
import { hostContractBindingState, useHostContractLabApi, type HostContractModule } from "~/composables/api";

const { t } = useI18n();
const route = useRoute();
const { probe, normalizeError } = useHostContractLabApi();
const refreshing = ref(false);
type Card = { module: HostContractModule; capabilityID: string; providerMode: string; traceID: string; reasonCode: string; loading: boolean; ready: boolean | null };
const modules = reactive<Card[]>([
  { module: "cache", capabilityID: "", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
  { module: "taskcenter", capabilityID: "", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
  { module: "iam", capabilityID: "", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
  { module: "knowledge", capabilityID: "", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
  { module: "media", capabilityID: "com.corex.media.assets.read", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
  { module: "agent", capabilityID: "", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
  { module: "ai", capabilityID: "", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
  { module: "capability_registry", capabilityID: "", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
  { module: "integration_gateway", capabilityID: "", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
  { module: "skills", capabilityID: "", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
  { module: "notifications", capabilityID: "", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
  { module: "plugin_runtime", capabilityID: "", providerMode: "", traceID: "", reasonCode: "", loading: false, ready: null },
]);
const requestedModule = computed(() => {
  const value = Array.isArray(route.query.module) ? route.query.module[0] : route.query.module;
  return typeof value === "string" && modules.some((item) => item.module === value) ? value as HostContractModule : "";
});
const visibleModules = computed(() => requestedModule.value ? modules.filter((item) => item.module === requestedModule.value) : modules);
async function runProbe(item: Card) {
  item.loading = true;
  item.reasonCode = "";
  try {
    const result = (await probe(item.module)).data;
    item.capabilityID = result.capability_id || item.capabilityID;
    item.providerMode = result.provider_mode;
    item.traceID = result.trace_id || "";
    item.reasonCode = result.reason_code || "";
    item.ready = hostContractBindingState(result);
  } catch (error) {
    const normalized = normalizeError(error);
    item.reasonCode = normalized.reasonCode;
    item.traceID = normalized.traceID || "";
    item.ready = false;
  } finally { item.loading = false; }
}
async function refreshAll() { refreshing.value = true; try { await Promise.all(visibleModules.value.map(runProbe)); } finally { refreshing.value = false; } }
function badgeColor(item: Card) { return item.ready === true ? "success" : item.ready === false ? "error" : "neutral"; }
function statusLabel(item: Card) { return item.ready === true ? t("hostContract.adapterAvailable") : item.ready === false ? t(item.reasonCode === "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" ? "hostContract.adapterMissing" : "hostContract.unavailable") : t("hostContract.notChecked"); }
onMounted(refreshAll);
</script>
