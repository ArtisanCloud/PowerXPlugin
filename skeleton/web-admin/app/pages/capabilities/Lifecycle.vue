<template>
  <div class="p-6 space-y-6">
    <UCard>
      <template #header>
        <div>
          <p class="text-xs uppercase text-gray-500">{{ $t("capabilities.lifecycle.badge") }}</p>
          <p class="text-xl font-semibold">{{ $t("capabilities.lifecycle.title") }}</p>
          <p class="text-sm text-gray-500">
            {{ $t("capabilities.lifecycle.description") }}
          </p>
        </div>
      </template>
      <div class="space-y-5">
        <div class="grid gap-4 md:grid-cols-2">
          <UFormField :label="$t('capabilities.lifecycle.fields.capabilityId')" required>
            <div class="flex gap-2">
              <UInput v-model="planForm.capability_id" :placeholder="$t('capabilities.lifecycle.placeholders.capability')" />
              <UButton variant="soft" color="primary" @click="handleLoadPlans">
                {{ $t("capabilities.lifecycle.actions.loadPlans") }}
              </UButton>
            </div>
          </UFormField>
          <UFormField :label="$t('capabilities.lifecycle.fields.changeType')" required>
            <USelectMenu
              v-model="planForm.change_type"
              :options="changeTypeOptions"
              :placeholder="$t('capabilities.lifecycle.placeholders.changeType')"
            />
          </UFormField>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <UFormField :label="$t('capabilities.lifecycle.fields.notificationChannels')">
            <USelectMenu
              v-model="planForm.notification_channels"
              :options="channelOptions"
              multiple
              searchable
              :placeholder="$t('capabilities.lifecycle.placeholders.channels')"
            />
          </UFormField>
          <div class="grid gap-4 md:grid-cols-2">
            <UFormField :label="$t('capabilities.lifecycle.fields.graceHours')" required>
              <UInput
                v-model.number="planForm.grace_period_hours"
                type="number"
                min="1"
                :placeholder="$t('capabilities.lifecycle.placeholders.graceHours')"
              />
            </UFormField>
            <UFormField :label="$t('capabilities.lifecycle.fields.dualRun')">
              <UInput v-model="planForm.dual_run_until" type="datetime-local" />
            </UFormField>
          </div>
        </div>

        <UFormField :label="$t('capabilities.lifecycle.fields.diffSummary')" required>
          <UTextarea
            v-model="planForm.diff_summary"
            :rows="4"
            :placeholder="$t('capabilities.lifecycle.placeholders.diffSummary')"
          />
        </UFormField>

        <UFormField :label="$t('capabilities.lifecycle.fields.rollbackPlan')">
          <UTextarea
            v-model="planForm.rollback_plan"
            :rows="3"
            :placeholder="$t('capabilities.lifecycle.placeholders.rollbackPlan')"
          />
        </UFormField>

        <div class="grid gap-4 md:grid-cols-2">
          <UFormField :label="$t('capabilities.lifecycle.fields.impactScope')">
            <UInput v-model="metadataForm.impact_scope" :placeholder="$t('capabilities.lifecycle.placeholders.impactScope')" />
          </UFormField>
          <UFormField :label="$t('capabilities.lifecycle.fields.migrationGuide')">
            <UInput
              v-model="metadataForm.migration_guide"
              :placeholder="$t('capabilities.lifecycle.placeholders.migrationGuide')"
            />
          </UFormField>
        </div>

        <div>
          <div class="flex items-center justify-between">
            <div>
              <p class="font-semibold">{{ $t("capabilities.lifecycle.sections.windows") }}</p>
              <p class="text-sm text-gray-500">
                {{ $t("capabilities.lifecycle.sections.windowHint") }}
              </p>
            </div>
            <UButton size="xs" variant="soft" @click="addWindow">
              {{ $t("capabilities.lifecycle.actions.addWindow") }}
            </UButton>
          </div>
          <div v-if="planForm.windows.length" class="mt-4 space-y-4">
            <div
              v-for="(window, index) in planForm.windows"
              :key="`window-${index}`"
              class="rounded-lg border border-gray-200 dark:border-gray-800 p-4 space-y-3"
            >
              <div class="flex items-center justify-between">
                <span class="font-medium">{{ window.label || $t("capabilities.lifecycle.sections.windowLabel") }}</span>
                <UButton color="rose" variant="ghost" size="xs" @click="removeWindow(index)">
                  {{ $t("capabilities.lifecycle.actions.removeWindow") }}
                </UButton>
              </div>
              <div class="grid gap-3 md:grid-cols-2">
                <UInput v-model="window.label" :placeholder="$t('capabilities.lifecycle.placeholders.windowLabel')" />
                <UInput
                  v-model="window.percent"
                  type="number"
                  min="0"
                  max="100"
                  :placeholder="$t('capabilities.lifecycle.placeholders.percent')"
                />
              </div>
              <div class="grid gap-3 md:grid-cols-2">
                <UInput v-model="window.start_at" type="datetime-local" />
                <UInput v-model="window.end_at" type="datetime-local" />
              </div>
              <UInput v-model="window.condition" :placeholder="$t('capabilities.lifecycle.placeholders.condition')" />
            </div>
          </div>
          <p v-else class="mt-3 text-sm text-gray-500">
            {{ $t("capabilities.lifecycle.sections.noWindow") }}
          </p>
        </div>

        <div class="flex justify-end gap-3">
          <UButton variant="ghost" color="neutral" @click="resetForm">
            {{ $t("common.reset") }}
          </UButton>
          <UButton color="primary" :loading="savingPlan" @click="handleCreatePlan">
            {{ $t("capabilities.lifecycle.actions.createPlan") }}
          </UButton>
        </div>
      </div>
    </UCard>

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <div>
            <p class="text-lg font-semibold">{{ $t("capabilities.lifecycle.list.title") }}</p>
            <p class="text-sm text-gray-500">
              {{ $t("capabilities.lifecycle.list.description") }}
            </p>
          </div>
          <UButton variant="soft" color="primary" :loading="loadingPlans" @click="handleLoadPlans">
            {{ $t("capabilities.lifecycle.actions.refresh") }}
          </UButton>
        </div>
      </template>

      <div v-if="loadingPlans" class="flex items-center justify-center py-8 text-sm text-gray-500">
        {{ $t("common.loading") }}
      </div>
      <div v-else-if="!plans.length" class="text-sm text-gray-500 py-6 text-center">
        {{ $t("capabilities.lifecycle.list.empty") }}
      </div>
      <div v-else class="space-y-4">
        <div
          v-for="plan in plans"
          :key="plan.id"
          class="rounded-lg border border-gray-200 dark:border-gray-800 p-4 space-y-4"
        >
          <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
            <div>
              <p class="text-xs uppercase text-gray-500">{{ plan.capability_id }}</p>
              <p class="text-lg font-semibold">{{ plan.change_type }}</p>
              <p class="text-sm text-gray-600 whitespace-pre-line">{{ plan.diff_summary }}</p>
              <p class="text-xs text-gray-400 mt-1">
                {{ $t("capabilities.lifecycle.list.planMeta", { author: plan.created_by || "system", time: formatTime(plan.created_at) }) }}
              </p>
            </div>
            <UBadge :color="statusColor(plan.status)" variant="soft" class="uppercase tracking-wide">
              {{ statusLabel(plan.status) }}
            </UBadge>
          </div>

          <div class="grid gap-4 md:grid-cols-3 text-sm">
            <div>
              <p class="font-semibold text-gray-700 dark:text-gray-200">
                {{ $t("capabilities.lifecycle.sections.channels") }}
              </p>
              <p class="text-gray-500">
                {{
                  plan.notification_channels?.length
                    ? plan.notification_channels.join(", ")
                    : $t("capabilities.lifecycle.sections.noChannels")
                }}
              </p>
            </div>
            <div>
              <p class="font-semibold text-gray-700 dark:text-gray-200">
                {{ $t("capabilities.lifecycle.sections.grace") }}
              </p>
              <p class="text-gray-500">
                {{ plan.grace_period_hours }} {{ $t("capabilities.lifecycle.units.hours") }}
              </p>
            </div>
            <div>
              <p class="font-semibold text-gray-700 dark:text-gray-200">
                {{ $t("capabilities.lifecycle.sections.dualRun") }}
              </p>
              <p class="text-gray-500">
                {{ plan.dual_run_until || $t("capabilities.lifecycle.sections.notSet") }}
              </p>
            </div>
          </div>

          <div v-if="plan.windows?.length" class="text-sm">
            <p class="font-semibold text-gray-700 dark:text-gray-200">
              {{ $t("capabilities.lifecycle.sections.windowTimeline") }}
            </p>
            <div class="mt-2 space-y-1">
              <div
                v-for="window in plan.windows"
                :key="`${plan.id}-${window.label}-${window.start_at}`"
                class="flex flex-col md:flex-row md:items-center md:justify-between gap-1"
              >
                <div>
                  <span class="font-medium">{{ window.label || $t("capabilities.lifecycle.sections.windowLabel") }}</span>
                  <span class="ml-2 text-gray-500">{{ window.percent }}%</span>
                </div>
                <div class="text-gray-500">
                  {{ window.start_at || "—" }} → {{ window.end_at || "—" }}
                  <span v-if="window.condition" class="ml-2 text-xs text-gray-400">
                    {{ window.condition }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div class="flex flex-col gap-2 md:flex-row md:items-center">
            <UInput
              v-model="statusNotes[plan.id]"
              class="md:flex-1"
              :placeholder="$t('capabilities.lifecycle.fields.statusNotes')"
            />
            <div class="flex flex-wrap gap-2">
              <UButton
                v-for="nextStatus in actionableStatuses(plan.status)"
                :key="`${plan.id}-${nextStatus}`"
                size="sm"
                variant="soft"
                :loading="isUpdating(plan.id, nextStatus)"
                @click="handleStatus(plan, nextStatus)"
              >
                {{ statusLabel(nextStatus) }}
              </UButton>
            </div>
          </div>
        </div>
      </div>
    </UCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n, useToast, useRoute } from "#imports";
