<template>
  <UContainer class="py-10 space-y-6">
    <section class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-col gap-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ $t("capabilities.exposure.title") }}
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          {{ $t("capabilities.exposure.description") }}
        </p>
      </div>
      <div class="flex flex-wrap gap-3">
        <UButton
          icon="i-heroicons-arrow-path"
          variant="ghost"
          color="neutral"
          :loading="catalogLoading"
          @click="loadCatalog"
        >
          {{ $t("capabilities.exposure.list.refresh") }}
        </UButton>
      </div>
    </section>

    <UCard>
      <template #header>
        <div class="flex flex-col gap-1">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ $t("capabilities.exposure.list.title") }}
          </h2>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ $t("capabilities.exposure.list.description") }}
          </p>
        </div>
      </template>
      <div v-if="catalogLoading" class="flex items-center justify-center py-16 text-gray-500 dark:text-gray-400">
        {{ $t("common.loading") }}
      </div>
      <div v-else>
        <UTable
          v-if="catalogRows.length"
          :rows="catalogRows"
          :columns="tableColumns"
          :ui="{ td: { base: 'align-top' } }"
        >
          <template #capability_id-cell="{ row }">
            <div class="flex flex-col">
              <span class="font-semibold text-gray-900 dark:text-white">
                {{ row.capability_id }}
              </span>
              <small class="text-xs text-gray-500 dark:text-gray-400">
                {{ $t("capabilities.list.versionLabel", { version: row.version }) }}
              </small>
            </div>
          </template>
          <template #descriptor-cell="{ row }">
            <p class="text-sm text-gray-600 dark:text-gray-300">{{ row.descriptor }}</p>
          </template>
          <template #exposure-cell="{ row }">
            <div class="flex flex-col gap-1">
              <UBadge :label="exposureBadge(row.syncStatus).label" :color="exposureBadge(row.syncStatus).color" variant="soft" />
              <span v-if="row.updatedAt" class="text-xs text-gray-500">
                {{ $t("capabilities.exposure.list.updatedAt", { time: row.updatedAt }) }}
              </span>
            </div>
          </template>
          <template #actions-cell="{ row }">
            <UButton
              size="xs"
              icon="i-heroicons-adjustments-horizontal"
              @click="openForm(row.capability_id)"
            >
              {{ $t("capabilities.exposure.list.actions.configure") }}
            </UButton>
          </template>
        </UTable>
        <div v-else class="py-16 text-center">
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ $t("capabilities.exposure.list.empty") }}
          </p>
        </div>
      </div>
    </UCard>

    <UModal
      v-model:open="formOpen"
      prevent-close
      :ui="{ content: 'max-w-6xl w-[95vw] mx-auto' }"
    >
      <template #body>
        <div class="space-y-6">
          <div class="flex flex-col gap-2">
            <div class="flex items-start justify-between gap-4">
              <div>
                <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
                  {{ $t("capabilities.exposure.title") }}
                </h2>
                <p class="text-gray-600 dark:text-gray-300">
                  {{ $t("capabilities.exposure.description") }}
                </p>
              </div>
              <UButton icon="i-heroicons-x-mark" variant="ghost" color="neutral" @click="closeForm" />
            </div>
            <div class="flex flex-wrap items-center gap-3 text-sm text-gray-600 dark:text-gray-300">
              <div class="inline-flex items-center gap-1">
                <UIcon name="i-heroicons-cube" />
                <span class="font-medium">{{ $t("capabilities.exposure.capability") }}:</span>
                <code class="rounded bg-gray-100 px-2 py-0.5 text-gray-900 dark:bg-gray-800 dark:text-gray-100">
                  {{ form.capability_id || "—" }}
                </code>
              </div>
              <div class="inline-flex items-center gap-2">
                <UBadge :label="modalStatus.label" :color="modalStatus.color" variant="soft" />
                <span class="text-xs text-gray-500">
                  {{ $t("capabilities.exposure.updatedAt", { time: packageInfo?.updated_at || "—" }) }}
                </span>
              </div>
            </div>
          </div>

          <UCard :ui="{ body: 'space-y-8' }">
            <section class="space-y-4">
              <div class="grid gap-4 md:grid-cols-2">
                <UFormField :label="$t('capabilities.exposure.fields.capabilityId')" required class="md:col-span-2">
                  <UInput v-model="form.capability_id" readonly />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.docsVersion')">
                  <UInput v-model="form.docs_version" placeholder="1.0.0" />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.sdkVersion')">
                  <UInput v-model="form.sdk_version" placeholder="1.0.0" />
                </UFormField>
              </div>
              <div class="flex flex-wrap gap-3">
                <UButton variant="ghost" color="neutral" :loading="loading" @click="handleLoadPackage">
                  {{ $t("capabilities.exposure.actions.load") }}
                </UButton>
                <UButton variant="ghost" color="neutral" @click="resetChannels">
                  {{ $t("capabilities.exposure.actions.resetChannels") }}
                </UButton>
                <UButton color="primary" :disabled="!form.capability_id" :loading="saving" @click="handleSave">
                  {{ $t("common.save") }}
                </UButton>
              </div>
            </section>

            <section class="space-y-4">
              <div class="flex items-center justify-between">
                <h2 class="text-lg font-semibold">
                  {{ $t("capabilities.exposure.sections.channels") }}
                </h2>
                <span class="text-sm text-gray-500">
                  {{ selectedChannels.length }}/{{ form.channels.length }}
                  {{ $t("capabilities.exposure.sections.enabled") }}
                </span>
              </div>
              <div class="grid gap-4 lg:grid-cols-2">
                <div
                  v-for="channel in form.channels"
                  :key="channel.type"
                  class="border rounded-lg border-gray-200 dark:border-gray-800 p-4 space-y-3"
                >
                  <div class="flex items-center justify-between">
                    <div class="flex flex-col">
                      <span class="font-semibold capitalize">{{ channel.type }}</span>
                      <span class="text-xs text-gray-500">
                        {{ $t("capabilities.exposure.sections.channelHint") }}
                      </span>
                    </div>
                    <USwitch v-model="channel.enabled" color="primary" />
                  </div>
                  <div class="grid gap-3">
                    <UFormField :label="$t('capabilities.exposure.fields.displayName')">
                      <UInput v-model="channel.name" />
                    </UFormField>
                    <div class="grid gap-3 md:grid-cols-2" v-if="channel.type === 'rest' || channel.type === 'webhook'">
                      <UFormField label="HTTP">
                        <div class="flex gap-2">
                          <USelectMenu v-model="channel.method" :options="httpMethods" class="w-24" />
                          <UInput v-model="channel.path" class="flex-1" />
                        </div>
                      </UFormField>
                    </div>
                    <UFormField v-if="channel.type !== 'rest'" :label="$t('capabilities.exposure.fields.target')">
                      <UInput v-model="channel.target" placeholder="powerx.capability.Service/Method" />
                    </UFormField>
                    <UFormField :label="$t('capabilities.exposure.fields.description')">
                      <UTextarea v-model="channel.description" :rows="2" />
                    </UFormField>
                    <UFormField :label="$t('capabilities.exposure.fields.scopes')">
                      <UInput
                        v-model="channel.scopesText"
                        :placeholder="$t('capabilities.exposure.fields.scopePlaceholder')"
                        @change="syncScopeList(channel)"
                      />
                    </UFormField>
                  </div>
                </div>
              </div>
            </section>

            <section class="grid gap-4 md:grid-cols-2">
              <div class="space-y-3">
                <h2 class="text-lg font-semibold">
                  {{ $t("capabilities.exposure.sections.auth") }}
                </h2>
                <UFormField :label="$t('capabilities.exposure.fields.strategy')">
                  <USelectMenu v-model="form.auth.strategy" :options="template?.auth_strategies || []" />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.audience')">
                  <UInput v-model="form.auth.audience" />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.scopeList')">
                  <UInput
                    v-model="authScopes"
                    :placeholder="$t('capabilities.exposure.fields.scopePlaceholder')"
                    @change="syncAuthScopes"
                  />
                </UFormField>
              </div>
              <div class="space-y-3">
                <h2 class="text-lg font-semibold">
                  {{ $t("capabilities.exposure.sections.rateLimit") }}
                </h2>
                <UFormField :label="$t('capabilities.exposure.fields.rpm')">
                  <UInput v-model.number="form.rate_limit.requests_per_minute" type="number" min="1" />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.burst')">
                  <UInput v-model.number="form.rate_limit.burst" type="number" min="1" />
                </UFormField>
                <UFormField :label="$t('capabilities.exposure.fields.concurrency')">
                  <UInput v-model.number="form.rate_limit.concurrency" type="number" min="1" />
                </UFormField>
              </div>
            </section>

            <section class="space-y-4">
              <div class="flex items-center justify-between">
                <h2 class="text-lg font-semibold">
                  {{ $t("capabilities.exposure.sections.quotas") }}
                </h2>
                <div class="text-sm text-gray-500">
                  {{ $t("capabilities.exposure.sections.tenantHint") }}
                </div>
              </div>
              <div class="overflow-x-auto border rounded-lg border-gray-200 dark:border-gray-800">
                <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-800 text-sm">
                  <thead class="bg-gray-50 dark:bg-gray-900">
                    <tr>
                      <th class="px-4 py-2 text-left font-semibold">{{ $t("capabilities.exposure.fields.tenant") }}</th>
                      <th class="px-4 py-2 text-left font-semibold">{{ $t("capabilities.exposure.fields.quota") }}</th>
                      <th class="px-4 py-2 text-left font-semibold">{{ $t("capabilities.exposure.fields.status") }}</th>
                      <th class="px-4 py-2 text-left font-semibold w-48">{{ $t("capabilities.exposure.fields.notes") }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-100 dark:divide-gray-800">
                    <tr v-if="quotasList.length === 0">
                      <td colspan="4" class="px-4 py-4 text-center text-gray-500">
                        {{ $t("capabilities.exposure.sections.noQuotas") }}
                      </td>
                    </tr>
                    <tr v-for="quota in quotasList" :key="quota.tenant_id">
                      <td class="px-4 py-2">
                        <div class="font-medium">{{ quota.tenant_id }}</div>
                        <div class="text-xs text-gray-500">{{ quota.tenant_name }}</div>
                      </td>
                      <td class="px-4 py-2">
                        {{ quota.used || 0 }} / {{ quota.quota }}
                      </td>
                      <td class="px-4 py-2">
                        <UBadge :label="quota.status || 'active'" color="primary" variant="soft" />
                      </td>
                      <td class="px-4 py-2 break-words">
                        <span class="text-xs text-gray-500">{{ quota.notes }}</span>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div class="border rounded-lg border-gray-200 dark:border-gray-800 p-4 space-y-3">
                <div class="grid gap-3 md:grid-cols-2">
                  <UFormField :label="$t('capabilities.exposure.fields.tenantId')" required>
                    <UInput v-model="newQuota.tenant_id" />
                  </UFormField>
                  <UFormField :label="$t('capabilities.exposure.fields.tenantName')">
                    <UInput v-model="newQuota.tenant_name" />
                  </UFormField>
                  <UFormField :label="$t('capabilities.exposure.fields.quota')" required>
                    <UInput v-model.number="newQuota.quota" type="number" min="0" />
                  </UFormField>
                  <UFormField :label="$t('capabilities.exposure.fields.status')">
                    <USelectMenu v-model="newQuota.status" :options="statusOptions" />
                  </UFormField>
                  <UFormField class="md:col-span-2" :label="$t('capabilities.exposure.fields.notes')">
                    <UInput v-model="newQuota.notes" />
                  </UFormField>
                </div>
                <div class="flex justify-end">
                  <UButton
                    color="neutral"
                    variant="soft"
                    :disabled="!form.capability_id"
                    :loading="quotaSaving"
                    @click="handleQuotaSave"
                  >
                    {{ $t("capabilities.exposure.actions.addTenant") }}
                  </UButton>
                </div>
              </div>
            </section>
          </UCard>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useToast, useRoute, useRouter, useI18n } from "#imports";
