<template>
  <UContainer class="py-10 space-y-6">
    <header class="space-y-2">
      <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
        <UIcon name="i-heroicons-queue-list" class="text-primary" />
        <span class="uppercase tracking-wide">Security · Lifecycle</span>
      </div>
      <div class="space-y-1">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          Lifecycle Events
        </h1>
        <p class="text-gray-600 dark:text-gray-300 max-w-3xl">
          Inspect retention, export, and erasure events emitted by the plugin. Payload previews are filtered for sensitive keys to comply with masking rules.
        </p>
      </div>
    </header>

    <UCard>
      <template #header>
        <div class="flex flex-wrap items-end gap-4">
          <UFormGroup label="Tenant ID" required class="w-full sm:w-64">
            <UInput
              v-model="tenantId"
              placeholder="tenant-123"
              @keyup.enter="loadEvents"
              :disabled="loading"
            />
          </UFormGroup>

          <UFormGroup label="Event Types" class="w-full sm:w-72">
            <USelectMenu
              v-model="selectedTypes"
              :options="eventTypeOptions"
              multiple
              searchable
              :disabled="loading"
            >
              <template #label>
                <div class="flex flex-wrap gap-1">
                  <UBadge
                    v-for="type in selectedTypes"
                    :key="type"
                    size="xs"
                    color="primary"
                    variant="soft"
                  >
                    {{ type }}
                  </UBadge>
                  <span v-if="!selectedTypes.length" class="text-gray-500 dark:text-gray-400"
                    >All types</span
                  >
                </div>
              </template>
            </USelectMenu>
          </UFormGroup>

          <UFormGroup label="Limit" class="w-full sm:w-32">
            <UInput
              v-model.number="limit"
              type="number"
              min="1"
              max="200"
              :disabled="loading"
            />
          </UFormGroup>

          <UButton
            color="primary"
            :disabled="!tenantId"
            :loading="loading"
            @click="loadEvents"
          >
            Refresh
          </UButton>
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

          <div class="space-y-3">
            <UCard
              v-for="event in events"
              :key="event.id"
              class="border border-gray-200/70 dark:border-gray-700/80"
            >
              <template #header>
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <div class="flex items-center gap-2">
                    <UBadge :color="eventStatusColor(event.status)" size="xs" class="uppercase">
                      {{ event.status }}
                    </UBadge>
                    <span class="font-semibold">{{ event.eventType }}</span>
                  </div>
                  <span class="text-sm text-gray-500 dark:text-gray-400">
                    {{ formatDate(event.occurredAt) }}
                  </span>
                </div>
              </template>

              <div class="space-y-3 text-sm">
                <div class="flex flex-wrap gap-3">
                  <span><strong>Asset:</strong> {{ event.assetKey }}</span>
                  <span><strong>Recorded By:</strong> {{ event.recordedBy }}</span>
                </div>
                <UDivider />
                <div>
                  <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-2">
                    Payload Preview
                  </p>
                  <UCode :code="prettyPayload(event.payload)" language="json" class="text-xs" />
                </div>
              </div>
            </UCard>
          </div>

          <div v-if="!loading && !events.length" class="text-center py-10">
            <UIcon name="i-heroicons-book-open" class="text-4xl text-gray-300 dark:text-gray-600" />
            <p class="mt-3 text-gray-500 dark:text-gray-400">
              {{ tenantId ? "No lifecycle events captured yet." : "Enter a tenant ID to inspect lifecycle events." }}
            </p>
          </div>
        </div>
      </template>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import type { LifecycleEventListResponse, LifecycleEventResponse } from "~/types/security";

const runtimeConfig = useRuntimeConfig();
const tenantId = ref("");
const selectedTypes = ref<string[]>([]);
const limit = ref(50);
const loading = ref(false);
const events = ref<LifecycleEventResponse[]>([]);
const error = ref<string | null>(null);

const eventTypeOptions = [
  "RETENTION_START",
  "RETENTION_PURGE",
  "EXPORT",
  "ERASURE",
  "CONSENT_REVOKE",
  "CONSENT_RENEW",
];

const formatDate = (value?: string) => {
  if (!value) return "";
  return new Date(value).toLocaleString();
};

const prettyPayload = (payload: any) => {
  if (!payload) return "{}";
  try {
    return JSON.stringify(payload, null, 2);
  } catch {
    return JSON.stringify({ error: "unserializable" }, null, 2);
  }
};

const eventStatusColor = (status: string) => {
  switch (status) {
    case "SUCCEEDED":
      return "green";
    case "FAILED":
      return "red";
    case "PENDING":
      return "orange";
    default:
      return "gray";
  }
};

const loadEvents = async () => {
  if (!tenantId.value) {
    return;
  }
  loading.value = true;
  error.value = null;

  try {
    const response = await $fetch<LifecycleEventListResponse>(
      `${runtimeConfig.public.apiBaseUrl}/admin/security/lifecycle-events`,
      {
        params: {
          tenant_id: tenantId.value,
          event_type: selectedTypes.value,
          limit: limit.value > 0 ? limit.value : undefined,
        },
        credentials: "include",
      }
    );
    events.value = response?.data ?? [];
  } catch (err: any) {
    error.value = err?.message || "Failed to load lifecycle events";
    events.value = [];
  } finally {
    loading.value = false;
  }
};

watch([selectedTypes, limit], () => {
  if (tenantId.value) {
    loadEvents();
  }
});

definePageMeta({
  layout: "embedded",
  title: "AdminSecurityLifecycle",
});
</script>
