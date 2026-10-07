<template>
  <KnowledgeSpaceOverviewPanel v-if="section === 'overview'" description-key="knowledgeLab.description" :spaces="overviewRows" :loading="providerLoading" :error="spacesError || providerError">
    <template #header-actions><UButton color="neutral" variant="outline" icon="i-heroicons-arrow-path" :loading="providerLoading" @click="refreshKnowledgeState">{{ t('common.refresh') }}</UButton><UButton icon="i-heroicons-plus-circle" :to="`${labBase}/create`">{{ t('knowledgeSpaces.ui.createSpace') }}</UButton></template>
    <template #row-actions="{ space }"><UButton size="xs" icon="i-heroicons-arrow-up-tray" :to="sectionPath(space.id, 'ingestion')">{{ t('knowledgeSpaces.ui.ingest') }}</UButton>
<UButton size="xs" color="neutral" variant="soft" icon="i-heroicons-link" :to="sectionPath(space.id, 'sources')">{{ t('knowledgeSpaces.ui.connectDataSource') }}</UButton>
<UButton size="xs" color="neutral" variant="soft" icon="i-heroicons-list-bullet" :to="sectionPath(space.id, 'records')">{{ t('knowledgeSpaces.ui.ingestionRecords') }}</UButton>
<UButton size="xs" color="neutral" variant="soft" icon="i-heroicons-adjustments-horizontal" :to="sectionPath(space.id, 'strategy')">{{ t('knowledgeSpaces.ui.strategy') }}</UButton>
<UButton size="xs" color="neutral" variant="soft" icon="i-heroicons-magnifying-glass" :to="sectionPath(space.id, 'playground')">{{ t('knowledgeSpaces.ui.playground') }}</UButton>
<UButton v-if="rawSpace(space.id).status !== 'retired'" size="xs" color="error" variant="soft" :loading="retiringSpaceId === space.id" @click="retireSpace(rawSpace(space.id))">{{ t('knowledgeLab.copy.text144') }}</UButton>
<UButton size="xs" color="error" variant="soft" icon="i-heroicons-trash" :disabled="rawSpace(space.id).status !== 'retired'" :loading="deletingSpaceId === space.id" @click="deleteSpace(rawSpace(space.id))">{{ t('common.delete') }}</UButton></template>
    <template #empty-actions><UButton class="mt-5" icon="i-heroicons-plus-circle" :to="`${labBase}/create`">{{ t('knowledgeSpaces.ui.createSpace') }}</UButton></template>
    <template #notices>
      <UAlert v-if="knowledgeNotice.message" :color="knowledgeNotice.color" :title="knowledgeNotice.title" :description="knowledgeNotice.message" /></template>
  </KnowledgeSpaceOverviewPanel>
  <KnowledgeSpaceCreatePanel v-else-if="section === 'create'"
    v-model:name="createForm.spaceName" v-model:department="createForm.departmentUUID"
    :strategy="createForm.strategyPackageKey" @update:strategy="onStrategyPackageChange"
    :overview-path="labBase" :department-options="creationDepartments" :strategy-options="strategyPackageItems"
    departments-empty-key="knowledgeLab.departmentsEmpty" :departments-error="creationDepartmentsError" :departments-loading="creationDepartmentsLoading" :strategy-error="catalogError || providerError || strategyUnavailableDescription"
    :strategy-summary="selectedStrategyPackage.key ? strategyUseCase(selectedStrategyPackage) : ''"
    :mapped-profile="selectedStrategyPackage.key ? profileLabel(selectedStrategyPackage.recommended_profile_key) : '-'"
    :execution-status="creationExecutionStatus" :status-color="selectedStrategyPackage.available ? 'success' : 'warning'"
    :can-save="canCreateSpace" :saving="createLoading"
    description-key="knowledgeLab.createDescription" basic-description-key="knowledgeLab.createBasicDescription"
    execution-status-key="knowledgeLab.executionStatus" @save="createKnowledgeSpace">
    <template #notices>
      <div class="flex justify-end"><UButton color="neutral" variant="outline" icon="i-heroicons-arrow-path" :loading="providerLoading" @click="refreshKnowledgeState">{{ t('common.refresh') }}</UButton></div>

      <UAlert v-if="providerError || createError" color="error" variant="soft" :title="t('knowledgeLab.copy.text084')" :description="providerError || createError" /></template>
    <template #strategy-notice>
      <UFormField :label="t('knowledgeLab.policyTemplate')" required>
        <USelectMenu v-model="createForm.policyTemplateUUID" :items="policyTemplateOptions" value-key="value" label-key="label" :portal="true" :ui="{ content: 'z-50' }" class="w-full" :disabled="createLoading || !!catalogError || !policyTemplateOptions.length" :placeholder="t('knowledgeLab.selectPolicyTemplate')" />
      </UFormField>
      <p v-if="knowledgeCatalog.quota_defaults" class="mt-3 text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeLab.quotaDefaults', knowledgeCatalog.quota_defaults) }}</p>
    </template>
  </KnowledgeSpaceCreatePanel>
  <KnowledgeHostWorkspace v-else :section="section" />
  <div class="mx-auto w-full px-5 pb-6 lg:px-8" :class="section === 'create' ? 'max-w-5xl' : 'max-w-[1600px]'"><details class="rounded-xl border border-slate-200 p-4 text-sm text-gray-600 dark:border-slate-700 dark:text-[#d6e2ff]"><summary class="cursor-pointer">{{ t('knowledgeLab.copy.text157') }}</summary><div class="mt-4 space-y-3"><UBadge variant="soft">{{ t('knowledgeLab.proxyMode') }}</UBadge><div>{{ providerState.provider }}</div><div class="flex flex-wrap gap-2"><UBadge v-for="operation in capabilities.operations" :key="operation" color="neutral" variant="soft">{{ operation }}</UBadge></div><pre v-if="result" class="max-h-56 overflow-auto rounded-md bg-gray-950 p-3 text-xs text-gray-100">{{ diagnosticsText }}</pre></div></details></div>
  <ToastAlert v-model="toast.visible" :title="toast.title" :message="toast.message" :color="toast.color" :duration="toast.duration" />
