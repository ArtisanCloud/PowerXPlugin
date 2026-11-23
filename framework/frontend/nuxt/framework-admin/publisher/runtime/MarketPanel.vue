<script setup lang="ts">
import RequestHistory from './RequestHistory.vue'
import type { PublisherClient } from '../client'
import UButton from '@nuxt/ui/components/Button.vue'
import UDropdownMenu from '@nuxt/ui/components/DropdownMenu.vue'
import UForm from '@nuxt/ui/components/Form.vue'
import * as v from 'valibot'
import UFormField from '@nuxt/ui/components/FormField.vue'
import UIcon from '@nuxt/ui/components/Icon.vue'
import UInput from '@nuxt/ui/components/Input.vue'
import UModal from '@nuxt/ui/components/Modal.vue'
import USelect from '@nuxt/ui/components/Select.vue'

import { computed, reactive, ref, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PublisherBuild, PublisherUpload } from '../types'
import type { createMarketWorkspace } from '../market-workspace'
import en from '../locales/en.json'
import zh from '../locales/zh.json'
const props = defineProps<{
  client: PublisherClient
  mode: 'connection' | 'upload'
  market: ReturnType<typeof createMarketWorkspace>
  build: PublisherBuild | null
  connected: boolean
}>()
const emit = defineEmits<{
  build: []
  connection: []
  upload: []
  review: [upload: PublisherUpload]
}>()
const { t, te, locale } = useI18n({ useScope: 'local', messages: { en, zh } })
const {
  markets,
  uploads,
  selectedMarketID,
  selectedUploadID,
  selectedMarket,
  selectedUpload,
  versions,
  versionsLoaded,
  versionsPage,
  versionsTotal,
  busy,
  canUpload,
  needsReconcile,
  pollPaused
} = props.market
const marketDialogOpen = ref(false)
const marketDialog = ref<'details' | 'add' | 'correct'>('add')
const confirmOpen = ref(false)
const bindOpen = ref(false), token = ref('')
const bindFailure = ref('')
watch(bindOpen, open => {
  bindFailure.value = ''
  if (!open) {
    token.value = ''
    if (import.meta.client) (document.activeElement as HTMLElement | null)?.blur()
  }
}, { flush: 'sync' })
onUnmounted(() => { token.value = '' })
async function bind() {
  const value = token.value
  token.value = ''
  if (await props.market.bind(value)) bindOpen.value = false
  else bindFailure.value = props.market.error.value
}
const correctionDraft = reactive({ name: '', api_root: '', console_base: '' })
const draft = computed(() =>
  marketDialog.value === 'correct' ? correctionDraft : props.market.draft
)
const options = computed(() =>
  markets.value.map((m) => {
    const siblings = markets.value.filter(other => other !== m && other.name === m.name)
    const address = siblings.some(other => other.api_root === m.api_root)
      ? `${m.api_root} · ${m.console_base}`
      : m.api_root
    return {
      label: siblings.length ? t('market.connectionOption', { name: m.name, address }) : m.name,
      value: m.connection_id
    }
  })
)
const marketHost = computed(() =>
  selectedMarket.value ? new URL(selectedMarket.value.console_base).host : ''
)
const connectionStatus = computed(() => {
  const market = selectedMarket.value
  const credential = market?.identity?.credential
  if (!props.connected) return 'offline'
  if (!market?.verified_at || !credential) return 'unbound'
  if (Date.parse(credential.expires_at) <= Date.now()) return 'expired'
  if (!credential.scopes.includes('packages:write')) return 'limited'
  return canUpload.value ? 'ready' : 'blocked'
})
const connectionAction = computed(() => {
  switch (connectionStatus.value) {
    case 'ready':
      return props.build?.stage === 'ready'
        ? { label: t('market.goToUpload'), icon: 'i-lucide-arrow-right', run: () => emit('upload') }
        : { label: t('workflow.nextBuild'), icon: 'i-lucide-arrow-right', run: () => emit('build') }
    case 'blocked':
      return { label: t('market.verify'), icon: 'i-lucide-shield-check', run: () => props.market.verify() }
    case 'offline':
      return null
    default:
      return {
        label: t(connectionStatus.value === 'unbound' ? 'market.bindAction' : 'market.rebindAction'),
        icon: 'i-lucide-key-round',
        run: () => { bindOpen.value = true }
      }
  }
})
const connectionMenu = computed(() => [
  {
    label: t('market.rebindAction'), icon: 'i-lucide-key-round',
    disabled: !props.connected || !!busy.value || needsReconcile.value || !selectedMarket.value?.verified_at,
    onSelect: () => { bindOpen.value = true }
  },
  {
    label: t('market.verify'), icon: 'i-lucide-shield-check',
    disabled: !props.connected || !!busy.value || !selectedMarket.value?.verified_at,
    onSelect: () => { void props.market.verify() }
  },
  {
    label: t('market.disconnect'), icon: 'i-lucide-unlink',
    disabled: !props.connected || !!busy.value || needsReconcile.value || !selectedMarket.value?.verified_at,
    onSelect: () => { void props.market.disconnect() }
  }
])
const taskOptions = computed(() =>
  uploads.value.map((u) => ({
    label: t('market.taskOption', {
      version: u.version,
      market:
        markets.value.find((m) => m.connection_id === u.connection_id)?.name ||
        t('market.savedConnection'),
      time: new Intl.DateTimeFormat(locale.value, {
        dateStyle: 'short',
        timeStyle: 'short'
      }).format(new Date(u.created_at))
    }),
    value: u.upload_id
  }))
)
const existingTask = computed(() =>
  uploads.value.find(
    (u) =>
      u.connection_id === selectedMarketID.value &&
      u.build_id === props.build?.build_id &&
      u.stage !== 'failed'
  )
)
const marketSchema = computed(() =>
  v.object({
    name: v.pipe(
      v.string(),
      v.nonEmpty(t('market.validationName')),
      v.maxLength(200, t('market.validationName'))
    ),
    api_root: v.pipe(v.string(), v.url(t('market.validationURL'))),
    console_base: v.pipe(v.string(), v.url(t('market.validationURL')))
  })
)
const canWriteConnection = computed(() =>
  props.connected && !busy.value && !needsReconcile.value
)
const canSaveConnection = computed(() => {
  const { name, api_root, console_base } = draft.value
  if (!canWriteConnection.value || !name.trim() || name.length > 200 || !api_root || !console_base)
    return false
  if (marketDialog.value !== 'correct') return true
  const current = selectedMarket.value
  return !!current && (api_root !== current.api_root || console_base !== current.console_base)
})
const tokenPage = computed(() => {
  if (!selectedMarket.value) return ''
  const base = new URL(selectedMarket.value.console_base)
  return base.href.replace(/\/$/, '') + '/developer/publish-tokens'
})
const stageLabel = (stage: string) =>
  te(`market.stages.${stage}`)
    ? t(`market.stages.${stage}`)
    : t('stages.unknown')