import {
  useCapabilityExposureApi,
  useCapabilityCatalogApi,
  type CapabilityCatalogEntry,
  type ExposureChannel,
  type ExposurePackage,
  type ExposureTemplate,
  type TenantQuota,
} from "~/composables/api";
import { useNormalizedColumns } from "~/utils/table";

definePageMeta({
  alias: ["/capabilities/exposure"],
});

type ChannelFormEntry = ExposureChannel & { scopesText?: string };
type ExposureMetaState = { status: string; updated_at?: string };
type StatusBadge = { label: string; color: string };

const httpMethods = ["GET", "POST", "PUT", "PATCH", "DELETE"];
const statusOptions = ["active", "suspended"];

const { t } = useI18n();
const toast = useToast();
const route = useRoute();
const router = useRouter();

const { getTemplate, getPackage, upsertPackage, listQuotas, upsertQuota } =
  useCapabilityExposureApi();
const { list: listCatalog } = useCapabilityCatalogApi();

const template = ref<ExposureTemplate | null>(null);
const packageInfo = ref<ExposurePackage | null>(null);
const formOpen = ref(false);
const catalogLoading = ref(false);
const catalog = ref<CapabilityCatalogEntry[]>([]);
const exposureMeta = ref<Record<string, ExposureMetaState>>({});

