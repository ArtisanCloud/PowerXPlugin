<template>
  <div class="space-y-4">
    <UForm class="grid gap-4 md:grid-cols-3" @submit.prevent="apply">
      <UFormGroup label="租户 ID">
        <UInput v-model="filters.tenant_id" placeholder="tenant-1" />
      </UFormGroup>
      <UFormGroup label="任务类型">
        <UInput v-model="filters.job_type" placeholder="webhook_replay" />
      </UFormGroup>
      <UFormGroup label="状态">
        <UInput v-model="filters.status" placeholder="failed" />
      </UFormGroup>
      <div class="md:col-span-3 flex justify-end gap-3">
        <UButton variant="soft" color="neutral" @click="reset">清除</UButton>
        <UButton type="submit" color="primary">筛选</UButton>
      </div>
    </UForm>

    <UTable :rows="runs" :columns="columns" :loading="loading">
      <template #created_at-data="{ row }">
        {{ formatDate(row.created_at) }}
      </template>
      <template #status-data="{ row }">
        <UBadge :color="statusColor(row.status)">{{ formatStatus(row.status) }}</UBadge>
      </template>
      <template #scope-data="{ row }">
        <span>
          {{ row.safe_op.scope_type || '-' }}
          <template v-if="row.safe_op.scope_ref">
            :{{ row.safe_op.scope_ref }}
          </template>
        </span>
      </template>
      <template #message-data="{ row }">
        {{ row.message || '-' }}
      </template>
      <template #actions-data="{ row }">
        <UButton
          v-if="canRetry(row)"
          size="xs"
          color="primary"
          :loading="retrying"
          @click="$emit('retry', row)"
        >
          重试
        </UButton>
      </template>
    </UTable>

    <div class="flex justify-between items-center">
      <UButton
        color="primary"
        variant="soft"
        :disabled="!nextCursor || loading"
        @click="$emit('load-more')"
      >
        加载更多
      </UButton>
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { JobRun, JobRunFilters } from '~/stores/dev-console/troubleshoot'

const props = defineProps<{
  runs: JobRun[]
  loading?: boolean
  nextCursor?: string
  modelValue: JobRunFilters
  retrying?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', payload: JobRunFilters): void
  (event: 'apply'): void
  (event: 'load-more'): void
  (event: 'retry', payload: JobRun): void
}>()

const filters = reactive<JobRunFilters>({ ...props.modelValue })

watch(
  () => props.modelValue,
  value => {
    Object.assign(filters, value)
  }
)

const columns = [
  { key: 'created_at', label: '创建时间' },
  { key: 'job_type', label: '任务类型' },
  { key: 'status', label: '状态' },
  { key: 'environment', label: '环境' },
  { key: 'trigger_source', label: '触发来源' },
  { key: 'scope', label: '作用域', sortable: false },
  { key: 'message', label: '备注' },
  { key: 'actions', label: '操作', sortable: false },
]

const nextCursor = computed(() => props.nextCursor)
const retrying = computed(() => props.retrying)

function apply() {
  const clean: JobRunFilters = {}
  for (const [key, value] of Object.entries(filters)) {
    if (value) {
      ;(clean as any)[key] = value
    }
  }
  emit('update:modelValue', clean)
  emit('apply')
}

function reset() {
  for (const key of Object.keys(filters)) {
    ;(filters as any)[key] = ''
  }
  apply()
}

function formatDate(value?: string) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

function formatStatus(status: string) {
  return status?.toUpperCase() || 'UNKNOWN'
}

function statusColor(status: string) {
  switch ((status || '').toLowerCase()) {
    case 'succeeded':
      return 'green'
    case 'failed':
      return 'red'
    case 'running':
      return 'primary'
    case 'pending':
      return 'orange'
    default:
      return 'neutral'
  }
}

function canRetry(run: JobRun) {
  const status = (run.status || '').toLowerCase()
  return status === 'failed' || status === 'cancelled'
}
</script>
