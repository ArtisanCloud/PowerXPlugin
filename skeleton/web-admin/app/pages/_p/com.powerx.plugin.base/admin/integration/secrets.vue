<template>
  <UContainer class="py-10 space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-key" class="text-primary" />
        <span class="uppercase tracking-wide">Integration · Secrets</span>
      </div>
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">外部凭证管理</h1>
        <p class="text-gray-600 dark:text-gray-300 max-w-3xl">
          维护外部系统的 API 凭证，查看轮换计划、执行双密钥切换并记录审计事件。
        </p>
      </div>
    </header>

    <div class="flex justify-end">
      <UButton color="primary" icon="i-heroicons-plus-circle" @click="openCreateModal">
        新建凭证
      </UButton>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <UIcon name="i-heroicons-shield-check" />
            <span>当前凭证</span>
          </div>
          <UButton color="gray" variant="soft" :loading="loading" icon="i-heroicons-arrow-path" @click="loadSecrets">
            刷新
          </UButton>
        </div>
      </template>

      <UTable :rows="secrets" :columns="columns" :loading="loading">
        <template #status-data="{ row }">
          <UBadge :color="statusBadgeColor(row.status)" variant="soft" class="uppercase tracking-wide">
            {{ row.status }}
          </UBadge>
        </template>
        <template #next_rotation_due_at-data="{ row }">
          {{ formatDate(row.next_rotation_due_at) || '-' }}
        </template>
        <template #actions-data="{ row }">
          <div class="flex flex-wrap gap-2">
            <UButton size="xs" color="primary" variant="soft" @click="rotate(row)">
              轮换
            </UButton>
            <UButton size="xs" color="primary" variant="soft" @click="completeRotation(row)" :disabled="!row.pending_secret_ref">
              完成轮换
            </UButton>
            <UButton size="xs" color="orange" variant="soft" @click="showAudit(row)">
              审计记录
            </UButton>
            <UButton size="xs" color="red" variant="ghost" @click="revoke(row)">
              吊销
            </UButton>
          </div>
        </template>
      </UTable>

      <div v-if="!loading && !secrets.length" class="py-6 text-center text-gray-500 dark:text-gray-400">
        暂无凭证记录，点击“新建凭证”开始管理。
      </div>
    </UCard>

    <UModal v-model:open="createOpen">
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-plus-circle" class="text-primary" />
            <span class="font-semibold">新建外部凭证</span>
          </div>
        </template>

        <form class="space-y-4" @submit.prevent="submitCreate">
          <UFormGroup label="集成类型" required>
            <UInput v-model="createForm.integrationType" placeholder="webhook.target" />
          </UFormGroup>
          <UFormGroup label="轮换间隔（天）" required>
            <UInput v-model.number="createForm.rotationIntervalDays" type="number" min="1" />
          </UFormGroup>
          <UFormGroup label="附加元数据 (JSON)" help='例如 {"owner":"security"}'>
            <UTextarea v-model="createForm.metadataInput" :rows="3" placeholder="可选" />
          </UFormGroup>
          <UCheckbox v-model="createForm.generate" label="立即生成新密钥" />
          <div v-if="!createForm.generate" class="space-y-2">
            <UFormGroup label="现有 Secret 引用">
              <UInput v-model="createForm.secretRef" placeholder="例如 secret://provider/ref" />
            </UFormGroup>
          </div>
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

    <UModal v-model:open="generatedOpen">
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-exclamation-circle" class="text-primary" />
            <span class="font-semibold">请立即保存生成的密钥</span>
          </div>
        </template>

        <div class="space-y-3">
          <p class="text-sm text-gray-600 dark:text-gray-300">
            以下密钥只展示一次，请复制妥善保管。
          </p>
          <UInput v-model="generatedSecret" readonly class="font-mono" />
        </div>
        <template #footer>
          <div class="flex justify-end">
            <UButton color="primary" @click="generatedOpen = false">
              已保存
            </UButton>
          </div>
        </template>
      </UCard>
    </UModal>

    <USlideover v-model:open="auditOpen">
      <UCard class="flex flex-col h-full">
        <template #header>
          <div class="flex flex-col gap-1">
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-clipboard-document" class="text-primary" />
              <span class="font-semibold">审计日志</span>
            </div>
            <p class="text-sm text-gray-500 dark:text-gray-400 truncate">
              {{ selectedSecret?.integration_type }}
            </p>
          </div>
        </template>

        <div class="flex-1 overflow-y-auto">
          <UTimeline :items="auditEntries" icon="i-heroicons-clock" />
          <div v-if="!auditEntries.length" class="py-6 text-center text-gray-500 dark:text-gray-400">
            暂无审计记录。
          </div>
        </div>

        <template #footer>
          <div class="flex justify-end">
            <UButton color="gray" variant="soft" @click="auditOpen = false">
              关闭
            </UButton>
          </div>
        </template>
      </UCard>
    </USlideover>
  </UContainer>
</template>

<script setup lang="ts">
import type { IntegrationSecret, IntegrationSecretAuditEntry } from "~/types/integration"