const form = reactive({
  capability_id: "",
  docs_version: "1.0.0",
  sdk_version: "1.0.0",
  auth: {
    strategy: "powerx_session",
    audience: "",
    scopes: [] as string[],
  },
  rate_limit: {
    requests_per_minute: 600,
    burst: 120,
    concurrency: 10,
  },
  channels: [] as ChannelFormEntry[],
});

const quotas = ref<TenantQuota[]>([]);
const newQuota = reactive({
  tenant_id: "",
  tenant_name: "",
  quota: 1000,
  status: "active",
  notes: "",
});

const loading = ref(false);
const saving = ref(false);
const quotaSaving = ref(false);

const statusPresets = computed<Record<string, StatusBadge>>(() => ({
  unconfigured: {
    label: t("capabilities.exposure.list.status.unconfigured"),
    color: "gray",
  },
  synced: {
    label: t("capabilities.exposure.list.status.synced"),
    color: "primary",
  },
  pending: {
    label: t("capabilities.exposure.list.status.pending"),
    color: "amber",
  },
  failed: {
    label: t("capabilities.exposure.list.status.failed"),
    color: "rose",
  },
}));

const catalogRows = computed(() =>
  (catalog.value || []).map((entry) => {
    const meta = exposureMeta.value[entry.id] || { status: "unconfigured" };
    return {
      capability_id: entry.id,
      version: entry.version,
      descriptor: entry.descriptor,
      syncStatus: meta.status,
      updatedAt: meta.updated_at || "",
    };
  }),
);

