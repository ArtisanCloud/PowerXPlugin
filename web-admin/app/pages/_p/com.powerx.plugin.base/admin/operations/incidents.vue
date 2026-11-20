<template>
  <UContainer>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-semibold">Incident Center</h1>
        <p class="text-sm text-gray-500">Track SEV incidents, communicate updates, and maintain readiness.</p>
      </div>
      <div class="space-x-2">
        <UButton variant="ghost" color="secondary" @click="refresh">刷新</UButton>
        <UButton color="primary" @click="startCreate">声明事故</UButton>
      </div>
    </div>

    <UAlert v-if="store.error" color="error" :title="store.error" class="mb-4" />

    <div class="grid gap-4 md:grid-cols-3">
      <UCard class="md:col-span-1">
        <template #header>事故列表</template>
        <div class="space-y-2">
          <div
            v-for="incident in store.incidents"
            :key="incident.id"
            class="border rounded-md p-3 cursor-pointer hover:border-primary"
            :class="{
              'border-primary ring-1 ring-primary': store.selected?.incident.id === incident.id,
            }"
            @click="selectIncident(incident.id)"
          >
            <div class="flex items-center justify-between text-sm">
              <span class="font-medium uppercase">{{ incident.severity }}</span>
              <UBadge :color="statusColor(incident.status)" size="xs">{{ incident.status }}</UBadge>
            </div>
            <p class="mt-2 text-sm line-clamp-2">{{ incident.summary }}</p>
            <div class="mt-2 text-xs text-gray-500">下一次更新：{{ formatDate(incident.next_update_at) }}</div>
          </div>
          <div v-if="store.incidents.length === 0" class="text-sm text-gray-500">暂无事故记录</div>
        </div>
      </UCard>

      <div class="md:col-span-2 space-y-4">
        <UCard v-if="store.selected">
          <template #header>
            <div class="flex items-center justify-between">
              <div>
                <div class="text-sm text-gray-500">{{ store.selected.incident.detection_source }}</div>
                <h2 class="text-xl font-semibold">{{ store.selected.incident.summary }}</h2>
              </div>
              <div class="flex gap-2 items-center">
                <UBadge :color="statusColor(store.selected.incident.status)">
                  {{ store.selected.incident.status }}
                </UBadge>
                <UBadge variant="soft">SEV: {{ store.selected.incident.severity.toUpperCase() }}</UBadge>
              </div>
            </div>
          </template>

          <div class="grid md:grid-cols-2 gap-4">
            <div>
              <h3 class="font-medium mb-2">事故详情</h3>
              <div class="space-y-2 text-sm">
                <div>检测时间：{{ formatDate(store.selected.incident.detected_at) }}</div>
                <div v-if="store.selected.incident.mitigation">缓解措施：{{ store.selected.incident.mitigation }}</div>
                <div v-if="store.selected.incident.root_cause">根因：{{ store.selected.incident.root_cause }}</div>
                <div class="space-x-2">
                  <UBadge v-for="label in visibleIncidentLabels" :key="label" variant="soft">#{{ label }}</UBadge>
                </div>
              </div>
            </div>
            <form class="space-y-3" @submit.prevent="updateStatus">
              <h3 class="font-medium">状态更新</h3>
              <USelect v-model="statusForm.status" :options="statusOptions" label="状态" size="sm" />
              <UTextarea v-model="statusForm.mitigation" label="Mitigation" size="sm" />
              <UInput v-model="statusForm.next_update_at" type="datetime-local" label="下一次更新" size="sm" />
              <div class="flex justify-end">
                <UButton type="submit" color="primary" :loading="store.loading">保存</UButton>
              </div>
            </form>
          </div>
        </UCard>

        <IncidentTimeline
          v-if="store.selected"
          :entries="store.selected.timeline"
          :checklist="store.selected.checklist_status"
          :saving="store.loading"
          @create="handleTimeline"
        />

        <UCard v-else class="text-sm text-gray-500">请选择左侧事故以查看详情</UCard>
      </div>
    </div>

    <UModal v-model="showCreate">
      <UCard>
        <template #header>声明新事故</template>
        <form class="space-y-3" @submit.prevent="submitCreate">
          <USelect v-model="createForm.severity" :options="severityOptions" label="严重级别" required />
          <USelect v-model="createForm.detection_source" :options="detectionSources" label="检测来源" required />
          <UInput v-model="createForm.summary" label="概述" placeholder="服务降级" required />
          <UTextarea v-model="createForm.mitigation" label="当前缓解步骤" />
          <UTextarea v-model="labelInput" label="标签 (用逗号分隔)" placeholder="#availability,#performance" />
          <UTextarea v-model="impactInput" label="影响 (JSON 可选)" placeholder='{ "tenants": ["acme"], "estimation": "$10k" }' />
          <div class="flex justify-end gap-2">
            <UButton variant="ghost" @click="showCreate = false">取消</UButton>
            <UButton type="submit" color="primary" :loading="store.loading">提交</UButton>
          </div>
        </form>
      </UCard>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import { useToast } from '#imports'
import { useIncidentsStore } from '~/stores/operations/useIncidentsStore'
import type { IncidentUpdatePayload, TimelineCreatePayload, IncidentDraftPayload } from '~/types/operations'
import IncidentTimeline from '~/components/operations/IncidentTimeline.vue'

