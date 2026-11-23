<script setup lang="ts">
import UButton from '@nuxt/ui/components/Button.vue'
import UDropdownMenu from '@nuxt/ui/components/DropdownMenu.vue'
import UForm from '@nuxt/ui/components/Form.vue'
import UFormField from '@nuxt/ui/components/FormField.vue'
import UIcon from '@nuxt/ui/components/Icon.vue'
import UInput from '@nuxt/ui/components/Input.vue'
import UModal from '@nuxt/ui/components/Modal.vue'
import UTabs from '@nuxt/ui/components/Tabs.vue'
import UTextarea from '@nuxt/ui/components/Textarea.vue'
import USelect from '@nuxt/ui/components/Select.vue'
import { useRoute, useRouter } from '#imports'

import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import * as v from 'valibot'
import { createPublisherClient, PublisherClientError } from '../client'
import { createPublisherWorkspace } from '../workspace'
import MarketPanel from './MarketPanel.vue'
import ReviewPanel from './ReviewPanel.vue'
import { createReviewWorkspace } from '../review-workspace'
import { createMarketWorkspace } from '../market-workspace'
import { activeBuildStages, type PublisherBuild, type PublisherPlan, type PublisherUpload } from '../types'
import en from '../locales/en.json'
import zh from '../locales/zh.json'

const { t, te } = useI18n({ useScope: 'local', messages: { en, zh } })
const { locale } = useI18n({ useScope: 'global' })
const route = useRoute(),
  router = useRouter()
const browserOrigin = import.meta.client ? window.location.origin : ''
const client = createPublisherClient(browserOrigin)
const workspace = createPublisherWorkspace(client)
const market = createMarketWorkspace(client)
const review = createReviewWorkspace(client, market.selectedMarket)
const reviewAttention = computed(() => review.attention.value.length)
const {
  error: marketErrorCode,
  errorRequestID: marketRequestID,
  ongoing: ongoingUploads,
  needsReconcile: marketReconcile,
  selectedMarket: currentMarket,
  canUpload: marketReady
} = market
const attentionUploads = computed(() =>
  market.uploads.value.filter((task) => task.error)
)
const marketErrorMessage = computed(() =>
  te(`errors.${marketErrorCode.value}`)
    ? t(`errors.${marketErrorCode.value}`)
    : t('errors.PUBLISHER_MARKET_PROTOCOL')
)
const {
  connection,
  executor,
  environment,
  executorUpdateRequired,
  needsReconcile,
  plans,
  builds,
  selectedPlan,
  selectedBuild,
  ongoingBuild,
  error,
  busy,
  running,
  canBuild,
  canPrepareContainer,
  verified,
  pollPaused
} = workspace
const buildMethods = computed(() => [
  { value: 'container', label: t('containerExecutor'), available: environment.value?.container_available },
  { value: 'native', label: t('nativeExecutor'), available: environment.value?.native_available }
].filter(method => method.available || method.value === executor.value))
const canStartBuild = computed(() => canBuild.value || canPrepareContainer.value || busy.value === 'prepare-build')
const buildSetupStatus = computed(() => {
  if (busy.value === 'reconnect') return t('workflow.reconnectingTools')
  if (executorUpdateRequired.value) return t('workflow.toolUpdateHelp')
  if (busy.value === 'prepare-build') return t('containerPreparingBuild')
  if (canStartBuild.value) return ''
  if (executor.value === 'native') return t('nativeBuildUnavailable')
  if (!environment.value?.container_available) return t('containerMissing')
  if (environment.value.container_ready) return t('platformUnavailable')
  return t('containerCheckHelp')
})
const buildSetupActions = computed(() => [
  ...(selectedBuild.value
    ? [{ label: t('logs'), icon: 'i-lucide-scroll-text', onSelect: () => showDetail('logs') }]
    : []),
  { label: t('workflow.buildSettings'), icon: 'i-lucide-settings-2', onSelect: () => showDetail('build-settings') },
  ...(builds.value.total
    ? [{ label: t('buildHistory'), icon: 'i-lucide-history', onSelect: () => showDetail('history') }]
    : [])
])
watch([() => market.error.value, () => review.error.value], codes => {
  if (codes.includes('PUBLISHER_UNAUTHORIZED')) connection.value = null
})
watch(connection, (value, previous) => {
  if (value && (!previous || value.expires_at !== previous.expires_at || value.info.plugin.plugin_id !== previous.info.plugin.plugin_id)) {
    void market.refresh()
    void review.refresh()
  }
})
const section = ref(
  ['versions', 'build', 'upload', 'review', 'connection'].includes(
    String(route.query.section)
  )
    ? String(route.query.section)
    : 'versions'
)
const workspaceElement = ref<HTMLElement | null>(null)
const narrow = ref(false),
  createOpen = ref(false),
  detail = ref(''),
  detailOpen = ref(false),
  detailBusy = ref(false),
  logs = ref('')
const draft = reactive({ version: '', release_notes: '' })
const schema = computed(() =>
  v.object({
    version: v.pipe(
      v.string(),
      v.maxLength(50, t('validation.version')),
      v.regex(
        /^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$/,
        t('validation.version')
      )
    ),
    release_notes: v.pipe(
      v.string(),
      v.check(
        (value) => new TextEncoder().encode(value).length <= 10000,
        t('validation.notes')
      )
    )
  })
)
const tabs = computed(() => [
  {
    value: 'versions',
    slot: 'versions',
    label: t(narrow.value ? 'compactTabs.versions' : 'tabs.versions'),
    icon: 'i-lucide-layers'
  },
  {
    value: 'build',
    slot: 'build',
    label: t(narrow.value ? 'compactTabs.build' : 'tabs.build'),
    icon: 'i-lucide-package'
  },
  {
    value: 'upload',
    slot: 'upload',
    label: t(narrow.value ? 'compactTabs.upload' : 'tabs.upload'),
    icon: 'i-lucide-cloud-upload'
  },
  {
    value: 'review',
    slot: 'review',
    label: t(narrow.value ? 'compactTabs.review' : 'tabs.review'),
    icon: 'i-lucide-send'
  },
  {
    value: 'connection',
    slot: 'connection',
    label: t(narrow.value ? 'compactTabs.connection' : 'tabs.connection'),
    icon: 'i-lucide-plug',
    ui: { trigger: 'publisher-tab-connection' }
  }
])
const stage = computed(() => selectedBuild.value?.stage || 'none')
const buildActive = computed(() => activeBuildStages.includes(stage.value))
const buildReady = computed(() => stage.value === 'ready' && !!selectedBuild.value?.artifact)
const buildSteps = activeBuildStages.filter((step) => step !== 'queued')
const buildStepIndex = computed(() => buildSteps.indexOf(stage.value))
const currentUpload = computed(() =>
  market.uploads.value.find((task) =>
    task.build_id === selectedBuild.value?.build_id &&
    task.connection_id === market.selectedMarketID.value &&
    task.stage !== 'failed'
  )
)
const packageUploaded = computed(() =>
  currentUpload.value?.stage === 'succeeded' && !!currentUpload.value.receipt
)
const completedSteps = computed(() => new Set([
  ...(selectedPlan.value ? ['versions'] : []),
  ...(buildReady.value ? ['build'] : []),
  ...(packageUploaded.value ? ['upload'] : [])
]))
const packageNextLabel = computed(() => {
  if (packageUploaded.value) return 'workflow.nextReview'
  if (currentUpload.value) return 'market.inspect'
  return marketReady.value ? 'workflow.uploadMarket' : 'workflow.connectMarket'
})
const packageNextHint = computed(() => {
  if (packageUploaded.value) return t('workflow.uploadReceived')
  if (currentUpload.value) return t(`market.stages.${currentUpload.value.stage}`)
  if (!currentMarket.value) return t('workflow.marketRequired')
  if (!currentMarket.value.identity) return t('workflow.marketTokenRequired')
  if (!marketReady.value) return t('workflow.marketAuthorizationRequired')
  return t('workflow.uploadTarget', { market: currentMarket.value.name })
})
const showBuildSetup = computed(() =>
  !!connection.value && !!selectedPlan.value && !buildActive.value && !buildReady.value
)
const stageLabel = (value: string) =>
  te(`stages.${value}`) ? t(`stages.${value}`) : t('stages.unknown')
