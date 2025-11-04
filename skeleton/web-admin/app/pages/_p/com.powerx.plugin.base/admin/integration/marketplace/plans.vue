<template>
  <UContainer class="py-10 space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-currency-dollar" class="text-primary" />
        <span class="uppercase tracking-wide">Integration · Marketplace</span>
      </div>
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Pricing Plans</h1>
        <p class="text-gray-600 dark:text-gray-300 max-w-3xl">
          维护 Marketplace Listing 的价格计划与 Usage Tiers。支持新增/删除计划、设置默认计划以及编辑分级计价配置。
        </p>
      </div>
    </header>

    <div class="flex flex-wrap items-center gap-4">
      <UInput
        v-model="filters.search"
        placeholder="搜索标题或插件"
        class="w-72"
        icon="i-heroicons-magnifying-glass"
        @keyup.enter="loadListings"
      />
      <USelectMenu
        v-model="filters.status"
        :options="statusOptions"
        multiple
        placeholder="筛选状态"
        class="w-64"
      />
      <UButton color="primary" icon="i-heroicons-arrow-path" :loading="loading" @click="loadListings">
        刷新
      </UButton>
      <UButton
        color="primary"
        icon="i-heroicons-plus-circle"
        class="ml-auto"
        :disabled="!selectedListing"
        @click="addPlan"
      >
        新增计划
      </UButton>
    </div>

    <div class="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,3fr)]">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
              <UIcon name="i-heroicons-list-bullet" />
              <span>Marketplace Listings</span>
            </div>
          </div>
        </template>

        <UTable
          :columns="columns"
          :rows="listings"
          :loading="loading"
          :sort="{ column: 'updated_at', direction: 'desc' }"
          @select="selectListing"
        >
          <template #title-data="{ row }">
            <div class="space-y-0.5">
              <span class="font-medium">{{ row.title }}</span>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ row.plugin_id }}</p>
            </div>
          </template>
          <template #status-data="{ row }">
            <UBadge :color="statusBadgeColor(row.status)" class="uppercase tracking-wide">{{ row.status }}</UBadge>
          </template>
        </UTable>

        <div v-if="!loading && !listings.length" class="py-8 text-center text-gray-500 dark:text-gray-400">
          尚无 Listing，请先完成上架。
        </div>
      </UCard>

      <UCard v-if="selectedListing">
        <template #header>
          <div class="flex items-center justify-between gap-2">
            <div class="space-y-1">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ selectedListing?.title }} 价格计划
              </h2>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                Plugin {{ selectedListing?.plugin_id }} · Tenant Listing
              </p>
            </div>
            <UButton color="primary" :loading="saving" @click="savePlans">保存变更</UButton>
          </div>
        </template>

        <div v-if="!plans.length" class="py-8 text-center text-gray-500 dark:text-gray-400">
          尚未配置价格计划，点击上方“新增计划”开始。
        </div>

        <div v-else class="space-y-4">
          <div class="flex flex-wrap gap-2">
            <UButton
              v-for="(plan, index) in plans"
              :key="plan.key"
              :color="index === selectedPlanIndex ? 'primary' : 'gray'"
              variant="soft"
              size="sm"
              @click="selectedPlanIndex = index"
            >
              {{ plan.plan_code || `计划 ${index + 1}` }}
            </UButton>
          </div>

          <div v-if="currentPlan" class="space-y-6">
            <div class="flex items-center justify-between">
              <div class="space-y-1">
                <div class="flex items-center gap-2">
                  <h3 class="text-base font-semibold text-gray-900 dark:text-white">计划配置</h3>
                  <UBadge v-if="currentPlan.is_default" color="primary" class="uppercase">默认</UBadge>
                </div>
                <p class="text-xs text-gray-500 dark:text-gray-400">唯一 Plan Code 将作为 SDK 与 Billing 的引用 ID。</p>
              </div>
              <div class="flex items-center gap-2">
                <UButton size="xs" color="primary" variant="soft" @click="markDefault(currentPlan)">
                  设为默认
                </UButton>
                <UButton size="xs" color="red" variant="soft" @click="removePlan(selectedPlanIndex)">
                  删除
                </UButton>
              </div>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <UFormGroup label="Plan Code" required>
                <UInput v-model="currentPlan.plan_code" placeholder="enterprise" />
              </UFormGroup>
              <UFormGroup label="Plan 类型" required>
                <USelect v-model="currentPlan.plan_type" :options="planTypeOptions" />
              </UFormGroup>
              <UFormGroup label="币种" required>
                <UInput v-model="currentPlan.currency" placeholder="USD" />
              </UFormGroup>
              <UFormGroup label="金额">
                <UInput v-model="currentPlan.amount" type="number" step="0.01" placeholder="29.99" />
              </UFormGroup>
              <UFormGroup label="计费周期">
                <USelect v-model="currentPlan.billing_period" :options="billingPeriodOptions" />
              </UFormGroup>
              <UFormGroup label="试用天数">
                <UInput v-model="currentPlan.trial_days" type="number" min="0" placeholder="14" />
              </UFormGroup>
              <UFormGroup label="额度上限">
                <UInput v-model="currentPlan.quota_limit" type="number" min="0" placeholder="1000" />
              </UFormGroup>
              <UFormGroup label="超额策略">
                <UInput v-model="currentPlan.overage_policy" placeholder="throttle|billable" />
              </UFormGroup>
            </div>

            <UFormGroup label="Feature Matrix (JSON)">
              <UTextarea v-model="currentPlan.feature_matrix_text" :rows="4" />
            </UFormGroup>

            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <h3 class="text-sm font-semibold text-gray-800 dark:text-gray-200">Usage Tiers</h3>
                <UButton size="xs" color="gray" variant="soft" icon="i-heroicons-plus" @click="addTier(currentPlan)">
                  添加分级
                </UButton>
              </div>
              <div v-if="!currentPlan.tiers.length" class="text-sm text-gray-500 dark:text-gray-400">
                暂无用量分级，默认按固定价格收费。
              </div>
              <div v-else class="space-y-3">
                <div
                  v-for="(tier, tierIndex) in currentPlan.tiers"
                  :key="tier.key"
                  class="rounded border border-gray-200 p-3 dark:border-gray-700"
                >
                  <div class="grid gap-3 md:grid-cols-2">
                    <UFormGroup label="Metric" required>
                      <UInput v-model="tier.metric" placeholder="requests" />
                    </UFormGroup>
                    <UFormGroup label="Range From" required>
                      <UInput v-model="tier.range_from" type="number" min="0" />
                    </UFormGroup>
                    <UFormGroup label="Range To">
                      <UInput v-model="tier.range_to" type="number" min="0" />
                    </UFormGroup>
                    <UFormGroup label="Unit Amount" required>
                      <UInput v-model="tier.unit_amount" type="number" step="0.0001" />
                    </UFormGroup>
                    <UFormGroup label="Unit Name">
                      <UInput v-model="tier.unit_name" placeholder="API call" />
                    </UFormGroup>
                  </div>
                  <div class="mt-2 flex justify-end">
                    <UButton size="xs" color="red" variant="ghost" @click="removeTier(currentPlan, tierIndex)">移除</UButton>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </UCard>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { useNuxtApp, useRuntimeConfig, useToast } from "#imports"
