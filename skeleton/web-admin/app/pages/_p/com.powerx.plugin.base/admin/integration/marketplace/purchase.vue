<template>
  <UContainer class="py-10 space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-shopping-cart" class="text-primary" />
        <span class="uppercase tracking-wide">Integration · Marketplace</span>
      </div>
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Tenant Purchase Flow</h1>
        <p class="text-gray-600 dark:text-gray-300 max-w-3xl">
          模拟租户选择价格计划并完成 License 购买，验证续费与离线延展能力。用于演示 Billing/License Server 的闭环体验。
        </p>
      </div>
    </header>

    <div class="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,3fr)]">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
              <UIcon name="i-heroicons-building-storefront" />
              <span>Listing 与价格计划</span>
            </div>
            <UButton color="gray" variant="soft" icon="i-heroicons-arrow-path" :loading="loading" @click="loadListings">
              刷新
            </UButton>
          </div>
        </template>

        <div class="space-y-4">
          <UFormGroup label="租户 ID">
            <UInput v-model="tenantId" placeholder="1" />
          </UFormGroup>

          <UFormGroup label="选择 Listing">
            <USelectMenu
              v-model="selectedListingId"
              :options="listingOptions"
              :loading="loading"
              placeholder="选择 Listing"
              @update:model-value="handleListingChange"
            />
          </UFormGroup>

          <div v-if="selectedListing">
            <UFormGroup label="选择价格计划">
              <USelectMenu
                v-model="selectedPlanId"
                :options="planOptions"
                placeholder="选择计划"
              />
            </UFormGroup>

            <div v-if="selectedPlan" class="rounded border border-gray-200 p-4 dark:border-gray-700">
              <h3 class="text-base font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                {{ selectedPlan.plan_code }}
                <UBadge v-if="selectedPlan.is_default" color="primary" class="uppercase">默认</UBadge>
              </h3>
              <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
                类型：{{ selectedPlan.plan_type }} · 币种：{{ selectedPlan.currency }}
                <span v-if="selectedPlan.amount != null"> · 金额：{{ selectedPlan.amount }}</span>
              </p>
              <ul v-if="selectedPlan.tiers?.length" class="mt-3 space-y-1 text-sm text-gray-600 dark:text-gray-300">
                <li v-for="tier in selectedPlan.tiers" :key="tier.id">
                  {{ tier.metric }}: {{ tier.range_from }} -
                  {{ tier.range_to ?? '∞' }} → {{ tier.unit_amount }}/{{ tier.unit_name || 'unit' }}
                </li>
              </ul>
            </div>
          </div>
        </div>
      </UCard>

      <UCard>
        <template #header>
          <div class="flex items-center justify-between gap-2">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">购买与 License 状态</h2>
            <div class="flex items-center gap-2">
              <UButton color="primary" :disabled="!selectedPlan" :loading="purchasing" @click="purchaseLicense">
                立即购买
              </UButton>
            </div>
          </div>
        </template>

        <div class="space-y-4">
          <UFormGroup label="Payment Intent ID">
            <UInput v-model="paymentIntent" placeholder="pi_mock_123" />
          </UFormGroup>

          <UAlert v-if="!license" color="gray" icon="i-heroicons-information-circle">
            购买成功后将在此显示 License 详情，可进一步续费或延长离线窗口。
          </UAlert>

          <div v-else class="space-y-4">
            <div class="rounded border border-gray-200 p-4 dark:border-gray-700 space-y-2">
              <div class="flex items-center justify-between">
                <h3 class="text-base font-semibold text-gray-900 dark:text-white">License #{{ license.id }}</h3>
                <UBadge :color="licenseBadgeColor" class="uppercase">{{ license.status }}</UBadge>
              </div>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                计划 {{ license.plan_id }} · 过期时间 {{ formatIso(license.expires_at) }}
              </p>
              <p v-if="license.offline_until" class="text-sm text-gray-500 dark:text-gray-400">
                离线可用至 {{ formatIso(license.offline_until) }}
              </p>
              <p class="text-xs break-words text-gray-400">Token: {{ license.token }}</p>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
              <UCard>
                <template #header>
                  <div class="flex items-center justify-between">
                    <span class="text-sm font-semibold text-gray-900 dark:text-white">续费 License</span>
                    <UButton
                      size="xs"
                      color="primary"
                      :disabled="!license.renewal_token"
                      :loading="renewing"
                      @click="renewLicense"
                    >
                      续费
                    </UButton>
                  </div>
                </template>
                <p class="text-sm text-gray-500 dark:text-gray-400">
                  使用当前 Renewal Token 触发续费并刷新离线窗口。
                </p>
              </UCard>

              <UCard>
                <template #header>
                  <span class="text-sm font-semibold text-gray-900 dark:text-white">离线续期</span>
                </template>
                <div class="space-y-2">
                  <UFormGroup label="延长小时数" help="最大 72 小时">
                    <UInput v-model="extendHours" type="number" min="1" max="72" />
                  </UFormGroup>
                  <UButton color="gray" :loading="extending" @click="extendOffline">
                    延长离线窗口
                  </UButton>
                </div>
              </UCard>
            </div>
          </div>
        </div>
      </UCard>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useNuxtApp, useRuntimeConfig, useToast } from "#imports"
