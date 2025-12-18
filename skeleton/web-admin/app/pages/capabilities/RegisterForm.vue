<template>
  <UContainer class="py-10 space-y-6">
    <section class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-col gap-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ $t("capabilities.list.title") }}
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          {{ $t("capabilities.list.description") }}
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
          {{ $t("capabilities.list.refresh") }}
        </UButton>
        <UButton icon="i-heroicons-plus" color="primary" @click="openForm">
          {{ $t("capabilities.list.createButton") }}
        </UButton>
      </div>
    </section>

    <UCard>
      <template #header>
        <div class="flex flex-col gap-1">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ $t("capabilities.list.tableTitle") }}
          </h2>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ $t("capabilities.list.tableHint") }}
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
          <template #execution-cell="{ row }">
            <UBadge
              :label="row.execution.mode.toUpperCase()"
              :color="row.execution.mode === 'async' ? 'primary' : 'gray'"
              variant="soft"
            />
            <p v-if="row.execution.callback_url" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ row.execution.callback_url }}
            </p>
          </template>
          <template #tags-cell="{ row }">
            <div class="flex flex-wrap gap-1.5">
              <UBadge
                v-for="tag in row.tags"
                :key="`${row.capability_id}-${tag}`"
                :label="tag"
                variant="subtle"
              />
              <span v-if="!row.tags.length" class="text-xs text-gray-400">
                {{ $t("capabilities.list.noTags") }}
              </span>
            </div>
          </template>
          <template #checksum-cell="{ row }">
            <code class="text-xs text-gray-500 dark:text-gray-400">{{ row.checksum }}</code>
          </template>
        </UTable>
        <div v-else class="py-16 text-center">
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ $t("capabilities.list.empty") }}
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
          <div class="flex flex-col gap-1">
            <div class="flex items-center justify-between">
              <div>
                <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
                  {{ $t("capabilities.form.title") }}
                </h2>
                <p class="text-gray-600 dark:text-gray-300">
                  {{ $t("capabilities.form.description") }}
                </p>
              </div>
              <UButton icon="i-heroicons-x-mark" variant="ghost" color="neutral" @click="closeForm" />
            </div>
            <div class="flex flex-wrap items-center gap-3 text-sm text-gray-600 dark:text-gray-300">
              <div class="inline-flex items-center gap-1">
                <UIcon name="i-heroicons-identification" />
                <span class="font-medium">{{ $t("capabilities.form.capabilityId") }}:</span>
                <code class="rounded bg-gray-100 px-2 py-0.5 text-gray-900 dark:bg-gray-800 dark:text-gray-100">
                  {{ capabilityId || "—" }}
                </code>
              </div>
              <UBadge :label="validationBadge.label" :color="validationBadge.color" variant="soft" />
            </div>
          </div>

          <UCard :ui="{ body: 'space-y-8' }">
            <template #header>
              <div class="flex flex-col gap-2">
                <div class="flex flex-wrap gap-3">
                  <button
                    v-for="(item, idx) in stepItems"
                    :key="item.label"
                    type="button"
                    class="flex items-center gap-2 rounded-full border px-4 py-1 text-sm transition"
                    :class="idx === currentStep
                      ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-950 dark:text-primary-300'
                      : 'border-gray-200 text-gray-500 dark:border-gray-800 dark:text-gray-400'"
                    @click="currentStep = idx"
                  >
                    <span
                      class="flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold"
                      :class="idx === currentStep
                        ? 'bg-primary-600 text-white'
                        : 'bg-gray-200 text-gray-600 dark:bg-gray-800 dark:text-gray-300'"
                    >
                      {{ idx + 1 }}
                    </span>
                    {{ item.label }}
                  </button>
                </div>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  {{ stepItems[currentStep]?.description }}
                </p>
              </div>
            </template>

            <div
              v-if="loadingTemplate"
              class="flex items-center justify-center py-16 text-gray-500 dark:text-gray-400"
            >
              {{ $t("common.loading") }}
            </div>

            <template v-else>
        <section v-if="currentStep === 0" class="space-y-6">
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.namespace')" required>
              <UInput v-model="form.namespace" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.resource')" required>
              <UInput v-model="form.resource" />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.action')" required>
              <UInput v-model="form.action" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.sensitivity')" required>
              <USelectMenu
                v-model="form.sensitivity"
                :options="template?.sensitivity_options || ['low','medium','high']"
              />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.tenantScope')">
              <UInput v-model="form.tenant_scope" placeholder="global" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.tags')" :description="$t('capabilities.form.tagsHint')">
              <UInput v-model="tagsText" placeholder="workflow,agent" />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField class="md:col-span-2" :label="$t('capabilities.form.scenario')" :description="$t('capabilities.form.scenarioHint')" required>
              <UTextarea v-model="form.scenario" :rows="3" />
            </UFormField>
          </div>

          <div class="space-y-2">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ $t("capabilities.form.displayName") }}
              <span class="text-red-500">*</span>
            </p>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField :label="$t('capabilities.form.localeZh')" required>
                <UInput v-model="form.name.zh" />
              </UFormField>
              <UFormField :label="$t('capabilities.form.localeEn')" required>
                <UInput v-model="form.name.en" />
              </UFormField>
            </div>
          </div>

          <div class="space-y-2">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ $t("capabilities.form.summary") }}
              <span class="text-red-500">*</span>
            </p>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField :label="$t('capabilities.form.localeZh')" required>
                <UInput v-model="form.summary.zh" />
              </UFormField>
              <UFormField :label="$t('capabilities.form.localeEn')" required>
                <UInput v-model="form.summary.en" />
              </UFormField>
            </div>
          </div>

          <div class="space-y-2">
            <p class="text-sm font-medium text-gray-700 dark:text-gray-200">
              {{ $t("capabilities.form.descriptionField") }}
            </p>
            <div class="grid gap-4 md:grid-cols-2">
              <UFormField :label="$t('capabilities.form.localeZh')">
                <UTextarea v-model="form.description.zh" :rows="4" />
              </UFormField>
              <UFormField :label="$t('capabilities.form.localeEn')">
                <UTextarea v-model="form.description.en" :rows="4" />
              </UFormField>
            </div>
          </div>

        </section>

        <section v-else-if="currentStep === 1" class="space-y-6">
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.inputSchema')" required :description="template?.field_hints?.['schemas.input']">
              <UInput v-model="form.schemas.input" placeholder="contracts/schema/input/xxx.json" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.outputSchema')" required :description="template?.field_hints?.['schemas.output']">
              <UInput v-model="form.schemas.output" placeholder="contracts/schema/output/xxx.json" />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.restPath')" :description="template?.protocol_samples?.rest_path">
              <div class="flex gap-3">
                <USelectMenu v-model="form.protocols.rest.method" :options="httpMethods" class="w-32" />
                <UInput v-model="form.protocols.rest.path" class="flex-1" />
              </div>
            </UFormField>
            <UFormField :label="$t('capabilities.form.grpcService')" :description="template?.protocol_samples?.grpc_service">
              <UInput v-model="form.protocols.grpc.service" placeholder="powerx.template.TemplateService/Create" />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.workflowTemplate')" :description="template?.protocol_samples?.workflow_template">
              <UInput v-model="form.protocols.workflow.template" placeholder="contracts/exposure/workflow/template-create.json" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.agentStream')">
              <UInput v-model="form.protocols.agent_stream.channel" placeholder="contracts/exposure/agent-streams/create.yaml" />
            </UFormField>
          </div>

          <div class="border-t border-gray-200 pt-4 dark:border-gray-800">
            <h3 class="text-base font-semibold">
              {{ $t("capabilities.form.asyncTitle") }}
            </h3>
            <div class="mt-3 grid gap-4 md:grid-cols-3">
              <UFormField :label="$t('capabilities.form.executionMode')">
                <USelectMenu v-model="form.async_mode" :options="template?.async_modes || ['sync','async']" />
              </UFormField>
              <UFormField :label="$t('capabilities.form.callbackUrl')" :disabled="form.async_mode !== 'async'">
                <UInput v-model="form.async_config.callback_url" :disabled="form.async_mode !== 'async'" />
              </UFormField>
              <UFormField :label="$t('capabilities.form.statusEndpoint')" :disabled="form.async_mode !== 'async'">
                <UInput v-model="form.async_config.status_endpoint" :disabled="form.async_mode !== 'async'" />
              </UFormField>
              <UFormField
                :label="$t('capabilities.form.sseChannel')"
                class="md:col-span-3"
                :disabled="form.async_mode !== 'async'"
              >
                <UInput v-model="form.async_config.sse_channel" :disabled="form.async_mode !== 'async'" />
              </UFormField>
            </div>
          </div>
        </section>

        <section v-else class="space-y-6">
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.sampleRequest')" :description="$t('capabilities.form.jsonHint')">
              <UTextarea v-model="form.samples.requestText" :rows="8" class="font-mono text-xs" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.sampleResponse')" :description="$t('capabilities.form.jsonHint')">
              <UTextarea v-model="form.samples.responseText" :rows="8" class="font-mono text-xs" />
            </UFormField>
          </div>

          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <h3 class="text-base font-semibold">
                {{ $t("capabilities.form.errorCodes") }}
              </h3>
              <UButton icon="i-heroicons-plus" size="xs" variant="soft" @click="addErrorCode">
                {{ $t("capabilities.form.addRow") }}
              </UButton>
            </div>
            <div class="space-y-2">
              <div
                v-for="(error, idx) in form.samples.errors"
                :key="`error-${idx}`"
                class="grid gap-2 rounded border border-gray-200 p-3 dark:border-gray-800 md:grid-cols-3"
              >
                <UInput v-model="error.code" :placeholder="$t('capabilities.form.errorCode')" />
                <UInput v-model="error.message" :placeholder="$t('capabilities.form.errorMessage')" />
                <div class="flex gap-2">
                  <UInput v-model="error.solution" class="flex-1" :placeholder="$t('capabilities.form.errorSolution')" />
                  <UButton icon="i-heroicons-x-mark" size="xs" variant="ghost" color="neutral" @click="removeErrorCode(idx)" />
                </div>
              </div>
              <p v-if="!form.samples.errors.length" class="text-sm text-gray-500">
                {{ $t("capabilities.form.errorEmpty") }}
              </p>
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.demoUrl')">
              <UInput v-model="form.demo.url" placeholder="https://demo.powerx.cloud/template" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.demoHint')">
              <UInput v-model="form.demo.credential_hint" placeholder="使用测试租户 demo-tenant / demo123" />
            </UFormField>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.form.ownerName')" required>
              <UInput v-model="form.owner.name" />
            </UFormField>
            <UFormField :label="$t('capabilities.form.ownerEmail')" required>
              <UInput v-model="form.owner.email" type="email" />
            </UFormField>
          </div>

          <UCard size="lg" :ui="{ body: 'space-y-2' }">
            <template #header>
              <div class="flex items-center gap-2">
                <UIcon name="i-heroicons-document-text" class="text-gray-400" />
                <h3 class="text-base font-semibold">
                  {{ $t("capabilities.form.previewTitle") }}
                </h3>
              </div>
            </template>
            <dl class="grid gap-2 md:grid-cols-2">
              <div>
                <dt class="text-sm text-gray-500">{{ $t("capabilities.form.previewName") }}</dt>
                <dd class="font-medium">{{ form.name.zh || "—" }}</dd>
              </div>
              <div>
                <dt class="text-sm text-gray-500">{{ $t("capabilities.form.previewOwner") }}</dt>
                <dd class="font-medium">{{ form.owner.email || "—" }}</dd>
              </div>
              <div class="md:col-span-2">
                <dt class="text-sm text-gray-500">{{ $t("capabilities.form.previewSummary") }}</dt>
                <dd class="font-medium">{{ form.summary.zh || $t("capabilities.form.previewPlaceholder") }}</dd>
              </div>
            </dl>
          </UCard>
        </section>

        <UAlert
          v-if="validationResult && !validationResult.valid"
          icon="i-heroicons-exclamation-triangle"
          color="rose"
          variant="soft"
        >
          <template #title>
            {{ $t("capabilities.form.validationFailed") }}
          </template>
          <template #description>
            <ul class="list-disc pl-5 text-sm">
              <li v-for="err in validationResult.errors" :key="`${err.field}-${err.message}`">
                <span class="font-semibold">{{ err.field }}:</span> {{ err.message }}
                <span v-if="err.suggestion" class="text-gray-500">({{ err.suggestion }})</span>
              </li>
            </ul>
          </template>
        </UAlert>

        <div class="flex flex-col gap-3 border-t border-gray-200 pt-4 dark:border-gray-800 md:flex-row md:items-center md:justify-between">
          <div class="text-sm text-gray-500">
            <span v-if="savedAt">
              {{ $t("capabilities.form.lastSaved", { time: savedAt }) }}
            </span>
          </div>
          <div class="flex flex-wrap gap-3">
            <UButton variant="ghost" color="neutral" :disabled="currentStep === 0" @click="currentStep = Math.max(0, currentStep - 1)">
              {{ $t("common.previous") }}
            </UButton>
            <UButton variant="ghost" color="neutral" :disabled="currentStep === stepItems.length - 1" @click="currentStep = Math.min(stepItems.length - 1, currentStep + 1)">
              {{ $t("common.next") }}
            </UButton>
            <UButton variant="outline" color="primary" :loading="validating" @click="handleValidate">
              {{ $t("capabilities.form.validateButton") }}
            </UButton>
            <UButton variant="soft" color="primary" :loading="savingDraft" @click="handleSaveDraft">
              {{ $t("capabilities.form.saveDraft") }}
            </UButton>
            <UButton color="primary" :loading="submitting" @click="handleSubmit">
              {{ $t("common.submit") }}
            </UButton>
          </div>
        </div>
      </template>
    </UCard>
  </div>