import type {
  MarketplaceListing,
  MarketplaceListingStatus,
  MarketplacePricingPlan,
  MarketplacePricingTier,
} from "~/types/integration"

interface TierForm {
  key: string
  id?: string
  metric: string
  range_from: string
  range_to: string
  unit_amount: string
  unit_name: string
}

interface PlanForm {
  key: string
  id?: string
  plan_code: string
  plan_type: MarketplacePricingPlan["plan_type"]
  currency: string
  amount: string
  billing_period: string
  trial_days: string
  quota_limit: string
  overage_policy: string
  feature_matrix_text: string
  is_default: boolean
  tiers: TierForm[]
}

definePageMeta({
  middleware: ["layout.global"],
})

const toast = useToast()
const nuxtApp = useNuxtApp()
const runtime = useRuntimeConfig()
const apiBase = computed(() => runtime.public.apiBaseUrl as string)

const loading = ref(false)
const saving = ref(false)
const listings = ref<MarketplaceListing[]>([])
const selectedListing = ref<MarketplaceListing | null>(null)
const plans = ref<PlanForm[]>([])
const selectedPlanIndex = ref(0)

const filters = reactive<{ status: MarketplaceListingStatus[]; search: string }>({
  status: [],
  search: "",
})

const columns = [
  { key: "title", label: "Listing" },
  { key: "status", label: "状态" },
  { key: "updated_at", label: "更新时间" },
]

