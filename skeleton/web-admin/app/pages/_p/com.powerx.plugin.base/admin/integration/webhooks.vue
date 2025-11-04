<template>
  <UContainer class="py-10 space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-bolt" class="text-primary" />
        <span class="uppercase tracking-wide">Integration · Webhooks</span>
      </div>
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">Webhook Subscriptions</h1>
        <p class="text-gray-600 dark:text-gray-300 max-w-3xl">
          管理事件订阅、重试策略与死信补投。创建新的回调目标，并追踪最近的投递尝试。
        </p>
      </div>
    </header>

    <div class="flex justify-end">
      <UButton color="primary" icon="i-heroicons-plus-circle" @click="openCreateModal">
        新建订阅
      </UButton>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <UIcon name="i-heroicons-link" />
            <span>当前订阅</span>
          </div>
          <UButton color="gray" variant="soft" :loading="loading" icon="i-heroicons-arrow-path" @click="loadSubscriptions">
            刷新
          </UButton>
        </div>
      </template>

      <UTable :rows="subscriptions" :columns="columns" :loading="loading">
        <template #status-data="{ row }">
          <UBadge :color="statusBadgeColor(row.status)" variant="soft" class="uppercase tracking-wide">
            {{ row.status }}
          </UBadge>
        </template>
        <template #retry_policy-data="{ row }">
          <span v-if="row.retry_policy?.length">{{ row.retry_policy.join('s · ') }}s</span>
          <span v-else class="text-gray-400">默认</span>
        </template>
        <template #actions-data="{ row }">
          <div class="flex flex-wrap gap-2">
            <UButton size="xs" color="primary" variant="soft" @click="showAttempts(row)">
              尝试记录
            </UButton>
            <UButton size="xs" color="orange" variant="soft" :disabled="row.status !== 'PAUSED'" @click="updateStatus(row, 'ACTIVE')">
              恢复
            </UButton>
            <UButton size="xs" color="orange" variant="soft" :disabled="row.status !== 'ACTIVE'" @click="updateStatus(row, 'PAUSED')">
              暂停
            </UButton>
            <UButton size="xs" color="red" variant="ghost" @click="removeSubscription(row)">
              删除
            </UButton>
          </div>
        </template>
      </UTable>

      <div v-if="!loading && !subscriptions.length" class="py-6 text-center text-gray-500 dark:text-gray-400">
        当前没有订阅。点击 “新建订阅” 创建第一个 webhook。
      </div>
    </UCard>

    <UModal v-model:open="createOpen">
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-plus-circle" class="text-primary" />
            <span class="font-semibold">新建订阅</span>
          </div>
        </template>

        <form class="space-y-4" @submit.prevent="submitCreate">
          <UFormGroup label="事件类型" required>
            <UInput v-model="createForm.eventType" placeholder="integration.envelope.dispatch" />
          </UFormGroup>

          <UFormGroup label="目标 URL" required>
            <UInput v-model="createForm.targetUrl" placeholder="https://example.com/webhooks" />
          </UFormGroup>

          <UFormGroup label="签名密钥 (可选)" help="用于签名头 X-PowerX-Signature 的密钥，建议使用随机字符串">
            <UInput v-model="createForm.secret" placeholder="留空则沿用默认密钥策略" />
          </UFormGroup>

          <UFormGroup label="重试策略 (秒)" help="填写逗号分隔的秒数，例如 60,300,900">
            <UInput v-model="createForm.retryPolicyInput" placeholder="60,300,900" />
          </UFormGroup>

          <div class="flex justify-end gap-2">
            <UButton color="gray" variant="soft" @click="createOpen = false">
              取消
            </UButton>
            <UButton type="submit" color="primary" :loading="creating">
              创建
            </UButton>
          </div>
        </form>
      </UCard>
    </UModal>

    <USlideover v-model:open="attemptPanelOpen">
      <UCard class="flex flex-col h-full">
        <template #header>
          <div class="flex flex-col gap-1">
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-clock" class="text-primary" />
              <span class="font-semibold">最近投递尝试</span>
            </div>
            <p class="text-sm text-gray-500 dark:text-gray-400 truncate">
              {{ selectedSubscription?.event_type }} → {{ selectedSubscription?.target_url }}
            </p>
          </div>
        </template>

        <div class="flex-1 overflow-y-auto">
          <UTable :rows="attempts" :columns="attemptColumns" :loading="attemptLoading">
            <template #status-data="{ row }">
              <UBadge :color="statusBadgeColor(row.status)" variant="soft" class="uppercase tracking-wide">
                {{ row.status }}
              </UBadge>
            </template>
            <template #payload_snapshot-data="{ row }">
              <pre class="max-h-32 overflow-auto bg-gray-50 dark:bg-gray-900 p-2 rounded text-xs">{{ formatPayload(row.payload_snapshot) }}</pre>
            </template>
            <template #actions-data="{ row }">
              <UButton
                size="xs"
                color="primary"
                variant="soft"
                :disabled="row.status !== 'DLQ' && row.status !== 'FAILED'"
                @click="replayAttempt(row)"
              >
                重新投递
              </UButton>
            </template>
          </UTable>
          <div v-if="!attemptLoading && !attempts.length" class="py-6 text-center text-gray-500 dark:text-gray-400">
            暂无记录。
          </div>
        </div>

        <template #footer>
          <div class="flex justify-end">
            <UButton color="gray" variant="soft" @click="attemptPanelOpen = false">
              关闭
            </UButton>
          </div>
        </template>
      </UCard>
    </USlideover>
  </UContainer>
