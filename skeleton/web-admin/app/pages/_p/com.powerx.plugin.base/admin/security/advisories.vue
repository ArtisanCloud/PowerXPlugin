<template>
  <UContainer class="py-10 space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-shield-check" class="text-primary" />
        <span class="uppercase tracking-wide">Security · Advisories</span>
      </div>
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          Vulnerability Advisories
        </h1>
        <p class="text-gray-600 dark:text-gray-300 max-w-3xl">
          Track vulnerability intake, publish signed advisories, and monitor delivery channels to keep tenants informed within SLA targets.
        </p>
      </div>
    </header>

    <div class="flex justify-end">
      <UButton color="primary" @click="openCreate">
        New Advisory
      </UButton>
    </div>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <UIcon name="i-heroicons-list-bullet" />
            <span>Advisory Backlog</span>
          </div>
          <UButton color="gray" variant="soft" :loading="loading" @click="loadAdvisories">
            Refresh
          </UButton>
        </div>
      </template>

      <UTable :rows="advisories" :columns="columns" :loading="loading">
        <template #severity-data="{ row }">
          <UBadge :color="severityColor(row.severity)" variant="soft" class="uppercase tracking-wide">
            {{ row.severity }}
          </UBadge>
        </template>
        <template #status-data="{ row }">
          <UBadge :color="statusBadgeColor(row.status)" variant="soft" class="uppercase tracking-wide">
            {{ row.status }}
          </UBadge>
        </template>
        <template #published_at-data="{ row }">
          {{ formatDate(row.published_at) }}
        </template>
        <template #actions-data="{ row }">
          <div class="flex flex-wrap gap-2">
            <UButton
              size="xs"
              color="primary"
              :disabled="row.status === 'PUBLISHED' || row.status === 'CLOSED'"
              @click="startPublish(row)"
            >
              Publish
            </UButton>
          </div>
        </template>
      </UTable>
      <div v-if="!loading && !advisories.length" class="py-6 text-center text-gray-500 dark:text-gray-400">
        No advisories recorded yet.
      </div>
    </UCard>

    <UModal v-model:open="createOpen">
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-plus-circle" class="text-primary" />
            <span class="font-semibold">Create Advisory</span>
          </div>
        </template>

        <form class="space-y-4" @submit.prevent="submitCreate">
          <UFormGroup label="Reference" required>
            <UInput v-model="newAdvisory.reference" placeholder="PX-ADV-2025-0001" />
          </UFormGroup>
          <UFormGroup label="Severity" required>
            <USelectMenu v-model="newAdvisory.severity" :options="severityOptions" />
          </UFormGroup>
          <UFormGroup label="Summary" required>
            <UTextarea v-model="newAdvisory.summary" :rows="3" placeholder="Short summary of the vulnerability" />
          </UFormGroup>
          <UFormGroup label="Affected Versions">
            <UInput
              v-model="affectedVersionInput"
              placeholder="Comma separated list (e.g. 1.2.0,1.2.1)"
            />
          </UFormGroup>
          <UFormGroup label="Details (Markdown)">
            <UTextarea v-model="newAdvisory.detailsMarkdown" :rows="4" placeholder="Remediation notes, CVSS scores, links..." />
          </UFormGroup>
          <div class="flex justify-end gap-2">
            <UButton color="gray" variant="soft" @click="createOpen = false">
              Cancel
            </UButton>
            <UButton type="submit" color="primary" :loading="creating">
              Create
            </UButton>
          </div>
        </form>
      </UCard>
    </UModal>

    <UModal v-model:open="publishOpen">
      <UCard>
        <template #header>
          <div class="flex flex-col gap-1">
            <div class="flex items-center gap-2">
              <UIcon name="i-heroicons-megaphone" class="text-primary" />
              <span class="font-semibold">Publish Advisory</span>
            </div>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ selectedAdvisory?.reference }} · {{ selectedAdvisory?.summary }}
            </p>
          </div>
        </template>

        <form class="space-y-4" @submit.prevent="submitPublish">
          <UFormGroup label="Patched in Version" required>
            <UInput v-model="publishForm.patchedInVersion" placeholder="2.0.1" />
          </UFormGroup>
          <UFormGroup label="Notify Channels" help="Choose delivery channels for this advisory">
            <USelectMenu
              v-model="publishForm.notifyChannels"
              :options="channelOptions"
              multiple
              searchable
              placeholder="Select channels"
            />
          </UFormGroup>
          <div class="flex justify-end gap-2">
            <UButton color="gray" variant="soft" @click="publishOpen = false">
              Cancel
            </UButton>
            <UButton type="submit" color="primary" :loading="publishing">
              Publish
            </UButton>
          </div>
        </form>
      </UCard>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import type { VulnerabilityAdvisory } from "~/types/security"

const runtimeConfig = useRuntimeConfig()
const toast = useToast()