const store = useIncidentsStore()
const toast = useToast()

const showCreate = ref(false)
const labelInput = ref('')
const impactInput = ref('')

const createForm = reactive<IncidentDraftPayload>({
  severity: 'sev1',
  detection_source: 'monitoring',
  summary: '',
  mitigation: '',
  labels: {},
})

const statusForm = reactive<IncidentUpdatePayload>({
  status: undefined,
  mitigation: '',
  next_update_at: undefined,
})

const severityOptions = [
  { label: 'SEV-0', value: 'sev0' },
  { label: 'SEV-1', value: 'sev1' },
  { label: 'SEV-2', value: 'sev2' },
  { label: 'SEV-3', value: 'sev3' },
  { label: 'SEV-4', value: 'sev4' },
]

const detectionSources = [
  { label: 'Monitoring', value: 'monitoring' },
  { label: 'Support', value: 'support' },
  { label: 'Vendor', value: 'vendor' },
  { label: 'Security', value: 'security' },
  { label: 'Dependency', value: 'dependency' },
]

const statusOptions = [
  { label: 'Detected', value: 'detected' },
  { label: 'Acknowledged', value: 'acknowledged' },
  { label: 'Mitigated', value: 'mitigated' },
  { label: 'Monitoring', value: 'monitoring' },
  { label: 'Resolved', value: 'resolved' },
  { label: 'Closed', value: 'closed' },
]

const startCreate = () => {
  labelInput.value = ''
  impactInput.value = ''
  Object.assign(createForm, {
    severity: 'sev1',
    detection_source: 'monitoring',
    summary: '',
    mitigation: '',
    labels: {},
  })
  showCreate.value = true
}

const parseLabels = (input: string) => {
  const map: Record<string, boolean> = {}
  input
    .split(',')
    .map((label) => label.trim())
    .filter(Boolean)
    .forEach((label) => {
      const normalized = label.replace(/^#/, '')
      map[normalized] = true
    })
  return map
}

const submitCreate = async () => {
  try {
    const payload: IncidentDraftPayload = {
      severity: createForm.severity,
      detection_source: createForm.detection_source,
      summary: createForm.summary,
      mitigation: createForm.mitigation,
      labels: parseLabels(labelInput.value),
    }
    if (impactInput.value.trim()) {
      try {
        payload.impact = JSON.parse(impactInput.value)
      } catch {
        toast.add({ title: '影响 JSON 无法解析', color: 'error' })
        return
      }
    }
    const record = await store.createIncident(payload)
    if (record?.incident.id) {
      await store.fetchIncident(record.incident.id)
    }
    showCreate.value = false
    toast.add({ title: '事故已声明', color: 'success' })
  } catch (err: any) {
    toast.add({ title: err?.message ?? '声明失败', color: 'error' })
  }
}

const selectIncident = async (incidentId: string) => {
  try {
    await store.fetchIncident(incidentId)
    if (store.selected?.incident.status) {
      statusForm.status = store.selected.incident.status
      statusForm.mitigation = store.selected.incident.mitigation
      statusForm.next_update_at = store.selected.incident.next_update_at || undefined
    }
  } catch (err: any) {
    toast.add({ title: err?.message ?? '加载事故失败', color: 'error' })
  }
}

const visibleIncidentLabels = computed(() => {
  const labels = store.selected?.incident.labels
  if (!labels) {
    return []
  }
  return Object.entries(labels)
    .filter(([, value]) => Boolean(value))
    .map(([label]) => label)
})

watch(() => store.selected?.incident, (incident) => {
  if (!incident) {
    statusForm.status = undefined
    statusForm.mitigation = ''
    statusForm.next_update_at = undefined
    return
  }
  statusForm.status = incident.status
  statusForm.mitigation = incident.mitigation
  statusForm.next_update_at = incident.next_update_at || undefined
}, { immediate: true })

const updateStatus = async () => {
  if (!store.selected) return
  try {
    await store.updateIncident(store.selected.incident.id, statusForm)
    toast.add({ title: '状态已更新', color: 'success' })
  } catch (err: any) {
    toast.add({ title: err?.message ?? '更新失败', color: 'error' })
  }
}

const handleTimeline = async (payload: TimelineCreatePayload) => {
  if (!store.selected) return
  try {
    await store.appendTimeline(store.selected.incident.id, payload)
    toast.add({ title: '已发布时间线更新', color: 'success' })
  } catch (err: any) {
    toast.add({ title: err?.message ?? '发布失败', color: 'error' })
  }
}

const refresh = async () => {
  await store.fetchIncidents()
  if (store.selected?.incident.id) {
    await store.fetchIncident(store.selected.incident.id)
  }
}

const formatDate = (value?: string | null) => {
  if (!value) return '未设定'
  return new Date(value).toLocaleString()
}

const statusColor = (status: string) => {
  switch (status) {
    case 'resolved':
    case 'closed':
      return 'success'
    case 'mitigated':
    case 'monitoring':
      return 'info'
    case 'acknowledged':
      return 'warning'
    default:
      return 'primary'
  }
}

onMounted(async () => {
  await store.fetchIncidents()
})
</script>
