<template>
  <UContainer class="py-10 space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-chart-bar-square" class="text-primary" />
        <span class="uppercase tracking-wide">Integration · Insights</span>
      </div>
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">关键指标</h1>
        <p class="text-gray-600 dark:text-gray-300 max-w-3xl">
          观察 Envelope 吞吐、Webhook 成功率与 Secrets 轮换健康度，及时发现异常。
        </p>
      </div>
    </header>

    <UCard>
      <template #header>
        <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
          <UIcon name="i-heroicons-bolt" />
          <span>近 5 分钟 Envelope 吞吐</span>
        </div>
      </template>
      <div class="text-3xl font-semibold text-primary">
        {{ formatNumber(metrics.envelope_rate) }} <span class="text-base font-normal text-gray-500">req/s</span>
      </div>
    </UCard>

    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <UCard>
        <template #header>
          <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <UIcon name="i-heroicons-wifi" />
            <span>Webhook 成功率 (5m)</span>
          </div>
        </template>
        <div class="space-y-2">
          <UProgress :value="metrics.webhook_success_rate" :max="100" color="primary" />
          <p class="text-sm text-gray-500 dark:text-gray-400">
            成功 {{ metrics.webhook_success_rate.toFixed(1) }}% · 重试 {{ metrics.webhook_retry_rate.toFixed(1) }}%
          </p>
        </div>
      </UCard>

      <UCard>
        <template #header>
          <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <UIcon name="i-heroicons-key" />
            <span>Secrets 待轮换</span>
          </div>
        </template>
        <div class="space-y-2">
          <div class="text-3xl font-semibold" :class="metrics.secrets_due_now > 0 ? 'text-orange-500' : 'text-primary'">
            {{ metrics.secrets_due_now }}
          </div>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            24 小时内待轮换：{{ metrics.secrets_due_24h }}
          </p>
        </div>
      </UCard>
      <UCard>
        <template #header>
          <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <UIcon name="i-heroicons-arrow-path-rounded-square" />
            <span>幂等冲突 (5m)</span>
          </div>
        </template>
        <div class="space-y-2">
          <div class="text-3xl font-semibold" :class="metrics.idempotency_conflict_rate > 0 ? 'text-amber-500' : 'text-primary'">
            {{ metrics.idempotency_conflict_rate.toFixed(2) }}
            <span class="text-base font-normal text-gray-500">/min</span>
          </div>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            监控 `powerx_integration_idempotency_events_total{outcome="conflict"}`。
          </p>
        </div>
      </UCard>
      <UCard>
        <template #header>
          <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <UIcon name="i-heroicons-queue-list" />
            <span>Webhook DLQ 速率 (5m)</span>
          </div>
        </template>
        <div class="space-y-2">
          <div class="text-3xl font-semibold" :class="metrics.webhook_dlq_rate > 0 ? 'text-red-500' : 'text-primary'">
            {{ metrics.webhook_dlq_rate.toFixed(2) }}
            <span class="text-base font-normal text-gray-500">/min</span>
          </div>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            来源于 `powerx_integration_webhook_attempts_total{status="dlq"}`。
          </p>
        </div>
      </UCard>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
          <UIcon name="i-heroicons-book-open" />
          <span>Runbook</span>
        </div>
      </template>
      <div class="space-y-2 text-sm text-gray-600 dark:text-gray-300">
        <p>· Envelope 指标来自 `powerx_integration_envelopes_total`，建议在 Grafana 套用 `docs/observability/integration-dashboard.json`。</p>
        <p>· Webhook 异常可在管理端 Webhooks 页面执行 replay，并关注 `integration.webhooks:*` 审批记录与 DLQ 速率。</p>
        <p>· Secrets 轮换提醒由 Secret Rotation Worker 触发，详见 `docs/security/integration.md`。</p>
        <p>· 幂等冲突上升时，交叉检查 GrantMatrix 与下游调用，确保 SC-005 告警触发。</p>
      </div>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
const runtimeConfig = useRuntimeConfig()
const toast = useToast()
const metrics = reactive({
  envelope_rate: 0,
  webhook_success_rate: 100,
  webhook_retry_rate: 0,
  secrets_due_now: 0,
  secrets_due_24h: 0,
  idempotency_conflict_rate: 0,
  webhook_dlq_rate: 0,
})

function formatNumber(value: number) {
  if (value >= 1000) return (value / 1000).toFixed(1) + 'k'
  return value.toFixed(1)
}

async function loadMetrics() {
  try {
    const response = await $fetch(`${runtimeConfig.public.apiBaseUrl}/admin/integration/metrics/summary`)
    Object.assign(metrics, response)
  } catch (error) {
    toast.add({ title: '无法加载指标', description: String(error), color: 'red' })
  }
}

onMounted(() => {
  void loadMetrics()
})
</script>
