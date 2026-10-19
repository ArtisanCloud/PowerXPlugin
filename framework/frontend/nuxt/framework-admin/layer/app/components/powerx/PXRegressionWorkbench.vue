<script setup lang="ts">
import { computed, nextTick, onMounted, ref, toRaw, watch } from 'vue';
import { useI18n } from '#imports';
import * as v from 'valibot';
import type { RegressionClient, CaseDefinition, Run } from '@artisan-cloud/plugin-framework-client';
import type { RegressionCasePresentation } from '../../types/regression';
import { usePowerXRegression } from '../../composables/usePowerXRegression';
import PXRegressionRunDetail from './PXRegressionRunDetail.vue';

const props = defineProps<{
  client: RegressionClient;
  canRead: boolean;
  canInvoke: boolean;
  canManage: boolean;
  identityKey?: string;
  defaultInputs: (definition: CaseDefinition) => Record<string, unknown>;
  describeCase: (definition: CaseDefinition) => RegressionCasePresentation | undefined;
}>();
const { t } = useI18n();
const state = usePowerXRegression(props.client);
const tab = ref('cases');
const showResult = ref(false);
const profile = ref('');
const suite = ref('all');
const filter = ref('');
const selected = ref<string[]>([]);
const inputValues = ref<Record<string, Record<string, any>>>({});
const caseOpen = ref(false);
const viewedCase = ref<CaseDefinition | null>(null);
const preparationOpen = ref(false);
const preparationIds = ref<string[]>([]);
const invalidInputCases = ref<string[]>([]);
const rerunOf = ref<string>();
const definitionTechnicalOpen = ref(false);
const resultView = ref<{ focus: () => void } | null>(null);
const focusAfterPreparation = ref(false);
const selectedProfile = computed(() => state.profiles.value.find(p => p.uuid === profile.value));
const tabItems = computed(() => [
  { label: t('regression.tabs.cases'), value: 'cases', icon: 'i-heroicons-clipboard-document-check' },
  { label: t('regression.tabs.runs'), value: 'runs', icon: 'i-heroicons-clock' },
]);
const profileItems = computed(() => state.profiles.value.map(p => ({ label: t(p.title_key), value: p.uuid })));
const suiteItems = computed(() => [{ label: t('regression.fields.allSuites'), value: 'all' }, ...state.suites.value.map(s => ({ label: t(s.title_key), value: s.uuid }))]);
const available = computed(() => selectedProfile.value ? state.cases.value.filter(c => c.supported_profile_kinds.includes(selectedProfile.value!.kind)) : []);
const visible = computed(() => available.value.filter(c => (suite.value === 'all' || c.suite_uuid === suite.value) && (!filter.value || t(c.title_key).toLowerCase().includes(filter.value.toLowerCase()) || c.tags.some(tag => tag.toLowerCase().includes(filter.value.toLowerCase())))));
const preparingCases = computed(() => preparationIds.value.map(id => state.cases.value.find(c => c.uuid === id)).filter((c): c is CaseDefinition => !!c));
const busy = computed(() => state.runs.value.some(run => ['queued', 'running', 'cancelling'].includes(run.phase) || run.cleanup_state === 'running'));
const visibleSelection = computed(() => visible.value.length > 0 && visible.value.every(item => selected.value.includes(item.uuid)) ? true : selected.value.length ? 'indeterminate' as const : false);
function selectVisible(value: boolean | 'indeterminate') { selected.value = value === true ? visible.value.map(item => item.uuid) : []; }
function preparationError(event: { errors: Array<{ name?: string }> }) { invalidInputCases.value = [...new Set(event.errors.map(error => error.name?.split('.')[0]).filter((id): id is string => !!id))]; }
const status = (value: string) => t('regression.states.' + value);
const color = (value: string) => value === 'passed' ? 'success' : value === 'failed' ? 'error' : value === 'blocked' ? 'warning' : 'neutral';
const latestRuns = computed(() => {
  const latest = new Map<string, Run>();
  for (const run of [...state.runs.value].sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at))) {
    if (run.profile_uuid !== profile.value) continue;
    for (const definition of run.selection_snapshot) if (!latest.has(definition.uuid)) latest.set(definition.uuid, run);
  }
  return latest;
});
function configured(definition: CaseDefinition) {
  const presentation = props.describeCase(definition);
  const properties = definition.input_schema.properties as Record<string, { type?: string }> | undefined;
  if (!presentation || !properties) return false;
  const fields = presentation.fields.map(field => field.key);
  return new Set(fields).size === fields.length && fields.length === Object.keys(properties).length &&
    fields.every(key => properties[key]?.type === 'string');
}
const preparationConfigured = computed(() => preparingCases.value.length === preparationIds.value.length && preparingCases.value.every(configured));
const formSchema = computed(() => {
  const cases: Record<string, any> = {};
  for (const definition of preparingCases.value) {
    const fields: Record<string, any> = {};
    const properties = definition.input_schema.properties as Record<string, { minLength?: number; maxLength?: number; format?: string }>;
    const required = definition.input_schema.required as string[] | undefined;
    for (const field of props.describeCase(definition)?.fields || []) {
      const property = properties[field.key];
      if (!property) continue;
      const rules: any[] = [v.string(t('regression.validation.required', { field: t(field.labelKey) }))];
      if (required?.includes(field.key)) rules.push(v.nonEmpty(t('regression.validation.required', { field: t(field.labelKey) })));
      if (property.minLength) rules.push(v.minLength(property.minLength, t('regression.validation.minLength', { field: t(field.labelKey), count: property.minLength })));
      if (property.maxLength) rules.push(v.maxLength(property.maxLength, t('regression.validation.maxLength', { field: t(field.labelKey), count: property.maxLength })));
      if (property.format === 'uuid') rules.push(v.uuid(t('regression.validation.uuid', { field: t(field.labelKey) })));
      fields[field.key] = v.pipe(...rules as [any, ...any[]]);
    }
    cases[definition.uuid] = v.object(fields);
  }
  return v.object(cases);
});
function initializeInputs(definition: CaseDefinition) {
  if (!inputValues.value[definition.uuid]) inputValues.value[definition.uuid] = structuredClone(props.defaultInputs(definition));
  for (const field of props.describeCase(definition)?.fields || []) {
    if (!Object.hasOwn(inputValues.value[definition.uuid], field.key)) inputValues.value[definition.uuid][field.key] = '';
  }
}
watch(state.cases, definitions => definitions.forEach(initializeInputs));
watch(state.profiles, items => { if (!items.some(p => p.uuid === profile.value)) profile.value = items[0]?.uuid || ''; });
watch(profile, () => { selected.value = []; suite.value = 'all'; filter.value = ''; preparationOpen.value = false; });
watch(visible, definitions => { selected.value = selected.value.filter(id => definitions.some(c => c.uuid === id)); });
watch(tab, (_value, previous) => { if (previous === 'runs') showResult.value = false; });
function clear() {
  state.reset(); tab.value = 'cases'; showResult.value = false; selected.value = []; profile.value = ''; suite.value = 'all'; filter.value = '';
  inputValues.value = {}; caseOpen.value = false; viewedCase.value = null; preparationOpen.value = false; preparationIds.value = []; rerunOf.value = undefined; definitionTechnicalOpen.value = false; focusAfterPreparation.value = false;
}
watch(() => props.canRead, value => { clear(); if (value) void state.load(); });
watch(() => props.identityKey, () => { clear(); if (props.canRead) void state.load(); });
onMounted(() => { if (props.canRead) void state.load(); });
function blurActive() { if (typeof document !== 'undefined' && document.activeElement instanceof HTMLElement) document.activeElement.blur(); }
function viewCase(definition: CaseDefinition) { viewedCase.value = definition; caseOpen.value = true; }
async function prepare(ids: string[], previous?: string) {
  blurActive(); caseOpen.value = false; definitionTechnicalOpen.value = false;
  await nextTick();
  preparationIds.value = [...ids]; invalidInputCases.value = []; rerunOf.value = previous;
  preparingCases.value.forEach(initializeInputs);
  state.error.value = ''; preparationOpen.value = true;
}
function closePreparation() { blurActive(); preparationOpen.value = false; }
async function execute() {
  if (!props.canInvoke || !preparationConfigured.value || !profile.value || busy.value) return;
  await state.run({
    client_request_uuid: crypto.randomUUID(), profile_uuid: profile.value,
    case_uuids: preparationIds.value, inputs: Object.fromEntries(preparationIds.value.map(id => [id, inputValues.value[id]])),
    ...(rerunOf.value ? { rerun_of_run_uuid: rerunOf.value } : {}),
  });
  if (!state.error.value) { focusAfterPreparation.value = true; closePreparation(); selected.value = []; tab.value = 'runs'; showResult.value = true; }
}
function preparationClosed() { if (focusAfterPreparation.value) { focusAfterPreparation.value = false; resultView.value?.focus(); } }
async function openRun(uuid: string) {
  await state.open(uuid);
  if (!state.error.value) { tab.value = 'runs'; showResult.value = true; await nextTick(); resultView.value?.focus(); }
}
async function rerun() {
  if (!state.detail.value) return;
  const run = state.detail.value.run;
  if (!state.profiles.value.some(p => p.uuid === run.profile_uuid)) { state.error.value = 'REGRESSION_PROFILE_UNAVAILABLE'; return; }
  const ids = state.detail.value.case_executions.filter(c => c.verdict !== 'passed').map(c => c.case_uuid);
  if (!ids.length) return;
  for (const id of ids) {
    const snapshot = run.input_snapshot[id];
    if (!snapshot || typeof snapshot !== 'object' || Array.isArray(snapshot)) { state.error.value = 'REGRESSION_INVALID_REQUEST'; return; }
    inputValues.value[id] = structuredClone(toRaw(snapshot)) as Record<string, unknown>;
  }
  profile.value = run.profile_uuid;
  await nextTick();
  await prepare(ids, run.uuid);
}
async function showDefinitionTechnical() { blurActive(); caseOpen.value = false; await nextTick(); definitionTechnicalOpen.value = true; }
</script>

