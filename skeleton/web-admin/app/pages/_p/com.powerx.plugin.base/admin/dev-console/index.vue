<template>
  <div class="space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-command-line" class="text-primary" />
        <span class="uppercase tracking-wide">Dev Console</span>
      </div>
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">配置与治理</h1>
        <p class="text-sm text-gray-600 dark:text-gray-300">管理 Dev Console 的默认参数、导出策略和安全操作。</p>
      </div>
    </header>

    <UTabs v-model="activeTab" :items="tabs" />

    <section v-if="activeTab === 'config'" class="space-y-6">
      <UAlert
        v-if="configStore.error"
        color="red"
        variant="soft"
        :title="configStore.error"
      />

      <USkeleton v-if="configStore.loading" class="h-32 w-full" repeat="3" />

      <div v-else class="space-y-6">
        <ConfigSectionCard
          v-for="section in configSections"
          :key="section.key"
          :section="section"
          :pending="savingKey === section.key"
          @submit="values => saveSection(section.key, values)"
        />
      </div>
    </section>

    <section v-else-if="activeTab === 'audit'" class="space-y-6">
      <UAlert
        v-if="auditStore.error"
        color="red"
        variant="soft"
        :title="auditStore.error"
      />

      <AuditHistoryTable
        v-model="auditFilters"
        :events="auditEvents"
        :loading="auditStore.loading"
        :next-cursor="auditStore.nextCursor"
        @apply="loadAuditEvents"
        @load-more="loadMoreAuditEvents"
      >
        <template #actions>
          <AuditExportDialog v-model="exportFormat" :pending="auditStore.exporting" @export="exportAuditEvents" />
        </template>
      </AuditHistoryTable>
    </section>

    <section v-else class="space-y-6">
      <UAlert
        v-if="runsError"
        color="red"
        variant="soft"
        :title="runsError"
      />

      <UAlert
        v-if="summaryError"
        color="red"
        variant="soft"
        :title="summaryError"
      />

      <JobRunsTable
        v-model="jobFilters"
        :runs="runs"
        :loading="loadingRuns"
        :next-cursor="runsNextCursor"
        :retrying="retrying"
        @apply="() => loadJobRuns()"
        @load-more="loadMoreRuns"
        @retry="handleRetry"
      >
        <template #actions>
          <UButton size="xs" variant="soft" color="primary" @click="loadJobRuns">刷新列表</UButton>
        </template>
      </JobRunsTable>

      <UCard>
        <template #header>
          <div class="font-medium">安全操作</div>
        </template>
        <form class="grid gap-4 md:grid-cols-3" @submit.prevent="submitSafeOp">
          <UFormGroup label="动作">
            <USelect v-model="safeOpForm.action" :options="safeOpActions" />
          </UFormGroup>
          <UFormGroup label="作用域类型">
            <USelect v-model="safeOpForm.scope_type" :options="safeOpScopeTypes" />
          </UFormGroup>
          <UFormGroup label="作用域标识">
            <UInput v-model="safeOpForm.scope_ref" placeholder="tenant-1" />
          </UFormGroup>
          <UFormGroup label="目标 ID" class="md:col-span-3">
            <UInput v-model="safeOpForm.target_id" placeholder="hook-evt-123" />
          </UFormGroup>
          <UFormGroup label="备注" class="md:col-span-3">
            <UTextarea v-model="safeOpForm.reason" placeholder="本次操作原因" />
          </UFormGroup>
          <UFormGroup label="Dry Run" class="md:col-span-3">
            <UToggle v-model="safeOpForm.dry_run" />
          </UFormGroup>
          <UFormGroup label="租户 ID" class="md:col-span-3">
            <UInput v-model="safeOpForm.tenant_id" placeholder="tenant-1" />
          </UFormGroup>
          <div class="md:col-span-3 flex justify-end gap-3">
            <UButton type="submit" color="primary" :loading="safeOpsLoading">提交操作</UButton>
          </div>
        </form>
      </UCard>

      <TroubleshootingDashboard
        :summary="summary"
        :loading="loadingSummary"
        @refresh="refreshSummary"
      />

      <TroubleshootingHelpPanel :items="summary?.guidance ?? []" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useDevConsoleConfigStore } from '~/stores/dev-console/config'
