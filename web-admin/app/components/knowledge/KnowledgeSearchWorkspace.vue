<template>
  <main class="mx-auto w-full max-w-5xl space-y-6 px-5 py-6 lg:px-8">
    <header class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-[#fdfcff]">
          {{ t('knowledgeSearch.title') }}
        </h1>
        <div
          class="mt-2 flex flex-wrap items-center gap-2 text-sm text-gray-600 dark:text-slate-300"
        >
          <span>{{ space?.name }}</span>
          <UBadge
            v-if="workspace.semanticCapabilities"
            :color="indexReady ? 'success' : indexError ? 'error' : 'neutral'"
            variant="soft"
          >
            {{
              t(
                indexLoading
                  ? 'knowledgeSearch.checking'
                  : indexError
                    ? 'knowledgeSearch.statusUnavailable'
                    : indexReady
                      ? 'knowledgeSearch.ready'
                      : 'knowledgeSearch.notReady'
              )
            }}
          </UBadge>
          <span v-if="capabilities?.index_counts?.length === 1">
            {{
              t('knowledgeSearch.documentCount', {
                count: capabilities.index_counts[0].documents
              })
            }}
          </span>
        </div>
        <p class="mt-2 text-sm text-gray-600 dark:text-slate-300">
          {{ t('knowledgeSearch.description') }}
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <UButton
          v-if="workspace.semanticCapabilities"
          color="neutral"
          variant="outline"
          @click="indexSettingsOpen = true"
        >
          {{ t('knowledgeSearch.indexSettings') }}
        </UButton>
        <UButton color="neutral" variant="ghost" :to="workspace.basePath">
          {{ t('knowledgeSpaces.detail.backToOverview') }}
        </UButton>
      </div>
    </header>
    <KnowledgeContentOverview
      v-if="space"
      :spaceUUID="space.uuid"
      :spaceName="space.name"
      :retrievalMode="workspace.semanticQuery ? mode : undefined"
      :corpusGeneration="capabilities?.generation?.corpus_generation"
      :disabled="loading"
      @question="selectQuestion"
    />
    <section
      class="rounded-2xl border border-slate-200 bg-white p-5 sm:p-6 text-gray-900 dark:border-slate-700 dark:bg-[#0f192a] dark:text-slate-100"
    >
      <UAlert
        v-if="indexError"
        color="error"
        variant="soft"
        class="mb-4"
        :title="t('knowledgeSearch.statusUnavailable')"
        :description="indexError"
      />
      <UAlert
        v-else-if="workspace.semanticQuery && capabilities && !indexReady"
        color="warning"
        variant="soft"
        class="mb-4"
        :title="t('knowledgeSearch.notReady')"
        :description="t('knowledgeSearch.prepareHint')"
      />
      <div
        v-if="indexError || (workspace.semanticQuery && capabilities && !indexReady)"
        class="mb-5 flex gap-2"
      >
        <UButton
          v-if="indexError"
          size="sm"
          color="neutral"
          variant="outline"
          :loading="indexLoading"
          @click="loadCapabilities"
        >
          {{ t('common.refresh') }}
        </UButton>
        <UButton size="sm" color="neutral" variant="outline" @click="indexSettingsOpen = true">
          {{ t('knowledgeSearch.prepareAction') }}
        </UButton>
      </div>
      <UAlert v-if="error" color="error" variant="soft" class="mb-4" :description="error" />
      <form class="space-y-4" @submit.prevent="search">
        <UFormField :label="t('knowledgeSearch.questionLabel')" required>
          <UTextarea
            ref="questionInput"
            v-model="query"
            :placeholder="t('knowledgeSearch.questionPlaceholder')"
            :rows="4"
            class="w-full"
            :disabled="loading"
          />
        </UFormField>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="text-xs text-gray-600 dark:text-slate-300">
            {{
              t('knowledgeSearch.activeSettings', {
                mode: workspace.semanticQuery
                  ? t(`knowledgeSemantic.modes.${mode}`)
                  : t('knowledgeSemantic.localLexical'),
                count: limit
              })
            }}
          </span>
          <UButton
            type="submit"
            icon="i-heroicons-magnifying-glass"
            :loading="loading"
            :disabled="!canSearch"
          >
            {{ t('knowledgeSearch.searchAction') }}
          </UButton>
        </div>
        <details class="border-t border-slate-100 pt-4 dark:border-slate-800">
          <summary class="cursor-pointer text-sm text-gray-600 dark:text-slate-300">
            {{ t('knowledgeSearch.advancedSettings') }}
          </summary>
          <div class="mt-4 space-y-4">
            <div class="grid gap-4 sm:grid-cols-2">
              <UFormField :label="t('knowledgeLab.copy.text153')">
                <UInput v-model.number="limit" type="number" min="1" max="100" />
              </UFormField>
              <UFormField v-if="workspace.semanticQuery" :label="t('knowledgeSemantic.queryMode')">
                <USelect v-model="mode" :items="modeItems" :disabled="!indexReady" class="w-full" />
              </UFormField>
            </div>
            <template v-if="workspace.semanticQuery">
              <UCheckbox
                v-model="requireGeneration"
                :label="t('knowledgeSemantic.requireGeneration')"
              />
              <p
                v-if="capabilities?.indexed && !capabilities.indexed_query_modes?.includes(mode)"
                class="text-sm text-amber-700 dark:text-amber-300"
              >
                {{ t('knowledgeSemantic.setup.modeUnavailable') }}
              </p>
              <fieldset
                class="space-y-4 rounded-xl border border-slate-200 p-4 dark:border-slate-700"
              >
                <legend class="px-2 text-sm">
                  {{ t('knowledgeSearch.optionalFilters') }}
                </legend>
                <UFormField :label="t('knowledgeSemantic.artifactRoles')">
                  <USelectMenu
                    v-model="filters.artifact_roles"
                    multiple
                    :items="roleItems"
                    value-key="value"
                    :portal="false"
                    :aria-label="t('knowledgeSemantic.artifactRoles')"
                    class="w-full"
                  />
                </UFormField>
                <UFormField v-for="field in filterFields" :key="field.key" :label="t(field.label)">
                  <div
                    v-for="(_, index) in filters[field.key]"
                    :key="index"
                    class="mb-2 flex gap-2"
                  >
                    <UInput v-model="filters[field.key][index]" class="flex-1" />
                    <UButton
                      type="button"
                      color="neutral"
                      variant="soft"
                      @click="filters[field.key].splice(index, 1)"
                    >
                      {{ t('common.delete') }}
                    </UButton>
                  </div>
                  <UButton
                    type="button"
                    color="neutral"
                    variant="soft"
                    @click="filters[field.key].push('')"
                  >
                    {{ t('common.add') }}
                  </UButton>
                </UFormField>
              </fieldset>
            </template>
          </div>
        </details>
      </form>
    </section>
    <section v-if="semanticResult || result" class="space-y-4 text-gray-900 dark:text-slate-100">
      <header class="flex flex-wrap items-center justify-between gap-2">
        <h2 class="font-semibold">
          {{
            t('knowledgeSearch.resultsTitle', {
              count: semanticResult ? semanticResult.items.length : result.chunks?.length || 0
            })
          }}
        </h2>
        <span class="text-xs text-gray-600 dark:text-slate-300">
          {{ t('knowledgeSearch.resultsHint') }}
        </span>
      </header>
      <template v-if="semanticResult">
        <article
          v-for="(match, index) in semanticResult.items"
          :key="match.chunk_uuid"
          class="rounded-2xl border border-slate-200 bg-white p-5 dark:border-slate-700 dark:bg-[#0f192a]"
        >
          <div class="flex items-start gap-3">
            <span class="shrink-0 rounded-lg bg-slate-100 px-2 py-1 text-xs dark:bg-slate-800">
              {{ index + 1 }}
            </span>
            <div class="min-w-0 flex-1">
              <h3 class="break-words font-semibold">{{ match.title }}</h3>
              <p class="mt-1 text-xs text-gray-600 dark:text-slate-300">
                {{
                  t('knowledgeSearch.sourceLabel', {
                    kind: t(`knowledgeSemantic.sources.${match.source_ref.kind}`)
                  })
                }}
                · {{ t(`knowledgeSemantic.roles.${match.artifact_role}`) }}
              </p>
            </div>
          </div>
          <p
            class="mt-4 whitespace-pre-wrap text-sm leading-6"
            :class="expandedChunks.includes(match.chunk_uuid) ? '' : 'line-clamp-6'"
          >
            {{ match.text }}
          </p>
          <UButton
            type="button"
            class="mt-2"
            size="xs"
            color="neutral"
            variant="ghost"
            @click="toggleChunk(match.chunk_uuid)"
          >
            {{
              t(
                expandedChunks.includes(match.chunk_uuid)
                  ? 'knowledgeSearch.collapseSnippet'
                  : 'knowledgeSearch.expandSnippet'
              )
            }}
          </UButton>
          <details class="mt-4 border-t border-slate-100 pt-3 dark:border-slate-800">
            <summary class="cursor-pointer text-xs text-gray-600 dark:text-slate-300">
              {{ t('knowledgeSearch.resultDetails') }}
            </summary>
            <div class="mt-3 space-y-3 text-sm">
              <p>
                {{ t(`knowledgeSemantic.scores.${match.score_type}`) }}:
                {{ match.score }}
              </p>
              <dl class="grid gap-2 sm:grid-cols-2">
                <div v-for="score in match.scores" :key="score.source + score.score_type">
                  <dt>
                    {{ t(`knowledgeSemantic.scores.${score.score_type}`) }} ·
                    {{ t(`knowledgeSemantic.directions.${score.direction}`) }}
                  </dt>
                  <dd>{{ score.score }}</dd>
                </div>
              </dl>
              <p>
                {{
                  t('knowledgeSemantic.sourcePosition', {
                    kind: t(`knowledgeSemantic.sources.${match.source_ref.kind}`),
                    version: match.source_ref.version,
                    start: match.source_ref.char_start,
                    end: match.source_ref.char_end
                  })
                }}
              </p>
              <pre class="max-h-64 overflow-auto whitespace-pre-wrap text-xs">{{
                JSON.stringify(
                  {
                    source_ref: match.source_ref,
                    external_ref: match.external_ref,
                    knowledge_profile_uuid: match.knowledge_profile_uuid,
                    case_uuid: match.case_uuid,
                    document_version: match.document_version,
                    bundle_generation: match.bundle_generation,
                    retrieval_sources: match.retrieval_sources
                  },
                  null,
                  2
                )
              }}</pre>
              <UButton
                v-if="workspace.setVisibility"
                type="button"
                size="xs"
                color="warning"
                variant="soft"
                @click="hideTarget = match"
              >
                {{ t('knowledgeSemantic.hideDocument') }}
              </UButton>
            </div>
          </details>
        </article>
        <details class="rounded-xl border border-slate-200 p-4 dark:border-slate-700">
          <summary class="cursor-pointer text-xs text-gray-600 dark:text-slate-300">
            {{ t('knowledgeSemantic.versionDetails') }}
          </summary>
          <pre class="mt-3 max-h-72 overflow-auto whitespace-pre-wrap text-xs">{{
            JSON.stringify(
              {
                generations: semanticResult.generations,
                trace_id: semanticResult.trace_id,
                invocation_trace_id: semanticResult.invocation_trace_id,
                request_id: semanticResult.request_id
              },
              null,
              2
            )
          }}</pre>
        </details>
      </template>
      <template v-else-if="result">
        <article
          v-for="(chunk, index) in result.chunks"
          :key="chunk.chunk_id || index"
          class="rounded-2xl border border-slate-200 bg-white p-5 dark:border-slate-700 dark:bg-[#0f192a]"
        >
          <h3 class="font-semibold">{{ chunk.citation?.title }}</h3>
          <p
            class="mt-4 whitespace-pre-wrap text-sm leading-6"
            :class="expandedChunks.includes(String(index)) ? '' : 'line-clamp-6'"
          >
            {{ chunk.text }}
          </p>
          <UButton
            type="button"
            class="mt-2"
            size="xs"
            color="neutral"
            variant="ghost"
            @click="toggleChunk(String(index))"
          >
            {{
              t(
                expandedChunks.includes(String(index))
                  ? 'knowledgeSearch.collapseSnippet'
                  : 'knowledgeSearch.expandSnippet'
              )
            }}
          </UButton>
          <details class="mt-4 border-t border-slate-100 pt-3 dark:border-slate-800">
            <summary class="cursor-pointer text-xs text-gray-600 dark:text-slate-300">
              {{ t('knowledgeSearch.resultDetails') }}
            </summary>
            <pre class="mt-3 max-h-64 overflow-auto whitespace-pre-wrap text-xs">{{
              JSON.stringify(chunk.citation, null, 2)
            }}</pre>
          </details>
        </article>
      </template>
      <p
        v-if="semanticResult ? !semanticResult.items.length : !result?.chunks?.length"
        class="text-sm text-gray-600 dark:text-slate-300"
      >
        {{ t('knowledgeLab.copy.text133') }}
      </p>
    </section>
    <UModal
      v-model:open="indexSettingsOpen"
      :title="t('knowledgeSearch.indexSettings')"
      :description="space?.name || ''"
      :close="{ 'aria-label': t('common.close') }"
      :ui="{ content: 'max-w-3xl w-[94vw] mx-auto', body: 'p-4 sm:p-5' }"
    >
      <template #body>
        <KnowledgeSemanticIndexPanel
          v-if="workspace.semanticCapabilities && space"
          :spaceUUID="space.uuid"
          :initialState="capabilities"
          @changed="setCapabilities"
          @failed="indexError = $event"
        />
      </template>
    </UModal>
    <UModal
      :open="Boolean(hideTarget)"
      :title="t('knowledgeSemantic.hideDocument')"
      :description="hideTarget?.title || ''"
      :close="{ 'aria-label': t('common.close') }"
      @update:open="!$event && (hideTarget = undefined)"
    >
      <template #body>
        <p class="text-gray-900 dark:text-slate-100">
          {{ t('knowledgeSemantic.hideHint') }}
        </p>
      </template>
      <template #footer>
        <UButton color="neutral" variant="soft" @click="hideTarget = undefined">
          {{ t('common.cancel') }}
        </UButton>
        <UButton color="warning" :loading="loading" @click="hideDocument">
          {{ t('common.confirm') }}
        </UButton>
      </template>
    </UModal>
  </main>
</template>
<script setup lang="ts">
import KnowledgeSemanticIndexPanel from './KnowledgeSemanticIndexPanel.vue'
import KnowledgeContentOverview from './KnowledgeContentOverview.vue'
import { formatKnowledgeRequestFailure } from '~/composables/knowledge/requestFailure'
import { computed, nextTick, onMounted, ref, reactive } from 'vue'
import type {
  SemanticCapabilities,
  SemanticFilters,
  SemanticResult,
  SemanticMatch,
  RetrievalMode
} from '~/composables/api/knowledgeSemanticTypes'
import { useKnowledgeWorkspace } from '~/composables/api/useKnowledgeWorkspace'
import type { LocalKnowledgeSpace } from '~/composables/api/useLocalKnowledge'
const workspace = useKnowledgeWorkspace()
const { t } = useI18n()
const route = useRoute()
const indexSettingsOpen = ref(false)
const indexLoading = ref(false)
const indexError = ref('')
const expandedChunks = ref<string[]>([])
const questionInput = ref<{ textareaRef?: HTMLTextAreaElement }>()
function selectQuestion(question: string) {
  if (loading.value) return
  query.value = question
  void nextTick(() => questionInput.value?.textareaRef?.focus())
}
function toggleChunk(id: string) {
  expandedChunks.value = expandedChunks.value.includes(id)
    ? expandedChunks.value.filter((value) => value !== id)
    : [...expandedChunks.value, id]
}
const indexReady = computed(() =>
  Boolean(capabilities.value?.configured && capabilities.value.ready && capabilities.value.indexed)
)
const space = ref<LocalKnowledgeSpace>()
const capabilities = ref<SemanticCapabilities>()
const mode = ref<RetrievalMode>('semantic')
const query = ref('')
const limit = ref(5)
const loading = ref(false)
const error = ref('')
const result = ref<any>(null)
const semanticResult = ref<SemanticResult>()
const hideTarget = ref<SemanticMatch>()
const requireGeneration = ref(true)
const filters = reactive<SemanticFilters>({
  document_uuids: [],
  category_codes: [],
  tag_uuids: [],
  artifact_roles: []
})
const filterFields = [
  { key: 'category_codes', label: 'knowledgeSemantic.categoryCodes' },
  { key: 'document_uuids', label: 'knowledgeSemantic.documentUUIDs' },
  { key: 'tag_uuids', label: 'knowledgeSemantic.tagUUIDs' }
] as const
const modeItems = computed(() =>
  ['semantic', 'hybrid'].map((value) => ({
    value,
    label: t(`knowledgeSemantic.modes.${value}`),
    disabled: !capabilities.value?.indexed_query_modes?.includes(value as RetrievalMode)
  }))
)
const roleItems = computed(() =>
  (capabilities.value?.artifact_roles || []).map((value) => ({
    value,
    label: t(`knowledgeSemantic.roles.${value}`)
  }))
)
const canSearch = computed(() =>
  Boolean(
    space.value &&
      query.value.trim() &&
      Array.from(query.value).length <= 8192 &&
      Number.isInteger(limit.value) &&
      limit.value >= 1 &&
      limit.value <= 100 &&
      (!workspace.semanticQuery ||
        (indexReady.value && capabilities.value?.indexed_query_modes?.includes(mode.value)))
  )
)
function setCapabilities(value: SemanticCapabilities | undefined) {
  capabilities.value = value
  if (value) indexError.value = ''
  semanticResult.value = undefined
  result.value = null
  expandedChunks.value = []
}
async function loadCapabilities() {
  if (!space.value || !workspace.semanticCapabilities) return
  indexLoading.value = true
  indexError.value = ''
  setCapabilities(undefined)
  try {
    setCapabilities(await workspace.semanticCapabilities(space.value.uuid))
  } catch (e) {
    indexError.value = message(e)
  } finally {
    indexLoading.value = false
  }
}
function message(e: unknown) {
  return formatKnowledgeRequestFailure(e, t)
}
onMounted(async () => {
  try {
    const out = await workspace.api.spaces()
    space.value = out.items.find((item) => item.uuid === String(route.params.uuid))
    if (!space.value) error.value = t('knowledgeSpaces.detail.notFound')
    else await loadCapabilities()
  } catch (e) {
    error.value = message(e)
  }
})
async function search() {
  if (!canSearch.value || !space.value || loading.value) return
  loading.value = true
  error.value = ''
  result.value = null
  semanticResult.value = undefined
  expandedChunks.value = []
  try {
    if (workspace.semanticQuery) {
      semanticResult.value = await workspace.semanticQuery({
        schema: 'powerx.knowledge.retrieval-query/v1',
        query: query.value,
        space_uuids: [space.value.uuid],
        mode: mode.value,
        top_k: limit.value,
        filters: {
          document_uuids: [...filters.document_uuids],
          category_codes: [...filters.category_codes],
          tag_uuids: [...filters.tag_uuids],
          artifact_roles: [...filters.artifact_roles]
        },
        ...(requireGeneration.value && capabilities.value?.generation
          ? {
              required_generations: {
                [space.value.uuid]: capabilities.value.generation.corpus_generation
              }
            }
          : {})
      })
    } else {
      result.value = await workspace.search(space.value.uuid, query.value, limit.value)
    }
  } catch (e) {
    error.value = message(e)
  } finally {
    loading.value = false
  }
}
async function hideDocument() {
  if (!hideTarget.value || !space.value || !workspace.setVisibility) return
  loading.value = true
  try {
    await workspace.setVisibility(space.value.uuid, hideTarget.value.document_uuid, {
      queryable: false
    })
    hideTarget.value = undefined
    semanticResult.value = undefined
    result.value = null
    await loadCapabilities()
  } catch (e) {
    error.value = message(e)
  } finally {
    loading.value = false
  }
}
</script>
