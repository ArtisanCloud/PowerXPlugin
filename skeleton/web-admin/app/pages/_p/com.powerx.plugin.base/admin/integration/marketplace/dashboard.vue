<template>
  <UContainer class="py-10 space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-chart-bar" class="text-primary" />
        <span class="uppercase tracking-wide">Integration · Marketplace</span>
      </div>
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Usage & Revenue Dashboard</h1>
        <p class="text-gray-600 dark:text-gray-300 max-w-3xl">
          实时洞察安装量、调用频率与分润报表。选择租户/License 后加载趋势，关注告警并导出分润数据。
        </p>
      </div>
    </header>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="space-y-1">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">查询条件</h2>
            <p class="text-xs text-gray-500 dark:text-gray-400">填写租户与 License ID，选择时间窗口后点击加载。</p>
          </div>
          <UButton :loading="loading" icon="i-heroicons-bolt" color="primary" @click="loadDashboard">
            加载仪表盘
          </UButton>
        </div>
      </template>

      <div class="grid gap-4 md:grid-cols-2">
        <div class="space-y-3">
          <UInput v-model="tenantId" label="Tenant ID" placeholder="tenant-123" />
          <UInput v-model="licenseId" label="License ID" placeholder="license-abc" />
          <USelectMenu
            v-model="window"
            label="聚合窗口"
            :options="windowOptions"
            option-attribute="label"
            class="w-full"
          />
          <UInput v-model="metric" label="指标过滤 (可选)" placeholder="calls" />
        </div>
        <div class="space-y-3">
          <UInput v-model="vendorId" label="Vendor ID (可选)" placeholder="vendor-xyz" />
          <div class="flex gap-2">
            <UInput v-model="periodStart" type="date" label="报表开始" />
            <UInput v-model="periodEnd" type="date" label="报表结束" />
          </div>
          <UAlert v-if="error" color="red" icon="i-heroicons-exclamation-triangle">
            {{ error }}
          </UAlert>
        </div>
      </div>
    </UCard>

    <div v-if="loading" class="space-y-4">
      <USkeleton class="h-32" />
      <USkeleton class="h-32" />
    </div>

    <div v-else class="space-y-6">
      <div v-if="alerts.length" class="space-y-3">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">告警</h2>
        <div class="grid gap-3 md:grid-cols-2">
          <UAlert
            v-for="alert in alerts"
            :key="alert.code + alert.message"
            :color="alertColor(alert.severity)"
            :icon="alertIcon(alert.code)"
          >
            <template #title>{{ alert.code }}</template>
            {{ alert.message }}
          </UAlert>
        </div>
      </div>

      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <div class="space-y-1">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Usage 趋势</h2>
              <p class="text-xs text-gray-500 dark:text-gray-400">展示累计调用量、剩余配额与收入估算。</p>
            </div>
            <UBadge color="gray" variant="soft">{{ series.length }} Points</UBadge>
          </div>
        </template>

        <div v-if="!series.length" class="py-10 text-center text-gray-500 dark:text-gray-400">
          暂无数据，请确认输入的 Tenant / License 是否正确。
        </div>

        <UTable v-else :columns="usageColumns" :rows="usageRows" :sort="{ column: 'timestamp', direction: 'desc' }">
          <template #quota_remaining-data="{ row }">
            <span v-if="row.quota_remaining !== null">{{ row.quota_remaining }}</span>
            <span v-else class="text-gray-400">—</span>
          </template>
        </UTable>
      </UCard>

      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <div class="space-y-1">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">分润报表</h2>
              <p class="text-xs text-gray-500 dark:text-gray-400">根据查询条件列出分润明细。</p>
            </div>
            <UBadge color="gray" variant="soft">{{ reports.length }} Reports</UBadge>
          </div>
        </template>

        <div v-if="reportsLoading" class="py-6"><USkeleton class="h-24" /></div>
        <div v-else-if="!reports.length" class="py-10 text-center text-gray-500 dark:text-gray-400">
          暂无报表记录。
        </div>

        <UTable v-else :columns="reportColumns" :rows="reportRows" />
      </UCard>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useRoute } from "vue-router"