import {
  useCapabilityLifecycleApi,
  type LifecyclePlan,
  type PlanTemplate,
} from "~/composables/api";

definePageMeta({
  alias: ["/capabilities/lifecycle"],
});

type WindowForm = {
  label: string;
  start_at: string;
  end_at: string;
  percent: number;
  condition: string;
};

const { t } = useI18n();
const toast = useToast();
const route = useRoute();

const { getTemplate, listPlans, createPlan, updateStatus } = useCapabilityLifecycleApi();

const template = ref<PlanTemplate | null>(null);
const plans = ref<LifecyclePlan[]>([]);
const loadingTemplate = ref(false);
const loadingPlans = ref(false);
const savingPlan = ref(false);

const planForm = reactive({
  capability_id: "",
  change_type: "",
  diff_summary: "",
  notification_channels: [] as string[],
  grace_period_hours: 72,
  dual_run_until: "",
  rollback_plan: "",
  windows: [] as WindowForm[],
});

const metadataForm = reactive({
  impact_scope: "",
  migration_guide: "",
});

const statusNotes = reactive<Record<string, string>>({});
const statusUpdating = reactive<Record<string, string>>({});

const changeTypeOptions = computed(() => template.value?.change_types || []);
const channelOptions = computed(() => template.value?.channel_options || []);
const statusOptions = computed(() => template.value?.status_options || []);