</template>

<script setup lang="ts">
import type { IntegrationWebhookSubscription, IntegrationWebhookAttempt } from "~/types/integration"

const runtimeConfig = useRuntimeConfig()
const toast = useToast()

const apiBase = computed(() => runtimeConfig.public.apiBaseUrl)

const loading = ref(false)
const creating = ref(false)
const subscriptions = ref<IntegrationWebhookSubscription[]>([])

const createOpen = ref(false)
const createForm = reactive({
  eventType: "",
  targetUrl: "",
  secret: "",
  retryPolicyInput: "60,300,900",
})

const attemptPanelOpen = ref(false)
const attemptLoading = ref(false)
const selectedSubscription = ref<IntegrationWebhookSubscription | null>(null)
const attempts = ref<IntegrationWebhookAttempt[]>([])

const columns = [
  { key: "event_type", label: "事件" },
  { key: "target_url", label: "目标" },
  { key: "status", label: "状态" },
  { key: "retry_policy", label: "重试策略" },
  { key: "updated_at", label: "最近更新时间" },
  { key: "actions", label: "操作" },
]

const attemptColumns = [
  { key: "status", label: "状态" },
  { key: "retry_count", label: "重试次数" },
  { key: "last_error", label: "最近错误" },
  { key: "next_delivery_at", label: "下次投递" },
  { key: "payload_snapshot", label: "快照" },
  { key: "actions", label: "操作" },
]

function statusBadgeColor(status: string) {
  switch (status) {
    case "ACTIVE":
    case "SUCCEEDED":
      return "primary"
    case "PAUSED":
    case "RETRYING":
      return "orange"
    case "FAILED":
    case "DLQ":
      return "red"
    default:
      return "gray"
  }
}

function parseRetryPolicy(input: string) {
  if (!input) return []
  return input
    .split(",")
    .map((item) => Number.parseInt(item.trim()))
    .filter((num) => Number.isFinite(num) && num > 0)
}

async function loadSubscriptions() {
  loading.value = true
  try {
    subscriptions.value = await $fetch<IntegrationWebhookSubscription[]>(`${apiBase.value}/admin/integration/webhooks`)
  } catch (error) {
    toast.add({ title: "加载失败", description: String(error), color: "red" })
  } finally {
    loading.value = false
  }
}

function openCreateModal() {
  createForm.eventType = ""
  createForm.targetUrl = ""
  createForm.secret = ""
  createForm.retryPolicyInput = "60,300,900"
  createOpen.value = true
}

async function submitCreate() {
  creating.value = true
  try {
    await $fetch(`${apiBase.value}/admin/integration/webhooks`, {
      method: "POST",
      body: {
        event_type: createForm.eventType,
        target_url: createForm.targetUrl,
        secret: createForm.secret,
        retry_policy: parseRetryPolicy(createForm.retryPolicyInput),
      },
    })
    toast.add({ title: "订阅已创建", color: "primary" })
    createOpen.value = false
    await loadSubscriptions()
  } catch (error) {
    toast.add({ title: "创建失败", description: String(error), color: "red" })
  } finally {
    creating.value = false
  }
}

async function updateStatus(sub: IntegrationWebhookSubscription, status: string) {
  try {
    await $fetch(`${apiBase.value}/admin/integration/webhooks/${sub.id}`, {
      method: "PUT",
      body: { status },
    })
    toast.add({ title: "状态已更新", color: "primary" })
    await loadSubscriptions()
  } catch (error) {
    toast.add({ title: "更新失败", description: String(error), color: "red" })
  }
}

async function removeSubscription(sub: IntegrationWebhookSubscription) {
  try {
    await $fetch(`${apiBase.value}/admin/integration/webhooks/${sub.id}`, {
      method: "DELETE",
    })
    toast.add({ title: "订阅已删除", color: "primary" })
    await loadSubscriptions()
  } catch (error) {
    toast.add({ title: "删除失败", description: String(error), color: "red" })
  }
}

function showAttempts(sub: IntegrationWebhookSubscription) {
  selectedSubscription.value = sub
  attemptPanelOpen.value = true
  void loadAttempts(sub.id)
}

async function loadAttempts(subscriptionID: string) {
  attemptLoading.value = true
  try {
    attempts.value = await $fetch<IntegrationWebhookAttempt[]>(`${apiBase.value}/admin/integration/webhooks/${subscriptionID}/attempts`)
  } catch (error) {
    toast.add({ title: "加载尝试记录失败", description: String(error), color: "red" })
  } finally {
    attemptLoading.value = false
  }
}

async function replayAttempt(attempt: IntegrationWebhookAttempt) {
  try {
    await $fetch(`${apiBase.value}/admin/integration/webhooks/attempts/${attempt.id}/replay`, {
      method: "POST",
    })
    toast.add({ title: "已重新排队", color: "primary" })
    if (selectedSubscription.value) {
      await loadAttempts(selectedSubscription.value.id)
    }
  } catch (error) {
    toast.add({ title: "操作失败", description: String(error), color: "red" })
  }
}

function formatPayload(payload: unknown) {
  if (!payload) return "-"
  try {
    return JSON.stringify(payload, null, 2)
  } catch {
    return String(payload)
  }
}

onMounted(() => {
  void loadSubscriptions()
})
</script>