import { useToast } from "#imports"
import { storeToRefs } from "pinia"
import { useUsageMetrics } from "~/composables/useUsageMetrics"
import { useMarketplaceAnalyticsStore } from "~/stores/marketplaceAnalytics"

const route = useRoute()
const toast = useToast()
const analyticsStore = useMarketplaceAnalyticsStore()
const { reports, reportsLoading } = storeToRefs(analyticsStore)
const { series, alerts, loading, error, load } = useUsageMetrics()

const tenantId = ref<string>((route.query.tenant as string) || "")
const licenseId = ref<string>((route.query.license as string) || "")
const window = ref<string>((route.query.window as string) || "day")
const metric = ref<string>((route.query.metric as string) || "")
const vendorId = ref<string>((route.query.vendor as string) || "")
const periodStart = ref<string>(new Date().toISOString().slice(0, 10))
const periodEnd = ref<string>(new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString().slice(0, 10))

const usageColumns = [
  { key: "timestamp", label: "时间" },
  { key: "metric", label: "指标" },
  { key: "value", label: "累计值" },
  { key: "quota_remaining", label: "剩余额度" },
  { key: "revenue", label: "收入" },
]

const reportColumns = [
  { key: "period", label: "结算区间" },
  { key: "gross_amount", label: "总收入" },
  { key: "vendor_share", label: "Vendor" },
  { key: "platform_share", label: "Platform" },
  { key: "fees", label: "费用" },
  { key: "status", label: "状态" },
]

const windowOptions = [
  { value: "hour", label: "小时" },
  { value: "day", label: "天" },
  { value: "month", label: "月" },
]

const usageRows = computed(() =>
  series.value.map((point) => ({
    timestamp: new Date(point.timestamp).toLocaleString(),
    metric: point.metric,
    value: Number(point.value.toFixed(2)),
    quota_remaining:
      typeof point.quota_remaining === "number"
        ? Number(point.quota_remaining.toFixed(2))
        : null,
    revenue: `${point.revenue.toFixed(2)} ${point.currency ?? ""}`.trim(),
  }))
)

const reportRows = computed(() =>
  reports.value.map((report) => ({
    period: `${report.period_start} → ${report.period_end}`,
    gross_amount: `${report.gross_amount.toFixed(2)} ${report.currency}`,
    vendor_share: `${report.vendor_share.toFixed(2)} ${report.currency}`,
    platform_share: `${report.platform_share.toFixed(2)} ${report.currency}`,
    fees: `${report.fees.toFixed(2)} ${report.currency}`,
    status: report.status,
  }))
)

function alertColor(severity: string) {
  switch (severity) {
    case "critical":
      return "red"
    case "warning":
      return "orange"
    default:
      return "gray"
  }
}

function alertIcon(code: string) {
  if (code === "quota_exceeded") return "i-heroicons-no-symbol"
  if (code === "usage_spike") return "i-heroicons-fire"
  return "i-heroicons-information-circle"
}

async function loadDashboard() {
  if (!tenantId.value || !licenseId.value) {
    toast.add({ title: "请输入 Tenant 与 License", color: "orange" })
    return
  }
  await load(tenantId.value, licenseId.value, {
    window: window.value,
    metric: metric.value || undefined,
  })
  await analyticsStore.fetchReports({
    vendorId: vendorId.value || undefined,
    periodStart: periodStart.value || undefined,
    periodEnd: periodEnd.value || undefined,
  })
  if (!series.value.length) {
    toast.add({ title: "未发现数据", description: "请确认租户与 License 是否正确", color: "gray" })
  }
}

onMounted(() => {
  if (tenantId.value && licenseId.value) {
    loadDashboard()
  }
})
</script>
