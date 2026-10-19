<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from '#imports';
import type { AssertionResult, CaseDefinition, CaseExecution, Evidence, FixtureResource, RunDetail } from '@artisan-cloud/plugin-framework-client';
import type { RegressionCasePresentation } from '../../types/regression';
import { regressionComparison } from '../../composables/regressionComparison';

const props = defineProps<{
  detail: RunDetail; canInvoke: boolean; canManage: boolean; busy: boolean; disconnected: boolean;
  describeCase: (definition: CaseDefinition) => RegressionCasePresentation | undefined;
}>();
const emit = defineEmits<{ back: []; cancel: []; cleanup: []; rerun: []; report: []; reconnect: [] }>();
const { t, locale } = useI18n();
type Section = 'overview' | 'checks' | 'resources' | 'diagnostics';
const section = ref<Section>('overview');
const selectedCase = ref('');
const comparisonId = ref('');
const comparisonOpen = ref(false);
const evidenceOpen = ref(false);
const viewedEvidence = ref<Evidence[]>([]);
const technicalOpen = ref(false);
const heading = ref<HTMLElement | null>(null);
defineExpose({ focus: () => heading.value?.focus() });
const status = (value: string) => t('regression.states.' + value);
const color = (value: string) => value === 'passed' || value === 'succeeded' ? 'success' : value === 'failed' ? 'error' : value === 'blocked' ? 'warning' : 'neutral';
const assertions = (result: CaseExecution) => result.steps.flatMap(step => step.assertions);
const current = computed(() => props.detail.case_executions.find(item => item.uuid === selectedCase.value));
const currentChecks = computed(() => current.value ? assertions(current.value) : []);
const selectedAssertion = computed(() => currentChecks.value.find(item => item.uuid === comparisonId.value));
const active = computed(() => ['queued', 'running', 'cancelling'].includes(props.detail.run.phase));
const headline = computed(() => t('regression.results.headlines.' + (props.detail.run.phase === 'completed' ? props.detail.run.verdict : props.detail.run.phase)));
const allChecks = computed(() => props.detail.case_executions.flatMap(assertions));
const requiredCount = computed(() => props.detail.case_executions.reduce((total, result) => total + result.definition_snapshot.required_assertion_keys.length, 0));
const passedChecks = computed(() => allChecks.value.filter(item => item.required && item.status === 'passed').length);
const allResources = computed(() => props.detail.case_executions.flatMap(item => item.fixture_resources));
const cleanCount = computed(() => allResources.value.filter(item => item.cleanup_state === 'succeeded').length);
const counts = computed(() => ['passed', 'failed', 'blocked', 'pending'].map(verdict => ({ verdict, count: props.detail.case_executions.filter(item => item.verdict === verdict).length })).filter(item => item.count));
const rerunnable = computed(() => !active.value && props.detail.case_executions.some(item => item.verdict !== 'passed' && item.verdict !== 'pending'));
const firstFailure = computed(() => {
  if (['cancelling', 'cancelled'].includes(props.detail.run.phase)) return undefined;
  for (const result of props.detail.case_executions) {
    const step = result.steps.find(item => item.status === 'failed' || item.status === 'blocked');
    if (step) return { result, step, assertion: step.assertions.find(item => item.status === 'failed' || item.status === 'blocked') };
    if (result.missing_assertion_keys.length) return { result, step: undefined, assertion: undefined };
  }
});
const failureLocation = computed(() => firstFailure.value ? firstFailure.value.step ? t('regression.results.failureLocation', { name: t(firstFailure.value.result.definition_snapshot.title_key), step: t(firstFailure.value.step.title_key) }) : t('regression.errors.REGRESSION_ASSERTION_MISSING') : '');
const sectionItems = computed(() => [
  { label: t('regression.resultSections.overview'), value: 'overview', icon: 'i-heroicons-chart-bar-square' },
  { label: t('regression.resultSections.checks'), value: 'checks', icon: 'i-heroicons-clipboard-document-check' },
  { label: t('regression.resultSections.resources'), value: 'resources', icon: 'i-heroicons-circle-stack' },
  { label: t('regression.resultSections.diagnostics'), value: 'diagnostics', icon: 'i-heroicons-command-line' },
]);
const caseItems = computed(() => props.detail.case_executions.map(item => ({ value: item.uuid, label: t(item.definition_snapshot.title_key) })));
const resourceGaps = computed(() => current.value?.fixture_resources.some(item => !item.display_name?.trim() || !item.role_key || item.cleanup_state === 'succeeded' && !item.cleanup_evidence?.length));
const missingChecks = computed(() => props.detail.case_executions.some(item => item.missing_assertion_keys.length));
watch(() => props.detail.run.uuid, () => { section.value = 'overview'; selectedCase.value = ''; comparisonId.value = ''; comparisonOpen.value = false; evidenceOpen.value = false; technicalOpen.value = false; }, { immediate: true });
watch(() => props.detail.case_executions.map(item => item.uuid), ids => { if (!ids.includes(selectedCase.value)) selectedCase.value = props.detail.case_executions.find(item => item.verdict === 'failed' || item.verdict === 'blocked')?.uuid || ids[0] || ''; }, { immediate: true });
function comparison(assertion: AssertionResult) {
  if (!current.value) return null;
  return regressionComparison(assertion, props.describeCase(current.value.definition_snapshot)?.results?.[assertion.key], t);
}
const comparedRows = computed(() => selectedAssertion.value ? comparison(selectedAssertion.value) : null);
function inspect(result: CaseExecution) { selectedCase.value = result.uuid; section.value = 'checks'; }
function inspectCleanup() {
  const result = props.detail.case_executions.find(item => item.cleanup_state === 'failed');
  if (result) selectedCase.value = result.uuid;
  section.value = 'resources';
}
function openComparison(assertion: AssertionResult) { comparisonId.value = assertion.uuid; comparisonOpen.value = true; }
function locateFailure() {
  if (!firstFailure.value) return;
  selectedCase.value = firstFailure.value.result.uuid;
  section.value = firstFailure.value.assertion ? 'checks' : 'diagnostics';
  if (firstFailure.value.assertion) openComparison(firstFailure.value.assertion);
}
function showEvidence(evidence: Evidence[]) { viewedEvidence.value = evidence; evidenceOpen.value = true; }
function blurActive() { if (typeof document !== 'undefined' && document.activeElement instanceof HTMLElement) document.activeElement.blur(); }
function closeOverlays() { blurActive(); comparisonOpen.value = false; evidenceOpen.value = false; technicalOpen.value = false; }
function formatTime(value: string | null | undefined) {
  return value ? new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'medium' }).format(new Date(value)) : t('regression.states.unrecorded');
}
function duration(record: { started_at: string | null; finished_at: string | null }) {
  return record.started_at && record.finished_at ? t('regression.timing.milliseconds', { value: Date.parse(record.finished_at) - Date.parse(record.started_at) }) : t('regression.timing.incomplete');
}
function progress(result: CaseExecution) { return t('regression.resultValues.ratio', { passed: assertions(result).filter(item => item.required && item.status === 'passed').length, total: result.definition_snapshot.required_assertion_keys.length }); }
function source(key: string) { const value = props.detail.run.source_snapshot[key]; return typeof value === 'string' && value.trim() ? value : t('regression.states.unrecorded'); }
function parentName(resource: FixtureResource) {
  if (!resource.parent_resource_uuid) return t('regression.resultValues.noParent');
  return current.value?.fixture_resources.find(item => item.resource_uuid === resource.parent_resource_uuid)?.display_name?.trim() || t('regression.states.unrecorded');
}
function stepName(assertion: AssertionResult) { return current.value?.steps.find(item => item.assertions.some(check => check.uuid === assertion.uuid))?.title_key; }
</script>

