<script setup lang="ts">
import RequestHistory from './RequestHistory.vue'
import type { PublisherClient } from '../client'
import UButton from '@nuxt/ui/components/Button.vue'
import UForm from '@nuxt/ui/components/Form.vue'
import UFormField from '@nuxt/ui/components/FormField.vue'
import UInput from '@nuxt/ui/components/Input.vue'
import UTextarea from '@nuxt/ui/components/Textarea.vue'
import USelect from '@nuxt/ui/components/Select.vue'
import UModal from '@nuxt/ui/components/Modal.vue'
import UTabs from '@nuxt/ui/components/Tabs.vue'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import * as v from 'valibot'
import type { createMarketWorkspace } from '../market-workspace'
import type { createReviewWorkspace } from '../review-workspace'
import en from '../locales/en.json'
import zh from '../locales/zh.json'
const props = defineProps<{
  client: PublisherClient
  market: ReturnType<typeof createMarketWorkspace>
  review: ReturnType<typeof createReviewWorkspace>
  connected: boolean
  narrow: boolean
}>()
const { t, te, locale } = useI18n({ useScope: 'local', messages: { en, zh } })
const {
  selectedMarketID,
  selectedMarket,
  markets,
  versions,
  versionsPage,
  versionsTotal,
  versionsLoaded,
  uploads
} = props.market
const {
  versionID,
  tasks,
  selectedID,
  selected,
  preparation,
  preview,
  draft,
  error,
  errorRequestID,
  busy,
  needsReconcile,
  pollPaused,
  capable,
  existing,
  collectionState,
  history,
  historyLoaded
} = props.review
const historyOpen = ref(false)
const validationFailed = ref(false)
const formOpen = ref(false),
  confirmOpen = ref(false),
  formSection = ref('general')
const marketOptions = computed(() =>
  markets.value.map((m) => ({ label: m.name, value: m.connection_id }))
)
const versionOptions = computed(() => {
  const items = versions.value.map((v) => ({
    label: v.version,
    value: v.version_id
  }))
  for (const u of uploads.value)
    if (
      u.connection_id === selectedMarketID.value &&
      u.receipt &&
      !items.some((v) => v.value === u.receipt!.version_id)
    )
      items.push({ label: u.version, value: u.receipt.version_id })
  if (
    preparation.value &&
    !items.some((v) => v.value === preparation.value!.version.version_id)
  )
    items.push({
      label: preparation.value.version.version,
      value: preparation.value.version.version_id
    })
  return items
})
const taskOptions = computed(() =>
  tasks.value.map((s) => ({
    label: t('review.taskOption', {
      version: s.version,
      market:
        markets.value.find((m) => m.connection_id === s.connection_id)?.name ||
        t('review.savedMarket'),
      time: formatDate(s.created_at)
    }),
    value: s.submission_id
  }))
)
const formTabs = computed(() => [
  { value: 'general', slot: 'general', label: t('review.general') },
  { value: 'policy', slot: 'policy', label: t('review.policy') }
])
const trim = (s: string) => s.replace(/^[ \t\r\n]+|[ \t\r\n]+$/g, '')
const textRule = () =>
  v.pipe(
    v.string(),
    v.check(
      (s) =>
        [...trim(s)].length >= 10 &&
        new TextEncoder().encode(trim(s)).length <= 10000 &&
        !s.includes('\0'),
      t('review.validationText')
    )
  )
const schema = computed(() =>
  v.object({
    description: textRule(),
    security_statement: textRule(),
    privacy_statement: textRule(),
    license: v.pipe(
      v.string(),
      v.check(
        (s) => !!preparation.value?.options.licenses.includes(s),
        t('review.validationChoice')
      )
    ),
    min_core: v.pipe(
      v.string(),
      v.check(
        (s) => !!preparation.value?.options.versions.includes(s),
        t('review.validationChoice')
      )
    ),
    support_url: v.pipe(
      v.string(),
      v.check(publicationURL, t('review.validationURL'))
    ),
    icon_url: v.pipe(
      v.string(),
      v.check((s) => s === '' || publicationURL(s), t('review.validationURL'))
    )
  })
)
const errorText = (code: string) =>
  te(`errors.${code}`)
    ? t(`errors.${code}`)
    : t('errors.PUBLISHER_MARKET_PROTOCOL')