const errorMessage = computed(() =>
  error.value
    ? te(`errors.${error.value}`)
      ? t(`errors.${error.value}`)
      : t('errors.PUBLISHER_PROXY_RESPONSE_INVALID')
    : ''
)
const taskError = computed(() =>
  selectedBuild.value?.error?.code &&
  te(`errors.${selectedBuild.value.error.code}`)
    ? t(`errors.${selectedBuild.value.error.code}`)
    : ''
)
const date = (value: string) =>
  new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'medium',
    timeStyle: 'short'
  }).format(new Date(value))
const size = (value: number) =>
  t('byteSize', { size: new Intl.NumberFormat(locale.value).format(value) })
const packageSize = (value: number) =>
  new Intl.NumberFormat(locale.value, {
    style: 'unit', unit: 'megabyte', maximumFractionDigits: 1
  }).format(value / 1_000_000)
const uuid =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/
const queryID = (value: unknown) =>
  typeof value === 'string' && uuid.test(value) ? value : undefined
const connectExecutor = async () => {
  await workspace.connect(queryID(route.query.plan), queryID(route.query.build))
}
function showUpload(id: string) {
  market.selectedUploadID.value = id
  section.value = 'upload'
}
function continueToUpload() {
  showUpload(currentUpload.value?.upload_id || '')
}
function continueWithPackage() {
  const upload = currentUpload.value
  if (upload?.stage === 'succeeded' && upload.receipt) void continueToReview(upload)
  else if (!upload && !marketReady.value) section.value = 'connection'
  else continueToUpload()
}
function viewPackage() {
  section.value = 'build'
}
async function continueToReview(upload: PublisherUpload) {
  if (upload.stage !== 'succeeded' || !upload.receipt) return
  market.selectedMarketID.value = upload.connection_id
  await nextTick()
  review.versionID.value = upload.receipt.version_id
  const task = review.tasks.value.find((item) =>
    item.connection_id === upload.connection_id && item.version_id === upload.receipt?.version_id
  )
  review.selectedID.value = task?.submission_id || ''
  section.value = 'review'
}
function showReview() {
  section.value = 'review'
  if (review.attention.value.length)
    review.selectedID.value = review.attention.value[0]!.submission_id
}
let resizeObserver: ResizeObserver | undefined
let restoringRoute = true
onMounted(async () => {
  resizeObserver = new ResizeObserver(([entry]) => {
    if (entry) narrow.value = entry.contentRect.width < 768
  })
  if (workspaceElement.value) resizeObserver.observe(workspaceElement.value)
  market.selectedMarketID.value = queryID(route.query.market) ?? ''
  const restored = await workspace.connect(queryID(route.query.plan), queryID(route.query.build))
  restoringRoute = false
  if (restored && workspaceElement.value) syncRoute()
})
onUnmounted(() => {
  workspace.dispose()
  market.dispose()
  review.dispose()
  resizeObserver?.disconnect()
})
for (const open of [createOpen, detailOpen])
  watch(
    open,
    (value) => {
      if (!value && import.meta.client)
        (document.activeElement as HTMLElement | null)?.blur()
    },
    { flush: 'sync' }
  )
function syncRoute() {
  if (restoringRoute) return
  void router.replace({
    query: {
      ...route.query,
      section: section.value,
      plan: selectedPlan.value?.release_plan_id,
      build: selectedBuild.value?.build_id,
      market: market.selectedMarketID.value || undefined
    }
  })
}
watch([
  section,
  () => selectedPlan.value?.release_plan_id,
  () => selectedBuild.value?.build_id,
  market.selectedMarketID
], syncRoute)
async function create() {
  if (
    await workspace.create({
      ...draft,
      channel: 'stable',
      target: { os: 'linux', arch: 'amd64' }
    })
  ) {
    createOpen.value = false
    draft.version = ''
    draft.release_notes = ''
    section.value = 'build'
  }
}
async function choosePlan(plan: PublisherPlan) {
  if (await workspace.selectPlan(plan)) detailOpen.value = false
}
async function chooseBuild(build: PublisherBuild) {
  if (await workspace.selectBuild(build)) detailOpen.value = false
}
async function showDetail(kind: string) {
  detail.value = kind
  detailOpen.value = true
  if (kind !== 'logs' || !selectedBuild.value) return
  logs.value = ''
  detailBusy.value = true
  try {
    logs.value = (await client.logs(selectedBuild.value.build_id)).text
  } catch (cause) {
    error.value =
      cause instanceof PublisherClientError
        ? cause.code
        : 'PUBLISHER_PROXY_UNAVAILABLE'
  } finally {
    detailBusy.value = false
  }
}
async function showOngoingBuild() {
  if (!ongoingBuild.value) return
  if (await workspace.selectBuild(ongoingBuild.value)) section.value = 'build'
}
</script>

