<template>
  <UContainer class="py-10 space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-beaker" class="text-primary" />
        <span class="uppercase tracking-wide">Integration · Marketplace · Recommendation</span>
      </div>
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Recommendation Experiments</h1>
        <p class="text-gray-600 dark:text-gray-300 max-w-3xl">
          调整推荐曝光参数、查看当前实验配置，并手动刷新推荐权重以验证曝光效果。
        </p>
      </div>
    </header>

    <div class="grid gap-6 lg:grid-cols-[minmax(0,1.5fr)_minmax(0,2fr)]">
      <UCard :ui="{ body: 'space-y-4' }">
        <template #header>
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
              <UIcon name="i-heroicons-adjustments-horizontal" />
              <span>实验配置</span>
            </div>
            <UButton color="gray" variant="soft" :loading="loading" icon="i-heroicons-arrow-path" @click="loadConfig">
              刷新
            </UButton>
          </div>
        </template>

        <div class="space-y-3">
          <div class="flex justify-between text-sm">
            <span class="text-gray-600 dark:text-gray-300">推荐服务是否启用</span>
            <UBadge :color="config.enabled ? 'primary' : 'gray'">{{ config.enabled ? '启用' : '停用' }}</UBadge>
          </div>
          <div class="flex justify-between text-sm text-gray-600 dark:text-gray-300">
            <span>默认权重</span>
            <span>{{ config.defaultWeight.toFixed(2) }}</span>
          </div>
          <div class="flex justify-between text-sm text-gray-600 dark:text-gray-300">
            <span>实验主题 Topic</span>
            <span>{{ config.experimentTopic || '—' }}</span>
          </div>
          <div class="flex justify-between text-sm text-gray-600 dark:text-gray-300">
            <span>刷新频率 (分钟)</span>
            <span>{{ config.frequencyMinutes }}</span>
          </div>
        </div>

        <div class="mt-4 space-y-4">
          <UFormGroup label="调整默认权重" help="实验用途，不会持久化到配置文件">
            <div class="flex gap-2">
              <UInput v-model.number="experiment.defaultWeight" type="number" min="0" step="0.05" class="w-32" />
              <UButton color="primary" :loading="saving" @click="updateDefaultWeight">应用</UButton>
            </div>
          </UFormGroup>
          <UButton color="primary" variant="soft" :loading="syncing" icon="i-heroicons-play" @click="triggerSync">
            手动刷新推荐权重
          </UButton>
        </div>
      </UCard>

      <UCard :ui="{ body: 'space-y-4' }">
        <template #header>
          <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <UIcon name="i-heroicons-sparkles" />
            <span>当前推荐结果 (Top 10)</span>
          </div>
        </template>

        <div v-if="loading" class="flex items-center justify-center py-12">
          <UProgress size="md" animation="carousel" />
        </div>

        <div v-else>
          <UTable :columns="columns" :rows="topListings">
            <template #title-data="{ row }">
              <div class="space-y-0.5">
                <span class="font-medium">{{ row.title }}</span>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ row.plugin_id }} · {{ row.vendor_id }}
                </p>
              </div>
            </template>
            <template #recommended_weight-data="{ row }">
              <span class="text-sm font-medium text-primary-600 dark:text-primary-300">
                {{ row.recommended_weight.toFixed(4) }}
              </span>
            </template>
          </UTable>
          <div v-if="!topListings.length" class="text-sm text-gray-500 dark:text-gray-400">
            暂无推荐数据。
          </div>
        </div>
      </UCard>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
const nuxtApp = useNuxtApp()
const config = reactive({
  enabled: true,
  defaultWeight: 0,
  experimentTopic: "",
  frequencyMinutes: 60,
})
const topListings = ref<Array<Record<string, any>>>([])
const loading = ref(false)
const syncing = ref(false)
const saving = ref(false)
const experiment = reactive({
  defaultWeight: 0,
})
const toast = useToast()
const runtime = useRuntimeConfig()
const apiBase = computed(() => runtime.public.apiBaseUrl as string)

const columns = [
  { key: "title", label: "Listing" },
  { key: "status", label: "状态" },
  { key: "recommended_weight", label: "权重" },
]

async function loadConfig() {
  loading.value = true
  try {
    const res = await nuxtApp.$fetch<{ config: any; top_listings: any[] }>(`${apiBase.value}/admin/marketplace/recommendation/config`)
    const cfg = res.config || {}
    config.enabled = cfg.enabled ?? true
    config.defaultWeight = cfg.default_weight ?? 0
    config.experimentTopic = cfg.experiment_topic ?? ""
    config.frequencyMinutes = cfg.frequency_minutes ?? 60
    experiment.defaultWeight = config.defaultWeight
    topListings.value = res.top_listings || []
  } catch (error) {
    toast.add({ title: "加载失败", description: String(error), color: "red" })
  } finally {
    loading.value = false
  }
}

async function triggerSync() {
  syncing.value = true
  try {
    const res = await nuxtApp.$fetch<{ updated: number }>(`${apiBase.value}/admin/marketplace/recommendation/sync`, {
      method: "POST",
    })
    toast.add({ title: "刷新完成", description: `更新 ${res.updated || 0} 个权重`, color: "primary" })
    await loadConfig()
  } catch (error) {
    toast.add({ title: "刷新失败", description: String(error), color: "red" })
  } finally {
    syncing.value = false
  }
}

async function updateDefaultWeight() {
  saving.value = true
  try {
    await nuxtApp.$fetch(`${apiBase.value}/admin/marketplace/recommendation/experiment`, {
      method: "PATCH",
      body: { default_weight: experiment.defaultWeight },
    })
    toast.add({ title: "已更新默认权重", color: "primary" })
    config.defaultWeight = experiment.defaultWeight
  } catch (error) {
    toast.add({ title: "更新失败", description: String(error), color: "red" })
  } finally {
    saving.value = false
  }
}

onMounted(loadConfig)
</script>