<template>
  <section class="min-w-0 space-y-4" data-testid="regression-detail" :data-run-uuid="detail.run.uuid">
    <UButton color="neutral" variant="link" icon="i-heroicons-arrow-left" :label="t('regression.actions.backToRuns')" class="px-0" @click="emit('back')" />
    <header class="rounded-xl border border-default bg-elevated/30 p-4 sm:p-5 space-y-4">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0"><h2 ref="heading" tabindex="-1" class="text-xl font-semibold text-highlighted outline-none">{{ headline }}</h2><p class="mt-1 text-sm text-muted">{{ formatTime(detail.run.created_at) }} · {{ t(detail.run.profile_snapshot.title_key) }}</p></div>
        <div class="flex flex-wrap gap-2">
          <UButton v-if="canManage && active" color="neutral" variant="outline" :label="t('regression.actions.cancel')" @click="emit('cancel')" />
          <UButton v-if="canInvoke && rerunnable" :disabled="busy" :label="t('regression.actions.rerun')" @click="emit('rerun')" />
          <UButton color="neutral" variant="outline" icon="i-heroicons-arrow-down-tray" :label="t('regression.actions.report')" @click="emit('report')" />
        </div>
      </div>
      <span class="sr-only" data-testid="regression-verdict">{{ status(detail.run.verdict) }}</span>
      <dl class="grid grid-cols-2 gap-3 xl:grid-cols-4 text-sm">
        <div data-testid="regression-result-counts"><dt class="text-muted">{{ t('regression.resultLabels.cases') }}</dt><dd class="mt-1 font-medium text-highlighted">{{ t('regression.results.total', { count: detail.case_executions.length }) }}</dd><dd class="mt-1 text-xs text-muted"><span v-for="item in counts" :key="item.verdict" class="mr-2">{{ t('regression.results.count', { status: status(item.verdict), count: item.count }) }}</span></dd></div>
        <div><dt class="text-muted">{{ t('regression.resultLabels.checks') }}</dt><dd class="mt-1 text-lg font-semibold text-highlighted" data-testid="regression-required-count">{{ t('regression.resultValues.ratio', { passed: passedChecks, total: requiredCount }) }}</dd></div>
        <div><dt class="text-muted">{{ t('regression.fields.duration') }}</dt><dd class="mt-1 font-medium text-highlighted">{{ duration(detail.run) }}</dd></div>
        <div data-testid="regression-cleanup"><dt class="text-muted">{{ t('regression.fields.cleanup') }}</dt><dd class="mt-1 font-medium" :class="detail.run.cleanup_state === 'failed' ? 'text-error' : 'text-highlighted'">{{ status(detail.run.cleanup_state) }}</dd><dd class="mt-1 text-xs text-muted">{{ t('regression.resultLabels.resourcesCleaned', { passed: cleanCount, total: allResources.length }) }}</dd></div>
      </dl>
    </header>
    <UAlert v-if="failureLocation" color="error" variant="subtle" :title="failureLocation" data-testid="regression-failure-location"><template #actions><UButton color="error" variant="outline" :label="t('regression.resultActions.locateFailure')" @click="locateFailure" /></template></UAlert>
    <UAlert v-if="missingChecks && !failureLocation" color="error" :title="t('regression.errors.REGRESSION_ASSERTION_MISSING')" />
    <UAlert v-if="detail.run.reason_code" :color="detail.run.phase === 'cancelled' ? 'neutral' : 'error'" :title="t('regression.errors.' + detail.run.reason_code)" />
    <UAlert v-if="detail.run.cleanup_state === 'failed'" color="error" :title="t('regression.errors.REGRESSION_CLEANUP_FAILED')"><template #actions><UButton v-if="canManage" color="error" :label="t('regression.actions.cleanup')" @click="emit('cleanup')" /><UButton color="neutral" variant="outline" :label="t('regression.resultActions.inspectResources')" @click="inspectCleanup" /></template></UAlert>
    <UAlert v-if="disconnected" color="warning" :title="t('regression.errors.REGRESSION_STREAM_INTERRUPTED')"><template #actions><UButton :label="t('regression.actions.reconnect')" @click="emit('reconnect')" /></template></UAlert>

    <div class="min-w-0 lg:grid lg:grid-cols-[190px_minmax(0,1fr)] lg:gap-6" data-testid="regression-result-layout">
      <nav class="hidden lg:block" :aria-label="t('regression.resultLabels.sections')" data-testid="regression-result-nav-desktop">
        <UTabs v-model="section" :items="sectionItems" orientation="vertical" :content="false" variant="pill" :ui="{ list: 'w-full', trigger: 'justify-start' }" />
      </nav>
      <nav class="mb-4 overflow-x-auto lg:hidden" :aria-label="t('regression.resultLabels.sections')" data-testid="regression-result-nav-mobile">
        <UTabs v-model="section" :items="sectionItems" :content="false" variant="link" :ui="{ list: 'w-max min-w-full', trigger: 'shrink-0' }" />
      </nav>
      <div class="min-w-0 rounded-xl border border-default p-4 sm:p-5 space-y-5" data-testid="regression-result-content">
        <template v-if="section === 'overview'">
          <div><h3 class="text-lg font-semibold text-highlighted">{{ t('regression.resultSections.overview') }}</h3><p class="mt-1 text-sm text-muted">{{ t('regression.resultHelp.overview') }}</p></div>
          <div class="divide-y divide-default" data-testid="regression-case-results">
            <article v-for="result in detail.case_executions" :key="result.uuid" class="grid grid-cols-2 gap-3 py-4 md:grid-cols-[minmax(0,2fr)_auto_auto_auto] md:items-center" :data-testid="'regression-case-result-' + result.definition_snapshot.key">
              <div class="col-span-2 min-w-0 md:col-span-1"><h4 class="font-medium text-highlighted">{{ t(result.definition_snapshot.title_key) }}</h4><p class="mt-1 text-sm text-muted">{{ t(result.definition_snapshot.description_key) }}</p></div>
              <div class="text-sm"><UBadge :color="color(result.verdict)" variant="subtle">{{ status(result.verdict) }}</UBadge><p class="mt-1 text-xs text-muted">{{ t('regression.resultLabels.checks') }} {{ progress(result) }}</p></div>
              <div class="text-sm text-muted"><p>{{ duration(result) }}</p><p class="mt-1 text-xs">{{ t('regression.fields.cleanup') }}: {{ status(result.cleanup_state) }}</p></div>
              <UButton color="neutral" variant="outline" class="col-span-2 justify-center md:col-span-1" :label="t('regression.resultActions.inspectChecks')" @click="inspect(result)" />
            </article>
          </div>
        </template>
        <template v-else-if="current">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div><h3 class="text-lg font-semibold text-highlighted">{{ t('regression.resultSections.' + section) }}</h3><p class="mt-1 text-sm text-muted" data-testid="regression-selected-case">{{ t(current.definition_snapshot.title_key) }}</p></div>
            <UFormField v-if="caseItems.length > 1" :label="t('regression.resultLabels.currentCase')"><USelect v-model="selectedCase" :items="caseItems" class="w-64 max-w-full" :aria-label="t('regression.resultLabels.currentCase')" /></UFormField>
          </div>
          <template v-if="section === 'checks'">
            <p class="text-sm text-muted">{{ t('regression.resultHelp.checks') }}</p>
            <div v-if="!currentChecks.length" class="rounded-lg bg-elevated/50 p-5 text-sm text-muted">{{ t('regression.resultHelp.noChecks') }}</div>
            <article v-for="assertion in currentChecks" :key="assertion.uuid" class="rounded-lg border border-default p-4 space-y-3" :class="assertion.status === 'failed' ? 'border-error/60' : ''" :data-testid="'regression-assertion-' + assertion.key">
              <div class="flex flex-wrap items-start justify-between gap-2"><div><h4 class="font-medium text-highlighted">{{ t(assertion.title_key) }}</h4><p v-if="stepName(assertion)" class="mt-1 text-xs text-muted">{{ t(stepName(assertion)!) }}</p></div><UBadge :color="color(assertion.status)" variant="subtle">{{ status(assertion.status) }}</UBadge></div>
              <UAlert v-if="comparison(assertion) === null" color="warning" :title="t('regression.resultHelp.presentationMissing')" />
              <UAlert v-if="comparison(assertion)?.some(row => !row.recorded)" color="warning" :title="t('regression.resultHelp.valueMissing')" />
              <div v-if="comparison(assertion) !== null && comparison(assertion)!.length <= 4" class="space-y-2">
                <div class="hidden grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)] gap-3 text-xs text-muted sm:grid"><span>{{ t('regression.resultLabels.field') }}</span><span>{{ t('regression.fields.expected') }}</span><span>{{ t('regression.fields.actual') }}</span></div>
                <div v-for="(row, index) in comparison(assertion)!" :key="index" class="grid grid-cols-1 gap-2 rounded-md bg-elevated/40 p-3 text-sm sm:grid-cols-3 sm:gap-3" :class="!row.matches ? 'ring-1 ring-error/40' : ''" data-testid="regression-check-comparison">
                  <span class="font-medium text-highlighted">{{ row.label }}</span><div><span class="mr-2 text-xs text-muted sm:hidden">{{ t('regression.fields.expected') }}</span><span class="whitespace-pre-wrap break-words">{{ row.expected }}</span></div><div><span class="mr-2 text-xs text-muted sm:hidden">{{ t('regression.fields.actual') }}</span><span class="whitespace-pre-wrap break-words">{{ row.actual }}</span></div>
                </div>
              </div>
              <div v-else-if="comparison(assertion) !== null" class="grid gap-3 text-sm sm:grid-cols-2"><p><span class="text-muted">{{ t('regression.fields.expected') }}: </span>{{ t('regression.resultLabels.expectedFields', { total: comparison(assertion)!.length }) }}</p><p><span class="text-muted">{{ t('regression.fields.actual') }}: </span>{{ t('regression.resultLabels.matchingFields', { passed: comparison(assertion)!.filter(row => row.matches).length, total: comparison(assertion)!.length }) }}</p></div>
              <div class="flex flex-wrap gap-2"><UButton color="neutral" variant="outline" size="sm" :label="t('regression.resultActions.compare')" @click="openComparison(assertion)" /><UButton color="neutral" variant="ghost" size="sm" :label="t('regression.resultActions.evidence')" @click="showEvidence(assertion.evidence)" /></div>
            </article>
          </template>
          <template v-else-if="section === 'resources'">
            <p class="text-sm text-muted">{{ t('regression.resultHelp.resources') }}</p>
            <UAlert v-if="resourceGaps" color="warning" :title="t('regression.resultHelp.resourceGaps')" />
            <div v-if="!current.fixture_resources.length" class="rounded-lg bg-elevated/50 p-5 text-sm text-muted">{{ t('regression.resultHelp.noResources') }}</div>
            <div v-else class="divide-y divide-default" data-testid="regression-resources">
              <article v-for="resource in current.fixture_resources" :key="resource.uuid" class="grid gap-3 py-4 md:grid-cols-[minmax(0,2fr)_minmax(0,1.5fr)_minmax(0,1.5fr)_auto] md:items-start" data-testid="regression-resource-row">
                <div class="min-w-0"><p class="text-xs text-muted">{{ resource.role_key ? t(resource.role_key) : t('regression.states.unrecorded') }}</p><h4 class="mt-1 break-words text-sm font-medium text-highlighted">{{ resource.display_name?.trim() || t('regression.states.unrecorded') }}</h4></div>
                <div class="min-w-0 text-sm"><p class="text-xs text-muted">{{ t('regression.resultLabels.source') }}</p><p class="mt-1 break-words">{{ parentName(resource) }}</p></div>
                <div class="text-sm"><UBadge :color="color(resource.cleanup_state)" variant="subtle">{{ status(resource.cleanup_state) }}</UBadge><p class="mt-1 text-xs text-muted">{{ formatTime(resource.cleaned_at) }}</p><p class="mt-1 text-xs text-muted">{{ resource.cleanup_evidence?.length ? t('regression.resultLabels.readbackRecorded', { count: resource.cleanup_evidence.length }) : t('regression.resultLabels.readbackMissing') }}</p></div>
                <UButton v-if="resource.cleanup_evidence?.length" color="neutral" variant="outline" size="sm" :label="t('regression.resultActions.cleanupEvidence')" @click="showEvidence(resource.cleanup_evidence)" />
              </article>
            </div>
          </template>
          <template v-else-if="section === 'diagnostics'">
            <ol class="space-y-3" data-testid="regression-result-steps">
              <li v-for="(definition, index) in current.definition_snapshot.steps" :key="definition.uuid" class="rounded-lg border border-default p-4 space-y-2">
                <div class="flex items-start justify-between gap-3"><h4 class="font-medium text-highlighted">{{ index + 1 }}. {{ t(definition.title_key) }}</h4><UBadge :color="color(current.steps.find(step => step.step_uuid === definition.uuid)?.status || 'pending')" variant="subtle">{{ status(current.steps.find(step => step.step_uuid === definition.uuid)?.status || 'pending') }}</UBadge></div>
                <template v-for="step in current.steps.filter(item => item.step_uuid === definition.uuid)" :key="step.uuid"><dl class="grid gap-2 text-xs text-muted sm:grid-cols-3"><div><dt>{{ t('regression.fields.started') }}</dt><dd class="mt-1">{{ formatTime(step.started_at) }}</dd></div><div><dt>{{ t('regression.resultLabels.finished') }}</dt><dd class="mt-1">{{ formatTime(step.finished_at) }}</dd></div><div><dt>{{ t('regression.fields.duration') }}</dt><dd class="mt-1">{{ duration(step) }}</dd></div></dl></template>
              </li>
            </ol>
            <dl class="grid gap-3 border-t border-default pt-4 text-sm sm:grid-cols-2" data-testid="regression-run-snapshot"><div><dt class="text-muted">{{ t('regression.fields.pluginVersion') }}</dt><dd class="mt-1">{{ source('plugin_version') }}</dd></div><div><dt class="text-muted">{{ t('regression.fields.sourceRevision') }}</dt><dd class="mt-1 break-words">{{ source('source_revision') }}</dd></div></dl>
            <div v-if="current.dependency_evidence.length" class="space-y-2 text-sm" data-testid="regression-dependency-evidence"><p v-for="dependency in current.dependency_evidence" :key="dependency.dependency_key">{{ t('regression.boundaries.' + dependency.boundary.replaceAll('.', '_')) }} · {{ t('regression.fields.declaredMode') }}: {{ t('regression.dependencyModes.' + dependency.declared_mode) }} · {{ t('regression.fields.actualMode') }}: {{ t('regression.dependencyModes.' + dependency.actual_mode) }}</p></div>
            <UButton color="neutral" variant="outline" :label="t('regression.actions.technical')" data-testid="regression-result-technical" @click="technicalOpen = true" />
          </template>
        </template>
      </div>
    </div>

    <USlideover v-model:open="comparisonOpen" :title="selectedAssertion ? t(selectedAssertion.title_key) : t('regression.resultActions.compare')" :description="t('regression.resultHelp.comparison')" :ui="{ content: 'max-w-3xl w-full' }">
      <template #body><div class="space-y-4" data-testid="regression-difference"><UAlert v-if="comparedRows === null" color="warning" :title="t('regression.resultHelp.presentationMissing')" /><div v-for="(row, index) in comparedRows || []" :key="index" class="rounded-lg border border-default p-4 space-y-3" :class="!row.matches ? 'border-error/60' : ''"><div class="flex items-start justify-between gap-2"><h4 class="text-sm font-medium text-highlighted">{{ row.label }}</h4><UBadge :color="!row.recorded ? 'warning' : row.matches ? 'success' : 'error'" variant="subtle">{{ t(!row.recorded ? 'regression.states.unrecorded' : row.matches ? 'regression.resultValues.matches' : 'regression.resultValues.differs') }}</UBadge></div><div class="grid gap-3 text-sm sm:grid-cols-2"><div><p class="mb-1 text-xs text-muted">{{ t('regression.fields.expected') }}</p><p class="whitespace-pre-wrap break-words">{{ row.expected }}</p></div><div><p class="mb-1 text-xs text-muted">{{ t('regression.fields.actual') }}</p><p class="whitespace-pre-wrap break-words">{{ row.actual }}</p></div></div></div></div></template>
      <template #footer><UButton color="neutral" variant="outline" :label="t('regression.actions.close')" @click="closeOverlays" /></template>
    </USlideover>
    <USlideover v-model:open="evidenceOpen" :title="t('regression.resultActions.evidence')" :description="t('regression.resultHelp.evidence')" :ui="{ content: 'max-w-3xl w-full' }">
      <template #body><p v-if="!viewedEvidence.length" class="text-sm text-muted">{{ t('regression.resultHelp.noEvidence') }}</p><article v-for="(evidence, index) in viewedEvidence" :key="index" class="mb-5 space-y-2 text-sm"><p>{{ t('regression.fields.boundary') }}: {{ t('regression.boundaries.' + evidence.boundary.replaceAll('.', '_')) }}</p><p class="text-muted">{{ formatTime(evidence.observed_at) }}</p><pre class="overflow-auto rounded-lg bg-elevated p-3 text-xs" data-testid="regression-evidence-snapshot">{{ JSON.stringify(evidence, null, 2) }}</pre></article></template>
      <template #footer><UButton color="neutral" variant="outline" :label="t('regression.actions.close')" @click="closeOverlays" /></template>
    </USlideover>
    <UModal v-model:open="technicalOpen" :title="t('regression.actions.technical')" :description="t('regression.technical.description')" :ui="{ content: 'max-w-4xl' }"><template #body><pre class="overflow-auto rounded-lg bg-elevated p-3 text-xs">{{ JSON.stringify(detail, null, 2) }}</pre></template><template #footer><UButton color="neutral" variant="outline" :label="t('regression.actions.close')" @click="closeOverlays" /></template></UModal>
  </section>
</template>