const tableColumns = useNormalizedColumns([
  { key: "capability_id", label: t("capabilities.exposure.list.columns.capability") },
  { key: "descriptor", label: t("capabilities.exposure.list.columns.descriptor") },
  { key: "exposure", label: t("capabilities.exposure.list.columns.exposure") },
  { key: "actions", label: "" },
]);

const selectedChannels = computed(() =>
  form.channels.filter((channel) => channel.enabled),
);
const quotasList = computed(() => quotas.value ?? []);
const authScopes = ref("");
const modalStatus = computed(() => exposureBadge(packageInfo.value?.sync_status));

onMounted(async () => {
  await hydrateTemplate();
  await loadCatalog();
  const capabilityFromQuery = route.query.capability as string | undefined;
  if (capabilityFromQuery) {
    await openForm(capabilityFromQuery);
  }
});

function exposureBadge(status?: string): StatusBadge {
  const presets = statusPresets.value;
  if (status && presets[status]) {
    return presets[status];
  }
  if (status && !presets[status]) {
    return { label: status, color: "gray" };
  }
  return presets.unconfigured;
}

async function loadCatalog() {
  catalogLoading.value = true;
  try {
    catalog.value = await listCatalog();
  } catch (error) {
    console.error("[capabilities] failed to load catalog", error);
    toast.add({
      title: t("capabilities.exposure.toast.catalogFailed"),
      color: "rose",
    });
  } finally {
    catalogLoading.value = false;
  }
}