</template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useDebounceFn } from "@vueuse/core";
import { useI18n, useToast } from "#imports";
import {
  useCapabilityRegistryApi,
  type CapabilityTemplate,
  type CapabilityRegisterPayload,
  type CapabilityValidationResult,
  useCapabilityCatalogApi,
  type CapabilityCatalogEntry,
} from "~/composables/api";
import { useNormalizedColumns } from "~/utils/table";

definePageMeta({
  alias: ["/capabilities/register"],
});

const { t } = useI18n();
const toast = useToast();
const {
  fetchTemplate,
  validateDraft,
  submitDraft,
} = useCapabilityRegistryApi();
const { list: listCatalog } = useCapabilityCatalogApi();

const formOpen = ref(false);
const loadingTemplate = ref(true);
const currentStep = ref(0);
const template = ref<CapabilityTemplate | null>(null);
const validating = ref(false);
const savingDraft = ref(false);
const submitting = ref(false);
const validationResult = ref<CapabilityValidationResult | null>(null);
const savedAt = ref<string | null>(null);
const autoValidate = ref(false);
const draftStorageKey = "powerxplugin::capability-register-draft";
const catalogLoading = ref(false);
const catalog = ref<CapabilityCatalogEntry[]>([]);

const form = reactive(createDefaultForm());

