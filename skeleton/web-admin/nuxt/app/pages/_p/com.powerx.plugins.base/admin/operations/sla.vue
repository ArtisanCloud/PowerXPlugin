<template>
  <UContainer>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-semibold">SLA Dashboard</h1>
        <p class="text-sm text-gray-500">Monitor commitments, actual performance, and automate incentives.</p>
      </div>
      <div class="space-x-2">
        <UButton color="primary" variant="ghost" @click="refresh" :loading="store.loading">刷新</UButton>
        <UButton color="primary" @click="recompute" :loading="store.loading">触发重算</UButton>
      </div>
    </div>

    <UAlert v-if="store.error" color="error" :title="store.error" class="mb-4" />

    <div class="grid gap-4 md:grid-cols-3 mb-8" v-if="store.profiles.length">
      <SlaScoreCard v-for="profile in store.profiles" :key="profile.id" :profile="profile" />
    </div>

    <UCard class="mb-6">
      <template #header>更新 SLA 目标</template>
      <form class="grid gap-4 md:grid-cols-5" @submit.prevent="submitTargets">
        <USelect v-model="targets.planType" :options="planOptions" label="Plan Type" required />
        <UInput v-model.number="targets.targets.uptimeTarget" type="number" step="0.01" label="Uptime %" required />
        <UInput v-model.number="targets.targets.responseTargetMs" type="number" label="Response ms" required />
        <UInput v-model.number="targets.targets.successTargetPct" type="number" step="0.01" label="Success %" required />
        <UInput v-model.number="targets.targets.supportFrtTargetHours" type="number" step="0.1" label="FRT Hours" required />
        <div class="md:col-span-5 flex justify-end">
          <UButton type="submit" color="primary" :loading="store.loading">保存目标</UButton>
        </div>
      </form>
    </UCard>

    <UCard>
      <template #header>同步实际指标</template>
      <form class="grid gap-4 md:grid-cols-5" @submit.prevent="submitActuals">
        <USelect v-model="actuals.planType" :options="planOptions" label="Plan Type" required />
        <UInput v-model.number="actuals.actuals.uptimeActual" type="number" step="0.01" label="Uptime %" required />
        <UInput v-model.number="actuals.actuals.responseActualMs" type="number" label="Response ms" required />
        <UInput v-model.number="actuals.actuals.successActualPct" type="number" step="0.01" label="Success %" required />
        <UInput v-model.number="actuals.actuals.supportFrtActualHours" type="number" step="0.1" label="FRT Hours" required />
        <div class="md:col-span-5 flex justify-end">
          <UButton type="submit" color="secondary" :loading="store.loading">上传实际值</UButton>
        </div>
      </form>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { useToast } from '#imports'
import SlaScoreCard from '~/components/operations/SlaScoreCard.vue'
import { useSlaStore } from '~/stores/operations/useSlaStore'
import type { SlaProfileUpdatePayload, SlaActualsPayload } from '~/types/operations'

const store = useSlaStore()
const toast = useToast()

const planOptions = [
  { label: 'Real-time', value: 'real_time' },
  { label: 'Transactional', value: 'transactional' },
  { label: 'Utility', value: 'utility' },
]

const targets = reactive<SlaProfileUpdatePayload>({
  planType: 'real_time',
  targets: {
    uptimeTarget: 99.9,
    responseTargetMs: 600,
    successTargetPct: 99.5,
    supportFrtTargetHours: 4,
  },
})

const actuals = reactive<SlaActualsPayload>({
  planType: 'real_time',
  actuals: {
    uptimeActual: 99.0,
    responseActualMs: 800,
    successActualPct: 98.5,
    supportFrtActualHours: 6,
  },
})

const refresh = async () => {
  await store.fetchProfiles()
}

const recompute = async () => {
  try {
    await store.recompute()
    toast.add({ title: '已触发重算', color: 'success' })
  } catch (err: any) {
    toast.add({ title: err?.message ?? '重算失败', color: 'error' })
  }
}

const submitTargets = async () => {
  try {
    await store.upsertProfile(targets)
    toast.add({ title: 'SLA 目标已更新', color: 'success' })
  } catch (err: any) {
    toast.add({ title: err?.message ?? '保存失败', color: 'error' })
  }
}

const submitActuals = async () => {
  try {
    await store.updateActuals(actuals)
    toast.add({ title: '实际指标已同步', color: 'success' })
  } catch (err: any) {
    toast.add({ title: err?.message ?? '同步失败', color: 'error' })
  }
}

await refresh()
</script>