import type { MarketplaceLicense, MarketplaceListing, MarketplacePricingPlan } from "~/types/integration"

definePageMeta({
  middleware: ["layout.global"],
})

const toast = useToast()
const nuxtApp = useNuxtApp()
const runtime = useRuntimeConfig()
const apiBase = computed(() => runtime.public.apiBaseUrl as string)

const loading = ref(false)
const purchasing = ref(false)
const renewing = ref(false)
const extending = ref(false)

const tenantId = ref("1")
const listings = ref<MarketplaceListing[]>([])
const selectedListingId = ref<string | null>(null)
const selectedPlanId = ref<string | null>(null)
const license = ref<MarketplaceLicense | null>(null)
const paymentIntent = ref("pi_mock_123")
const extendHours = ref(24)

const listingOptions = computed(() =>
  listings.value.map((listing) => ({ label: listing.title, value: listing.id }))
)

const selectedListing = computed(() =>
  listings.value.find((item) => item.id === selectedListingId.value) || null
)

const planOptions = computed(() =>
  (selectedListing.value?.pricing_plans || []).map((plan) => ({
    label: `${plan.plan_code} (${plan.plan_type})`,
    value: plan.id,
  }))
)

const selectedPlan = computed<MarketplacePricingPlan | null>(() => {
  if (!selectedListing.value || !selectedPlanId.value) {
    return null
  }
  return selectedListing.value.pricing_plans?.find((plan) => plan.id === selectedPlanId.value) || null
})

const licenseBadgeColor = computed(() => {
  switch (license.value?.status) {
    case "active":
      return "primary"
    case "trial":
      return "emerald"
    case "expired":
      return "orange"
    case "revoked":
      return "red"
    default:
      return "gray"
  }
})

async function loadListings() {
  loading.value = true
  try {
    const response = await nuxtApp.$fetch<{ data: { items: MarketplaceListing[] } }>(
      `${apiBase.value}/admin/marketplace/listings`,
      {
        query: { limit: 50 },
      }
    )
    listings.value = response?.data?.items ?? []
    if (!listings.value.length) {
      selectedListingId.value = null
      selectedPlanId.value = null
      return
    }
    if (!selectedListingId.value) {
      selectedListingId.value = listings.value[0].id
    }
    await loadListingDetail(selectedListingId.value)
  } catch (error) {
    toast.add({ title: "加载 Listing 失败", description: String(error), color: "red" })
  } finally {
    loading.value = false
  }
}