import { useDevConsoleAuditStore } from '~/stores/dev-console/audit'
import { useDevConsoleTroubleshootStore } from '~/stores/dev-console/troubleshoot'
import type { JobRun, JobRunFilters, SafeOpPayload } from '~/stores/dev-console/troubleshoot'
import ConfigSectionCard from '~/components/dev-console/ConfigSectionCard.vue'
import AuditHistoryTable from '~/components/dev-console/AuditHistoryTable.vue'
import AuditExportDialog from '~/components/dev-console/AuditExportDialog.vue'
import JobRunsTable from '~/components/dev-console/JobRunsTable.vue'
import TroubleshootingDashboard from '~/components/dev-console/TroubleshootingDashboard.vue'
import TroubleshootingHelpPanel from '~/components/dev-console/TroubleshootingHelpPanel.vue'
import { useSafeOps } from '~/composables/useSafeOps'

const tabs = [
  { label: '配置管理', value: 'config', icon: 'i-heroicons-adjustments-horizontal' },
  { label: '审计历史', value: 'audit', icon: 'i-heroicons-clock' },
  { label: '故障排查', value: 'troubleshoot', icon: 'i-heroicons-wrench-screwdriver' },
]

const configStore = useDevConsoleConfigStore()
const auditStore = useDevConsoleAuditStore()
const troubleshootStore = useDevConsoleTroubleshootStore()
const { sections: configSections } = storeToRefs(configStore)
const { events: auditEvents } = storeToRefs(auditStore)
const {
  runs,
  nextCursor: runsNextCursor,
  loadingRuns,
  runsError,
  summary,
  loadingSummary,
  summaryError,
  runsFilters,
} = storeToRefs(troubleshootStore)
const toast = useToast()
const savingKey = ref('')
const activeTab = ref<'config' | 'audit' | 'troubleshoot'>('config')
const auditFilters = ref(auditStore.filters)
const exportFormat = ref<'csv' | 'json'>('csv')
const jobFilters = ref<JobRunFilters>({ ...runsFilters.value })
const retrying = ref(false)
const safeOps = useSafeOps()
const safeOpsLoading = computed(() => safeOps.loading.value)
const safeOpForm = reactive<SafeOpPayload>({
  action: 'replay',
  scope_type: 'tenant',
  scope_ref: '',
  target_id: '',
  reason: '',
  dry_run: false,
  tenant_id: '',
  environment: '',
})

const safeOpActions = [
  { label: '重放 Webhook', value: 'replay' },
  { label: '任务重试', value: 'retry' },
  { label: '队列 Drain', value: 'drain' },
  { label: '禁用', value: 'disable' },
]

const safeOpScopeTypes = [
  { label: '租户', value: 'tenant' },
  { label: '环境', value: 'environment' },
  { label: '订阅', value: 'subscription' },
]

onMounted(async () => {
  if (!configSections.value.length) {
    await configStore.fetchSections().catch(() => {})
  }
})

watch(
  runsFilters,
  value => {
    jobFilters.value = { ...value }
  },
  { deep: true }
)

watch(
  () => jobFilters.value.tenant_id,
  value => {
    const cleaned = value?.trim() ?? ''
    safeOpForm.tenant_id = cleaned
    if (safeOpForm.scope_type === 'tenant') {
      safeOpForm.scope_ref = cleaned
    }
  },
  { immediate: true }
)

watch(
  () => safeOpForm.scope_type,
  value => {
    if (value === 'tenant' && jobFilters.value.tenant_id) {
      safeOpForm.scope_ref = jobFilters.value.tenant_id
    }
  }
)