const statusOptions = [
  { label: "草稿", value: "draft" },
  { label: "审核中", value: "in_review" },
  { label: "已发布", value: "published" },
  { label: "已暂停", value: "suspended" },
]

const planTypeOptions = [
  { label: "免费", value: "free" },
  { label: "一次性", value: "one_time" },
  { label: "订阅", value: "subscription" },
  { label: "按量", value: "usage" },
]

const billingPeriodOptions = [
  { label: "不适用", value: "" },
  { label: "Monthly", value: "monthly" },
  { label: "Quarterly", value: "quarterly" },
  { label: "Yearly", value: "yearly" },
]

const currentPlan = computed(() => plans.value[selectedPlanIndex.value] ?? null)

function statusBadgeColor(status: MarketplaceListingStatus) {
  switch (status) {
    case "published":
      return "primary"
    case "in_review":
      return "orange"
    case "suspended":
      return "red"
    default:
      return "gray"
  }
}

async function loadListings() {
  loading.value = true
  try {
    const response = await nuxtApp.$fetch<{ data: { items: MarketplaceListing[] } }>(
      `${apiBase.value}/admin/marketplace/listings`,
      {
        query: {
          search: filters.search || undefined,
          status: filters.status.join(",") || undefined,
        },
      }
    )
    const payload = response?.data
    listings.value = payload?.items ?? []
    if (!listings.value.length) {
      selectedListing.value = null
      plans.value = []
      return
    }
    if (!selectedListing.value) {
      selectedListing.value = listings.value[0]
    } else {
      const updated = listings.value.find((item) => item.id === selectedListing.value?.id)
      if (updated) {
        selectedListing.value = updated
      }
    }
    if (selectedListing.value) {
      await loadListingDetail(selectedListing.value.id)
    }
  } catch (error) {
    toast.add({ title: "加载失败", description: String(error), color: "red" })
  } finally {
    loading.value = false
  }
}

async function loadListingDetail(id: string) {
  try {
    const response = await nuxtApp.$fetch<{ data: MarketplaceListing }>(
      `${apiBase.value}/admin/marketplace/listings/${id}`
    )
    const listing = response?.data
    if (!listing) {
      return
    }
    selectedListing.value = listing
    plans.value = (listing.pricing_plans || []).map(mapPlan)
    if (plans.value.length === 0) {
      selectedPlanIndex.value = 0
    } else if (selectedPlanIndex.value >= plans.value.length) {
      selectedPlanIndex.value = plans.value.length - 1
    }
  } catch (error) {
    toast.add({ title: "加载详情失败", description: String(error), color: "red" })
  }
}

function mapPlan(plan: MarketplacePricingPlan): PlanForm {
  return {
    key: plan.id || uuid(),
    id: plan.id,
    plan_code: plan.plan_code,
    plan_type: plan.plan_type,
    currency: plan.currency,
    amount: plan.amount != null ? String(plan.amount) : "",
    billing_period: plan.billing_period || "",
    trial_days: plan.trial_period_days != null ? String(plan.trial_period_days) : "",
    quota_limit: plan.quota_limit != null ? String(plan.quota_limit) : "",
    overage_policy: plan.overage_policy || "",
    feature_matrix_text: plan.feature_matrix ? JSON.stringify(plan.feature_matrix, null, 2) : "",
    is_default: Boolean(plan.is_default),
    tiers: (plan.tiers || []).map(mapTier),
  }
}

function mapTier(tier: MarketplacePricingTier): TierForm {
  return {
    key: tier.id || uuid(),
    id: tier.id,
    metric: tier.metric,
    range_from: String(tier.range_from ?? 0),
    range_to: tier.range_to != null ? String(tier.range_to) : "",
    unit_amount: String(tier.unit_amount ?? 0),
    unit_name: tier.unit_name || "",
  }
}