</template>
<script setup lang="ts">
import KnowledgeHostWorkspace from './KnowledgeHostWorkspace.vue';
import KnowledgeSpaceCreatePanel from './KnowledgeSpaceCreatePanel.vue';

import KnowledgeSpaceOverviewPanel, { type KnowledgeOverviewRow } from './KnowledgeSpaceOverviewPanel.vue';

import { computed, onMounted, reactive, ref } from "vue";

import ToastAlert from "~/components/ToastAlert.vue";

import { useApiClient } from "~/composables/api/_client";


type ProviderResponse = {
  provider?: string;
  mode?: string;
  production?: boolean;
  capabilities?: {
    provider?: string;
    mode?: string;
    operations?: string[];
    health?: string;
    limits?: Record<string, unknown>;
  };
};


type SearchResult = {
  provider?: string;
  total?: number;
  chunks?: Array<{
    chunk_id?: string;
    document_id?: string;
    space_id?: string;
    text?: string;
    score?: number;
    citation?: {
      title?: string;
      document_id?: string;
      chunk_id?: string;
      provider?: string;
      version?: string;
    };
  }>;
  citations?: Array<{
    title?: string;
    document_id?: string;
    chunk_id?: string;
    provider?: string;
    version?: string;
  }>;
  diagnostics?: Record<string, unknown>;
};


type SpaceRow = {
  id: string;
  name: string;
  spaceId?: string;
  spaceName?: string;
  tenant_uuid?: string;
  department: string;
  departmentCode?: string;
  type: string;
  typeLabel?: string;
  type_label?: string;
  status: string;
  statusLabel?: string;
  status_label?: string;
  statusColor?: "success" | "warning" | "neutral" | "error" | "info";
  status_color?: "success" | "warning" | "neutral" | "error" | "info";
  provider?: string;
  provider_mode?: string;
  policyTemplateVersionId?: string;
  ingestionProfileKey?: string;
  indexProfileKey?: string;
  ragProfileKey?: string;
  featureFlags?: string[];
  quotas?: Record<string, unknown>;
  contract_gaps?: string[];
  icon?: string;
};


type StrategyDependency = {
  index?: string[];
  runtime?: string[];
  assets?: string[];
};


type KnowledgeScenePreset = {
  key: string;
  label: string;
  description?: string;
  default_strategy_package?: string;
  default_bundle?: string;
};


type ProfileRef = { uuid: string; key: string; version: number };

type StrategyPackagePreset = {
  profiles?: { ingestion?: ProfileRef; index?: ProfileRef; rag?: ProfileRef };
  available?: boolean;
  unavailable_reasons?: string[];
  activation_dependencies?: string[];
  recommended_scenes?: string[];
  key: string;
  label: string;
  display_label?: string;
  summary?: string;
  use_case?: string;
  not_for?: string;
  recommended_profile_key?: string;
  dependencies?: StrategyDependency;
};