<template>
  <section v-if="canRead" class="mx-auto max-w-7xl space-y-5 p-4 text-default" data-testid="regression-workbench">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div><h1 class="text-2xl font-semibold text-highlighted">{{ t('regression.title') }}</h1><p class="mt-1 text-sm text-muted">{{ t('regression.description') }}</p></div>
      <UButton color="neutral" variant="outline" icon="i-heroicons-arrow-path" :label="t('regression.actions.refresh')" :loading="state.loading.value" @click="state.load" />
    </header>
    <UAlert v-if="state.error.value && !preparationOpen" color="error" :title="t('regression.errors.' + state.error.value)" data-testid="regression-error" />
    <UTabs v-model="tab" :items="tabItems" :content="false" variant="link" class="w-full" :aria-label="t('regression.title')" />

    <section v-if="tab === 'cases'" class="space-y-4" data-testid="regression-catalog">
      <div class="flex flex-wrap items-end justify-between gap-3">
        <div><h2 class="text-lg font-semibold text-highlighted" data-testid="regression-case-count">{{ state.loading.value && !selectedProfile ? t('regression.states.loading') : selectedProfile ? t('regression.overview.available', { count: available.length }) : t('regression.overview.unavailable') }}</h2><p v-if="selectedProfile" class="mt-1 text-sm text-muted">{{ t('regression.catalog.environment', { name: t(selectedProfile.title_key) }) }}</p></div>
        <UFormField v-if="state.profiles.value.length > 1" :label="t('regression.fields.profile')"><USelect v-model="profile" :items="profileItems" class="w-64" /></UFormField>
      </div>
      <UAlert v-if="busy" color="info" :title="t('regression.overview.busy')"><template #actions><UButton color="neutral" variant="outline" :label="t('regression.actions.viewRuns')" @click="tab = 'runs'; showResult = false" /></template></UAlert>
      <div v-if="available.length > 1 || filter || suite !== 'all'" class="flex flex-wrap items-end gap-3">
        <UFormField v-if="state.suites.value.length > 1" :label="t('regression.fields.suite')"><USelect v-model="suite" :items="suiteItems" class="w-48" /></UFormField>
        <UFormField :label="t('regression.fields.filter')"><UInput v-model="filter" icon="i-heroicons-magnifying-glass" class="w-64" /></UFormField>
        <span class="pb-2 text-sm text-muted">{{ t('regression.overview.showing', { count: visible.length, total: available.length }) }}</span>
      </div>
      <div v-if="canInvoke && selected.length" class="flex items-center justify-between gap-3 rounded-lg bg-primary/10 p-3" data-testid="regression-batch-actions"><span class="text-sm">{{ t('regression.overview.selected', { count: selected.length }) }}</span><UButton :label="t('regression.actions.runSelected', { count: selected.length })" :disabled="busy || state.loading.value" data-testid="regression-run-selected" @click="prepare(selected)" /></div>
      <div class="overflow-x-auto rounded-xl border border-default">
        <table class="w-full text-left text-sm">
          <thead class="bg-elevated/60 text-muted"><tr><th v-if="canInvoke" class="w-12 p-4"><UCheckbox :model-value="visibleSelection" :disabled="!visible.length" :aria-label="t('regression.actions.selectVisible')" data-testid="regression-select-visible" @update:model-value="selectVisible" /></th><th scope="col" class="p-4 font-medium">{{ t('regression.catalog.case') }}</th><th scope="col" class="p-4 font-medium">{{ t('regression.catalog.purpose') }}</th><th scope="col" class="p-4 font-medium">{{ t('regression.catalog.latest') }}</th><th scope="col" class="p-4 font-medium">{{ t('regression.catalog.actions') }}</th></tr></thead>
          <tbody>
            <tr v-for="item in visible" :key="item.uuid" class="border-t border-default align-top hover:bg-elevated/30" :data-testid="'regression-case-' + item.key">
              <td v-if="canInvoke" class="p-4"><UCheckbox :model-value="selected.includes(item.uuid)" :aria-label="t('regression.actions.selectCase', { name: t(item.title_key) })" @update:model-value="value => { selected = value ? [...selected, item.uuid] : selected.filter(id => id !== item.uuid) }" /></td>
              <td class="min-w-44 p-4 font-medium text-highlighted">{{ t(item.title_key) }}</td>
              <td class="min-w-64 max-w-md p-4 text-muted">{{ t(item.description_key) }}</td>
              <td class="min-w-32 p-4"><UBadge v-if="latestRuns.has(item.uuid)" :color="color(latestRuns.get(item.uuid)!.verdict)" variant="subtle" data-testid="regression-latest-run">{{ status(latestRuns.get(item.uuid)!.phase === 'completed' ? latestRuns.get(item.uuid)!.verdict : latestRuns.get(item.uuid)!.phase) }}</UBadge><span v-else class="text-muted">{{ t('regression.catalog.notRun') }}</span></td>
              <td class="p-4"><div class="flex flex-wrap gap-2"><UButton color="neutral" variant="outline" :label="t('regression.actions.viewCase')" @click="viewCase(item)" /><UButton v-if="canInvoke" :label="t('regression.actions.runCase')" :disabled="busy || state.loading.value || !profile" data-testid="regression-run-case" @click="prepare([item.uuid])" /></div></td>
            </tr>
            <tr v-if="!visible.length"><td :colspan="canInvoke ? 5 : 4" class="p-8 text-center text-muted">{{ state.loading.value ? t('regression.states.loading') : state.error.value ? t('regression.overview.unavailable') : t('regression.states.empty') }}</td></tr>
          </tbody>
        </table>
      </div>
      <p class="text-xs text-muted">{{ t('regression.catalog.historyScope') }}</p>
    </section>

    <section v-else-if="!showResult || !state.detail.value" class="space-y-4" data-testid="regression-history">
      <div><h2 class="text-lg font-semibold text-highlighted">{{ t('regression.tabs.runs') }}</h2><p class="mt-1 text-sm text-muted">{{ t('regression.history.description') }}</p></div>
      <div v-if="!state.runs.value.length" class="rounded-xl border border-default p-10 text-center"><p class="font-medium text-highlighted">{{ t('regression.states.noRuns') }}</p><p class="mt-2 text-sm text-muted">{{ t('regression.history.emptyHelp') }}</p><UButton class="mt-4" color="neutral" variant="outline" :label="t('regression.actions.backToCases')" @click="tab = 'cases'" /></div>
      <div v-else class="overflow-x-auto rounded-xl border border-default"><table class="w-full text-left text-sm"><thead class="bg-elevated/60 text-muted"><tr><th v-for="key in ['time', 'cases', 'result', 'cleanup', 'actions']" :key="key" scope="col" class="p-4 font-medium">{{ t('regression.history.' + key) }}</th></tr></thead><tbody><tr v-for="run in state.runs.value" :key="run.uuid" class="border-t border-default align-top"><td class="whitespace-nowrap p-4 text-muted">{{ new Date(run.created_at).toLocaleString() }}</td><td class="min-w-52 p-4"><p class="font-medium text-highlighted">{{ run.selection_snapshot.map(definition => t(definition.title_key)).join(' · ') }}</p><p class="mt-1 text-xs text-muted">{{ t(run.profile_snapshot.title_key) }} · {{ t('regression.results.total', { count: run.selection_snapshot.length }) }}</p></td><td class="p-4"><UBadge :color="color(run.verdict)" variant="subtle">{{ status(run.phase === 'completed' ? run.verdict : run.phase) }}</UBadge></td><td class="p-4">{{ status(run.cleanup_state) }}</td><td class="p-4"><UButton color="neutral" variant="outline" :label="t('regression.actions.viewResult')" @click="openRun(run.uuid)" /></td></tr></tbody></table></div>
    </section>
    <PXRegressionRunDetail v-else ref="resultView" :detail="state.detail.value" :describe-case="describeCase" :can-invoke="canInvoke" :can-manage="canManage" :busy="busy || state.loading.value" :disconnected="state.connection.value === 'error'" @back="showResult = false" @cancel="state.cancel" @cleanup="state.cleanup" @rerun="rerun" @report="state.download" @reconnect="state.reconnect" />

    <USlideover v-model:open="caseOpen" :title="viewedCase ? t(viewedCase.title_key) : t('regression.actions.viewCase')" :description="t('regression.caseDetail.description')" :ui="{ content: 'max-w-xl' }">
      <template #body><div v-if="viewedCase" class="space-y-6" data-testid="regression-case-detail">
        <section><h3 class="font-semibold text-highlighted">{{ t('regression.caseDetail.purpose') }}</h3><p class="mt-2 text-sm text-muted">{{ t(viewedCase.description_key) }}</p></section>
        <section><h3 class="font-semibold text-highlighted">{{ t('regression.caseDetail.steps', { count: viewedCase.steps.length }) }}</h3><ol class="mt-3 space-y-3"><li v-for="(step, index) in viewedCase.steps" :key="step.uuid" class="flex gap-3 text-sm"><span class="flex size-6 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">{{ index + 1 }}</span><span>{{ t(step.title_key) }}</span></li></ol></section>
        <template v-if="describeCase(viewedCase)"><section><h3 class="font-semibold text-highlighted">{{ t('regression.caseDetail.expected') }}</h3><ul class="mt-2 list-disc space-y-2 pl-5 text-sm text-muted"><li v-for="key in describeCase(viewedCase)!.expectationKeys" :key="key">{{ t(key) }}</li></ul></section><UAlert color="info" variant="subtle" :title="t('regression.caseDetail.impact')" :description="t(describeCase(viewedCase)!.impactKey)" /></template>
        <UAlert v-else color="error" :title="t('regression.errors.REGRESSION_PRESENTATION_MISSING')" />
        <UButton color="neutral" variant="link" :label="t('regression.actions.technical')" class="px-0" @click="showDefinitionTechnical" />
      </div></template>
      <template #footer><div class="flex w-full justify-end gap-2"><UButton color="neutral" variant="outline" :label="t('regression.actions.close')" @click="blurActive(); caseOpen = false" /><UButton v-if="canInvoke && viewedCase" :label="t('regression.actions.runCase')" :disabled="busy || state.loading.value" @click="prepare([viewedCase.uuid])" /></div></template>
    </USlideover>

    <UModal v-model:open="preparationOpen" :title="t('regression.preparation.title', { count: preparingCases.length })" :description="t('regression.preparation.description')" :dismissible="!state.loading.value" :close="!state.loading.value" :ui="{ content: 'max-w-2xl' }" @after:leave="preparationClosed">
      <template #body>
        <div class="space-y-5" data-testid="regression-preparation">
          <p v-if="selectedProfile" class="text-sm font-medium text-highlighted">{{ t('regression.catalog.environment', { name: t(selectedProfile.title_key) }) }}</p>
          <p v-if="selectedProfile" class="text-sm text-muted">{{ t('regression.profileDescriptions.' + selectedProfile.kind) }}</p>
          <UAlert v-if="state.error.value" color="error" :title="t('regression.errors.' + state.error.value)" data-testid="regression-error" />
          <UAlert v-if="!preparationConfigured" color="error" :title="t('regression.errors.REGRESSION_PRESENTATION_MISSING')" />
          <UForm v-if="preparationConfigured" id="regression-run-form" :state="inputValues" :schema="formSchema" class="space-y-6" @submit="execute" @error="preparationError">
            <section v-for="definition in preparingCases" :key="definition.uuid" class="space-y-4">
              <div><h3 class="font-semibold text-highlighted">{{ t(definition.title_key) }}</h3><p class="mt-1 text-sm text-muted">{{ t(definition.description_key) }}</p></div>
              <UAlert color="info" variant="subtle" :title="t('regression.caseDetail.impact')" :description="t(describeCase(definition)!.impactKey)" />
              <details :open="preparingCases.length === 1 || invalidInputCases.includes(definition.uuid)" class="rounded-lg border border-default p-3" :data-testid="'regression-inputs-' + definition.key">
              <summary class="cursor-pointer text-sm font-medium text-highlighted">{{ t('regression.preparation.reviewInputs') }}</summary>
              <div class="mt-4 grid gap-4 sm:grid-cols-2">
              <UFormField v-for="field in describeCase(definition)!.fields" :key="field.key" :class="field.control === 'textarea' ? 'sm:col-span-2' : ''" :name="definition.uuid + '.' + field.key" :label="t(field.labelKey)" :help="field.helpKey ? t(field.helpKey) : undefined" :required="(definition.input_schema.required as string[] | undefined)?.includes(field.key)">
                <UTextarea v-if="field.control === 'textarea'" v-model="inputValues[definition.uuid][field.key]" :rows="3" class="w-full" :disabled="state.loading.value" />
                <UInput v-else v-model="inputValues[definition.uuid][field.key]" class="w-full" :disabled="state.loading.value" />
              </UFormField>
              </div>
              </details>
            </section>
          </UForm>
        </div>
      </template>
      <template #footer><div class="flex w-full justify-end gap-2"><UButton color="neutral" variant="outline" :disabled="state.loading.value" :label="t('regression.actions.cancelPreparation')" @click="closePreparation" /><UButton v-if="canInvoke" type="submit" form="regression-run-form" :label="t('regression.actions.start')" :disabled="!preparationConfigured || busy || !profile" :loading="state.loading.value" data-testid="regression-run" /></div></template>
    </UModal>
    <UModal v-model:open="definitionTechnicalOpen" :title="t('regression.actions.technical')" :description="t('regression.technical.definition')" :ui="{ content: 'max-w-3xl' }"><template #body><pre class="overflow-auto rounded-lg bg-elevated p-3 text-xs">{{ JSON.stringify({ definition: viewedCase, profile: selectedProfile }, null, 2) }}</pre></template><template #footer><UButton color="neutral" variant="outline" :label="t('regression.actions.close')" @click="blurActive(); definitionTechnicalOpen = false" /></template></UModal>
  </section>
</template>