const marketError = (code: string) =>
  te(`errors.${code}`)
    ? t(`errors.${code}`)
    : t('errors.PUBLISHER_MARKET_PROTOCOL')
const link = computed(() => {
  const u = selectedUpload.value,
    raw =
      u?.current_version?.market_version_url || u?.receipt?.market_version_url
  const m = markets.value.find((m) => m.connection_id === u?.connection_id)
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
      ? target.href
      : ''
  } catch {
    return ''
  }
})
function openMarketDialog(view: 'details' | 'add') {
  marketDialog.value = view
  marketDialogOpen.value = true
}
function correctAddress() {
  const current = selectedMarket.value
  if (!current || !canWriteConnection.value) return
  const { name, api_root, console_base } = current
  Object.assign(correctionDraft, { name, api_root, console_base })
  marketDialog.value = 'correct'
}
function cancelConnectionForm() {
  if (marketDialog.value === 'correct') marketDialog.value = 'details'
  else marketDialogOpen.value = false
}
async function saveConnection() {
  if (!canSaveConnection.value) return
  const form = draft.value
  if (await props.market.add({ ...form })) {
    (document.activeElement as HTMLElement | null)?.blur()
    marketDialogOpen.value = false
    Object.assign(form, { name: '', api_root: '', console_base: '' })
  }
}
async function start() {
  confirmOpen.value = false
  if (props.build) await props.market.start(props.build.build_id)
}
</script>
<template>
  <section
    :aria-label="
      t(mode === 'connection' ? 'market.connectionTitle' : 'market.uploadTitle')
    "
  >
    <div class="publisher-section-heading">
      <h2>
        {{
          t(
            mode === 'connection'
              ? 'market.connectionTitle'
              : 'market.uploadTitle'
          )
        }}
      </h2>
      <UButton
        v-if="mode === 'upload'"
        color="neutral"
        variant="outline"
        :disabled="!connected || !!busy"
        @click="void market.refresh()"
        >{{ t('refresh') }}</UButton
      >
      <div v-else class="market-toolbar">
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-lucide-refresh-cw"
          :aria-label="t('refresh')"
          :disabled="!connected || !!busy"
          :loading="busy === 'refresh'"
          @click="void market.refresh()"
        />
        <UButton
          color="neutral"
          :variant="markets.length ? 'outline' : 'solid'"
          icon="i-lucide-plus"
          :disabled="!canWriteConnection"
          @click="openMarketDialog('add')"
        >{{ t('market.add') }}</UButton>
      </div>
    </div>
    <UFormField
      v-if="mode === 'upload' || markets.length > 1"
      :ui="{ error: 'publisher-form-error' }"
      :label="t('market.selectedConnection')"
      class="mb-5"
    >
      <USelect
        v-model="selectedMarketID"
        :items="options"
        :ui="{ itemLabel: 'whitespace-normal [overflow-wrap:anywhere]' }"
        :placeholder="t('market.noConnection')"
        :disabled="!connected || !!busy || !markets.length"
        class="w-full"
      />
    </UFormField>
    <template v-if="mode === 'connection'">
      <article v-if="selectedMarket" class="market-card" data-testid="publisher-market-card">
        <div class="market-card-header">
          <span class="market-symbol" aria-hidden="true">
            <UIcon name="i-lucide-store" />
          </span>
          <div class="market-identity">
            <h3>{{ selectedMarket.name }}</h3>
            <p class="publisher-muted">{{ marketHost }}</p>
          </div>
          <span class="market-status" :data-status="connectionStatus" role="status">
            {{ t(`market.connectionStatus.${connectionStatus}`) }}
          </span>
        </div>
        <p class="market-hint">
          {{ t(connectionStatus === 'ready' && build?.stage !== 'ready'
            ? 'market.needBuild'
            : `market.connectionHint.${connectionStatus}`) }}
        </p>
        <footer class="market-card-footer">
          <UButton
            v-if="connectionAction"
            color="neutral"
            :icon="connectionStatus === 'ready' ? undefined : connectionAction.icon"
            :trailing-icon="connectionStatus === 'ready' ? connectionAction.icon : undefined"
            :loading="connectionStatus === 'blocked' && busy === 'verify'"
            :disabled="!!busy || needsReconcile"
            :data-testid="connectionStatus === 'ready' || connectionStatus === 'blocked' ? 'publisher-market-next' : 'publisher-market-bind'"
            @click="connectionAction.run"
          >{{ connectionAction.label }}</UButton>
          <div class="market-secondary-actions">
            <UButton
              color="neutral"
              variant="ghost"
              icon="i-lucide-info"
              @click="openMarketDialog('details')"
            >{{ t('market.connectionDetails') }}</UButton>
            <UDropdownMenu :items="connectionMenu">
              <UButton
                color="neutral"
                variant="ghost"
                icon="i-lucide-ellipsis"
                :aria-label="t('workflow.moreActions')"
              />
            </UDropdownMenu>
          </div>
        </footer>
      </article>
      <div v-else class="market-card market-empty">
        <span class="market-symbol" aria-hidden="true">
          <UIcon name="i-lucide-store" />
        </span>
        <div>
          <h3>{{ t('market.noConnection') }}</h3>
          <p class="publisher-muted">{{ t('market.emptyConnection') }}</p>
        </div>
      </div>
    </template>
    <template v-else>
      <div v-if="!existingTask || existingTask.upload_id !== selectedUploadID" class="publisher-card">
        <p>{{ t('market.uploadHelp') }}</p>
        <p v-if="build?.artifact">
          {{
            t('market.buildSummary', {
              version: build.version,
              size: new Intl.NumberFormat(locale).format(build.artifact.size)
            })
          }}
        </p>
        <p v-else>{{ t('market.needBuild') }}</p>
        <p v-if="!selectedMarket">{{ t('market.noConnection') }}</p>
        <p v-else-if="!canUpload">{{ t('market.capabilityClosed') }}</p>
        <p v-if="existingTask">{{ t('market.existingTask') }}</p>
        <div class="publisher-actions">
          <UButton
            v-if="build?.stage !== 'ready'"
            color="neutral"
            @click="emit('build')"
            >{{ t('workflow.nextBuild') }}</UButton
          >
          <UButton
            v-else-if="existingTask"
            color="neutral"
            @click="selectedUploadID = existingTask.upload_id"
            >{{ t('market.inspect') }}</UButton
          >
          <UButton
            v-else-if="!canUpload"
            color="neutral"
            :disabled="!connected"
            @click="emit('connection')"
            >{{ t('workflow.configureMarket') }}</UButton
          >
          <UButton
            v-else
            color="neutral"
            :disabled="!connected || !!busy || needsReconcile"
            @click="void (confirmOpen = true)"
            data-testid="publisher-market-upload"
            >{{ t('upload') }}</UButton
          >
          <UButton
            v-if="selectedMarket?.verified_at"
            color="neutral"
            variant="outline"
            :disabled="!!busy"
            @click="void market.readVersions()"
            >{{ t('market.readVersions') }}</UButton
          >
        </div>
      </div>
      <UFormField
        v-if="uploads.length"
        :ui="{ error: 'publisher-form-error' }"
        :label="t('market.selectedTask')"
        class="mb-5"
        ><USelect
          v-model="selectedUploadID"
          :items="taskOptions"
          :placeholder="t('market.noTasks')"
          :disabled="!uploads.length || !!busy"
          class="w-full"
      /></UFormField>
      <div
        v-if="selectedUpload"
        class="publisher-card"
        data-testid="publisher-upload-task"
      >
        <h3>
          {{ selectedUpload.version }} · {{ stageLabel(selectedUpload.stage) }}
        </h3>
        <p>
          {{
            t('market.sent', {
              sent: new Intl.NumberFormat(locale).format(
                selectedUpload.bytes_sent
              ),
              total: new Intl.NumberFormat(locale).format(
                selectedUpload.artifact.size
              )
            })
          }}
        </p>
        <p
          v-if="
            selectedUpload.stage === 'processing' ||
            selectedUpload.stage === 'reconciling'
          "
        >
          {{ t('market.serverProcessing') }}
        </p>
        <p
          v-if="selectedUpload.error"
          role="alert"
          class="publisher-form-error"
        >
          {{ marketError(selectedUpload.error.code) }}
          <code v-if="selectedUpload.error.params?.request_id"
            >{{ t('trace.serverID') }}
            {{ selectedUpload.error.params.request_id }}</code
          >
        </p>
        <p v-if="selectedUpload.current_version">
          {{ t('market.currentState') }}
          {{
            t(
              `market.availability.${selectedUpload.current_version.availability_status}`
            )
          }}
        </p>
        <div class="publisher-actions">
          <UButton
            v-if="selectedUpload.stage === 'succeeded' && selectedUpload.receipt"
            color="neutral"
            trailing-icon="i-lucide-arrow-right"
            @click="emit('review', selectedUpload)"
            >{{ t('workflow.nextReview') }}</UButton
          >
          <UButton
            color="neutral"
            variant="outline"
            :disabled="
              !!busy ||
              needsReconcile ||
              ['queued', 'uploading', 'processing', 'reconciling'].includes(
                selectedUpload.stage
              )
            "
            @click="void market.resume()"
            >{{ t('market.resume') }}</UButton
          >
          <UButton
            v-if="
              ['queued', 'uploading', 'processing', 'reconciling'].includes(
                selectedUpload.stage
              )
            "
            color="neutral"
            variant="outline"
            :disabled="!!busy"
            @click="void market.cancel(selectedUpload.upload_id)"
            >{{ t('market.cancel') }}</UButton
          >
          <UButton
            v-if="link"
            color="neutral"
            variant="outline"
            :to="link"
            target="_blank"
            rel="noopener noreferrer"
            >{{ t('market.openVersion') }}</UButton
          >
        </div>
        <details class="mt-5">
          <summary>{{ t('technicalDetails') }}</summary>
          <dl class="publisher-technical">
            <dt>{{ t('market.operationReference') }}</dt>
            <dd>{{ selectedUpload.operation_id }}</dd>
            <dt>{{ t('sha256') }}</dt>
            <dd>{{ selectedUpload.artifact.sha256 }}</dd>
            <dt>{{ t('market.attempts') }}</dt>
            <dd>{{ selectedUpload.attempts }}</dd>
          </dl>
          <RequestHistory
            :client="client"
            kind="uploads"
            :task-id="selectedUpload.upload_id"
          />
        </details>
      </div>
      <div v-if="pollPaused" role="status" class="publisher-notice">
        <p>{{ t('market.pollPaused') }}</p>
        <UButton color="neutral" @click="void market.resumePolling()">{{
          t('resumePolling')
        }}</UButton>
      </div>
      <div v-if="versionsLoaded" class="publisher-card">
        <h3>{{ t('market.versionsTitle') }}</h3>
        <p v-if="!versions.length">{{ t('market.noVersions') }}</p>
        <p>
          {{
            t('market.pageSummary', {
              page: versionsPage,
              total: versionsTotal
            })
          }}
        </p>
        <ul>
          <li v-for="item in versions" :key="item.version_id">
            {{ item.version }} ·
            {{ t(`market.availability.${item.availability_status}`) }}
          </li>
        </ul>
        <div class="publisher-actions mt-4">
          <UButton
            color="neutral"
            variant="outline"
            :disabled="versionsPage <= 1 || !!busy"
            @click="void market.readVersions(versionsPage - 1)"
            >{{ t('market.previous') }}</UButton
          ><UButton
            color="neutral"
            variant="outline"
            :disabled="versionsPage * 20 >= versionsTotal || !!busy"
            @click="void market.readVersions(versionsPage + 1)"
            >{{ t('market.next') }}</UButton
          >
        </div>
      </div>
    </template>
    <UModal
      v-model:open="marketDialogOpen"
      :title="t(marketDialog === 'details' ? 'market.connectionDetails' : marketDialog === 'correct' ? 'market.correctAddress' : 'market.add')"
      :description="marketDialog === 'details' ? selectedMarket?.name : t(marketDialog === 'correct' ? 'market.correctAddressHelp' : 'market.connectionHelp')"
      :close="busy !== 'add'"
      :dismissible="busy !== 'add'"
      :ui="{
        content: 'publisher-dialog market-connection-dialog max-w-xl',
        title: 'publisher-dialog-title',
        body: 'overflow-y-auto',
        footer: 'justify-end'
      }"
    >
      <template #body>
        <dl v-if="marketDialog === 'details' && selectedMarket" class="market-details">
          <div>
            <dt>{{ t('market.apiRoot') }}</dt>
            <dd>{{ selectedMarket.api_root }}</dd>
          </div>
          <div>
            <dt>{{ t('market.consoleBase') }}</dt>
            <dd>{{ selectedMarket.console_base }}</dd>
          </div>
          <template v-if="selectedMarket.verified_at && selectedMarket.identity">
            <div>
              <dt>{{ t('market.authorizedIdentity') }}</dt>
              <dd>{{ t('market.identity', {
                user: selectedMarket.identity.user.name,
                developer: selectedMarket.identity.developer.name
              }) }}</dd>
            </div>
            <div>
              <dt>{{ t('market.credentialExpiry') }}</dt>
              <dd>{{ new Intl.DateTimeFormat(locale, {
                dateStyle: 'medium',
                timeStyle: 'short'
              }).format(new Date(selectedMarket.identity.credential.expires_at)) }}</dd>
            </div>
            <div>
              <dt>{{ t('market.scopes') }}</dt>
              <dd class="market-scopes">
                <code v-for="scope in selectedMarket.identity.credential.scopes" :key="scope">
                  {{ scope }}
                </code>
              </dd>
            </div>
          </template>
        </dl>
        <UForm
          v-else-if="marketDialog !== 'details'"
          :key="marketDialog"
          :schema="marketSchema"
          :state="draft"
          class="space-y-5"
          novalidate
          @submit="saveConnection"
        >
          <UFormField
            v-if="marketDialog === 'add'"
            :ui="{ error: 'publisher-form-error' }"
            :label="t('market.name')"
            name="name"
            required
            ><UInput
              v-model="draft.name"
              :readonly="busy === 'add'"
              class="w-full"
              :maxlength="200"
              required
          /></UFormField>
          <UFormField
            :ui="{ error: 'publisher-form-error' }"
            :label="t('market.apiRoot')"
            name="api_root"
            required
            ><UInput
              v-model="draft.api_root"
              :autofocus="marketDialog === 'correct'"
              :readonly="busy === 'add'"
              type="url"
              class="w-full"
              required
          /></UFormField>
          <UFormField
            :ui="{ error: 'publisher-form-error' }"
            :label="t('market.consoleBase')"
            name="console_base"
            required
            ><UInput
              v-model="draft.console_base"
              :readonly="busy === 'add'"
              type="url"
              class="w-full"
              required
          /></UFormField>
          <p
            v-if="market.error.value"
            role="alert"
            class="publisher-form-error"
          >
            {{ marketError(market.error.value) }}
          </p>
          <div class="publisher-actions market-form-actions">
            <UButton
              color="neutral"
              variant="ghost"
              :disabled="busy === 'add'"
              @click="cancelConnectionForm"
            >{{ t('cancel') }}</UButton>
            <UButton
              type="submit"
              color="neutral"
              :disabled="!canSaveConnection"
              :loading="busy === 'add'"
            >{{ t(marketDialog === 'correct' ? 'market.saveAndSwitch' : 'market.save') }}</UButton>
          </div>
        </UForm>
      </template>
      <template v-if="marketDialog === 'details' && selectedMarket" #footer>
        <UButton
          color="neutral"
          variant="outline"
          icon="i-lucide-pencil"
          :disabled="!canWriteConnection"
          @click="correctAddress"
        >{{ t('market.correctAddress') }}</UButton>
      </template>
    </UModal>
    <UModal
      v-model:open="confirmOpen"
      :title="t('market.confirmTitle')"
      :description="t('market.confirmDescription')"
      :ui="{
        content: 'publisher-dialog max-w-xl',
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
        ><p>{{ selectedMarket?.name }} · {{ build?.version }}</p>
        <p>{{ t('market.confirmBoundary') }}</p>
        <div class="publisher-actions mt-5">
          <UButton color="neutral" @click="start">{{
            t('market.confirm')
          }}</UButton
          ><UButton
            color="neutral"
            variant="outline"
            @click="void (confirmOpen = false)"
            >{{ t('close') }}</UButton
          >
        </div></template
      >
    </UModal>
  </section>
  <UModal v-model:open="bindOpen" class="publisher-modal" :title="t('market.bindTitle')" :description="t('market.bindHelp')" :ui="{ content: 'publisher-dialog max-w-xl', title: 'publisher-dialog-title', body: 'max-h-[80vh] overflow-y-auto' }">
    <template #close="{ close }"><UButton color="neutral" variant="ghost" icon="i-lucide-x" :aria-label="t('close')" @click="void close()" /></template>
    <template #body>
      <p v-if="bindFailure" class="publisher-form-error" role="alert">{{ marketError(bindFailure) }}</p>
      <p class="publisher-muted">{{ t('market.bindSafety') }}</p>
      <UButton v-if="tokenPage" :to="tokenPage" target="_blank" rel="noopener noreferrer" color="neutral" variant="outline" class="mt-4" :label="t('market.openTokenPage')" />
      <form class="mt-5 space-y-5" autocomplete="off" @submit.prevent="bind">
        <UFormField :label="t('market.tokenLabel')" :description="t('market.tokenHelp')">
          <UInput v-model="token" type="password" autocomplete="off" autocapitalize="off" :spellcheck="false" name="publisher-token" class="w-full" data-testid="publisher-market-token" />
        </UFormField>
        <UButton type="submit" color="neutral" :loading="busy === 'bind'" :disabled="!!busy || !/^pxpub_[A-Za-z0-9_-]{43}$/.test(token)" data-testid="publisher-market-bind-submit">{{ t('market.bindSubmit') }}</UButton>
      </form>
    </template>
  </UModal>