type StrategyBundlePreset = {
  key: string;
  label: string;
  description?: string;
};


type KnowledgeCatalog = {
  policy_templates?: Array<{ uuid: string; name: string; version: string }>;
  default_policy_template_uuid?: string;
  quota_defaults?: { cpu_cores: number; storage_gb: number; ingestion_concurrency: number };
  quota_override_allowed?: boolean;
  version?: string;
  source?: string;
  scenes?: KnowledgeScenePreset[];
  strategy_packages?: StrategyPackagePreset[];
  strategy_bundles?: StrategyBundlePreset[];
  metadata?: Record<string, unknown>;
};


const { get, post, delete: del } = useApiClient();

const route = useRoute();

const { section = 'overview' } = defineProps<{ section?: 'overview' | 'create' | 'ingestion' | 'sources' | 'records' | 'strategy' | 'playground' }>();

const labBase = computed(() => route.path.slice(0, route.path.indexOf('/knowledge-lab') + '/knowledge-lab'.length));

function sectionPath(id: string, target: string) { return `${labBase.value}/${encodeURIComponent(id)}/${target}`; }

function rawSpace(id: string) { return spaces.value.find(space => space.id === id)!; }

const overviewRows = computed<KnowledgeOverviewRow[]>(() => spaces.value.map(space => ({ id: space.id, name: space.spaceName || space.name, department: space.department || '-', statusLabel: space.statusLabel || statusLabel(space.status), statusColor: space.statusColor || statusColor(space.status) })));


const { t, te } = useI18n();


const providerLoading = ref(false);

const createLoading = ref(false);

const retiringSpaceId = ref("");

const deletingSpaceId = ref("");

const providerError = ref("");

const catalogFailure = ref<unknown | null>(null);

const catalogError = computed(() => providerError.value || (catalogFailure.value ? errorMessage(catalogFailure.value) : ""));

const spacesError = ref("");

const createError = ref("");

const providerState = ref<ProviderResponse>({ mode: "delegated" });

const knowledgeCatalog = ref<KnowledgeCatalog>({});

// Host departments are read separately from the knowledge catalog.
const creationDepartments = ref<Array<{ label: string; value: string }>>([]);

const creationDepartmentsFailure = ref<unknown | null>(null);

const creationDepartmentsError = computed(() => providerError.value || (creationDepartmentsFailure.value ? errorMessage(creationDepartmentsFailure.value) : ""));

const creationDepartmentsLoading = ref(false);

const strategyUnavailableDescription = computed(() => {
 const pkg = selectedStrategyPackage.value;
 if (!pkg.key || pkg.available === true || catalogError.value) return '';
 const reasons = (pkg.unavailable_reasons || []).map(reason => te(`knowledgeLab.reasons.${reason}`) ? t(`knowledgeLab.reasons.${reason}`) : t('knowledgeLab.dependencyUnavailable', { dependency: reason }));
 return t('knowledgeLab.strategyUnavailable', { reasons: reasons.join(' · ') || t('knowledgeLab.readinessUnavailable') });
});

const creationExecutionStatus = computed(() => catalogError.value || !selectedStrategyPackage.value.key ? t('knowledgeLab.readinessUnavailable') : selectedStrategyPackage.value.available ? t('knowledgeLab.creatablePendingActivation') : t('knowledgeLab.strategyNotCreatable'));

const policyTemplateOptions = computed(() => (knowledgeCatalog.value.policy_templates || []).map(item => ({ label: `${item.name} · ${item.version}`, value: item.uuid })));

const canCreateSpace = computed(() => !providerLoading.value && !createLoading.value && !creationDepartmentsLoading.value && !providerError.value && !catalogError.value && !creationDepartmentsError.value
  && !!createForm.spaceName.trim() && new TextEncoder().encode(createForm.spaceName.trim()).length <= 128
  && creationDepartments.value.some(item => item.value === createForm.departmentUUID)
  && selectedStrategyPackage.value.available === true
  && !!selectedStrategyPackage.value.profiles?.ingestion?.uuid && !!selectedStrategyPackage.value.profiles?.index?.uuid && !!selectedStrategyPackage.value.profiles?.rag?.uuid
  && policyTemplateOptions.value.some(item => item.value === createForm.policyTemplateUUID));

const result = ref<SearchResult | null>(null);

const selectedSpaceId = ref("");

const contractGaps = ref<string[]>([]);