const httpMethods = ["POST", "PUT", "PATCH", "GET", "DELETE"];

const capabilityId = computed(() =>
  buildCapabilityId(
    form.namespace || template.value?.namespace || "",
    form.resource,
    form.action,
  ),
);

const stepItems = computed(() => [
  {
    label: t("capabilities.form.steps.basic"),
    description: t("capabilities.form.steps.basicDesc"),
  },
  {
    label: t("capabilities.form.steps.protocol"),
    description: t("capabilities.form.steps.protocolDesc"),
  },
  {
    label: t("capabilities.form.steps.samples"),
    description: t("capabilities.form.steps.samplesDesc"),
  },
]);

const validationBadge = computed(() => {
  if (!validationResult.value) {
    return { label: t("capabilities.form.validationUnknown"), color: "neutral" };
  }
  if (validationResult.value.valid) {
    return { label: t("capabilities.form.validationPassed"), color: "green" };
  }
  return { label: t("capabilities.form.validationFailedShort"), color: "rose" };
});

const tagsText = computed({
  get() {
    return form.tags.join(", ");
  },
  set(value: string) {
    form.tags = value
      .split(",")
      .map((tag) => tag.trim())
      .filter(Boolean);
  },
});

const tableColumns = useNormalizedColumns([
  { key: "capability_id", label: t("capabilities.list.column.capability") },
  { key: "execution", label: t("capabilities.list.column.execution") },
  { key: "tags", label: t("capabilities.list.column.tags") },
  { key: "checksum", label: t("capabilities.list.column.checksum") },
]);