</template>
<style scoped>
section {
  color: var(--ui-text);
}
h2,
h3 {
  color: var(--ui-text-highlighted);
}
.market-toolbar,
.market-card-footer,
.market-secondary-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}
.market-card {
  padding: clamp(1rem, 3vw, 1.5rem);
  border: 1px solid var(--ui-border-accented);
  border-radius: 1rem;
  background: var(--ui-bg);
}
.market-card-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
}
.market-symbol {
  display: inline-grid;
  place-items: center;
  flex: none;
  width: 2.75rem;
  height: 2.75rem;
  border-radius: 0.75rem;
  background: color-mix(in srgb, var(--ui-info) 13%, var(--ui-bg));
  color: var(--ui-text-highlighted);
  font-size: 1.25rem;
}
.market-identity {
  flex: 1 1 10rem;
  min-width: 0;
  overflow-wrap: anywhere;
}
.market-identity h3,
.market-empty h3 {
  font-size: 1.125rem;
  font-weight: 650;
  line-height: 1.35;
}
.market-identity p,
.market-empty p {
  margin-top: 0.2rem;
  font-size: 0.875rem;
}
.market-identity p {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.market-status {
  --market-status-color: var(--ui-warning);
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.25rem 0.6rem;
  border-radius: 999px;
  color: var(--ui-text-highlighted);
  background: color-mix(in srgb, var(--market-status-color) 12%, var(--ui-bg));
  font-size: 0.8rem;
  font-weight: 600;
}
.market-status::before {
  content: '';
  width: 0.45rem;
  height: 0.45rem;
  flex: none;
  border-radius: 50%;
  background: var(--market-status-color);
}
.market-status[data-status='ready'] {
  --market-status-color: var(--ui-success);
}
.market-status[data-status='offline'] {
  --market-status-color: var(--ui-text-muted);
}
.market-hint {
  margin-top: 1rem;
  color: var(--ui-text-toned);
  font-size: 0.925rem;
  line-height: 1.6;
}
.market-card-footer {
  margin-top: 1rem;
  padding-top: 0.85rem;
  border-top: 1px solid var(--ui-border);
  gap: 0.65rem 1rem;
}
.market-secondary-actions {
  margin-inline-start: auto;
}
.market-empty {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.market-details {
  display: grid;
  gap: 1rem;
  color: var(--ui-text);
}
.market-details dt {
  color: var(--ui-text-toned);
  font-size: 0.875rem;
}
.market-details dd {
  margin-top: 0.25rem;
  color: var(--ui-text-highlighted);
  overflow-wrap: anywhere;
}
.market-form-actions {
  justify-content: flex-end;
}
.market-connection-dialog :deep(button:disabled) {
  opacity: 1;
  color: var(--ui-text-toned);
  background: var(--ui-bg-elevated);
}
.market-connection-dialog :deep(button:focus-visible) {
  outline: 2px solid var(--ui-text-highlighted);
  outline-offset: 2px;
}
.market-scopes {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}
.market-scopes code {
  padding: 0.15rem 0.4rem;
  border-radius: 0.35rem;
  background: var(--ui-bg-elevated);
  font-size: 0.8rem;
}
.publisher-muted {
  color: var(--ui-text-toned);
}
.publisher-card {
  background: var(--ui-bg-elevated);
  border: 1px solid var(--ui-border);
  border-radius: 0.75rem;
  padding: 1.25rem;
  margin-bottom: 1.25rem;
  overflow-wrap: anywhere;
}
.publisher-actions,
.publisher-section-heading {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
}
.publisher-section-heading {
  justify-content: space-between;
  margin-bottom: 1rem;
}
.publisher-technical {
  display: grid;
  grid-template-columns: minmax(6rem, 1fr) minmax(0, 3fr);
  gap: 0.5rem;
  overflow-wrap: anywhere;
  margin: 1rem 0;
}
.publisher-technical dt {
  color: var(--ui-text-toned);
}
.publisher-command {
  display: block;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  background: var(--ui-bg);
  padding: 0.75rem;
  margin: 0.75rem 0;
}
.publisher-form-error {
  color: var(--ui-text-highlighted);
  font-weight: 600;
}
p + p {
  margin-top: 0.75rem;
}
summary {
  cursor: pointer;
  color: var(--ui-text-highlighted);
}
</style>