const taskStage = (stage: string) =>
  te(`review.stages.${stage}`)
    ? t(`review.stages.${stage}`)
    : t('stages.unknown')
function publicationURL(raw: string) {
  try {
    const u = new URL(trim(raw))
    return (
      ['https:', 'http:'].includes(u.protocol) &&
      !u.username &&
      !u.password &&
      !/[\x00-\x20\\]/.test(trim(raw)) &&
      new TextEncoder().encode(trim(raw)).length <= 2000
    )
  } catch {
    return false
  }
}
function formatDate(raw: string) {
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'short',
    timeStyle: 'short'
  }).format(new Date(raw))
}
const trustedLink = computed(() => {
  const s = selected.value,
    raw =
      s?.current_publication?.market_publication_url ||
      s?.receipt?.market_publication_url,
    m = markets.value.find((m) => m.connection_id === s?.connection_id)
  if (!raw || !m) return ''
  try {
    const base = new URL(m.console_base),
      target = new URL(raw)
    return target.origin === base.origin &&
      ['https:', 'http:'].includes(target.protocol) &&
      !target.username &&
      !target.password &&
      !target.hash &&
      (target.pathname === base.pathname ||
        target.pathname.startsWith(base.pathname.replace(/\/$/, '') + '/'))
      ? raw
      : ''
  } catch {
    return ''
  }
})
watch([selectedMarketID, versionID], () => {
  formOpen.value = false
  confirmOpen.value = false
})
for (const open of [formOpen, confirmOpen, historyOpen])
  watch(
    open,
    (value) => {
      if (!value && import.meta.client)
        (document.activeElement as HTMLElement | null)?.blur()
    },
    { flush: 'sync' }
  )
