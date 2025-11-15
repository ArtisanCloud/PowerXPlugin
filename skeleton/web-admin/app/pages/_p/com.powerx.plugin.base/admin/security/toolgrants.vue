<template>
  <UContainer class="py-10 space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-key" class="text-primary" />
        <span class="uppercase tracking-wide">Security · ToolGrants</span>
      </div>
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          ToolGrant Overview
        </h1>
        <p class="text-gray-600 dark:text-gray-300 max-w-3xl">
          Track active ToolGrants, review revocation history, and inspect usage events to ensure least-privilege enforcement across agents.
        </p>
      </div>
    </header>

    <UCard>
      <template #header>
        <div class="flex flex-wrap items-end gap-4">
          <UFormGroup label="Tenant ID" required class="w-full sm:w-64">
            <UInput v-model="tenantId" placeholder="tenant-123" :disabled="loading" />
          </UFormGroup>
          <UButton color="primary" :loading="loading" :disabled="!tenantId" @click="loadData">
            Refresh
          </UButton>
        </div>
      </template>

      <template #default>
        <div class="grid gap-6 lg:grid-cols-2">
          <UCard class="border border-gray-200/70 dark:border-gray-700/80">
            <template #header>
              <div class="flex items-center gap-2">
                <UIcon name="i-heroicons-no-symbol" class="text-red-500" />
                <span class="font-semibold">Revocations</span>
              </div>
            </template>
            <UTable :rows="revocations" :columns="revocationColumns" :loading="loading">
              <template #revokedAt-data="{ row }">
                {{ formatDate(row.revokedAt) }}
              </template>
              <template #ttlExpiry-data="{ row }">
                {{ formatDate(row.ttlExpiry) }}
              </template>
            </UTable>
            <div v-if="!loading && !revocations.length" class="py-6 text-center text-gray-500 dark:text-gray-400">
              No revocations recorded for this tenant.
            </div>
          </UCard>

          <UCard class="border border-gray-200/70 dark:border-gray-700/80">
            <template #header>
              <div class="flex items-center gap-2">
                <UIcon name="i-heroicons-document-text" class="text-primary" />
                <span class="font-semibold">Usage Events</span>
              </div>
            </template>
            <UTable :rows="usageEvents" :columns="usageColumns" :loading="loading">
              <template #occurredAt-data="{ row }">
                {{ formatDate(row.occurredAt) }}
              </template>
              <template #metadata-data="{ row }">
                <UButton size="xs" variant="soft" color="gray" @click="showMetadata(row)">
                  View
                </UButton>
              </template>
            </UTable>
            <div v-if="!loading && !usageEvents.length" class="py-6 text-center text-gray-500 dark:text-gray-400">
              No usage events recorded.
            </div>
          </UCard>
        </div>
      </template>
    </UCard>

    <UModal v-model:open="metadataOpen">
      <UCard>
        <template #header>
          <div class="flex items-center gap-2">
            <UIcon name="i-heroicons-information-circle" class="text-primary" />
            <span class="font-semibold">Usage Metadata</span>
          </div>
        </template>
        <UCode :code="metadataJson" language="json" class="text-xs" />
        <template #footer>
          <div class="flex justify-end">
            <UButton color="primary" @click="metadataOpen = false">Close</UButton>
          </div>
        </template>
      </UCard>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import type { ToolGrantRevocation, ToolGrantUsageEvent } from "~/types/security";

const runtimeConfig = useRuntimeConfig();
const toast = useToast();

const tenantId = ref("");
const loading = ref(false);
const revocations = ref<ToolGrantRevocation[]>([]);
const usageEvents = ref<ToolGrantUsageEvent[]>([]);
const metadataOpen = ref(false);
const metadataJson = ref("{}");

const revocationColumns = [
  { key: "toolGrantId", label: "ToolGrant ID" },
  { key: "revokedBy", label: "Revoked By" },
  { key: "reason", label: "Reason" },
  { key: "revokedAt", label: "Revoked At" },
  { key: "ttlExpiry", label: "TTL Expiry" },
];

const usageColumns = [
  { key: "toolGrantId", label: "ToolGrant ID" },
  { key: "eventType", label: "Event" },
  { key: "capability", label: "Capability" },
  { key: "agentId", label: "Agent" },
  { key: "occurredAt", label: "Timestamp" },
  { key: "metadata", label: "Metadata" },
];

const formatDate = (value?: string) => (value ? new Date(value).toLocaleString() : "");

const loadData = async () => {
  if (!tenantId.value) {
    toast.add({ title: "Tenant ID required", color: "red" });
    return;
  }
  loading.value = true;
  try {
    const [revResp, usageResp] = await Promise.all([
      $fetch<{ data: ToolGrantRevocation[] }>(`${runtimeConfig.public.apiBaseUrl}/admin/security/toolgrants/revocations`, {
        params: { tenant_id: tenantId.value },
        credentials: "include",
      }),
      $fetch<{ data: ToolGrantUsageEvent[] }>(`${runtimeConfig.public.apiBaseUrl}/admin/security/toolgrants/usage`, {
        params: { tenant_id: tenantId.value },
        credentials: "include",
      }),
    ]);
    revocations.value = revResp?.data || [];
    usageEvents.value = usageResp?.data || [];
  } catch (err) {
    console.error(err);
    toast.add({ title: "Failed to load ToolGrant data", color: "red" });
  } finally {
    loading.value = false;
  }
};

const showMetadata = (row: ToolGrantUsageEvent) => {
  metadataOpen.value = true;
  metadataJson.value = row.metadata ? JSON.stringify(row.metadata, null, 2) : "{}";
};

definePageMeta({
  layout: "embedded",
  title: "AdminSecurityToolGrants",
});
</script>