onMounted(async () => {
  await hydrateTemplate();
  const capabilityFromQuery = (route.query.capability as string) || "";
  if (capabilityFromQuery) {
    planForm.capability_id = capabilityFromQuery;
    await fetchPlans(capabilityFromQuery);
  }
});

async function hydrateTemplate() {
  loadingTemplate.value = true;
  try {
    template.value = await getTemplate();
    if (!planForm.change_type && template.value?.change_types?.length) {
      planForm.change_type = template.value.change_types[0];
    }
  } catch (error) {
    console.error("[capabilities] failed to load lifecycle template", error);
    toast.add({
      title: t("capabilities.lifecycle.toast.templateFailed"),
      color: "rose",
    });
  } finally {
    loadingTemplate.value = false;
  }
}

async function fetchPlans(capabilityId?: string) {
  const target = capabilityId || planForm.capability_id;
  if (!target) {
    plans.value = [];
    return;
  }
  loadingPlans.value = true;
  try {
    const result = await listPlans(target);
    plans.value = result?.plans || [];
    plans.value.forEach((plan) => {
      if (!statusNotes[plan.id]) {
        statusNotes[plan.id] = "";
      }
    });
  } catch (error) {
    console.error("[capabilities] failed to load lifecycle plans", error);
    toast.add({
      title: t("capabilities.lifecycle.toast.loadFailed"),
      color: "rose",
    });
  } finally {
    loadingPlans.value = false;
  }
}