const catalogRows = computed(() =>
  (catalog.value || []).map((entry) => ({
    capability_id: entry.id,
    version: entry.version,
    descriptor: entry.descriptor,
    tags: entry.tags || [],
    checksum: entry.checksum,
    execution: entry.execution || { mode: "sync" },
  })),
);

onMounted(async () => {
  await Promise.all([loadTemplate(), loadCatalog()]);
  hydrateDraft();
  watchFormForPersist();
});

watch(formOpen, (isOpen) => {
  if (!isOpen) {
    currentStep.value = 0;
  }
});

async function loadTemplate() {
  loadingTemplate.value = true;
  try {
    template.value = await fetchTemplate();
    if (!form.namespace) {
      form.namespace = template.value?.namespace || "";
    }
  } catch (error) {
    console.error("[capabilities] failed to load template", error);
    toast.add({
      title: t("capabilities.form.toast.templateFailed"),
      description: String(error),
      color: "rose",
    });
  } finally {
    loadingTemplate.value = false;
  }
}

async function loadCatalog() {
  catalogLoading.value = true;
  try {
    catalog.value = await listCatalog();
  } catch (error) {
    console.error("[capabilities] failed to load catalog", error);
    toast.add({
      title: t("capabilities.list.toast.loadFailed"),
      description: String(error),
      color: "rose",
    });
  } finally {
    catalogLoading.value = false;
  }
}