const toast = reactive({
  visible: false,
  title: "",
  message: "",
  color: "success" as "primary" | "secondary" | "success" | "info" | "warning" | "error" | "neutral",
  duration: 4500,
});

const knowledgeNotice = reactive({
  title: "",
  message: "",
  color: "info" as "primary" | "secondary" | "success" | "info" | "warning" | "error" | "neutral",
  icon: "i-heroicons-information-circle",
});


const createForm = reactive({
  spaceName: "",
  departmentUUID: "",
  policyTemplateUUID: "",
  sceneKey: "",
  strategyPackageKey: "",
  ingestionProfileKey: "",
  indexProfileKey: "",
  ragProfileKey: "",
  featureFlags: [] as string[],
  cpuCores: 4,
  storageGb: 200,
  ingestionConcurrency: 2,
});


const spaces = ref<SpaceRow[]>([]);


const profileLabels: Record<string, string> = {
  p0_basic: t('knowledgeLab.copy.text002'),
  p1_general: t('knowledgeLab.copy.text003'),
  p2_high_accuracy: t('knowledgeLab.copy.text004'),
  p3_kg_strong: t('knowledgeLab.copy.text005'),
};

const strategyPackagesByKey = computed(() => Object.fromEntries((knowledgeCatalog.value.strategy_packages || []).map((pkg) => [pkg.key, pkg])));

const strategyPackageItems = computed(() => (knowledgeCatalog.value.strategy_packages || []).map((pkg) => ({ label: `${strategyDisplayLabel(pkg)} · ${pkg.summary || pkg.label || pkg.key}`, value: pkg.key })));

const selectedStrategyPackage = computed<StrategyPackagePreset>(() => strategyPackagesByKey.value[createForm.strategyPackageKey] || { key: "", label: "" });


function profileLabel(key?: string) {
  return profileLabels[key || ""] || key || "-";
}


function normalizeStrategyPackageKey(value: unknown) {
  if (typeof value === "string") return value;
  if (value && typeof value === "object" && "value" in value) {
    return String((value as { value?: unknown }).value || "");
  }
  return "";
}


function setCreateStrategyPackage(value: unknown, sceneKey?: string) {
  const key = normalizeStrategyPackageKey(value);
  const packageKey = strategyPackagesByKey.value[key] ? key : "";
  const pkg = strategyPackagesByKey.value[packageKey];
  const profileKey = pkg?.recommended_profile_key || "";
  const resolvedSceneKey = sceneKey && pkg?.recommended_scenes?.includes(sceneKey) ? sceneKey : pkg?.recommended_scenes?.[0] || "";
  createForm.sceneKey = resolvedSceneKey;
  createForm.strategyPackageKey = packageKey;
  createForm.ingestionProfileKey = profileKey;
  createForm.indexProfileKey = profileKey;
  createForm.ragProfileKey = profileKey;
  createForm.featureFlags = [
    packageKey ? "rag.strategy_package:" + packageKey : "",
    resolvedSceneKey ? "rag.scene:" + resolvedSceneKey : "",
    profileKey ? "rag.bundle:" + profileKey : "",
  ].filter(Boolean);
  if (resolvedSceneKey === "custom_expert") {
    createForm.featureFlags.push("rag.guided");
  }
}


function onStrategyPackageChange(value: unknown) {
  setCreateStrategyPackage(value);
}


function strategyDisplayLabel(pkg: StrategyPackagePreset) {
  return pkg.display_label || pkg.label || pkg.key || "-";
}


function strategyUseCase(pkg: StrategyPackagePreset) {
  return pkg.use_case || pkg.summary || pkg.label || "-";
}


async function createKnowledgeSpace() {
  if (createLoading.value) return;
  if (!canCreateSpace.value) { createError.value = catalogError.value || creationDepartmentsError.value || strategyUnavailableDescription.value || t('knowledgeLab.completeCreateForm'); return; }
  createLoading.value = true;
  createError.value = '';
  try {
    const pkg = selectedStrategyPackage.value;
    const payload = unwrap<{ item: { space_uuid: string; status: string } }>(await post('/admin/runtime/knowledge-lab/spaces', {
      name: createForm.spaceName.trim(), department_uuid: createForm.departmentUUID, strategy_key: pkg.key,
      ...(createForm.sceneKey ? { scene_key: createForm.sceneKey } : {}), policy_template_uuid: createForm.policyTemplateUUID,
      ingestion_profile_uuid: pkg.profiles!.ingestion!.uuid, index_profile_uuid: pkg.profiles!.index!.uuid, rag_profile_uuid: pkg.profiles!.rag!.uuid,
    }));
    if (!payload.item?.space_uuid) throw { data: { error: 'knowledgeLab.invalidCreatedSpace' } };
    await navigateTo(labBase.value);
  } catch (error: any) { createError.value = errorMessage(error); }
  finally { createLoading.value = false; }
}