function selectListing(row: MarketplaceListing | null) {
  if (!row) {
    selectedListing.value = null
    plans.value = []
    return
  }
  selectedListing.value = row
  loadListingDetail(row.id)
}

function addPlan() {
  if (!selectedListing.value) {
    return
  }
  plans.value.push({
    key: uuid(),
    plan_code: "",
    plan_type: "subscription",
    currency: "USD",
    amount: "",
    billing_period: "monthly",
    trial_days: "",
    quota_limit: "",
    overage_policy: "",
    feature_matrix_text: "",
    is_default: plans.value.length === 0,
    tiers: [],
  })
  selectedPlanIndex.value = plans.value.length - 1
}

function removePlan(index: number) {
  plans.value.splice(index, 1)
  if (selectedPlanIndex.value >= plans.value.length) {
    selectedPlanIndex.value = plans.value.length - 1
  }
}

function addTier(plan: PlanForm) {
  plan.tiers.push({
    key: uuid(),
    metric: "",
    range_from: "0",
    range_to: "",
    unit_amount: "0",
    unit_name: "",
  })
}

function removeTier(plan: PlanForm, index: number) {
  plan.tiers.splice(index, 1)
}

function markDefault(plan: PlanForm) {
  plans.value.forEach((item) => {
    item.is_default = item.key === plan.key
  })
}

function parseNumber(value: string): number | undefined {
  if (!value || Number.isNaN(Number(value))) {
    return undefined
  }
  return Number(value)
}

async function savePlans() {
  if (!selectedListing.value) {
    return
  }
  if (!plans.value.length) {
    toast.add({ title: "至少需要一个计划", color: "orange" })
    return
  }
  if (!plans.value.some((plan) => plan.is_default)) {
    plans.value[0].is_default = true
  }

  const payload = [] as any[]
  for (const plan of plans.value) {
    if (!plan.plan_code.trim()) {
      toast.add({ title: "Plan Code 不能为空", color: "orange" })
      return
    }
    let featureMatrix: Record<string, any> | undefined
    if (plan.feature_matrix_text.trim()) {
      try {
        featureMatrix = JSON.parse(plan.feature_matrix_text)
      } catch (error) {
        toast.add({ title: "Feature Matrix 需要合法 JSON", description: String(error), color: "red" })
        return
      }
    }
    const tiers = plan.tiers.map((tier) => ({
      id: tier.id,
      metric: tier.metric,
      range_from: parseNumber(tier.range_from) ?? 0,
      range_to: parseNumber(tier.range_to),
      unit_amount: parseNumber(tier.unit_amount) ?? 0,
      unit_name: tier.unit_name || "",
    }))
    payload.push({
      id: plan.id,
      plan_code: plan.plan_code.trim(),
      plan_type: plan.plan_type,
      currency: plan.currency.trim(),
      amount: parseNumber(plan.amount),
      billing_period: plan.billing_period,
      trial_days: parseNumber(plan.trial_days),
      quota_limit: parseNumber(plan.quota_limit),
      overage_policy: plan.overage_policy.trim(),
      feature_matrix: featureMatrix,
      is_default: plan.is_default,
      tiers,
    })
  }

  saving.value = true
  try {
    await nuxtApp.$fetch(`${apiBase.value}/admin/marketplace/listings/${selectedListing.value.id}` as string, {
      method: "PATCH",
      body: {
        pricing_plans: payload,
      },
    })
    toast.add({ title: "计划已保存", color: "green" })
    await loadListingDetail(selectedListing.value.id)
  } catch (error) {
    toast.add({ title: "保存失败", description: String(error), color: "red" })
  } finally {
    saving.value = false
  }
}

function uuid() {
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID()
  }
  return Math.random().toString(36).slice(2)
}

watch(
  () => plans.value.length,
  (length) => {
    if (length === 0) {
      selectedPlanIndex.value = 0
    } else if (selectedPlanIndex.value >= length) {
      selectedPlanIndex.value = length - 1
    }
  }
)

onMounted(() => {
  loadListings()
})
</script>