<template>
  <section ref="workspaceElement" class="publisher-workspace" data-testid="publisher-workspace" aria-labelledby="publisher-title">
    <header class="publisher-header">
      <h1 id="publisher-title">{{ t('title') }}</h1>
      <div class="publisher-context">
        <strong data-testid="publisher-plugin">{{ connection?.info.plugin.name || t('notConnected') }}</strong>
        <span v-if="selectedPlan && section !== 'versions'" class="publisher-version" data-testid="publisher-selected-version">
          {{ t('workflow.releaseVersion', { version: selectedPlan.version }) }}
        </span>
      </div>
    </header>

    <div
      v-if="error && (section !== 'build' || !showBuildSetup || needsReconcile)"
      class="publisher-notice publisher-error"
      role="alert"
      data-testid="publisher-error"
    >
      <strong>{{ t('operationFailed') }}</strong>
      <p>{{ errorMessage }}</p>
      <p v-if="needsReconcile">{{ t('failureRecovery') }}</p>
      <p v-if="needsReconcile">{{ t('unconfirmedWrite') }}</p>
      <UButton
        color="neutral"
        variant="outline"
        :disabled="!connection || !!busy"
        @click="void workspace.refresh()"
        data-testid="publisher-reconcile"
        >{{ t('refresh') }}</UButton
      >
    </div>
    <div
      v-if="marketErrorCode"
      class="publisher-notice publisher-error"
      role="alert"
    >
      <p>
        {{ marketErrorMessage }}
        <code v-if="marketRequestID"
          >{{ t('trace.serverID') }} {{ marketRequestID }}</code
        >
      </p>
      <p v-if="marketReconcile">{{ t('unconfirmedWrite') }}</p>
      <UButton
        color="neutral"
        :disabled="!connection || !!market.busy.value"
        @click="void market.refresh()"
        >{{ t('refresh') }}</UButton
      >
    </div>
    <div
      v-for="task in ongoingUploads"
      :key="task.upload_id"
      class="publisher-notice"
      role="status"
    >
      <strong>{{ t('market.ongoing', { version: task.version }) }}</strong>
      <p>{{ t(`market.stages.${task.stage}`) }}</p>
      <UButton
        color="neutral"
        variant="outline"
        :disabled="!!market.busy.value"
        @click="void market.cancel(task.upload_id)"
        >{{ t('market.cancel') }}</UButton
      >
    </div>
    <div
      v-for="task in attentionUploads"
      :key="`attention-${task.upload_id}`"
      class="publisher-notice publisher-error"
      role="alert"
    >
      <p>{{ t('market.attention', { version: task.version }) }}</p>
      <UButton
        color="neutral"
        variant="outline"
        @click="void showUpload(task.upload_id)"
        >{{ t('market.inspect') }}</UButton
      >
    </div>
    <div
      v-if="review.error.value || reviewAttention"
      role="alert"
      class="publisher-notice publisher-error"
    >
      <p v-if="review.error.value">
        {{
          te(`errors.${review.error.value}`)
            ? t(`errors.${review.error.value}`)
            : t('errors.PUBLISHER_MARKET_PROTOCOL')
        }}
      </p>
      <p v-if="reviewAttention">
        {{ t('review.attention', { count: reviewAttention }) }}
      </p>
      <UButton color="neutral" variant="outline" @click="showReview">{{
        t('review.inspect')
      }}</UButton>
    </div>
    <div
      v-if="review.ongoing.value.length"
      role="status"
      class="publisher-notice"
    >
      <p>{{ t('review.ongoing', { count: review.ongoing.value.length }) }}</p>
      <UButton color="neutral" variant="outline" @click="showReview">{{
        t('review.inspect')
      }}</UButton>
    </div>
    <div v-if="taskError && section !== 'build'" class="publisher-notice publisher-error" role="alert">
      {{ taskError }}
    </div>
    <div v-if="!connection" class="publisher-notice" role="status" data-testid="publisher-executor-connection">
      <div>
        <strong>{{ t(busy === 'connect' ? 'managedStarting' : 'managedUnavailable') }}</strong>
        <p>{{ t('managedDescription') }}</p>
      </div>
      <div class="publisher-actions">
        <UButton
          color="neutral"
          :loading="busy === 'connect'"
          :disabled="!!busy"
          @click="connectExecutor"
          data-testid="publisher-connect"
          >{{ t('managedRetry') }}</UButton>
      </div>
    </div>
    <div
      v-if="busy === 'prepare-build' && section !== 'build'"
      class="publisher-notice"
      role="status"
    >
      {{ t('containerPreparingBuild') }}
    </div>
    <div
      v-if="running && (section !== 'build' || ongoingBuild?.build_id !== selectedBuild?.build_id)"
      class="publisher-notice"
      role="status"
      data-testid="publisher-running-summary"
    >
      <div>
        <strong>{{
          t('runningSummary', {
            version: ongoingBuild?.version,
            stage: stageLabel(ongoingBuild?.stage || 'unknown')
          })
        }}</strong>
        <p v-if="pollPaused">{{ t('pollPaused') }}</p>
      </div>
      <div class="publisher-actions">
        <UButton
          v-if="ongoingBuild"
          color="neutral"
          variant="outline"
          :disabled="!!busy"
          @click="showOngoingBuild"
          >{{ t('details') }}</UButton
        ><UButton
          v-if="pollPaused"
          color="neutral"
          variant="outline"
          @click="void workspace.resume()"
          >{{ t('resume') }}</UButton
        ><UButton
          color="neutral"
          variant="outline"
          :disabled="!!busy"
          @click="void workspace.cancel()"
          data-testid="publisher-cancel"
          >{{ t('cancelBuild') }}</UButton
        >
      </div>
    </div>

    <UTabs
      v-model="section"
      :items="tabs"
      :orientation="narrow ? 'horizontal' : 'vertical'"
      :unmount-on-hide="false"
      variant="link"
      color="neutral"
      :ui="{
        root: 'publisher-tabs',
        list: 'publisher-tab-list',
        trigger: 'publisher-tab',
        content: 'publisher-panel',
        indicator: 'hidden'
      }"
    >
      <template #list-leading>
        <p v-if="!narrow" class="publisher-rail-label">{{ t('workflow.releaseSteps') }}</p>
      </template>
      <template #leading="{ item }">
        <span
          class="publisher-nav-icon"
          :data-complete="completedSteps.has(item.value || '') && section !== item.value"
        >
          <UIcon
            :name="completedSteps.has(item.value || '') && section !== item.value ? 'i-lucide-check' : item.icon"
            aria-hidden="true"
          />
        </span>
      </template>
      <template #trailing="{ item }">
        <span v-if="completedSteps.has(item.value || '') && section !== item.value" class="sr-only">
          {{ t('workflow.stepComplete') }}
        </span>
      </template>

      <template #versions>
        <section v-if="connection && busy !== 'connect'" :aria-label="t('tabs.versions')">
          <div class="publisher-section-heading">
            <h2>{{ t('workflow.chooseReleaseContent') }}</h2>
            <UButton
              v-if="plans.total > 1"
              color="neutral"
              variant="outline"
              @click="void showDetail('versions')"
              >{{ t('workflow.switchVersion') }}</UButton
            >
          </div>

          <div class="publisher-version-choices" :class="{ 'has-plan': selectedPlan }">
            <article v-if="selectedPlan" class="publisher-version-choice">
              <div class="publisher-choice-heading">
                <span class="publisher-choice-icon" :data-ready="buildReady" aria-hidden="true">
                  <UIcon :name="buildReady ? 'i-lucide-package-check' : 'i-lucide-package'" />
                </span>
                <div>
                  <h3>{{ t(buildReady ? 'workflow.builtPackage' : 'workflow.existingPlan') }}</h3>
                  <p class="publisher-choice-meta">
                    <strong>{{ selectedPlan.version }}</strong>
                    <span>{{ stageLabel(stage) }}</span>
                  </p>
                </div>
              </div>
              <p class="publisher-muted">
                {{ t(buildReady ? 'workflow.frozenChoice' : 'workflow.unfinishedChoice') }}
              </p>
              <div class="publisher-choice-actions">
                <UButton
                  color="neutral"
                  variant="outline"
                  trailing-icon="i-lucide-arrow-right"
                  :disabled="!!busy"
                  @click="viewPackage"
                  data-testid="publisher-view-package"
                  >{{ t(buildReady ? 'workflow.openVersionPackage' : 'workflow.openVersionBuild', { version: selectedPlan.version }) }}</UButton
                >
                <UButton
                  v-if="selectedPlan.release_notes"
                  color="neutral"
                  variant="ghost"
                  @click="void showDetail('plan')"
                  >{{ t('releaseNotes') }}</UButton
                >
              </div>
            </article>

            <article class="publisher-version-choice">
              <div class="publisher-choice-heading">
                <span class="publisher-choice-icon" aria-hidden="true">
                  <UIcon name="i-lucide-file-plus-2" />
                </span>
                <h3>{{ t('workflow.releaseCurrentCode') }}</h3>
              </div>
              <p class="publisher-muted">{{ t('workflow.versionHelp') }}</p>
              <div class="publisher-choice-actions">
                <UButton
                  color="neutral"
                  :variant="selectedPlan ? 'outline' : 'solid'"
                  trailing-icon="i-lucide-arrow-right"
                  :disabled="!!busy || needsReconcile"
                  @click="createOpen = true"
                  data-testid="publisher-create-open"
                  >{{ t('workflow.setNewVersion') }}</UButton
                >
              </div>
            </article>
          </div>
        </section>
      </template>

      <template #build>
        <section :aria-label="t('tabs.build')">
          <div v-if="!selectedPlan && connection" class="publisher-card">
            <p>{{ t('selectPlanFirst') }}</p>
            <UButton color="neutral" @click="section = 'versions'">
              {{ t('workflow.chooseVersion') }}
            </UButton>
          </div>

          <article v-else-if="showBuildSetup && selectedPlan" class="publisher-card publisher-build-card" data-testid="publisher-build-setup">
            <div class="publisher-build-heading">
              <h2>{{ t('workflow.generatePackage', { version: selectedPlan.version }) }}</h2>
              <p class="publisher-muted">{{ t('workflow.linuxAmd64') }}</p>
            </div>
            <div v-if="executorUpdateRequired || buildSetupStatus || taskError || (errorMessage && !needsReconcile)" class="publisher-build-status">
              <h3 v-if="executorUpdateRequired || (selectedBuild && canStartBuild)" data-testid="publisher-stage">
                {{ executorUpdateRequired ? t('workflow.toolUpdateTitle') : t('workflow.previousBuildStatus', { status: stageLabel(stage) }) }}
              </h3>
              <p v-if="errorMessage && !needsReconcile" role="alert">{{ errorMessage }}</p>
              <p v-else role="status">{{ buildSetupStatus || taskError }}</p>
            </div>
            <div class="publisher-actions">
              <UButton
                v-if="executorUpdateRequired"
                color="neutral"
                icon="i-lucide-refresh-cw"
                :loading="busy === 'reconnect'"
                :disabled="!!busy || running || needsReconcile"
                @click="void workspace.reconnect()"
                data-testid="publisher-executor-reconnect"
                >{{ t('workflow.reconnectTools') }}</UButton>
              <UButton
                v-else-if="canStartBuild"
                color="neutral"
                :loading="busy === 'run' || busy === 'prepare-build'"
                :disabled="!!busy || running || needsReconcile"
                @click="void workspace.run()"
                data-testid="publisher-build-run"
                >{{ t(selectedBuild ? 'workflow.retryBuild' : 'workflow.startBuild') }}</UButton>
              <UButton
                v-else
                color="neutral"
                :loading="busy === 'environment'"
                :disabled="!!busy || running"
                @click="void workspace.checkEnvironment()"
                >{{ t('containerCheck') }}</UButton>
              <UDropdownMenu :items="buildSetupActions">
                <UButton
                  color="neutral"
                  variant="ghost"
                  trailing-icon="i-lucide-chevron-down"
                  :aria-label="t('workflow.moreBuildActions')"
                  data-testid="publisher-build-setup-actions"
                  >{{ t('workflow.moreActions') }}</UButton>
              </UDropdownMenu>
            </div>
          </article>

          <article
            v-else-if="buildReady && selectedBuild?.artifact"
            class="publisher-package"
            data-testid="publisher-current-build"
          >
            <div class="publisher-package-summary">
              <span class="publisher-package-icon" aria-hidden="true">
                <UIcon name="i-lucide-package-check" />
              </span>
              <div>
                <h2 data-testid="publisher-stage">{{ stageLabel(stage) }}</h2>
              </div>
            </div>
            <dl class="publisher-package-facts">
              <div>
                <dt>{{ t('workflow.packagePlatform') }}</dt>
                <dd>{{ t('workflow.linuxAmd64') }}</dd>
              </div>
              <div>
                <dt>{{ t('workflow.packageSize') }}</dt>
                <dd><data :value="selectedBuild.artifact.size">{{ packageSize(selectedBuild.artifact.size) }}</data></dd>
              </div>
              <div>
                <dt>{{ t('workflow.packageBuiltAt') }}</dt>
                <dd><time :datetime="selectedBuild.updated_at">{{ date(selectedBuild.updated_at) }}</time></dd>
              </div>
            </dl>
            <aside class="publisher-package-next" :aria-label="t('workflow.nextStep')">
              <UButton
                color="neutral"
                trailing-icon="i-lucide-arrow-right"
                :disabled="!connection || !!market.busy.value"
                @click="continueWithPackage"
                data-testid="publisher-package-next"
                >{{ t(packageNextLabel) }}</UButton
              >
              <p class="publisher-muted">{{ packageNextHint }}</p>
            </aside>
            <div class="publisher-package-tools">
              <div class="publisher-actions">
                <UButton
                  color="neutral"
                  variant="ghost"
                  icon="i-lucide-file-archive"
                  @click="void showDetail('artifact')"
                  >{{ t('artifactDetails') }}</UButton
                >
                <UButton
                  color="neutral"
                  variant="ghost"
                  icon="i-lucide-shield-check"
                  :disabled="!connection || !!busy"
                  :loading="busy === 'verify'"
                  @click="void workspace.verify()"
                  data-testid="publisher-verify"
                  >{{ t('verify') }}</UButton
                >
                <UDropdownMenu
                  :items="[
                    { label: t('buildHistory'), icon: 'i-lucide-history', onSelect: () => showDetail('history') },
                    { label: t('logs'), icon: 'i-lucide-scroll-text', onSelect: () => showDetail('logs') },
                    { label: t('workflow.newBuild'), icon: 'i-lucide-rotate-cw', disabled: !connection || !canStartBuild || !!busy || running || needsReconcile || !!currentUpload, onSelect: () => workspace.run() }
                  ]"
                >
                  <UButton
                    color="neutral"
                    variant="ghost"
                    trailing-icon="i-lucide-chevron-down"
                    :aria-label="t('workflow.moreBuildActions')"
                    data-testid="publisher-build-actions"
                  >{{ t('workflow.moreActions') }}</UButton>
                </UDropdownMenu>
              </div>
              <p v-if="verified" role="status" class="publisher-muted" data-testid="publisher-verified">
                {{ t('verified') }}
              </p>
            </div>
          </article>

          <div
            v-else-if="selectedBuild && buildActive"
            class="publisher-card"
            data-testid="publisher-current-build"
          >
            <div class="publisher-card-heading">
              <h3>{{ t('currentTask', { version: selectedBuild.version }) }}</h3>
              <strong data-testid="publisher-stage" role="status">{{ stageLabel(stage) }}</strong>
            </div>
            <ol class="publisher-build-steps" :aria-label="t('workflow.buildProgress')">
              <li
                v-for="(step, index) in buildSteps"
                :key="step"
                :aria-current="index === buildStepIndex ? 'step' : undefined"
                :data-complete="index < buildStepIndex"
              >
                <UIcon
                  class="publisher-step-marker"
                  :name="index < buildStepIndex ? 'i-lucide-check' : index === buildStepIndex ? 'i-lucide-circle-dot' : 'i-lucide-circle'"
                  aria-hidden="true"
                />
                {{ t(`workflow.buildSteps.${step}`) }}
              </li>
            </ol>
            <p v-if="pollPaused" role="status">{{ t('pollPaused') }}</p>
            <div class="publisher-actions">
              <UButton
                v-if="pollPaused"
                color="neutral"
                @click="void workspace.resume()"
                >{{ t('resume') }}</UButton
              >
              <UButton
                color="neutral"
                variant="outline"
                :disabled="!!busy || ongoingBuild?.build_id !== selectedBuild.build_id"
                @click="void workspace.cancel()"
                data-testid="publisher-cancel"
                >{{ t('cancelBuild') }}</UButton
              >
              <UButton color="neutral" variant="ghost" @click="void showDetail('logs')" data-testid="publisher-logs">
                {{ t('logs') }}
              </UButton>
            </div>
          </div>
        </section>
      </template>

      <template #upload
        ><MarketPanel
          :client="client"
          mode="upload"
          :market="market"
          :build="selectedBuild"
          :connected="!!connection"
          @build="section = 'build'"
          @connection="section = 'connection'"
          @review="continueToReview"
      /></template>

      <template #review
        ><ReviewPanel
          :client="client"
          :market="market"
          :review="review"
          :connected="!!connection"
          :narrow="narrow"
      /></template>
      <template #connection
        ><section :aria-label="t('tabs.connection')">
          <MarketPanel
            :client="client"
            mode="connection"
            :market="market"
            :build="selectedBuild"
            :connected="!!connection"
            @build="section = 'build'"
            @upload="continueToUpload"
          />
          <details class="publisher-executor-details">
            <summary>{{ t('localExecutor') }}</summary>
            <p>{{ connection ? t('connected') : t('notConnected') }}</p>
            <p v-if="connection">
              {{ t('sessionExpiry', { time: date(connection.expires_at) }) }}
            </p>
            <div class="publisher-actions">
              <UButton
                color="neutral"
                :disabled="!!busy"
                @click="connectExecutor"
                >{{ t('connect') }}</UButton
              >
            </div>
          </details>
        </section
      ></template>
    </UTabs>

    <UModal
      v-model:open="createOpen"
      :title="t('workflow.setNewVersion')"
      :description="t('createDescription')"
      :close="{ 'aria-label': t('close'), 'data-testid': 'publisher-create-close' }"
      :ui="{
        content: 'publisher-dialog publisher-version-dialog max-w-md',
        header: 'px-5 py-4',
        wrapper: 'pr-9',
        title: 'publisher-dialog-title text-lg font-semibold',
        description: 'mt-1 text-toned',
        body: 'p-5',
        footer: 'justify-end border-t border-default px-5 py-3'
      }"
    >
      <template #body
        ><UForm
          id="publisher-new-version-form"
          :schema="schema"
          :state="draft"
          :validate-on="['input']"
          :validate-on-input-delay="0"
          class="space-y-4"
          @submit="create"
        >
          <UFormField
            :label="t('workflow.packageVersion')"
            name="version"
            :ui="{ error: 'publisher-form-error' }"
            ><UInput
              v-model="draft.version"
              class="w-full"
              color="neutral"
              autocomplete="off"
              spellcheck="false"
              :maxlength="50"
              data-testid="publisher-version-input"
          /></UFormField>
          <UFormField
            :label="t('workflow.optionalNotes')"
            name="release_notes"
            :ui="{ error: 'publisher-form-error' }"
            ><UTextarea
              v-model="draft.release_notes"
              class="w-full"
              color="neutral"
              :rows="3"
              data-testid="publisher-notes-input"
            />
          </UFormField
          >
          <p v-if="error" class="publisher-form-error" role="alert">
            {{ errorMessage }}
          </p>
        </UForm></template
      >
      <template #footer>
        <UButton
          color="neutral"
          variant="ghost"
          type="button"
          @click="createOpen = false"
          >{{ t('workflow.back') }}</UButton
        >
        <UButton
          color="neutral"
          type="submit"
          form="publisher-new-version-form"
          :loading="busy === 'create'"
          :disabled="!!busy || needsReconcile"
          data-testid="publisher-create-submit"
          >{{ t('workflow.continueToBuild') }}</UButton
        >
      </template>
    </UModal>
    <UModal
      v-model:open="detailOpen"
      :title="t(`detailTitles.${detail || 'plan'}`)"
      :description="detail === 'build-settings' && selectedPlan
        ? t('workflow.releaseVersion', { version: selectedPlan.version })
        : t('detailDescription')"
      :close="{ 'aria-label': t('close') }"
      :ui="{
        content: detail === 'build-settings' ? 'publisher-dialog max-w-md' : 'publisher-dialog max-w-3xl',
        wrapper: 'pr-8',
        title: 'publisher-dialog-title'
      }"
    >
      <template #body>
        <template v-if="detail === 'build-settings'">
          <dl class="publisher-build-facts">
            <div>
              <dt>{{ t('workflow.packagePlatform') }}</dt>
              <dd>{{ t('workflow.linuxAmd64') }}</dd>
            </div>
            <div>
              <dt>{{ t('workflow.releaseChannel') }}</dt>
              <dd>{{ t('workflow.stableChannel') }}</dd>
            </div>
            <div v-if="buildMethods.length === 1">
              <dt>{{ t('buildExecutor') }}</dt>
              <dd data-testid="publisher-build-method">{{ buildMethods[0]?.label }}</dd>
            </div>
          </dl>
          <UFormField v-if="buildMethods.length > 1" :label="t('buildExecutor')">
            <USelect
              v-model="executor"
              :items="buildMethods"
              :portal="false"
              :disabled="!!busy || running"
              color="neutral"
              class="w-full"
              data-testid="publisher-executor-select"
            />
          </UFormField>
          <p v-if="canPrepareContainer" class="publisher-muted mt-3 text-sm">
            {{ t('containerStartsWithBuild') }}
          </p>
        </template>
        <template v-else-if="detail === 'versions'">
          <ul v-if="plans.items.length" class="publisher-records">
            <li
              v-for="plan in plans.items"
              :key="plan.release_plan_id"
              :class="{
                selected: selectedPlan?.release_plan_id === plan.release_plan_id
              }"
            >
              <div>
                <strong>{{ plan.version }}</strong>
                <p class="publisher-muted">
                  {{ date(plan.created_at) }} · {{ t('target') }}
                </p>
              </div>
              <div class="publisher-actions">
                <UButton
                  color="neutral"
                  variant="outline"
                  :disabled="!!busy"
                  :aria-label="t('selectVersion', { version: plan.version })"
                  @click="void choosePlan(plan)"
                  >{{
                    selectedPlan?.release_plan_id === plan.release_plan_id
                      ? t('selected')
                      : t('select')
                  }}</UButton
                >
              </div>
            </li>
          </ul>
          <div v-if="plans.total > plans.size" class="publisher-pagination">
            <UButton
              color="neutral"
              variant="outline"
              :disabled="plans.page === 1 || !!busy"
              @click="void workspace.pagePlans(plans.page - 1)"
              >{{ t('previous') }}</UButton
            ><span>{{
              t('page', {
                page: plans.page,
                pages: Math.ceil(plans.total / plans.size)
              })
            }}</span
            ><UButton
              color="neutral"
              variant="outline"
              :disabled="plans.page * plans.size >= plans.total || !!busy"
              @click="void workspace.pagePlans(plans.page + 1)"
              >{{ t('next') }}</UButton
            >
          </div>
        </template>
        <template v-else-if="detail === 'history'">
          <p v-if="!builds.items.length" class="publisher-empty">
            {{ t('noBuilds') }}
          </p>
          <ul v-else class="publisher-records">
            <li v-for="build in builds.items" :key="build.build_id">
              <div>
                <strong
                  >{{ build.version }} · {{ stageLabel(build.stage) }}</strong
                >
                <p class="publisher-muted">{{ date(build.created_at) }}</p>
              </div>
              <UButton
                color="neutral"
                variant="outline"
                :disabled="!!busy"
                :aria-label="
                  t('selectTask', {
                    version: build.version,
                    time: date(build.created_at)
                  })
                "
                @click="void chooseBuild(build)"
                >{{ t('select') }}</UButton
              >
            </li>
          </ul>
          <div v-if="builds.total > builds.size" class="publisher-pagination">
            <UButton
              color="neutral"
              variant="outline"
              :disabled="builds.page === 1 || !!busy"
              @click="void workspace.pageBuilds(builds.page - 1)"
              >{{ t('previous') }}</UButton
            ><span>{{
              t('page', {
                page: builds.page,
                pages: Math.ceil(builds.total / builds.size)
              })
            }}</span
            ><UButton
              color="neutral"
              variant="outline"
              :disabled="builds.page * builds.size >= builds.total || !!busy"
              @click="void workspace.pageBuilds(builds.page + 1)"
              >{{ t('next') }}</UButton
            >
          </div>
        </template>
        <template v-else-if="detail === 'plan'"
          ><h3>{{ selectedPlan?.version }}</h3>
          <p class="whitespace-pre-wrap mt-3">
            {{ selectedPlan?.release_notes || t('noNotes') }}
          </p></template
        >
        <template v-else-if="detail === 'artifact'"
          ><p>
            {{ selectedBuild?.version }} ·
            {{ selectedBuild?.artifact && size(selectedBuild.artifact.size) }}
          </p>
          <p class="publisher-muted mt-3">{{ t('sha256') }}</p>
          <code class="publisher-digest">{{
            selectedBuild?.artifact?.sha256
          }}</code>
          <details class="mt-4">
            <summary>{{ t('technicalDetails') }}</summary>
            <dl class="publisher-technical">
              <dt>{{ t('buildReference') }}</dt>
              <dd>{{ selectedBuild?.build_id }}</dd>
              <dt>{{ t('artifactPath') }}</dt>
              <dd>{{ selectedBuild?.artifact?.path }}</dd>
              <dt>{{ t('sourceCommit') }}</dt>
              <dd>
                {{ selectedBuild?.source?.commit || t('filesystemSource') }}
              </dd>
            </dl>
          </details></template
        >
        <template v-else-if="detail === 'logs'"
          ><p v-if="detailBusy">{{ t('loading') }}</p>
          <pre v-else class="publisher-log" data-testid="publisher-log-text">{{
            logs || t('noLogs')
          }}</pre>
        </template>
        <p v-if="error" class="publisher-form-error mt-4" role="alert">
          {{ errorMessage }}
        </p>
      </template>
    </UModal>
  </section>