async function retireSpace(space: SpaceRow) {
  const spaceID = String(space.id || space.spaceId || "").trim();
  if (!spaceID) return;
  const name = String(space.spaceName || space.name || t("knowledgeLab.unnamedSpace"));
  const ok = window.confirm(t("knowledgeLab.archiveConfirm", { name }));
  if (!ok) return;
  retiringSpaceId.value = spaceID;
  providerError.value = "";
  try {
    await post(`/admin/runtime/knowledge-lab/spaces/${encodeURIComponent(spaceID)}/retire`, {
      reason: "plugin knowledge lab cleanup",
      requestedBy: "plugin-knowledge-lab",
      dropVectors: false,
    });
    await loadSpaces();
    if (selectedSpaceId.value === spaceID && spaces.value.length && !spaces.value.some((item) => item.id === spaceID && item.status !== "retired")) {
      selectedSpaceId.value = spaces.value.find((item) => item.status !== "retired")?.id || spaces.value[0]?.id || "";
    }
  } catch (error: any) {
    providerError.value = errorMessage(error);
  } finally {
    retiringSpaceId.value = "";
  }
}


async function deleteSpace(space: SpaceRow) {
  const spaceID = String(space.id || space.spaceId || "").trim();
  if (!spaceID) return;
  const name = String(space.spaceName || space.name || t("knowledgeLab.unnamedSpace"));
  const ok = window.confirm(t("knowledgeLab.deleteConfirm", { name }));
  if (!ok) return;
  deletingSpaceId.value = spaceID;
  providerError.value = "";
  try {
    await del(`/admin/runtime/knowledge-lab/spaces/${encodeURIComponent(spaceID)}`, {
      body: JSON.stringify({
        requestedBy: "plugin-knowledge-lab",
        force: false,
        dropVectors: true,
      }),
      headers: {
        "Content-Type": "application/json",
      },
    });
    await loadSpaces();
    if (selectedSpaceId.value === spaceID) {
      selectedSpaceId.value = spaces.value[0]?.id || "";
    }
  } catch (error: any) {
    providerError.value = errorMessage(error);
  } finally {
    deletingSpaceId.value = "";
  }
}


const capabilities = computed(() => providerState.value.capabilities || {});

const diagnosticsText = computed(() => JSON.stringify(result.value?.diagnostics || {}, null, 2));


function unwrap<T>(payload: any): T {
  if (payload && typeof payload === "object" && "data" in payload) {
    return payload.data as T;
  }
  return payload as T;
}


function errorMessage(error: any) {
  const message = String(error?.data?.error?.message || error?.data?.error || error?.message || "");
  if (te(message) && message !== "knowledgeLab.gatewayFailed") return t(message);
  const code = String(error?.data?.code || error?.data?.error?.code || error?.statusCode || error?.status || "");
  const trace = String(error?.data?.trace_id || "");
  return t("knowledgeLab.requestFailed", { code: code || "UNKNOWN" }) + (trace ? ` (${trace})` : "");
}


async function loadProvider() {
  providerLoading.value = true;
  providerError.value = "";
  try {
    providerState.value = unwrap<ProviderResponse>(await get("/admin/runtime/knowledge-lab/provider"));
  } catch (error: any) {
    providerError.value = errorMessage(error);
  } finally {
    providerLoading.value = false;
  }
}


async function loadKnowledgeCatalog() {
  catalogFailure.value = null;
  knowledgeCatalog.value = {};
  try {
    knowledgeCatalog.value = unwrap<KnowledgeCatalog>(await get("/admin/runtime/knowledge-lab/catalog"));
    if (!knowledgeCatalog.value.scenes?.some((scene) => scene.key === createForm.sceneKey)) {
      createForm.sceneKey = knowledgeCatalog.value.scenes?.[0]?.key || "";
    }
    if (!knowledgeCatalog.value.strategy_packages?.some((pkg) => pkg.key === createForm.strategyPackageKey)) {
      createForm.strategyPackageKey = knowledgeCatalog.value.strategy_packages?.find(pkg => pkg.available === true)?.key || knowledgeCatalog.value.strategy_packages?.[0]?.key || "";
    }
    setCreateStrategyPackage(createForm.strategyPackageKey, createForm.sceneKey);
    if (!knowledgeCatalog.value.policy_templates?.some(item => item.uuid === createForm.policyTemplateUUID)) createForm.policyTemplateUUID = knowledgeCatalog.value.default_policy_template_uuid || '';
  } catch (error: any) {
    createForm.sceneKey = "";
    setCreateStrategyPackage("");
    catalogFailure.value = error;
  }
}