async function edit() {
  if (!preparation.value) await props.review.prepare()
  if (preparation.value) formOpen.value = true
}
function validationError(event: { errors: { name?: string }[] }) {
  validationFailed.value = true
  const name = event.errors[0]?.name
  formSection.value =
    name === 'security_statement' || name === 'privacy_statement'
      ? 'policy'
      : 'general'
}
async function showPreview() {
  validationFailed.value = false
  if (await props.review.previewInput()) {
    if (preview.value) {
      formOpen.value = false
      confirmOpen.value = true
    }
  }
}
async function submit() {
  confirmOpen.value = false
  await props.review.submit()
}
async function showHistory() {
  historyOpen.value = true
  await props.review.readHistory(history.value.page)
}
async function revise() {
  if (await props.review.revise()) {
    if (preparation.value) formOpen.value = true
  }
}
</script>
<template>
  <section class="publisher-review" :aria-label="t('review.title')">
    <div class="review-heading">
      <h2>{{ t('review.title') }}</h2>
      <UButton
        color="neutral"
        variant="outline"
        :disabled="!connected || !!busy"
        @click="void review.refresh()"
        >{{ t('review.refreshTasks') }}</UButton
      >
    </div>
    <p class="mb-5">{{ t('review.intro') }}</p>
    <div class="review-selectors">
      <UFormField :label="t('market.selectedConnection')"
        ><USelect
          v-model="selectedMarketID"
          :items="marketOptions"
          class="w-full"
          :disabled="!connected || !!busy"
          :placeholder="t('market.noConnection')"
      /></UFormField>
      <UFormField :label="t('review.version')"
        ><USelect
          v-model="versionID"
          :items="versionOptions"
          class="w-full"
          :disabled="!selectedMarket || !!busy"
          :placeholder="t('review.selectVersion')"
      /></UFormField>
    </div>
    <div class="review-actions mb-5">
      <UButton
        color="neutral"
        variant="outline"
        :disabled="!selectedMarket || !!busy || !!market.busy.value"
        @click="void market.readVersions()"
        >{{ t('market.readVersions') }}</UButton
      ><UButton
        color="neutral"
        variant="outline"
        :disabled="!selectedMarket || !!busy"
        @click="void market.verify()"
        >{{ t('market.verify') }}</UButton
      >
    </div>
    <div v-if="versionsLoaded" class="review-pagination">
      <p>
        {{
          t('review.pageSummary', { page: versionsPage, total: versionsTotal })
        }}
      </p>
      <UButton
        color="neutral"
        variant="outline"
        :disabled="versionsPage <= 1 || !!market.busy.value"
        @click="void market.readVersions(versionsPage - 1)"
        >{{ t('market.previous') }}</UButton
      ><UButton
        color="neutral"
        variant="outline"
        :disabled="versionsPage * 20 >= versionsTotal || !!market.busy.value"
        @click="void market.readVersions(versionsPage + 1)"
        >{{ t('market.next') }}</UButton
      >
    </div>
    <div v-if="!capable" role="status" class="review-notice">
      <p>{{ t('review.unavailable') }}</p>
      <p>{{ t('review.scopeHelp') }}</p>
    </div>
    <div v-if="error" role="alert" class="review-notice">
      <p>{{ errorText(error) }}</p>
      <p v-if="errorRequestID">
        {{ t('trace.serverID') }} <code>{{ errorRequestID }}</code>
      </p>
      <p v-if="needsReconcile">{{ t('unconfirmedWrite') }}</p>
    </div>
    <div class="review-card" data-testid="publisher-review-collection">
      <h3>{{ t('review.collection') }}</h3>
      <p v-if="!preparation">{{ t('review.collectionHelp') }}</p>
      <template v-else>
        <p>{{ preparation.version.version }}</p>
        <p data-testid="publisher-review-collection-status" role="status">
          {{ t('review.collectionStatus') }}
          {{
            collectionState?.source === 'publication'
              ? t(`review.statuses.${collectionState.status}`)
              : collectionState?.source === 'version'
                ? t(`market.availability.${collectionState.status}`)
                : t('review.collectionStatusUnknown')
          }}
        </p>
        <ul class="review-artifacts">
          <li v-for="a in preparation.version.artifacts" :key="a.artifact_id">
            <strong>{{ a.os }} / {{ a.arch }}</strong
            ><span>{{
              t('byteSize', {
                size: new Intl.NumberFormat(locale).format(a.size)
              })
            }}</span>
            <details>
              <summary>{{ t('review.integrity') }}</summary>
              <dl>
                <dt>{{ t('review.artifactReference') }}</dt>
                <dd>{{ a.artifact_id }}</dd>
                <dt>{{ t('sha256') }}</dt>
                <dd>{{ a.sha256 }}</dd>
              </dl>
            </details>
          </li>
        </ul>
        <details>
          <summary>{{ t('review.optionsSource') }}</summary>
          <p>{{ t(`review.sources.${preparation.options.version_source}`) }}</p>
          <code>{{ preparation.options.source }}</code>
        </details>
      </template>
      <p v-if="existing" class="mt-3">{{ t('review.existingTask') }}</p>
      <div class="review-actions mt-4">
        <UButton
          color="neutral"
          :disabled="
            !connected ||
            !capable ||
            !versionID ||
            !!busy ||
            needsReconcile ||
            !!existing
          "
          @click="edit"
          data-testid="publisher-review-edit"
          >{{ t('review.edit') }}</UButton
        ><UButton
          color="neutral"
          variant="outline"
          :disabled="!capable || !versionID || !!busy"
          @click="void review.prepare()"
          >{{ t('review.refreshCollection') }}</UButton
        >
      </div>
    </div>
    <UFormField :label="t('review.task')" class="mb-5"
      ><USelect
        v-model="selectedID"
        :items="taskOptions"
        :placeholder="t('review.noTasks')"
        :disabled="!tasks.length || !!busy"
        class="w-full"
    /></UFormField>
    <div
      v-if="selected"
      class="review-card"
      data-testid="publisher-review-task"
    >
      <h3>{{ selected.version }} · {{ taskStage(selected.stage) }}</h3>
      <p>
        {{
          markets.find((m) => m.connection_id === selected?.connection_id)
            ?.name || t('review.savedMarket')
        }}
      </p>
      <p v-if="selected.error" role="alert">
        {{ errorText(selected.error.code) }}
      </p>
      <p v-if="selected.error?.params?.request_id">
        {{ t('trace.serverID') }}
        <code>{{ selected.error.params.request_id }}</code>
      </p>
      <p v-if="selected.receipt">{{ t('review.receiptBoundary') }}</p>
      <template v-if="selected.current_publication"
        ><p>
          {{ t('review.currentStatus') }}
          {{ t(`review.statuses.${selected.current_publication.status}`) }}
        </p>
        <p v-if="selected.current_publication.status === 'approved'">
          {{ t('review.approvedPrivate') }}
        </p>
        <p
          v-if="selected.current_publication.review_reason"
          class="review-reason"
          role="status"
        >
          {{ selected.current_publication.review_reason }}
        </p></template
      >
      <div class="review-actions mt-4">
        <UButton
          color="neutral"
          variant="outline"
          :disabled="
            !!busy ||
            needsReconcile ||
            [
              'queued',
              'reconciling',
              'sending',
              'processing',
              'submitted'
            ].includes(selected.stage)
          "
          @click="void review.resume()"
          >{{
            t(selected.receipt ? 'review.refreshStatus' : 'review.resume')
          }}</UButton
        ><UButton
          v-if="selected.current_publication?.status === 'rejected'"
          color="neutral"
          :disabled="
            !!busy ||
            !capable ||
            selected.connection_id !== selectedMarketID ||
            needsReconcile
          "
          @click="revise"
          >{{ t('review.revise') }}</UButton
        ><UButton
          v-if="
            [
              'queued',
              'reconciling',
              'sending',
              'processing',
              'submitted'
            ].includes(selected.stage)
          "
          color="neutral"
          variant="outline"
          :disabled="!!busy"
          @click="void review.cancel(selected.submission_id)"
          >{{ t('review.cancel') }}</UButton
        ><UButton
          v-if="trustedLink"
          color="neutral"
          variant="outline"
          :to="trustedLink"
          target="_blank"
          rel="noopener noreferrer"
          >{{ t('review.openPublication') }}</UButton
        >
      </div>
      <details class="mt-4">
        <summary>{{ t('technicalDetails') }}</summary>
        <dl>
          <dt v-if="selected.receipt">
            {{ t('review.publicationReference') }}
          </dt>
          <dd v-if="selected.receipt">{{ selected.receipt.publication_id }}</dd>
          <dt>{{ t('review.versionReference') }}</dt>
          <dd>{{ selected.version_id }}</dd>
          <dt>{{ t('market.operationReference') }}</dt>
          <dd>{{ selected.operation_id }}</dd>
          <dt>{{ t('review.fingerprint') }}</dt>
          <dd>{{ selected.request_sha256 }}</dd>
        </dl>
        <RequestHistory
          :client="client"
          kind="submissions"
          :task-id="selected.submission_id"
        />
      </details>
    </div>
    <div v-if="pollPaused" class="review-notice">
      <p>{{ t('review.pollPaused') }}</p>
      <UButton color="neutral" @click="void review.resumePolling()">{{
        t('resumePolling')
      }}</UButton>
    </div>
    <UButton
      color="neutral"
      variant="outline"
      :disabled="
        !selectedMarket?.identity?.credential.scopes.includes(
          'publications:read'
        ) || !!busy
      "
      @click="showHistory"
      >{{ t('review.history') }}</UButton
    >
    <UModal
      v-model:open="historyOpen"
      :title="t('review.history')"
      :description="t('review.historyHelp')"
      :ui="{
        content: 'publisher-dialog max-w-3xl',
        title: 'publisher-dialog-title'
      }"
    >
      <template #close="{ close }"
        ><UButton
          color="neutral"
          variant="ghost"
          icon="i-lucide-x"
          :aria-label="t('close')"
          @click="void close()"
      /></template>
      <template #body
        ><p v-if="error" class="publisher-form-error" role="alert">
          {{ errorText(error) }}
        </p>
        <p v-if="errorRequestID" class="request-reference">
          {{ t('trace.serverID') }} <code>{{ errorRequestID }}</code>
        </p>
        <div class="review-card">
          <div class="review-heading">
            <h3>{{ t('review.history') }}</h3>
            <UButton
              color="neutral"
              variant="outline"
              :disabled="
                !selectedMarket?.identity?.credential.scopes.includes(
                  'publications:read'
                ) || !!busy
              "
              @click="void review.readHistory()"
              >{{ t('review.readHistory') }}</UButton
            >
          </div>
          <template v-if="historyLoaded"
            ><p v-if="!history.items.length">{{ t('review.noHistory') }}</p>
            <ul>
              <li v-for="p in history.items" :key="p.publication_id">
                {{ p.version }} · {{ t(`review.statuses.${p.status}`) }} ·
                {{ formatDate(p.submitted_at) }}
              </li>
            </ul>
            <p>
              {{
                t('review.pageSummary', {
                  page: history.page,
                  total: history.total
                })
              }}
            </p>
            <div class="review-actions">
              <UButton
                color="neutral"
                variant="outline"
                :disabled="history.page <= 1 || !!busy"
                @click="void review.readHistory(history.page - 1)"
                >{{ t('market.previous') }}</UButton
              ><UButton
                color="neutral"
                variant="outline"
                :disabled="history.page * 20 >= history.total || !!busy"
                @click="void review.readHistory(history.page + 1)"
                >{{ t('market.next') }}</UButton
              >
            </div></template
          >
        </div>
      </template>
    </UModal>
    <UModal
      v-model:open="formOpen"
      :title="t('review.metadataTitle')"
      :description="t('review.metadataHelp')"
      :ui="{
        content: 'publisher-dialog max-w-3xl',
        title: 'publisher-dialog-title'
      }"
    >
      <template #close="{ close }"
        ><UButton
          color="neutral"
          variant="ghost"
          icon="i-lucide-x"
          :aria-label="t('close')"
          @click="void close()"
      /></template>
      <template #body
        ><UForm
          :schema="schema"
          :state="draft"
          class="space-y-5"
          @submit="showPreview"
          @error="validationError"
        >
          <UTabs
            variant="link"
            color="neutral"
            v-model="formSection"
            :items="formTabs"
            :orientation="narrow ? 'horizontal' : 'vertical'"
            :ui="{
              root: narrow
                ? 'review-form-tabs'
                : 'review-form-tabs flex-row items-start gap-5',
              trigger: 'review-form-tab',
              indicator: 'hidden',
              list: narrow ? '' : 'w-44 shrink-0',
              label: 'whitespace-normal text-left',
              content: 'flex-1 min-w-0'
            }"
          >
            <template #general
              ><div class="space-y-5">
                <UFormField
                  name="description"
                  :label="t('review.fields.description')"
                  required
                  :ui="{ error: 'publisher-form-error' }"
                  ><UTextarea
                    v-model="draft.description"
                    :rows="5"
                    class="w-full" /></UFormField
                ><UFormField
                  name="license"
                  :label="t('review.fields.license')"
                  required
                  :ui="{ error: 'publisher-form-error' }"
                  ><USelect
                    v-model="draft.license"
                    :items="preparation?.options.licenses || []"
                    class="w-full"
                    :placeholder="t('review.selectChoice')"
                    :portal="false" /></UFormField
                ><UFormField
                  name="min_core"
                  :label="t('review.fields.min_core')"
                  required
                  :ui="{ error: 'publisher-form-error' }"
                  ><USelect
                    v-model="draft.min_core"
                    :items="preparation?.options.versions || []"
                    class="w-full"
                    :placeholder="t('review.selectChoice')"
                    :portal="false" /></UFormField
                ><UFormField
                  name="support_url"
                  :label="t('review.fields.support_url')"
                  required
                  :ui="{ error: 'publisher-form-error' }"
                  ><UInput
                    v-model="draft.support_url"
                    class="w-full" /></UFormField
                ><UFormField
                  name="icon_url"
                  :label="t('review.fields.icon_url')"
                  :ui="{ error: 'publisher-form-error' }"
                  ><UInput v-model="draft.icon_url" class="w-full"
                /></UFormField></div
            ></template>
            <template #policy
              ><div class="space-y-5">
                <UFormField
                  name="security_statement"
                  :label="t('review.fields.security_statement')"
                  required
                  :ui="{ error: 'publisher-form-error' }"
                  ><UTextarea
                    v-model="draft.security_statement"
                    :rows="5"
                    class="w-full" /></UFormField
                ><UFormField
                  name="privacy_statement"
                  :label="t('review.fields.privacy_statement')"
                  required
                  :ui="{ error: 'publisher-form-error' }"
                  ><UTextarea
                    v-model="draft.privacy_statement"
                    :rows="5"
                    class="w-full"
                /></UFormField></div
            ></template>
          </UTabs>
          <p v-if="validationFailed" role="alert" class="publisher-form-error">
            {{ t('review.validationMissing') }}
          </p>
          <p v-if="error" role="alert" class="publisher-form-error">
            {{ errorText(error) }}
          </p>
          <div class="review-actions">
            <UButton
              type="submit"
              color="neutral"
              :disabled="!!busy || !capable || needsReconcile"
              >{{ t('review.preview') }}</UButton
            ><UButton
              color="neutral"
              variant="outline"
              @click="void (formOpen = false)"
              >{{ t('close') }}</UButton
            >
          </div>
        </UForm></template
      >
    </UModal>
    <UModal
      v-model:open="confirmOpen"
      :title="t('review.confirmTitle')"
      :description="t('review.confirmHelp')"
      :ui="{
        content: 'publisher-dialog max-w-3xl',
        title: 'publisher-dialog-title'
      }"
    >
      <template #close="{ close }"
        ><UButton
          color="neutral"
          variant="ghost"
          icon="i-lucide-x"
          :aria-label="t('close')"
          @click="void close()"
      /></template>
      <template #body
        ><template v-if="preview"
          ><p>{{ selectedMarket?.name }} · {{ preview.input.version }}</p>
          <ul class="review-artifacts">
            <li
              v-for="a in preview.input.expected_artifacts"
              :key="a.artifact_id"
            >
              {{ a.os }} / {{ a.arch }} ·
              {{
                t('byteSize', {
                  size: new Intl.NumberFormat(locale).format(a.size)
                })
              }}
            </li>
          </ul>
          <details class="review-confirm-details">
            <summary>{{ t('review.normalizedMetadata') }}</summary>
            <dl>
              <template
                v-for="(value, key) in preview.input.metadata"
                :key="key"
                ><dt>{{ t(`review.fields.${key}`) }}</dt>
                <dd>{{ value || t('review.notProvided') }}</dd></template
              >
            </dl>
          </details>
          <p>{{ t('review.confirmBoundary') }}</p>
          <div class="review-actions mt-4">
            <UButton
              color="neutral"
              :disabled="!!busy || !capable || needsReconcile"
              @click="submit"
              data-testid="publisher-review-confirm"
              >{{ t('review.submit') }}</UButton
            ><UButton
              color="neutral"
              variant="outline"
              @click="void (confirmOpen = false)"
              >{{ t('close') }}</UButton
            >
          </div></template
        ></template
      >
    </UModal>
  </section>