function handleLoadPlans() {
  if (!planForm.capability_id) {
    toast.add({
      title: t("capabilities.lifecycle.toast.capabilityRequired"),
      color: "amber",
    });
    return;
  }
  fetchPlans(planForm.capability_id);
}

function addWindow() {
  planForm.windows.push({
    label: `wave-${planForm.windows.length + 1}`,
    start_at: "",
    end_at: "",
    percent: 50,
    condition: "",
  });
}

function removeWindow(index: number) {
  planForm.windows.splice(index, 1);
}

function resetForm() {
  planForm.diff_summary = "";
  planForm.notification_channels = [];
  planForm.grace_period_hours = 72;
  planForm.dual_run_until = "";
  planForm.rollback_plan = "";
  planForm.windows = [];
  metadataForm.impact_scope = "";
  metadataForm.migration_guide = "";
}

async function handleCreatePlan() {
  if (!planForm.capability_id || !planForm.change_type || !planForm.diff_summary) {
    toast.add({
      title: t("capabilities.lifecycle.toast.required"),
      color: "amber",
    });
    return;
  }
  savingPlan.value = true;
  try {
    const payload = {
      capability_id: planForm.capability_id.trim(),
      change_type: planForm.change_type,
      diff_summary: planForm.diff_summary,
      notification_channels: [...planForm.notification_channels],
      grace_period_hours: Number(planForm.grace_period_hours) || 72,
      dual_run_until: planForm.dual_run_until,
      rollback_plan: planForm.rollback_plan,
      windows: planForm.windows.map((window) => ({
        label: window.label,
        start_at: window.start_at,
        end_at: window.end_at,
        percent: Number(window.percent) || 0,
        condition: window.condition,
      })),
      metadata: buildMetadata(),
    };
    await createPlan(payload);
    toast.add({
      title: t("capabilities.lifecycle.toast.planCreated"),
      color: "green",
    });
    await fetchPlans(payload.capability_id);
    resetForm();
  } catch (error) {
    console.error("[capabilities] failed to create lifecycle plan", error);
    toast.add({
      title: t("capabilities.lifecycle.toast.createFailed"),
      color: "rose",
    });
  } finally {
    savingPlan.value = false;
  }
}

function buildMetadata() {
  const meta: Record<string, string> = {};
  if (metadataForm.impact_scope) {
    meta.impact_scope = metadataForm.impact_scope;
  }
  if (metadataForm.migration_guide) {
    meta.migration_guide = metadataForm.migration_guide;
  }
  return meta;
}

async function handleStatus(plan: LifecyclePlan, nextStatus: string) {
  if (plan.status === nextStatus) {
    return;
  }
  statusUpdating[plan.id] = nextStatus;
  try {
    await updateStatus(plan.id, {
      status: nextStatus,
      notes: statusNotes[plan.id] || "",
    });
    toast.add({
      title: t("capabilities.lifecycle.toast.statusUpdated"),
      description: t("capabilities.lifecycle.toast.statusDesc", {
        status: statusLabel(nextStatus),
      }),
      color: "green",
    });
    statusNotes[plan.id] = "";
    await fetchPlans(plan.capability_id);
  } catch (error) {
    console.error("[capabilities] failed to update lifecycle status", error);
    toast.add({
      title: t("capabilities.lifecycle.toast.statusFailed"),
      color: "rose",
    });
  } finally {
    delete statusUpdating[plan.id];
  }
}

function actionableStatuses(current: string) {
  return (statusOptions.value || []).filter((status) => status !== current);
}

function statusColor(status: string) {
  const normalized = (status || "").toLowerCase();
  if (normalized === "approved" || normalized === "completed") return "green";
  if (normalized === "paused") return "amber";
  if (normalized === "pending") return "blue";
  if (normalized === "draft") return "neutral";
  return "gray";
}

function statusLabel(status: string) {
  return t(`capabilities.lifecycle.status.${status}`, status);
}

function isUpdating(planId: string, status: string) {
  return statusUpdating[planId] === status;
}

function formatTime(value?: string) {
  if (!value) return "—";
  try {
    return new Date(value).toLocaleString();
  } catch {
    return value;
  }
}
</script>