</template>

<style scoped>
.publisher-workspace {
  --ui-text-muted: var(--ui-text-toned);
  --ui-text-dimmed: var(--ui-text-toned);
  min-height: 0;
  container-type: inline-size;
  background: var(--ui-bg);
  color: var(--ui-text);
  padding: 0;
  max-width: 1440px;
  margin: auto;
  font-size: 1rem;
  line-height: 1.6;
}
.publisher-header {
  display: grid;
  gap: 0.38rem;
  margin-bottom: 1.5rem;
}
.publisher-section-heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.5rem;
}
.publisher-workspace h1 {
  font-size: 1.55rem;
  font-weight: 700;
  color: var(--ui-text-highlighted);
}
.publisher-workspace h2 {
  font-size: 1.25rem;
  font-weight: 650;
  color: var(--ui-text-highlighted);
}
.publisher-workspace h3 {
  font-size: 1rem;
  font-weight: 650;
  color: var(--ui-text-highlighted);
}
.publisher-muted {
  color: var(--ui-text-muted);
}
.publisher-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.65rem;
}
.publisher-context {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.25rem 0.85rem;
  font-size: 0.875rem;
  color: var(--ui-text-muted);
  overflow-wrap: anywhere;
}
.publisher-context strong {
  color: var(--ui-text-highlighted);
}
.publisher-version {
  padding-inline-start: 0.75rem;
  border-inline-start: 1px solid var(--ui-border-accented);
}
.publisher-version-choices {
  display: grid;
  grid-template-columns: minmax(0, 31rem);
  gap: 1rem;
}
.publisher-version-choices.has-plan {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}
.publisher-version-choice {
  display: grid;
  grid-template-rows: auto 1fr auto;
  gap: 1rem;
  padding: clamp(1rem, 2.5cqw, 1.5rem);
  border: 1px solid var(--ui-border-accented);
  border-radius: 0.9rem;
  background: var(--ui-bg);
}
.publisher-choice-heading {
  display: flex;
  align-items: center;
  gap: 0.8rem;
}
.publisher-choice-heading h3 {
  font-size: 1.125rem;
  line-height: 1.35;
}
.publisher-choice-icon {
  display: grid;
  flex: none;
  place-items: center;
  width: 2.65rem;
  height: 2.65rem;
  border-radius: 0.7rem;
  background: var(--ui-bg-elevated);
  color: var(--ui-text-highlighted);
  font-size: 1.25rem;
}
.publisher-choice-icon[data-ready='true'] {
  background: color-mix(in srgb, var(--ui-success) 16%, var(--ui-bg));
}
.publisher-choice-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.15rem 0.6rem;
  margin-top: 0.2rem;
  color: var(--ui-text-muted);
  font-size: 0.875rem;
}
.publisher-choice-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.3rem;
}
.publisher-package {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(13rem, 32%);
  grid-template-areas: 'summary next' 'facts next' 'tools tools';
  align-items: start;
  gap: 1.35rem 1.75rem;
  padding: clamp(1rem, 2.75cqw, 1.85rem);
  border: 1px solid var(--ui-border-accented);
  border-radius: 1rem;
  background: var(--ui-bg);
}
.publisher-package-summary {
  grid-area: summary;
  display: flex;
  align-items: center;
  gap: 0.9rem;
  min-width: 0;
}
.publisher-package-summary h2 {
  font-size: clamp(1.25rem, 2.2cqw, 1.65rem);
  line-height: 1.25;
  margin-bottom: 0.25rem;
}
.publisher-package-icon {
  display: grid;
  flex: none;
  place-items: center;
  width: 3rem;
  height: 3rem;
  border-radius: 0.8rem;
  background: color-mix(in srgb, var(--ui-success) 16%, var(--ui-bg));
  color: var(--ui-text-highlighted);
  font-size: 1.5rem;
}
.publisher-package-facts {
  grid-area: facts;
  display: grid;
  grid-template-columns: repeat(2, auto) minmax(0, 1fr);
  gap: 0.75rem 1.4rem;
  margin: 0;
}
.publisher-package-facts dt {
  color: var(--ui-text-muted);
  font-size: 0.82rem;
}
.publisher-package-facts dd {
  margin: 0.22rem 0 0;
  color: var(--ui-text-highlighted);
  font-size: 0.9rem;
  font-weight: 650;
}
.publisher-build-facts {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 1.5rem;
  margin: 0 0 1rem;
}
.publisher-card.publisher-build-card {
  display: grid;
  gap: 1.25rem;
  padding: clamp(1rem, 2.75cqw, 1.85rem);
}
.publisher-build-card > .publisher-actions {
  margin: 0;
}
.publisher-build-card .publisher-build-heading p {
  margin: 0.2rem 0 0;
  font-size: 0.875rem;
}
.publisher-build-status {
  padding-inline-start: 0.9rem;
  border-inline-start: 2px solid var(--ui-border-accented);
  color: var(--ui-text);
}
.publisher-build-card .publisher-build-status p {
  margin: 0.25rem 0 0;
  max-width: 34rem;
  font-size: 0.9rem;
}
.publisher-build-facts dt {
  color: var(--ui-text-muted);
  font-size: 0.82rem;
}
.publisher-build-facts dd {
  color: var(--ui-text-highlighted);
  font-weight: 600;
  margin-top: 0.15rem;
}
.publisher-package-next {
  grid-area: next;
  color: var(--ui-text);
}
.publisher-package-next p {
  margin: 0.55rem 0 0;
  font-size: 0.875rem;
  overflow-wrap: anywhere;
}
.publisher-package-next button {
  white-space: normal;
  text-align: start;
}
.publisher-package-tools {
  grid-area: tools;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem 1rem;
  border-top: 1px solid var(--ui-border);
  padding-top: 0.75rem;
  font-size: 0.875rem;
}
.publisher-package-tools .publisher-actions {
  gap: 0.3rem;
}
.publisher-executor-details {
  color: var(--ui-text-muted);
  margin-top: 1rem;
}
.publisher-executor-details summary {
  width: fit-content;
  cursor: pointer;
}
.publisher-executor-details .publisher-actions {
  margin-top: 0.5rem;
}
.publisher-card-heading {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.25rem 0.75rem;
}
.publisher-build-steps {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 1.25rem;
  margin: 1rem 0;
  color: var(--ui-text-muted);
}
.publisher-build-steps li {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}
.publisher-build-steps [aria-current='step'] {
  color: var(--ui-text-highlighted);
  font-weight: 700;
}
.publisher-step-marker {
  display: inline-block;
  width: 0.85em;
  height: 0.85em;
}
.publisher-notice {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.25rem;
  border: 1px solid var(--ui-border-accented);
  border-radius: 0.8rem;
  background: var(--ui-bg-elevated);
  margin-bottom: 1rem;
  color: var(--ui-text);
}
.publisher-error {
  display: block;
  border-left: 4px solid var(--ui-error);
  color: var(--ui-text-highlighted);
}
.publisher-workspace :deep(.publisher-tabs) {
  display: grid;
  grid-template-columns: 10.5rem minmax(0, 1fr);
  align-items: start;
  gap: clamp(1.25rem, 3cqw, 2.75rem);
}
.publisher-workspace :deep(.publisher-tab-list) {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 0.3rem;
  background: var(--ui-bg);
  border: 0;
  padding: 0;
}
.publisher-workspace :deep(.publisher-tab) {
  justify-content: flex-start;
  white-space: nowrap;
  color: var(--ui-text);
  gap: 0.65rem;
  padding: 0.45rem 0.75rem;
  border-radius: 0.65rem;
}
.publisher-rail-label {
  padding: 0 0.75rem 0.25rem;
  font-size: 0.78rem;
  color: var(--ui-text-muted);
}
.publisher-nav-icon {
  display: grid;
  flex: none;
  place-items: center;
  width: 1.7rem;
  height: 1.7rem;
  border-radius: 50%;
  color: var(--ui-text);
  background: var(--ui-bg-elevated);
}
.publisher-nav-icon[data-complete='true'] {
  background: color-mix(in srgb, var(--ui-success) 18%, var(--ui-bg));
  color: var(--ui-text-highlighted);
}
.publisher-workspace :deep(.publisher-tab-connection) {
  margin-top: 0.7rem;
}
.publisher-workspace :deep(.publisher-tab[data-state='active']) {
  background: var(--ui-bg-elevated);
  color: var(--ui-text-highlighted);
  font-weight: 650;
  box-shadow: inset 3px 0 0 var(--ui-primary);
}
.publisher-workspace :deep(.publisher-tab[data-state='active']) .publisher-nav-icon {
  background: var(--ui-bg);
  color: var(--ui-text-highlighted);
}
.publisher-workspace :deep(.publisher-tab:hover) {
  background: var(--ui-bg-elevated);
  color: var(--ui-text-highlighted);
}
.publisher-workspace :deep(.publisher-panel) {
  min-width: 0;
  margin: 0;
  padding: 0;
}
.publisher-card {
  padding: 1.25rem;
  border: 1px solid var(--ui-border);
  border-radius: 0.8rem;
  margin: 1rem 0;
  background: var(--ui-bg);
}
.publisher-card > .publisher-actions,
.publisher-card > button {
  margin-top: 1rem;
}
.publisher-card p {
  margin: 0.5rem 0;
}
.publisher-records {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0;
  list-style: none;
}
.publisher-records li {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: center;
  border: 1px solid var(--ui-border);
  border-radius: 0.8rem;
  padding: 1rem;
}
.publisher-records li.selected {
  background: var(--ui-bg-elevated);
  border-color: var(--ui-border-accented);
}
.publisher-empty {
  padding: 2rem;
  border: 1px dashed var(--ui-border-accented);
  border-radius: 0.8rem;
  text-align: center;
  color: var(--ui-text-muted);
}
.publisher-pagination {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 1rem;
  margin-top: 1rem;
}
.publisher-log {
  background: var(--ui-bg-elevated);
  color: var(--ui-text);
  border: 1px solid var(--ui-border);
  border-radius: 0.65rem;
  padding: 1rem;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  max-height: 55vh;
  overflow: auto;
  font-size: 0.85rem;
}
.publisher-digest,
.publisher-technical dd {
  display: block;
  overflow-wrap: anywhere;
  font-size: 0.85rem;
}
.publisher-technical dt {
  margin-top: 1rem;
  font-weight: 600;
}
:global(.publisher-dialog) {
  --ui-text-muted: var(--ui-text-toned);
  --ui-text-dimmed: var(--ui-text-toned);
  color: var(--ui-text);
  background: var(--ui-bg);
}
:global(.publisher-dialog-title) {
  color: var(--ui-text-highlighted);
}
:global(.publisher-version-dialog button:disabled) {
  opacity: 1;
  color: var(--ui-text-toned);
  background: var(--ui-bg-elevated);
}
:global(.publisher-form-error) {
  color: var(--ui-text-highlighted);
  font-weight: 600;
}
.publisher-workspace :deep(button:disabled) {
  opacity: 1;
  color: var(--ui-text-muted);
  background: var(--ui-bg-elevated);
}
.publisher-workspace :deep(button:focus-visible) {
  outline: 2px solid var(--ui-text-highlighted);
  outline-offset: 3px;
}
@container (max-width: 767px) {
  .publisher-header {
    margin-bottom: 1.15rem;
  }
  .publisher-package {
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas: 'summary' 'next' 'facts' 'tools';
    gap: 1rem;
  }
  .publisher-package-facts {
    gap: 0.55rem 0.75rem;
  }
  .publisher-build-steps {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .publisher-workspace :deep(.publisher-tabs) {
    display: block;
  }
  .publisher-workspace :deep(.publisher-tab-list) {
    display: grid;
    grid-template-columns: repeat(5, minmax(0, 1fr));
    overflow-x: auto;
    margin-bottom: 1.25rem;
    max-width: 100%;
  }
  .publisher-workspace :deep(.publisher-tab) {
    flex-shrink: 0;
    flex-direction: column;
    gap: 0.2rem;
    padding: 0.45rem 0.18rem;
    min-width: 0;
    font-size: 0.75rem;
    justify-content: center;
  }
  .publisher-workspace :deep(.publisher-tab-connection) {
    margin-top: 0;
  }
  .publisher-workspace :deep(.publisher-tab[data-state='active']) {
    box-shadow: inset 0 -2px 0 var(--ui-primary);
  }
  .publisher-section-heading,
  .publisher-records li,
  .publisher-notice {
    align-items: flex-start;
    flex-direction: column;
  }
  .publisher-section-heading .publisher-actions {
    width: 100%;
  }
}
@container (max-width: 590px) {
  .publisher-version-choices.has-plan {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