function openForm() {
  formOpen.value = true;
}

function closeForm() {
  formOpen.value = false;
}

function createDefaultForm() {
  return {
    namespace: "",
    resource: "",
    action: "",
    name: { zh: "", en: "" },
    summary: { zh: "", en: "" },
    description: { zh: "", en: "" },
    scenario: "",
    sensitivity: "medium",
    tags: [] as string[],
    tenant_scope: "global",
    schemas: { input: "", output: "" },
    protocols: {
      rest: { method: "POST", path: "" },
      grpc: { service: "" },
      workflow: { template: "" },
      agent_stream: { channel: "" },
    },
    samples: {
      requestText: "{\n  \n}",
      responseText: "{\n  \n}",
      errors: [] as Array<{ code: string; message: string; solution?: string }>,
    },
    demo: { url: "", credential_hint: "" },
    owner: { name: "", email: "", slack: "" },
    async_mode: "sync",
    async_config: { callback_url: "", sse_channel: "", status_endpoint: "" },
    draft: true,
    metadata: { source: "web-admin" } as Record<string, string>,
  };
}

function resetFormState() {
  Object.assign(form, createDefaultForm());
  currentStep.value = 0;
  validationResult.value = null;
  autoValidate.value = false;
  savedAt.value = null;
}

function hydrateDraft() {
  if (typeof window === "undefined") return;
  const raw = window.localStorage.getItem(draftStorageKey);
  if (!raw) return;
  try {
    const saved = JSON.parse(raw);
    Object.assign(form, saved);
    savedAt.value = new Date().toLocaleString();
  } catch (error) {
    console.warn("[capabilities] failed to parse draft", error);
  }
}

function persistDraft() {
  if (typeof window === "undefined") return;
  const payload = JSON.stringify(form);
  window.localStorage.setItem(draftStorageKey, payload);
  savedAt.value = new Date().toLocaleString();
}

function clearDraft() {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(draftStorageKey);
  savedAt.value = null;
}

function watchFormForPersist() {
  const debounced = useDebounceFn(persistDraft, 500);
  watch(
    form,
    () => debounced(),
    { deep: true },
  );
  watch(
    () => [form.namespace, form.resource, form.action],
    () => {
      if (autoValidate.value) {
        debouncedValidate();
      }
    },
  );
}

const debouncedValidate = useDebounceFn(async () => {
  await runValidation();
}, 800);

async function handleValidate() {
  await runValidation();
  autoValidate.value = true;
}

async function runValidation() {
  const payload = buildPayload();
  if (!payload) return;
  validating.value = true;
  try {
    const result = await validateDraft(payload);
    validationResult.value = result;
  } catch (error: any) {
    validationResult.value =
      error?.response?._data?.error?.details ??
      error?.response?._data?.data ??
      null;
    toast.add({
      title: t("capabilities.form.toast.validateFailed"),
      description: extractErrorMessage(error),
      color: "rose",
    });
  } finally {
    validating.value = false;
  }
}

async function handleSaveDraft() {
  form.draft = true;
  const payload = buildPayload();
  if (!payload) return;
  savingDraft.value = true;
  try {
    const record = await submitDraft(payload);
    validationResult.value = {
      capability_id: record.capability_id,
      valid: true,
      errors: [],
    };
    persistDraft();
    toast.add({
      title: t("capabilities.form.toast.saved"),
      description: t("capabilities.form.toast.savedDesc"),
      color: "primary",
    });
  } catch (error) {
    handleSubmitError(error);
  } finally {
    savingDraft.value = false;
  }
}