async function loadListingDetail(id: string) {
  try {
    const detail = await nuxtApp.$fetch<{ data: MarketplaceListing }>(
      `${apiBase.value}/admin/marketplace/listings/${id}`
    )
    const listing = detail?.data
    if (!listing) {
      return
    }
    const index = listings.value.findIndex((item) => item.id === id)
    if (index >= 0) {
      listings.value[index] = listing
    } else {
      listings.value.push(listing)
    }
    selectedListingId.value = listing.id
    selectedPlanId.value = listing.pricing_plans?.[0]?.id ?? null
  } catch (error) {
    toast.add({ title: "加载计划失败", description: String(error), color: "red" })
  }
}

function handleListingChange(value: string | null) {
  if (!value) {
    selectedListingId.value = null
    selectedPlanId.value = null
    return
  }
  selectedListingId.value = value
  loadListingDetail(value)
}

function ensureTenantQuery(): Record<string, string> {
  const id = tenantId.value.trim()
  return id ? { tenant_id: id } : {}
}

async function purchaseLicense() {
  if (!selectedListing.value || !selectedPlanId.value) {
    toast.add({ title: "请选择价格计划", color: "orange" })
    return
  }
  if (!paymentIntent.value.trim()) {
    toast.add({ title: "Payment Intent ID 不能为空", color: "orange" })
    return
  }
  purchasing.value = true
  try {
    const response = await nuxtApp.$fetch<{ data: MarketplaceLicense }>(
      `${apiBase.value}/marketplace/licenses`,
      {
        method: "POST",
        query: ensureTenantQuery(),
        body: {
          listing_id: selectedListing.value.id,
          plan_id: selectedPlanId.value,
          payment_intent_id: paymentIntent.value.trim(),
        },
      }
    )
    license.value = response?.data ?? null
    toast.add({ title: "购买成功", color: "green" })
  } catch (error) {
    toast.add({ title: "购买失败", description: String(error), color: "red" })
  } finally {
    purchasing.value = false
  }
}

async function renewLicense() {
  if (!license.value) {
    return
  }
  if (!license.value.renewal_token) {
    toast.add({ title: "不存在 Renewal Token", color: "orange" })
    return
  }
  renewing.value = true
  try {
    const response = await nuxtApp.$fetch<{ data: MarketplaceLicense }>(
      `${apiBase.value}/marketplace/licenses/${license.value.id}`,
      {
        method: "POST",
        query: ensureTenantQuery(),
        body: {
          renewal_token: license.value.renewal_token,
          plan_id: license.value.plan_id,
        },
      }
    )
    license.value = response?.data ?? null
    toast.add({ title: "续费成功", color: "green" })
  } catch (error) {
    toast.add({ title: "续费失败", description: String(error), color: "red" })
  } finally {
    renewing.value = false
  }
}

async function extendOffline() {
  if (!license.value) {
    return
  }
  if (extendHours.value < 1 || extendHours.value > 72) {
    toast.add({ title: "延长小时需在 1-72 之间", color: "orange" })
    return
  }
  extending.value = true
  try {
    await nuxtApp.$fetch(
      `${apiBase.value}/marketplace/licenses/${license.value.id}/offline-extend`,
      {
        method: "POST",
        query: ensureTenantQuery(),
        body: { requested_hours: extendHours.value },
      }
    )
    await refreshLicense()
    toast.add({ title: "离线窗口已延长", color: "green" })
  } catch (error) {
    toast.add({ title: "离线续期失败", description: String(error), color: "red" })
  } finally {
    extending.value = false
  }
}

async function refreshLicense() {
  if (!license.value) {
    return
  }
  try {
    const response = await nuxtApp.$fetch<{ data: MarketplaceLicense }>(
      `${apiBase.value}/marketplace/licenses/${license.value.id}`,
      {
        query: ensureTenantQuery(),
      }
    )
    license.value = response?.data ?? license.value
  } catch (error) {
    toast.add({ title: "刷新 License 失败", description: String(error), color: "red" })
  }
}

function formatIso(value: string | null | undefined) {
  if (!value) return "N/A"
  return new Date(value).toLocaleString()
}

onMounted(() => {
  loadListings()
})
</script>
