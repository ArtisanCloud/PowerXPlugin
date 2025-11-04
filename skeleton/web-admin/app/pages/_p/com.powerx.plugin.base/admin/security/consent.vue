<template>
  <UContainer class="py-10 space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-shield-check" class="text-primary" />
        <span class="uppercase tracking-wide">Security · Privacy</span>
      </div>
      <div class="flex flex-wrap items-end justify-between gap-4">
        <div class="space-y-1">
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
            Consent Tokens
          </h1>
          <p class="text-gray-600 dark:text-gray-300 max-w-2xl">
            Review tenant-issued consent tokens, confirm authorised data scopes, and revoke access when obligations change. Token previews are masked to avoid accidental PII exposure.
          </p>
        </div>
      </div>
    </header>

    <section>
      <UCard>
        <template #header>
          <div class="flex flex-wrap items-end gap-4">
            <UFormGroup label="Tenant ID" required class="w-full sm:w-64">
              <UInput
                v-model="tenantId"
                placeholder="tenant-123"
                @keyup.enter="loadTokens"
                :disabled="loading"
              />
            </UFormGroup>

            <UFormGroup label="Status" class="w-full sm:w-48">
              <USelectMenu
                v-model="selectedStatuses"
                :options="statusOptions"
                multiple
                searchable
                :disabled="loading"
              >
                <template #label>
                  <div class="flex flex-wrap gap-1">
                    <UBadge
                      v-for="status in selectedStatuses"
                      :key="status"
                      size="xs"
                      color="primary"
                    >
                      {{ status }}
                    </UBadge>
                    <span v-if="!selectedStatuses.length" class="text-gray-500 dark:text-gray-400"
                      >All statuses</span
                    >
                  </div>
                </template>
              </USelectMenu>
            </UFormGroup>

            <div class="flex items-center gap-3">
              <UButton
                color="primary"
                :disabled="!tenantId"
                :loading="loading"
                @click="loadTokens"
              >
                Load Tokens
              </UButton>
              <UButton
                variant="soft"
                color="gray"
                :disabled="!tenantId || loading"
                @click="exportAudit"
              >
                Export Audit
              </UButton>
            </div>
          </div>
        </template>

        <template #default>
          <div class="space-y-4">
            <UAlert
              color="red"
              variant="subtle"
              icon="i-heroicons-exclamation-triangle"
              :title="error"
              v-if="error"
            />

            <UTable :rows="rows" :columns="columns" :loading="loading">
              <template #token-data="{ row }">
                <span class="font-mono">{{ row.token }}</span>
              </template>
              <template #scope-data="{ row }">
                <div class="flex flex-wrap gap-2">
                  <UBadge
                    v-for="scope in row.scope"
                    :key="scope"
                    size="xs"
                    color="primary"
                    variant="soft"
                  >
                    {{ scope }}
                  </UBadge>
                </div>
              </template>
              <template #status-data="{ row }">
                <UBadge
                  :color="statusColor(row.status)"
                  size="xs"
                  class="uppercase"
                >
                  {{ row.status }}
                </UBadge>
              </template>
            </UTable>

            <div v-if="!loading && !rows.length" class="text-center py-10">
              <UIcon name="i-heroicons-document-magnifying-glass" class="text-4xl text-gray-300 dark:text-gray-600" />
              <p class="mt-3 text-gray-500 dark:text-gray-400">
                {{ tenantId ? "No consent tokens found for this tenant." : "Enter a tenant ID to begin." }}
              </p>
            </div>
          </div>
        </template>
      </UCard>
    </section>
  </UContainer>
</template>

<script setup lang="ts">
import type { ConsentTokenResponse, ConsentTokenListResponse } from "~/types/security";

const runtimeConfig = useRuntimeConfig();
const toast = useToast();

const tenantId = ref("");
const selectedStatuses = ref<string[]>(["ACTIVE"]);
const loading = ref(false);
const rows = ref<ConsentTokenResponse[]>([]);
const error = ref<string | null>(null);

const columns = [
  { key: "token", label: "Consent Token" },
  { key: "scope", label: "Scopes" },
  { key: "status", label: "Status" },
  { key: "issuedAt", label: "Issued At" },
  { key: "expiresAt", label: "Expires At" },
  { key: "issuedBy", label: "Issued By" },
];

const statusOptions = ["ACTIVE", "REVOKED", "EXPIRED"];

const maskToken = (token: string) => {
  if (!token) return "";
  if (token.length <= 8) return token;
  return `${token.slice(0, 4)}••••${token.slice(-4)}`;
};

const mapTokenRow = (token: ConsentTokenResponse) => ({
  id: token.id,
  token: maskToken(token.token),
  scope: token.scope || [],
  status: token.status,
  issuedAt: token.issuedAt ? new Date(token.issuedAt).toLocaleString() : "",
  expiresAt: token.expiresAt ? new Date(token.expiresAt).toLocaleString() : "—",
  issuedBy: token.issuedBy,
});

const statusColor = (status: string) => {
  switch (status) {
    case "ACTIVE":
      return "green";
    case "EXPIRED":
      return "orange";
    case "REVOKED":
      return "red";
    default:
      return "gray";
  }
};

const loadTokens = async () => {
  if (!tenantId.value) {
    toast.add({ title: "Tenant ID required", color: "red" });
    return;
  }

  loading.value = true;
  error.value = null;

  try {
    const response = await $fetch<ConsentTokenListResponse>(
      `${runtimeConfig.public.apiBaseUrl}/admin/security/consent-tokens`,
      {
        params: {
          tenant_id: tenantId.value,
          status: selectedStatuses.value,
        },
        credentials: "include",
      }
    );
    rows.value = (response?.data || []).map(mapTokenRow);
  } catch (err: any) {
    error.value = err?.message || "Failed to load consent tokens";
    rows.value = [];
  } finally {
    loading.value = false;
  }
};

const exportAudit = async () => {
  const path = runtimeConfig.public.apiBaseUrl.replace("/api/v1", "") + "/admin/security/lifecycle-events";
  await navigator.clipboard.writeText(
    `${path}?tenant_id=${encodeURIComponent(tenantId.value || "")}`
  );
  toast.add({
    title: "Audit endpoint copied",
    description: "Lifecycle export URL copied to clipboard.",
    color: "primary",
  });
};

onMounted(() => {
  if (tenantId.value) {
    loadTokens();
  }
});

watch(selectedStatuses, () => {
  if (tenantId.value) {
    loadTokens();
  }
});

definePageMeta({
  layout: "embedded",
  title: "AdminSecurityConsent",
});
</script>