async function saveSection(key: string, values: Record<string, any>) {
  savingKey.value = key
  try {
    await configStore.updateSection(key, { values })
    toast.add({ title: '配置已更新', color: 'green' })
  } catch (error: any) {
    toast.add({ title: '保存失败', description: error?.message ?? '请稍后再试', color: 'red' })
  } finally {
    savingKey.value = ''
  }
}

watch(
  () => activeTab.value,
  async value => {
    if (value !== 'troubleshoot') {
      troubleshootStore.stopAutoRefresh()
    }
    if (value === 'audit' && !auditEvents.value.length && !auditStore.loading) {
      await loadAuditEvents().catch(() => {})
    }
    if (value === 'troubleshoot') {
      if (!runs.value.length && !loadingRuns.value) {
        await loadJobRuns().catch(() => {})
      }
      await refreshSummary().catch(() => {})
    }
  }
)

watch(
  () => auditStore.filters,
  value => {
    auditFilters.value = { ...value }
  }
)

onBeforeUnmount(() => {
  troubleshootStore.stopAutoRefresh()
})

async function loadAuditEvents() {
  auditStore.setFilters({ ...auditFilters.value })
  await auditStore.fetchEvents().catch(() => {})
}

async function loadMoreAuditEvents() {
  if (!auditStore.nextCursor) {
    return
  }
  await auditStore.fetchEvents(auditStore.nextCursor).catch(() => {})
}

async function exportAuditEvents() {
  await auditStore.exportEvents(exportFormat.value).catch(() => {})
}

async function loadJobRuns(cursor?: string) {
  const payload: Record<string, any> = { ...jobFilters.value }
  if (payload.tenant_id) payload.tenant_id = payload.tenant_id.trim()
  if (payload.status) payload.status = payload.status.trim()
  if (payload.job_type) payload.job_type = payload.job_type.trim()
  if (cursor) {
    payload.cursor = cursor
  }
  try {
    await troubleshootStore.fetchRuns(payload)
  } catch (error: any) {
    toast.add({ title: '加载任务失败', description: error?.message ?? '请稍后再试', color: 'red' })
  }
}

async function loadMoreRuns() {
  if (!runsNextCursor.value) {
    return
  }
  await loadJobRuns(runsNextCursor.value)
}

async function handleRetry(run: JobRun) {
  retrying.value = true
  try {
    await troubleshootStore.retryRun(run.id, { tenant_id: jobFilters.value.tenant_id })
    toast.add({ title: '已发起重试', color: 'green' })
  } catch (error: any) {
    toast.add({ title: '重试失败', description: error?.message ?? '请稍后再试', color: 'red' })
  } finally {
    retrying.value = false
  }
}

async function refreshSummary() {
  const tenant = jobFilters.value.tenant_id?.trim()
  try {
    await troubleshootStore.fetchSummary({ tenant_id: tenant || undefined })
  } catch (error: any) {
    toast.add({ title: '加载看板失败', description: error?.message ?? '请稍后再试', color: 'red' })
  }
}

async function submitSafeOp() {
  const scopeRef = safeOpForm.scope_ref?.trim()
  if (!scopeRef) {
    toast.add({ title: '请输入作用域标识', color: 'red' })
    return
  }
  const payload: SafeOpPayload = {
    action: safeOpForm.action,
    scope_type: safeOpForm.scope_type,
    scope_ref: scopeRef,
    target_id: safeOpForm.target_id?.trim() || undefined,
    reason: safeOpForm.reason?.trim() || undefined,
    dry_run: safeOpForm.dry_run,
    tenant_id: safeOpForm.tenant_id?.trim() || undefined,
    environment: safeOpForm.environment?.trim() || undefined,
  }
  try {
    await safeOps.execute(payload)
    safeOpForm.reason = ''
    safeOpForm.target_id = ''
  } catch {
    // toast handled by composable
  }
}
</script>