const runtimeConfig = useRuntimeConfig()
const toast = useToast()

const apiBase = computed(() => runtimeConfig.public.apiBaseUrl)

const secrets = ref<IntegrationSecret[]>([])
const loading = ref(false)
const creating = ref(false)

const createOpen = ref(false)
const generatedOpen = ref(false)
const generatedSecret = ref("")

const auditOpen = ref(false)
const auditEntries = ref<{ title: string; description?: string; timestamp?: string }[]>([])
const selectedSecret = ref<IntegrationSecret | null>(null)

const columns = [
  { key: "integration_type", label: "集成类型" },
  { key: "status", label: "状态" },
  { key: "rotation_interval_days", label: "轮换间隔" },
  { key: "next_rotation_due_at", label: "下次轮换" },
  { key: "actions", label: "操作" },
]

const createForm = reactive({
  integrationType: "",
  rotationIntervalDays: 30,
  metadataInput: "",
  generate: true,
  secretRef: "",
})

function statusBadgeColor(status: string) {
  switch (status) {
    case "ACTIVE":
      return "primary"
    case "ROTATING":
      return "orange"
    case "REVOKED":
      return "red"
    default:
      return "gray"
  }
}

function formatDate(date?: string | null) {
  if (!date) return ""
  return new Date(date).toLocaleString()
}

async function loadSecrets() {
  loading.value = true
  try {
    secrets.value = await $fetch<IntegrationSecret[]>(`${apiBase.value}/admin/integration/secrets`)
  } catch (error) {
    toast.add({ title: "加载失败", description: String(error), color: "red" })
  } finally {
    loading.value = false
  }
}

function openCreateModal() {
  createForm.integrationType = ""
  createForm.rotationIntervalDays = 30
  createForm.metadataInput = ""
  createForm.generate = true
  createForm.secretRef = ""
  createOpen.value = true
}

function parseMetadata(input: string) {
  if (!input) return undefined
  try {
    return JSON.parse(input)
  } catch (error) {
    toast.add({ title: "元数据格式错误", description: String(error), color: "red" })
    throw error
  }
}

async function submitCreate() {
  creating.value = true
  try {
    const result = await $fetch<{ secret: IntegrationSecret; generated_secret?: string }>(`${apiBase.value}/admin/integration/secrets`, {
      method: "POST",
      body: {
        integration_type: createForm.integrationType,
        rotation_interval_days: createForm.rotationIntervalDays,
        metadata: parseMetadata(createForm.metadataInput),
        generate: createForm.generate,
        secret_ref: createForm.generate ? undefined : createForm.secretRef,
      },
    })
    toast.add({ title: "凭证已创建", color: "primary" })
    createOpen.value = false
    await loadSecrets()
    if (result.generated_secret) {
      generatedSecret.value = result.generated_secret
      generatedOpen.value = true
    }
  } catch (error) {
    toast.add({ title: "创建失败", description: String(error), color: "red" })
  } finally {
    creating.value = false
  }
}

async function rotate(secret: IntegrationSecret) {
  try {
    const result = await $fetch<{ generated_secret?: string; secret: IntegrationSecret }>(`${apiBase.value}/admin/integration/secrets/${secret.id}/rotate`, {
      method: "POST",
      body: { generate: true },
    })
    toast.add({ title: "已触发轮换", color: "primary" })
    await loadSecrets()
    if (result.generated_secret) {
      generatedSecret.value = result.generated_secret
      generatedOpen.value = true
    }
  } catch (error) {
    toast.add({ title: "轮换失败", description: String(error), color: "red" })
  }
}

async function completeRotation(secret: IntegrationSecret) {
  try {
    await $fetch(`${apiBase.value}/admin/integration/secrets/${secret.id}/rotate/complete`, {
      method: "POST",
    })
    toast.add({ title: "轮换已完成", color: "primary" })
    await loadSecrets()
  } catch (error) {
    toast.add({ title: "操作失败", description: String(error), color: "red" })
  }
}

async function revoke(secret: IntegrationSecret) {
  try {
    await $fetch(`${apiBase.value}/admin/integration/secrets/${secret.id}/revoke`, { method: "POST" })
    toast.add({ title: "已吊销", color: "primary" })
    await loadSecrets()
  } catch (error) {
    toast.add({ title: "吊销失败", description: String(error), color: "red" })
  }
}

async function showAudit(secret: IntegrationSecret) {
  selectedSecret.value = secret
  auditOpen.value = true
  try {
    const entries = await $fetch<IntegrationSecretAuditEntry[]>(`${apiBase.value}/admin/integration/secrets/${secret.id}/audit`)
    auditEntries.value = entries.map((entry) => ({
      title: entry.action,
      description: entry.details ? JSON.stringify(entry.details) : undefined,
      timestamp: new Date(entry.timestamp).toLocaleString(),
    }))
  } catch (error) {
    toast.add({ title: "获取审计失败", description: String(error), color: "red" })
    auditEntries.value = []
  }
}

onMounted(() => {
  void loadSecrets()
})
</script>