async function loadCreationDepartments() {
  creationDepartmentsLoading.value = true;
  creationDepartmentsFailure.value = null;
  creationDepartments.value = [];
  try {
    const payload = unwrap<{ items: Array<{ department_uuid: string; name: string }> }>(await get('/admin/runtime/knowledge-lab/departments'));
    creationDepartments.value = payload.items.map(item => ({ label: item.name, value: item.department_uuid }));
    if (!creationDepartments.value.some(item => item.value === createForm.departmentUUID)) createForm.departmentUUID = "";
  } catch (error: any) {
    createForm.departmentUUID = "";
    creationDepartmentsFailure.value = error;
  } finally {
    creationDepartmentsLoading.value = false;
  }
}


async function loadSpaces() {
  spacesError.value = "";
  spaces.value = [];
  try {
    const payload = unwrap<{ spaces?: SpaceRow[]; contract_gaps?: string[] }>(await get("/admin/runtime/knowledge-lab/spaces"));
    contractGaps.value = payload.contract_gaps || [];
    spaces.value = (payload.spaces || []).map(normalizeSpace);
    const requestedSpaceID = typeof route.params.uuid === "string" ? route.params.uuid : (typeof route.query.space_uuid === "string" ? route.query.space_uuid : "");
    if (requestedSpaceID && !spaces.value.some(space => space.id === requestedSpaceID)) { selectedSpaceId.value = ""; spacesError.value = t("knowledgeLab.spaceUnavailable"); return; }
    if (requestedSpaceID && spaces.value.some((space) => space.id === requestedSpaceID)) {
      selectedSpaceId.value = requestedSpaceID;
    } else if (spaces.value.length && !spaces.value.some((space) => space.id === selectedSpaceId.value)) {
      selectedSpaceId.value = spaces.value[0]?.id || "";
    }
    if (!spaces.value.length) {
      selectedSpaceId.value = "";
    }
  } catch (error: any) {
    spacesError.value = errorMessage(error);
    selectedSpaceId.value = "";
  }
}


async function refreshKnowledgeState() {
  providerLoading.value = true;
  providerError.value = "";
  try {
    await loadProvider();
    if (providerError.value) { spaces.value = []; selectedSpaceId.value = ""; return; }
    if (section === 'create') await Promise.all([loadKnowledgeCatalog(), loadCreationDepartments()]);
    else await Promise.all([loadKnowledgeCatalog(), loadSpaces()]);
  } finally {
    providerLoading.value = false;
  }
}


function normalizeSpace(space: SpaceRow): SpaceRow {
  const status = space.status || "pending";
  const id = space.id || space.spaceId || "";
  const name = space.name || space.spaceName || t("knowledgeLab.unnamedSpace");
  return {
    ...space,
    id,
    name,
    spaceId: space.spaceId || id,
    spaceName: space.spaceName || name,
    department: space.department || space.departmentCode || "-",
    departmentCode: space.departmentCode || space.department || "-",
    typeLabel: space.typeLabel || space.type_label || space.type,
    statusLabel: statusLabel(status),
    statusColor: space.statusColor || space.status_color || statusColor(status),
    icon: space.icon || "i-heroicons-circle-stack",
  };
}


function statusColor(status: string): "success" | "warning" | "neutral" | "error" | "info" {
  if (status === "active" || status === "ready") return "success";
  if (status === "pending" || status === "pending_iam") return "warning";
  if (status === "unavailable") return "error";
  return "neutral";
}


function statusLabel(status: string) {
  if (status === "active") return t('knowledgeLab.copy.text016');
  if (status === "pending_iam") return t('knowledgeLab.copy.text017');
  if (status === "retired") return t('knowledgeLab.copy.text018');
  if (status === "ready") return t('knowledgeLab.copy.text055');
  if (status === "pending") return t('knowledgeLab.copy.text019');
  if (status === "unavailable") return t('knowledgeLab.copy.text020');
  return status;
}


onMounted(async () => { if (section === 'overview' || section === 'create') await refreshKnowledgeState(); });
</script>