async function handleSubmit() {
  form.draft = false;
  const payload = buildPayload();
  if (!payload) return;
  submitting.value = true;
  try {
    const validation = await validateDraft(payload);
    validationResult.value = validation;
    if (!validation.valid) {
      toast.add({
        title: t("capabilities.form.toast.validateBlock"),
        description: t("capabilities.form.toast.fixBeforeSubmit"),
        color: "rose",
      });
      submitting.value = false;
      autoValidate.value = true;
      return;
    }
    const record = await submitDraft(payload);
    validationResult.value = {
      capability_id: record.capability_id,
      valid: true,
      errors: [],
    };
    autoValidate.value = true;
    toast.add({
      title: t("capabilities.form.toast.submitted"),
      description: t("capabilities.form.toast.submittedDesc", {
        id: record.capability_id,
      }),
      color: "green",
    });
    clearDraft();
    await loadCatalog();
    resetFormState();
    closeForm();
  } catch (error) {
    handleSubmitError(error);
  } finally {
    submitting.value = false;
  }
}

function handleSubmitError(error: any) {
  const details =
    error?.response?._data?.error?.details ??
    error?.response?._data?.data ??
    null;
  if (details?.capability_id) {
    validationResult.value = details;
  }
  toast.add({
    title: t("capabilities.form.toast.submitFailed"),
    description: extractErrorMessage(error),
    color: "rose",
  });
}

function buildPayload(): CapabilityRegisterPayload | null {
  const localErrors: CapabilityValidationResult = {
    capability_id: capabilityId.value || "",
    valid: false,
    errors: [],
  };
  const parseJSON = (value: string, field: string) => {
    if (!value || !value.trim()) return null;
    try {
      return JSON.parse(value);
    } catch (error: any) {
      localErrors.errors.push({
        field,
        message: error?.message || "JSON 解析失败",
      });
      return null;
    }
  };

  const requestPayload = parseJSON(form.samples.requestText, "samples.request");
  const responsePayload = parseJSON(form.samples.responseText, "samples.response");
  if (localErrors.errors.length) {
    validationResult.value = localErrors;
    toast.add({
      title: t("capabilities.form.toast.invalidJson"),
      description: t("capabilities.form.toast.fixJson"),
      color: "rose",
    });
    return null;
  }

  return {
    namespace: form.namespace || template.value?.namespace || "",
    resource: form.resource,
    action: form.action,
    name: { ...form.name },
    summary: { ...form.summary },
    description: { ...form.description },
    scenario: form.scenario,
    sensitivity: form.sensitivity,
    tags: [...form.tags],
    tenant_scope: form.tenant_scope,
    schemas: { ...form.schemas },
    protocols: buildProtocols(),
    samples: {
      request: requestPayload,
      response: responsePayload,
      errors: form.samples.errors.filter(
        (err) => err.code || err.message || err.solution,
      ),
    },
    demo: { ...form.demo },
    owner: { ...form.owner },
    async_mode: form.async_mode,
    async_config: { ...form.async_config },
    draft: form.draft,
    metadata: { ...form.metadata },
  };
}

function buildProtocols() {
  const matrix: Record<string, unknown> = {};
  if (form.protocols.rest.path) {
    matrix.rest = {
      path: form.protocols.rest.path,
      method: form.protocols.rest.method,
    };
  }
  if (form.protocols.grpc.service) {
    matrix.grpc = { service: form.protocols.grpc.service };
  }
  if (form.protocols.workflow.template) {
    matrix.workflow_step = { template: form.protocols.workflow.template };
  }
  if (form.protocols.agent_stream.channel) {
    matrix.agent_stream = { channel: form.protocols.agent_stream.channel };
  }
  return matrix;
}

function addErrorCode() {
  form.samples.errors.push({ code: "", message: "", solution: "" });
}

function removeErrorCode(index: number) {
  form.samples.errors.splice(index, 1);
}

function buildCapabilityId(namespace: string, resource: string, action: string) {
  const clean = (value: string) =>
    value
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9.-]/g, "-")
      .replace(/^-+|[-.]+$/g, "");
  const ns = clean(namespace);
  const res = clean(resource);
  const act = clean(action);
  return [ns, res, act].filter(Boolean).join(".");
}

function extractErrorMessage(error: any) {
  return (
    error?.response?._data?.error?.message ||
    error?.message ||
    t("capabilities.form.toast.genericError")
  );
}
</script>