async function openForm(capabilityId: string) {
  if (!capabilityId) {
    toast.add({
      title: t("capabilities.exposure.toast.capabilityRequired"),
      color: "rose",
    });
    return;
  }
  form.capability_id = capabilityId;
  formOpen.value = true;
  await handleLoadPackage();
}

function closeForm() {
  formOpen.value = false;
  const nextQuery = { ...route.query };
  if (Reflect.has(nextQuery, "capability")) {
    delete (nextQuery as Record<string, any>).capability;
    router.replace({ query: nextQuery });
  }
}

function setExposureMeta(capabilityId: string, status?: string, updatedAt?: string) {
  if (!capabilityId) {
    return;
  }
  exposureMeta.value = {
    ...exposureMeta.value,
    [capabilityId]: {
      status: status || "unconfigured",
      updated_at: updatedAt,
    },
  };
}

async function hydrateTemplate() {
  try {
    template.value = await getTemplate();
    if (template.value?.default_rate) {
      form.rate_limit = { ...template.value.default_rate };
    }
    if (!form.channels.length) {
      form.channels = buildChannelEntries(template.value?.channel_types || []);
    }
  } catch (error) {
    console.error(error);
    toast.add({
      title: t("capabilities.exposure.toast.templateFail"),
      color: "rose",
    });
  }
}

function buildChannelEntries(types: string[]): ChannelFormEntry[] {
  return types.map((type) => ({
    type,
    name: type.toUpperCase(),
    method: type === "rest" ? "POST" : "",
    path: "",
    target: "",
    description: "",
    enabled: false,
    scopes: [],
    scopesText: "",
  }));
}

function resetChannels() {
  form.channels = buildChannelEntries(template.value?.channel_types || []);
}

async function handleLoadPackage() {
  if (!form.capability_id) {
    toast.add({
      title: t("capabilities.exposure.toast.capabilityRequired"),
      color: "rose",
    });
    return;
  }
  loading.value = true;
  try {
    const data = await getPackage(form.capability_id);
    const pkg = data?.package || null;
    packageInfo.value = pkg;
    if (pkg) {
      form.docs_version = pkg.docs_version || "1.0.0";
      form.sdk_version = pkg.sdk_version || "1.0.0";
      form.auth = {
        strategy: pkg.auth?.strategy || "powerx_session",
        audience: pkg.auth?.audience || "",
        scopes: pkg.auth?.scopes || [],
      };
      authScopes.value = (pkg.auth?.scopes || []).join(", ");
      form.rate_limit = { ...pkg.rate_limit };
      syncChannelEntries(pkg.channels || []);
      setExposureMeta(pkg.capability_id, pkg.sync_status, pkg.updated_at);
    } else {
      packageInfo.value = null;
      resetChannels();
      setExposureMeta(form.capability_id, "unconfigured", undefined);
    }
    const quotaResp = await listQuotas(form.capability_id);
    quotas.value = quotaResp?.quotas || [];
    await router.replace({
      query: {
        ...route.query,
        capability: form.capability_id,
      },
    });
  } catch (error) {
    console.error(error);
    toast.add({
      title: t("capabilities.exposure.toast.loadFail"),
      color: "rose",
    });
  } finally {
    loading.value = false;
  }
}