</template>
<style scoped>
:global(.publisher-dialog button:disabled) {
  opacity: 1;
  color: var(--ui-text-toned);
  background: var(--ui-bg-elevated);
}
:global(.review-form-tab) {
  color: var(--ui-text-toned);
}
:global(.review-form-tab[data-state='active']) {
  color: var(--ui-text-highlighted);
  background: var(--ui-bg-elevated);
}
.publisher-review {
  color: var(--ui-text);
}
h2,
h3 {
  color: var(--ui-text-highlighted);
  font-weight: 600;
}
h2 {
  font-size: 1.25rem;
}
p + p {
  margin-top: 0.75rem;
}
.review-heading,
.review-actions,
.review-pagination {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
}
.review-heading {
  justify-content: space-between;
  margin-bottom: 1rem;
}
.review-pagination {
  margin-bottom: 1rem;
}
.review-selectors {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
  margin-bottom: 1rem;
}
.review-card,
.review-notice {
  padding: 1.25rem;
  border: 1px solid var(--ui-border);
  border-radius: 0.75rem;
  background: var(--ui-bg-elevated);
  margin-bottom: 1.25rem;
  overflow-wrap: anywhere;
}
.review-notice {
  border-left: 4px solid var(--ui-error);
}
.review-artifacts {
  display: grid;
  gap: 0.75rem;
  margin: 1rem 0;
}
.review-artifacts li {
  border: 1px solid var(--ui-border);
  border-radius: 0.5rem;
  padding: 0.75rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  justify-content: space-between;
}
.review-artifacts details {
  width: 100%;
}
dl {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}
dt {
  font-weight: 600;
  margin-top: 0.75rem;
  color: var(--ui-text-highlighted);
}
dd {
  color: var(--ui-text);
}
.review-confirm-details {
  margin: 1rem 0;
  max-height: 22rem;
  overflow: auto;
}
.review-reason {
  white-space: pre-wrap;
  max-height: 12rem;
  overflow: auto;
}
summary {
  cursor: pointer;
  color: var(--ui-text-highlighted);
}
@media (max-width: 767px) {
  .review-selectors {
    grid-template-columns: 1fr;
  }
}
</style>