const advisories = ref<VulnerabilityAdvisory[]>([])
const loading = ref(false)
const creating = ref(false)
const publishing = ref(false)
const createOpen = ref(false)
const publishOpen = ref(false)
const selectedAdvisory = ref<VulnerabilityAdvisory | null>(null)

const newAdvisory = reactive({
  reference: "",
  severity: "HIGH",
  summary: "",
  detailsMarkdown: "",
})

const affectedVersionInput = ref("")

const publishForm = reactive({
  patchedInVersion: "",
  notifyChannels: ["MARKETPLACE"] as string[],
})

const severityOptions = [
  { label: "Critical", value: "CRITICAL" },
  { label: "High", value: "HIGH" },
  { label: "Medium", value: "MEDIUM" },
  { label: "Low", value: "LOW" },
]

const channelOptions = [
  { label: "Marketplace", value: "MARKETPLACE" },
  { label: "Email", value: "EMAIL" },
  { label: "Webhook", value: "WEBHOOK" },
]

const columns = [
  { key: "reference", label: "Reference" },
  { key: "severity", label: "Severity" },
  { key: "status", label: "Status" },
  { key: "published_at", label: "Published" },
  { key: "actions", label: "Actions" },
]

const apiBase = computed(() => runtimeConfig.public.apiBaseUrl)

const formatDate = (value?: string) => (value ? new Date(value).toLocaleString() : "—")

const severityColor = (severity: string) => {
  switch (severity) {
    case "CRITICAL":
      return "red"
    case "HIGH":
      return "orange"
    case "MEDIUM":
      return "yellow"
    default:
      return "gray"
  }
}

const statusBadgeColor = (status: string) => {
  switch (status) {
    case "OPEN":
      return "gray"
    case "PATCHED":
      return "yellow"
    case "PUBLISHED":
      return "green"
    case "CLOSED":
      return "blue"
    default:
      return "gray"
  }
}

const resetCreateForm = () => {
  newAdvisory.reference = ""
  newAdvisory.severity = "HIGH"
  newAdvisory.summary = ""
  newAdvisory.detailsMarkdown = ""
  affectedVersionInput.value = ""
}

const loadAdvisories = async () => {
  loading.value = true
  try {
    const response = await $fetch<{ data: VulnerabilityAdvisory[] }>(`${apiBase.value}/admin/security/advisories`, {
      credentials: "include",
    })
    advisories.value = response?.data || []
  } catch (error) {
    console.error(error)
    toast.add({ title: "Failed to load advisories", color: "red" })
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  resetCreateForm()
  createOpen.value = true
}

const submitCreate = async () => {
  if (!newAdvisory.reference || !newAdvisory.summary) {
    toast.add({ title: "Reference and summary are required", color: "red" })
    return
  }
  creating.value = true
  try {
    const versionList = affectedVersionInput.value
      .split(",")
      .map((entry) => entry.trim())
      .filter(Boolean)
    await $fetch(`${apiBase.value}/admin/security/advisories`, {
      method: "POST",
      credentials: "include",
      body: {
        reference: newAdvisory.reference,
        severity: newAdvisory.severity,
        summary: newAdvisory.summary,
        details_markdown: newAdvisory.detailsMarkdown || undefined,
        affected_versions: versionList,
      },
    })
    toast.add({ title: "Advisory created", color: "green" })
    createOpen.value = false
    await loadAdvisories()
  } catch (error) {
    console.error(error)
    toast.add({ title: "Failed to create advisory", color: "red" })
  } finally {
    creating.value = false
  }
}

const startPublish = (advisory: VulnerabilityAdvisory) => {
  selectedAdvisory.value = advisory
  publishForm.patchedInVersion = advisory.patched_in_version || ""
  publishForm.notifyChannels = ["MARKETPLACE"]
  publishOpen.value = true
}

const submitPublish = async () => {
  if (!selectedAdvisory.value) {
    return
  }
  if (!publishForm.patchedInVersion) {
    toast.add({ title: "Patched version required", color: "red" })
    return
  }
  publishing.value = true
  try {
    const channels = publishForm.notifyChannels.length ? publishForm.notifyChannels : ["MARKETPLACE"]
    await $fetch(`${apiBase.value}/admin/security/advisories/${selectedAdvisory.value.id}/publish`, {
      method: "POST",
      credentials: "include",
      body: {
        patched_in_version: publishForm.patchedInVersion,
        notify_channels: channels,
      },
    })
    toast.add({ title: "Advisory published", color: "green" })
    publishOpen.value = false
    await loadAdvisories()
  } catch (error) {
    console.error(error)
    toast.add({ title: "Failed to publish advisory", color: "red" })
  } finally {
    publishing.value = false
  }
}

onMounted(() => {
  loadAdvisories()
})

definePageMeta({
  layout: "embedded",
  title: "AdminSecurityAdvisories",
})
</script>