function syncChannelEntries(existing: ExposureChannel[]) {
  const combined = buildChannelEntries(template.value?.channel_types || []);
  const map = new Map(
    existing.map((channel) => [channel.type, channel] as const),
  );
  for (const entry of combined) {
    const defined = map.get(entry.type);
    if (!defined) continue;
    entry.enabled = true;
    entry.name = defined.name || entry.name;
    entry.method = defined.method || entry.method;
    entry.path = defined.path || entry.path;
    entry.target = defined.target || "";
    entry.description = defined.description || "";
    entry.scopes = defined.scopes || [];
    entry.scopesText = (defined.scopes || []).join(", ");
  }
  form.channels = combined;
}

function syncScopeList(channel: ChannelFormEntry) {
  channel.scopes = (channel.scopesText || "")
    .split(",")
    .map((scope) => scope.trim())
    .filter(Boolean);
}

function syncAuthScopes() {
  form.auth.scopes = authScopes.value
    .split(",")
    .map((scope) => scope.trim())
    .filter(Boolean);
}

async function handleSave() {
  if (!form.capability_id) {
    toast.add({
      title: t("capabilities.exposure.toast.capabilityRequired"),
      color: "rose",
    });
    return;
  }
  saving.value = true;
  try {
    const payload = {
      capability_id: form.capability_id,
      docs_version: form.docs_version,
      sdk_version: form.sdk_version,
      auth: { ...form.auth },
      rate_limit: { ...form.rate_limit },
      channels: form.channels.map((channel) => ({
        type: channel.type,
        name: channel.name,
        enabled: channel.enabled,
        method: channel.method,
        path: channel.path,
        target: channel.target,
        description: channel.description,
        scopes: channel.scopes,
      })),
      tenants: quotas.value,
    };
    const record = await upsertPackage(payload);
    packageInfo.value = record;
    setExposureMeta(record.capability_id, record.sync_status, record.updated_at);
    await loadCatalog();
    toast.add({
      title: t("capabilities.exposure.toast.saveSuccess"),
      color: "primary",
    });
  } catch (error) {
    console.error(error);
    toast.add({
      title: t("capabilities.exposure.toast.saveFailed"),
      color: "rose",
    });
  } finally {
    saving.value = false;
  }
}

async function handleQuotaSave() {
  if (!form.capability_id) {
    toast.add({
      title: t("capabilities.exposure.toast.capabilityRequired"),
      color: "rose",
    });
    return;
  }
  if (!newQuota.tenant_id) {
    toast.add({
      title: t("capabilities.exposure.toast.tenantRequired"),
      color: "rose",
    });
    return;
  }
  quotaSaving.value = true;
  try {
    const payload: TenantQuota = {
      tenant_id: newQuota.tenant_id,
      tenant_name: newQuota.tenant_name,
      quota: newQuota.quota,
      status: newQuota.status,
      notes: newQuota.notes,
    };
    const record = await upsertQuota(form.capability_id, payload);
    quotas.value = record.tenants || [];
    packageInfo.value = record;
    setExposureMeta(record.capability_id, record.sync_status, record.updated_at);
    Object.assign(newQuota, {
      tenant_id: "",
      tenant_name: "",
      quota: 1000,
      status: "active",
      notes: "",
    });
    toast.add({
      title: t("capabilities.exposure.toast.quotaSaved"),
      color: "primary",
    });
  } catch (error) {
    console.error(error);
    toast.add({
      title: t("capabilities.exposure.toast.quotaFailed"),
      color: "rose",
    });
  } finally {
    quotaSaving.value = false;
  }
}
</script>
