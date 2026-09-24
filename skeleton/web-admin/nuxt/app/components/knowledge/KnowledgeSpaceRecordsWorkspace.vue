<template>
  <main class="mx-auto w-full max-w-6xl space-y-6 px-5 py-6 lg:px-8">
    <section class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-[#172536]"><div class="flex items-start justify-between gap-4"><div><p class="text-sm text-gray-500 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.overview.eyebrow') }}</p><h1 class="mt-1 text-2xl font-semibold text-gray-900 dark:text-[#fdfcff]">{{ t('knowledgeSpaces.records.title') }}</h1><p class="mt-2 text-sm text-gray-600 dark:text-[#d6e2ff]">{{ space?.name }}</p></div><div class="flex flex-wrap gap-2"><UButton color="neutral" variant="outline" :loading="loading" @click="load">{{ t('common.refresh') }}</UButton><UButton color="neutral" variant="outline" :to="detailPath">{{ t('knowledgeSpaces.detail.backToOverview') }}</UButton></div></div></section>
    <section class="grid gap-3 md:grid-cols-2">
      <div class="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-[#0f192a]"><h2 class="font-semibold text-gray-900 dark:text-[#fdfcff]">{{ t('knowledgeSpaces.records.currentVersionTitle') }}</h2><p class="mt-1 text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.records.currentVersionDescription') }}</p></div>
      <div class="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-[#0f192a]"><h2 class="font-semibold text-gray-900 dark:text-[#fdfcff]">{{ t('knowledgeSpaces.records.historyTitle') }}</h2><p class="mt-1 text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.records.historyDescription') }}</p></div>
    </section>
    <section class="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-[#0f192a]"><header class="border-b border-slate-200 p-5 dark:border-slate-700"><h2 class="font-semibold text-gray-900 dark:text-[#fdfcff]">{{ t('knowledgeSpaces.records.listTitle') }}</h2><p class="mt-1 text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.records.description') }}</p></header><div v-if="loading" class="p-10 text-center text-sm text-gray-700 dark:text-slate-200">{{ t('common.loading') }}</div><div v-else-if="!documents.length" class="p-10 text-center text-sm text-gray-700 dark:text-slate-200">{{ t('knowledgeSpaces.ui.noDocuments') }}</div><div v-else class="divide-y divide-slate-100 dark:divide-slate-800"><article v-for="document in documents" :key="document.uuid" class="flex flex-wrap items-center justify-between gap-4 p-5"><div><p class="font-medium text-gray-900 dark:text-[#fdfcff]">{{ document.title }}</p><p class="mt-1 text-xs text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.records.chunkTotal', { count: document.chunk_count }) }} · {{ formatDate(document.updated_at) }}</p></div><div class="flex gap-2"><UBadge :color="document.status === 'indexed' ? 'success' : document.status === 'failed' ? 'error' : 'warning'" variant="soft">{{ documentStatus(document.status) }}</UBadge><UButton size="xs" color="primary" variant="soft" @click="inspect(document)">{{ t('knowledgeSpaces.records.inspect') }}</UButton><UButton size="xs" color="neutral" variant="soft" @click="openJobs(document)">{{ t('knowledgeSpaces.records.jobsAction') }}</UButton><UButton size="xs" :loading="indexing === document.uuid" @click="reindex(document)">{{ t('knowledgeSpaces.records.reindex') }}</UButton></div></article></div></section>
    <UModal v-model:open="jobListOpen" :title="t('knowledgeSpaces.records.jobsTitle')" :description="jobDocument?.title || ''" :ui="{ content: 'max-w-5xl w-[88vw] mx-auto', body: 'p-0' }"><template #body><section class="rounded-2xl bg-white dark:bg-[#0f192a]">
      <header class="border-b border-slate-200 p-5 text-sm text-gray-600 dark:border-slate-700 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.records.jobsDescription') }}</header>
      <div v-if="jobListLoading" class="p-8 text-center text-sm">{{ t('common.loading') }}</div>
      <div v-else-if="!jobPage.items.length" class="p-8 text-center text-sm text-gray-600 dark:text-slate-200">{{ t('knowledgeSpaces.records.noJobs') }}</div>
      <div v-else class="divide-y divide-slate-100 dark:divide-slate-800">
        <article v-for="job in jobPage.items" :key="job.uuid" class="flex flex-wrap items-center justify-between gap-4 p-5">
          <div class="min-w-0 flex-1"><p class="font-medium text-gray-900 dark:text-[#fdfcff]">{{ jobTitle(job) }}</p><p class="mt-1 text-xs text-gray-600 dark:text-[#d6e2ff]">{{ formatDateTime(job.created_at) }} · {{ sourceTypeLabel(job.source_type) }} · {{ t('knowledgeSpaces.records.jobChunkCount', { count: job.chunk_total }) }} · {{ t('knowledgeSpaces.records.retryCount', { count: job.retry_count }) }}</p><div v-if="job.status === 'running' || job.status === 'retrying'" class="mt-2 max-w-sm"><UProgress :value="job.progress_percent" :max="100" /><p class="mt-1 text-xs text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.records.progressValue', { count: job.progress_percent }) }}</p></div><p v-if="job.error_code" class="mt-1 text-xs text-red-600 dark:text-red-300">{{ jobError(job.error_code) }}</p><p v-if="job.blocked_reason" class="mt-1 text-xs text-amber-700 dark:text-amber-300">{{ job.blocked_reason }}</p></div>
          <div class="flex items-center gap-2"><UBadge :color="jobColor(job.status)" variant="soft">{{ jobStatus(job.status) }}</UBadge><UButton size="xs" color="primary" variant="soft" @click="openJobFromList(job)">{{ t('knowledgeSpaces.records.viewJob') }}</UButton></div>
        </article>
      </div>
      <footer v-if="jobPage.total > jobPage.page_size" class="flex items-center justify-end gap-2 border-t border-slate-200 p-4 dark:border-slate-700"><span class="text-xs text-gray-600 dark:text-slate-300">{{ t('knowledgeSpaces.records.jobPage', { page: jobPage.page, total: jobPage.total }) }}</span><UButton size="xs" color="neutral" variant="soft" :disabled="jobPage.page <= 1" @click="loadJobs(jobPage.page - 1)">{{ t('common.previous') }}</UButton><UButton size="xs" color="neutral" variant="soft" :disabled="jobPage.page * jobPage.page_size >= jobPage.total" @click="loadJobs(jobPage.page + 1)">{{ t('common.next') }}</UButton></footer>
    </section></template></UModal>
    <UModal v-model:open="jobOpen" :title="t('knowledgeSpaces.records.jobDetailTitle')" :description="selectedJob ? jobTitle(selectedJob) : ''" :ui="{ content: 'max-w-5xl w-[88vw] mx-auto', body: 'p-5 sm:p-6' }">
      <template #body>
        <div v-if="jobLoading" class="p-6 text-center">{{ t('common.loading') }}</div>
        <div v-else-if="selectedJob" class="space-y-5 text-gray-900 dark:text-slate-100">
          <dl class="grid gap-4 text-sm sm:grid-cols-2">
            <div><dt>{{ t('knowledgeSpaces.records.jobStatus') }}</dt><dd class="font-semibold">{{ jobStatus(selectedJob.status) }}</dd></div><div><dt>{{ t('knowledgeSpaces.records.jobSourceType') }}</dt><dd>{{ sourceTypeLabel(selectedJob.source_type) }}</dd></div><div><dt>{{ t('knowledgeSpaces.records.jobRetryCount') }}</dt><dd>{{ selectedJob.retry_count }}</dd></div>
            <div><dt>{{ t('knowledgeSpaces.records.jobCreated') }}</dt><dd>{{ formatDateTime(selectedJob.created_at) }}</dd></div>
            <div><dt>{{ t('knowledgeSpaces.records.jobStarted') }}</dt><dd>{{ formatDateTime(selectedJob.started_at) }}</dd></div>
            <div><dt>{{ t('knowledgeSpaces.records.jobCompleted') }}</dt><dd>{{ formatDateTime(selectedJob.completed_at) }}</dd></div>
            <div><dt>{{ t('knowledgeSpaces.records.jobChunks') }}</dt><dd>{{ selectedJob.chunk_total }}</dd></div>
            <div><dt>{{ t('knowledgeSpaces.records.jobProgress') }}</dt><dd>{{ selectedJob.progress_percent }}%</dd></div>
            <div v-if="selectedJob.error_code" class="sm:col-span-2"><dt>{{ t('knowledgeSpaces.records.jobError') }}</dt><dd class="text-red-600 dark:text-red-300">{{ jobError(selectedJob.error_code) }}</dd></div>
            <div v-if="selectedJob.blocked_reason" class="sm:col-span-2"><dt>{{ t('knowledgeSpaces.records.jobReason') }}</dt><dd>{{ selectedJob.blocked_reason }}</dd></div>
          </dl>
          <section v-if="selectedJob.metrics_snapshot?.ingestion" class="border-t border-slate-200 pt-4 dark:border-slate-700">
            <h3 class="font-semibold">{{ t('knowledgeSpaces.records.strategyTitle') }}</h3>
            <dl class="mt-3 grid gap-3 text-sm sm:grid-cols-2">
              <div><dt>{{ t('knowledgeSpaces.records.strategyProfile') }}</dt><dd>{{ selectedJob.metrics_snapshot.ingestion.ingestion_profile }}</dd></div>
              <div><dt>{{ t('knowledgeSpaces.records.strategyMode') }}</dt><dd>{{ t(`knowledgeSpaces.ingestion.modeOptions.${selectedJob.metrics_snapshot.ingestion.segment_mode}`) }}</dd></div>
              <div><dt>{{ t('knowledgeSpaces.records.strategySize') }}</dt><dd>{{ selectedJob.metrics_snapshot.ingestion.chunk_size }}</dd></div>
              <div><dt>{{ t('knowledgeSpaces.records.strategyOverlap') }}</dt><dd>{{ selectedJob.metrics_snapshot.ingestion.chunk_overlap }}</dd></div>
              <div><dt>{{ t('knowledgeSpaces.records.strategyPagePriority') }}</dt><dd>{{ t(selectedJob.metrics_snapshot.ingestion.page_priority ? 'knowledgeSpaces.records.yes' : 'knowledgeSpaces.records.no') }}</dd></div>
              <div><dt>{{ t('knowledgeSpaces.records.strategySizePolicy') }}</dt><dd>{{ t(`knowledgeSpaces.ingestion.sizePolicyOptions.${selectedJob.metrics_snapshot.ingestion.segment_size_policy}`) }}</dd></div>
              <div v-if="selectedJob.metrics_snapshot.ingestion.segment_order?.length" class="sm:col-span-2"><dt>{{ t('knowledgeSpaces.records.strategyOrder') }}</dt><dd>{{ selectedJob.metrics_snapshot.ingestion.segment_order.map(step => t(`knowledgeSpaces.ingestion.segmentOrderLabels.${step}`)).join(' → ') }}</dd></div>
            </dl>
          </section>
          <section class="border-t border-slate-200 pt-4 dark:border-slate-700">
            <h3 class="font-semibold">{{ t('knowledgeSpaces.records.jobChunkPreview') }}</h3>
            <p v-if="!jobChunks.items.length" class="mt-2 text-sm text-gray-600 dark:text-slate-300">{{ t('knowledgeSpaces.records.noJobChunks') }}</p>
            <div v-else class="mt-3 space-y-3">
              <article v-for="chunk in jobChunks.items" :key="chunk.uuid" class="rounded-xl border border-slate-200 p-4 dark:border-slate-700">
                <p class="text-xs text-gray-600 dark:text-slate-300">{{ t('knowledgeSpaces.records.chunkOrdinal', { count: chunk.ordinal + 1 }) }} · {{ chunk.kind }}</p>
                <pre class="mt-2 max-h-44 overflow-auto whitespace-pre-wrap text-sm">{{ chunk.content }}</pre>
              </article>
              <div class="flex justify-end gap-2"><UButton size="xs" color="neutral" variant="soft" :disabled="jobChunks.page <= 1" @click="loadJobChunks(jobChunks.page - 1)">{{ t('common.previous') }}</UButton><UButton size="xs" color="neutral" variant="soft" :disabled="jobChunks.page * jobChunks.page_size >= jobChunks.total" @click="loadJobChunks(jobChunks.page + 1)">{{ t('common.next') }}</UButton></div>
            </div>
          </section>
          <p class="text-xs text-gray-600 dark:text-slate-300">{{ t('knowledgeSpaces.records.currentContentHint') }}</p>
        </div>
      </template>
    </UModal>
    <UModal v-model:open="inspectionOpen" :title="t('knowledgeSpaces.records.inspectionTitle')" :description="inspection?.document.title || ''" :ui="{ content: 'max-w-6xl w-[92vw] mx-auto', body: 'p-5 sm:p-6' }"><template #body><div v-if="inspectionLoading" class="p-10 text-center text-gray-700 dark:text-slate-200">{{ t('common.loading') }}</div><div v-else-if="inspection" class="space-y-6 text-gray-900 dark:text-slate-100"><div class="grid gap-4 sm:grid-cols-3"><div class="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-[#111a2b]"><p class="text-xs text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.records.persistedChunks') }}</p><b class="mt-1 block text-2xl text-gray-900 dark:text-[#fdfcff]">{{ inspection.persisted_chunk_count }}</b></div><div class="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-[#111a2b]"><p class="text-xs text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.records.vectors') }}</p><b class="mt-1 block text-2xl text-gray-900 dark:text-[#fdfcff]">{{ inspection.vector_count }}</b></div><div class="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-[#111a2b]"><p class="text-xs text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.records.vectorCheck') }}</p><UBadge class="mt-2" :color="vectorStatusColor" variant="soft">{{ t(`knowledgeSpaces.records.vectorStatus.${inspection.vector_status}`) }}</UBadge></div></div><section><div class="flex items-center justify-between gap-3"><h3 class="font-semibold text-gray-900 dark:text-[#fdfcff]">{{ t('knowledgeSpaces.records.originalContent') }}</h3><UButton size="xs" variant="soft" @click="openSourceEditor">{{ t('knowledgeSpaces.records.editOriginal') }}</UButton></div><pre class="mt-2 max-h-48 overflow-auto whitespace-pre-wrap rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm text-gray-800 dark:border-slate-700 dark:bg-[#111a2b] dark:text-slate-100">{{ inspection.document.content }}</pre></section><section><div class="flex items-center justify-between gap-3"><h3 class="font-semibold text-gray-900 dark:text-[#fdfcff]">{{ t('knowledgeSpaces.records.chunkPreview') }}</h3><span class="text-xs text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.records.pageInfo', { page: chunkPage.page, total: chunkPage.total }) }}</span></div><div class="mt-2 overflow-x-auto rounded-xl border border-slate-200 dark:border-slate-700"><table class="w-full text-left text-sm"><thead class="bg-slate-50 text-gray-700 dark:bg-[#111a2b] dark:text-[#d6e2ff]"><tr class="border-b border-slate-200 dark:border-slate-700"><th class="p-3">{{ t('knowledgeSpaces.records.chunkNumber') }}</th><th class="p-3">{{ t('knowledgeSpaces.records.chunkKind') }}</th><th class="p-3">{{ t('knowledgeSpaces.records.charCount') }}</th><th class="p-3" /></tr></thead><tbody class="text-gray-900 dark:text-slate-100"><tr v-for="chunk in chunkPage.items" :key="chunk.uuid" class="border-b border-slate-200 last:border-0 dark:border-slate-700"><td class="p-3">{{ chunk.ordinal + 1 }}</td><td class="p-3">{{ chunk.kind }}</td><td class="p-3">{{ chunk.char_count }}</td><td class="p-3 text-right"><UButton size="xs" variant="soft" @click="openChunk(chunk.uuid)">{{ t('knowledgeSpaces.records.viewChunk') }}</UButton></td></tr></tbody></table></div><div class="mt-3 flex justify-end gap-2"><UButton size="xs" color="neutral" variant="soft" :disabled="chunkPage.page <= 1" @click="loadChunks(chunkPage.page - 1)">{{ t('common.previous') }}</UButton><UButton size="xs" color="neutral" variant="soft" :disabled="chunkPage.page * chunkPage.page_size >= chunkPage.total" @click="loadChunks(chunkPage.page + 1)">{{ t('common.next') }}</UButton></div></section></div></template></UModal>
    <UModal v-model:open="chunkOpen" :title="t('knowledgeSpaces.records.chunkDetailTitle')" :description="selectedChunk ? t('knowledgeSpaces.records.chunkOrdinal', { count: selectedChunk.ordinal + 1 }) : ''" :ui="{ content: 'max-w-5xl w-[88vw] mx-auto', body: 'p-5 sm:p-6' }"><template #body><div v-if="chunkLoading" class="p-10 text-center text-gray-700 dark:text-slate-200">{{ t('common.loading') }}</div><div v-else-if="selectedChunk" class="space-y-4 text-gray-900 dark:text-slate-100"><div class="flex justify-end"><UButton size="xs" variant="soft" @click="openChunkEditor">{{ t('knowledgeSpaces.records.editChunk') }}</UButton></div><pre class="max-h-[55vh] overflow-auto whitespace-pre-wrap rounded-xl border border-slate-200 bg-slate-50 p-4 text-sm dark:border-slate-700 dark:bg-[#111a2b] dark:text-slate-100">{{ selectedChunk.content }}</pre><section class="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-[#111a2b]"><h3 class="text-sm font-semibold">{{ t('knowledgeSpaces.records.metadata') }}</h3><pre v-if="hasMetadata(selectedChunk.metadata)" class="mt-2 max-h-48 overflow-auto whitespace-pre-wrap text-xs">{{ prettyMetadata(selectedChunk.metadata) }}</pre><p v-else class="mt-2 text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.records.noMetadata') }}</p></section></div></template></UModal>
    <UModal v-model:open="sourceEditOpen" :title="t('knowledgeSpaces.records.editOriginalTitle')" :description="t('knowledgeSpaces.records.originalEditDescription')" :ui="{ content: 'max-w-5xl w-[88vw] mx-auto', body: 'p-5 sm:p-6' }"><template #body><UTextarea v-model="sourceDraft" :rows="18" class="w-full" /></template><template #footer><div class="flex w-full justify-end gap-2"><UButton color="neutral" variant="outline" @click="sourceEditOpen = false">{{ t('common.cancel') }}</UButton><UButton :loading="savingSource" @click="saveSourceAndReindex">{{ t('knowledgeSpaces.records.saveAndReindex') }}</UButton></div></template></UModal>
    <UModal v-model:open="chunkEditOpen" :title="t('knowledgeSpaces.records.editChunkTitle')" :description="t('knowledgeSpaces.records.chunkEditDescription')" :ui="{ content: 'max-w-5xl w-[88vw] mx-auto', body: 'p-5 sm:p-6' }"><template #body><UTextarea v-model="chunkDraft" :rows="18" class="w-full" /></template><template #footer><div class="flex w-full justify-end gap-2"><UButton color="neutral" variant="outline" @click="chunkEditOpen = false">{{ t('common.cancel') }}</UButton><UButton :loading="savingChunk" @click="saveChunk">{{ t('knowledgeSpaces.records.saveChunk') }}</UButton></div></template></UModal>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useLocalKnowledgeApi, type LocalKnowledgeChunk, type LocalKnowledgeChunkPage, type LocalKnowledgeDocument, type LocalKnowledgeDocumentInspection, type LocalKnowledgeIngestionJob, type LocalKnowledgeIngestionJobPage, type LocalKnowledgeJobChunkPage, type LocalKnowledgeSpace } from '~/composables/api/useLocalKnowledge'
import { getAuthToken, resolveApiBase } from '~/composables/api/_base'
import { createPluginWsClient } from '@artisan-cloud/plugin-framework-client'

const { t, locale } = useI18n()
const toast = useToast()
const api = useLocalKnowledgeApi()
const route = useRoute()
const runtimeConfig = useRuntimeConfig()
const documents = ref<LocalKnowledgeDocument[]>([])
const jobPage = ref<LocalKnowledgeIngestionJobPage>({ items: [], page: 1, page_size: 25, total: 0 })
const jobListOpen = ref(false)
const jobListLoading = ref(false)
const jobDocument = ref<LocalKnowledgeDocument | null>(null)
const jobOpen = ref(false)
const jobLoading = ref(false)
const selectedJob = ref<LocalKnowledgeIngestionJob | null>(null)
const jobChunks = ref<LocalKnowledgeJobChunkPage>({ items: [], page: 1, page_size: 25, total: 0 })
let ingestionWS: WebSocket | null = null
const space = ref<LocalKnowledgeSpace | null>(null)
const loading = ref(false)
const indexing = ref('')
const inspectionOpen = ref(false)
const inspectionLoading = ref(false)
const inspection = ref<LocalKnowledgeDocumentInspection | null>(null)
const chunkPage = ref<LocalKnowledgeChunkPage>({ items: [], page: 1, page_size: 25, total: 0 })
const chunkOpen = ref(false)
const chunkLoading = ref(false)
const selectedChunk = ref<LocalKnowledgeChunk | null>(null)
const sourceEditOpen = ref(false)
const sourceDraft = ref('')
const savingSource = ref(false)
const chunkEditOpen = ref(false)
const chunkDraft = ref('')
const savingChunk = ref(false)
const uuid = computed(() => String(route.params.uuid || ''))
const adminBase = computed(() => String(runtimeConfig.public?.pluginAdminBase || '/_p/com.powerx.plugins.base/admin/').replace(/\/+$/, ''))
const detailPath = computed(() => `${adminBase.value}/knowledge/${uuid.value}`)
const vectorStatusColor = computed(() => inspection.value?.vector_status === 'verified' ? 'success' : inspection.value?.vector_status === 'mismatch' ? 'error' : 'warning')

function formatDate(value: string) { return value ? new Intl.DateTimeFormat(locale.value).format(new Date(value)) : '-' }
function formatDateTime(value?: string) { return value ? new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value)) : '—' }
function documentStatus(status: string) { return t(`knowledgeSpaces.records.documentStatus.${status || 'queued'}`) }
function jobStatus(status: string) { return t(`knowledgeSpaces.records.jobStatuses.${status || 'pending'}`) }
function jobColor(status: string): 'success' | 'error' | 'warning' | 'neutral' { return status === 'completed' ? 'success' : status === 'failed' ? 'error' : status === 'blocked' ? 'warning' : 'neutral' }
function jobTitle(job: LocalKnowledgeIngestionJob) { return job.metrics_snapshot?.source_title || documents.value.find(item => item.uuid === job.source_id)?.title || t('knowledgeSpaces.records.sourceUnavailable') }
function sourceTypeLabel(sourceType: 'manual' | 'upload') { return t(`knowledgeSpaces.sourceType.${sourceType}`) }
function jobError(code: string) { const key = `knowledgeSpaces.ingestion.uploadErrors.${code}`; return t(key) === key ? t('knowledgeSpaces.ingestion.uploadErrors.UNKNOWN', { code }) : t(key) }
function hasMetadata(metadata: Record<string, unknown> | undefined) { return Boolean(metadata && Object.keys(metadata).length) }
function prettyMetadata(metadata: Record<string, unknown> | undefined) { return JSON.stringify(metadata, null, 2) }
function showError(message: string) { toast.add({ title: message, color: 'error' }) }
async function load() { if (!uuid.value) return; loading.value = true; try { const [out, spaces] = await Promise.all([api.documents(uuid.value), api.spaces()]); documents.value = out.items || []; space.value = (spaces.items || []).find(item => item.uuid === uuid.value) || null; if (jobListOpen.value) await loadJobs(jobPage.value.page) } catch { showError(t('knowledgeSpaces.records.loadFailed')) } finally { loading.value = false } }
async function loadJobs(page: number) { if (!uuid.value || !jobDocument.value) return; jobListLoading.value = true; try { jobPage.value = await api.ingestionJobs(uuid.value, page, jobPage.value.page_size, jobDocument.value.uuid) } catch { showError(t('knowledgeSpaces.records.loadFailed')) } finally { jobListLoading.value = false } }
async function openJobs(document: LocalKnowledgeDocument) { jobDocument.value = document; jobPage.value = { items: [], page: 1, page_size: 25, total: 0 }; jobListOpen.value = true; await loadJobs(1) }
async function openJobFromList(job: LocalKnowledgeIngestionJob) { jobListOpen.value = false; await openJob(job) }
async function openJob(job: LocalKnowledgeIngestionJob) { jobOpen.value = true; jobLoading.value = true; selectedJob.value = null; jobChunks.value = { items: [], page: 1, page_size: 25, total: 0 }; try { const [detail, chunks] = await Promise.all([api.ingestionJob(uuid.value, job.uuid), api.ingestionJobChunks(uuid.value, job.uuid)]); selectedJob.value = detail; jobChunks.value = chunks } catch { showError(t('knowledgeSpaces.records.loadFailed')) } finally { jobLoading.value = false } }
async function loadJobChunks(page: number) { if (!selectedJob.value) return; try { jobChunks.value = await api.ingestionJobChunks(uuid.value, selectedJob.value.uuid, page, jobChunks.value.page_size) } catch { showError(t('knowledgeSpaces.records.loadFailed')) } }
async function inspect(document: LocalKnowledgeDocument) { inspectionOpen.value = true; inspectionLoading.value = true; inspection.value = null; try { inspection.value = await api.inspectDocument(document.uuid); await loadChunks(1) } catch { showError(t('knowledgeSpaces.records.inspectionFailed')) } finally { inspectionLoading.value = false } }
async function loadChunks(page: number) { if (!inspection.value) return; chunkPage.value = await api.documentChunks(inspection.value.document.uuid, page, chunkPage.value.page_size) }
async function openChunk(chunkUUID: string) { if (!inspection.value) return; chunkOpen.value = true; chunkLoading.value = true; selectedChunk.value = null; try { selectedChunk.value = await api.documentChunk(inspection.value.document.uuid, chunkUUID) } catch { showError(t('knowledgeSpaces.records.inspectionFailed')) } finally { chunkLoading.value = false } }
async function reindex(document: LocalKnowledgeDocument) { indexing.value = document.uuid; try { await api.indexDocument(document.uuid); await load(); if (inspection.value?.document.uuid === document.uuid) { inspection.value = await api.inspectDocument(document.uuid); await loadChunks(1) } } catch { showError(t('knowledgeSpaces.records.reindexFailed')) } finally { indexing.value = '' } }
function openSourceEditor() { if (!inspection.value) return; sourceDraft.value = inspection.value.document.content; sourceEditOpen.value = true }
async function saveSourceAndReindex() { if (!inspection.value || !sourceDraft.value.trim()) return; savingSource.value = true; const document = inspection.value.document; try { await api.saveDocument(document.space_uuid, { uuid: document.uuid, title: document.title, content: sourceDraft.value, source_type: document.source_type, tags: document.tags, effective_from: document.effective_from, effective_to: document.effective_to }); await api.indexDocument(document.uuid); sourceEditOpen.value = false; inspection.value = await api.inspectDocument(document.uuid); await loadChunks(1); await load(); toast.add({ title: t('knowledgeSpaces.records.originalSavedAndReindexed'), color: 'success' }) } catch { showError(t('knowledgeSpaces.records.originalSaveFailed')) } finally { savingSource.value = false } }
function openChunkEditor() { if (!selectedChunk.value) return; chunkDraft.value = selectedChunk.value.content; chunkEditOpen.value = true }
async function saveChunk() { if (!inspection.value || !selectedChunk.value || !chunkDraft.value.trim()) return; savingChunk.value = true; try { selectedChunk.value = await api.updateDocumentChunk(inspection.value.document.uuid, selectedChunk.value.uuid, chunkDraft.value); chunkEditOpen.value = false; await loadChunks(chunkPage.value.page); inspection.value = await api.inspectDocument(inspection.value.document.uuid); toast.add({ title: t('knowledgeSpaces.records.chunkSaved'), color: 'success' }) } catch { showError(t('knowledgeSpaces.records.chunkSaveFailed')) } finally { savingChunk.value = false } }
function startIngestionWS() {
  if (typeof window === 'undefined' || ingestionWS) return
  const apiBase = new URL(resolveApiBase(), window.location.origin)
  const protocol = apiBase.protocol === 'https:' ? 'wss:' : 'ws:'
  ingestionWS = createPluginWsClient({ pluginId: String(runtimeConfig.public?.powerxPluginId || 'com.powerx.plugins.base'), wsBaseURL: `${protocol}//${apiBase.host}`, wsPath: '/api/ws', token: getAuthToken() }).connect()
  ingestionWS.onopen = () => ingestionWS?.send(JSON.stringify({ type: 'subscribe', topics: ['_topic.knowledge.ingestion.progress'] }))
  ingestionWS.onmessage = (event) => {
    let message: { type?: string; topic?: string; payload?: { job_uuid?: string; document_uuid?: string; space_uuid?: string; status?: string; progress_percent?: number; chunk_total?: number; error_code?: string } }
    try { message = JSON.parse(String(event.data || '{}')) } catch { return }
    if (message.type !== 'event' || message.topic !== '_topic.knowledge.ingestion.progress' || message.payload?.space_uuid !== uuid.value || !jobListOpen.value || message.payload?.document_uuid !== jobDocument.value?.uuid) return
    const payload = message.payload
    const job = jobPage.value.items.find(item => item.uuid === payload.job_uuid)
    if (!job) { if (jobListOpen.value) void loadJobs(1); return }
    job.status = payload.status || job.status
    job.progress_percent = Number(payload.progress_percent ?? job.progress_percent)
    job.chunk_total = Number(payload.chunk_total ?? job.chunk_total)
    if (payload.error_code) job.error_code = payload.error_code
    if (jobListOpen.value && (payload.status === 'completed' || payload.status === 'failed')) void loadJobs(jobPage.value.page)
  }
  ingestionWS.onclose = () => { ingestionWS = null }
}
onMounted(async () => { await load(); startIngestionWS() })
onBeforeUnmount(() => { ingestionWS?.close(); ingestionWS = null })
</script>
