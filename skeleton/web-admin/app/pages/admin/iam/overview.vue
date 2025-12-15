<template>
  <div class="space-y-6">
    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold">
              {{ $t("iam.overview.title") }}
            </h2>
            <p class="text-sm text-gray-500">
              {{ $t("iam.overview.caption") }}
            </p>
          </div>
          <div class="flex items-center gap-2">
            <UButton icon="i-heroicons-plus" color="primary" @click="openCreateTenant">
              {{ $t("iam.overview.createTenant") }}
            </UButton>
            <UButton
              icon="i-heroicons-arrow-path"
              variant="soft"
              :loading="loading"
              @click="fetchTenants"
            >
              {{ $t("common.refresh") }}
            </UButton>
          </div>
        </div>
      </template>
      <UTable :rows="tenantRows" :columns="columns" :loading="loading">
        <template #status-data="{ row }">
          <UBadge :color="row.status === 'active' ? 'green' : 'yellow'">
            {{ formatStatus(row.status) }}
          </UBadge>
        </template>
        <template #actions-data="{ row }">
          <div class="flex items-center gap-2">
            <USelectMenu
              :options="statusOptions"
              v-model="row.status"
              size="sm"
              @update:model-value="(value) => changeTenantStatus(row, value)"
            />
            <UButton
              size="xs"
              variant="soft"
              color="neutral"
              @click="() => openPlanDrawer(row)"
            >
              {{ $t("iam.overview.adjustPlan") }}
            </UButton>
          </div>
        </template>
      </UTable>
    </UCard>

    <USlideover v-model="showPlanDrawer">
      <UCard class="w-full max-w-md">
        <template #header>
          <div class="flex items-center justify-between">
            <h3 class="text-base font-semibold">
              {{ $t("iam.overview.planDrawer.title") }}
            </h3>
            <UButton
              icon="i-heroicons-x-mark"
              size="xs"
              color="neutral"
              variant="ghost"
              @click="showPlanDrawer = false"
            />
          </div>
        </template>
        <div v-if="selectedTenant">
          <UForm :state="planForm">
            <UFormGroup :label="$t('iam.overview.planDrawer.planLabel')">
              <USelect v-model="planForm.plan" :options="planOptions" />
            </UFormGroup>
            <UFormGroup class="mt-3" :label="$t('iam.overview.planDrawer.displayName')">
              <UInput v-model="planForm.name" />
            </UFormGroup>
          </UForm>
          <div class="mt-6 flex justify-end gap-2">
            <UButton variant="soft" @click="showPlanDrawer = false">
              {{ $t("common.cancel") }}
            </UButton>
            <UButton color="primary" :loading="planSaving" @click="submitPlanForm">
              {{ $t("common.save") }}
            </UButton>
          </div>
        </div>
      </UCard>
    </USlideover>

    <UModal v-model="showCreateModal">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-base font-semibold">
                {{ $t("iam.overview.createModal.title") }}
              </h3>
              <p class="text-sm text-gray-500">
                {{ $t("iam.overview.createModal.caption") }}
              </p>
            </div>
            <UButton icon="i-heroicons-x-mark" variant="ghost" color="neutral" @click="closeCreate" />
          </div>
        </template>
        <UForm :state="createForm" class="space-y-4">
          <UFormGroup :label="$t('iam.overview.createModal.key')" required>
            <UInput v-model="createForm.key" placeholder="tenant-key" />
          </UFormGroup>
          <UFormGroup :label="$t('iam.overview.createModal.name')" required>
            <UInput v-model="createForm.name" />
          </UFormGroup>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormGroup :label="$t('iam.overview.createModal.plan')">
              <USelect v-model="createForm.plan" :options="planOptions" />
            </UFormGroup>
            <UFormGroup :label="$t('iam.overview.createModal.status')">
              <USelect v-model="createForm.status" :options="statusOptions" />
            </UFormGroup>
          </div>
        </UForm>
        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton variant="soft" @click="closeCreate">
              {{ $t("common.cancel") }}
            </UButton>
            <UButton color="primary" :loading="creating" @click="submitCreate">
              {{ $t("common.save") }}
            </UButton>
          </div>
        </template>
      </UCard>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from "vue";
import { useToast } from "#imports";
import {
  useIAMService,
  type TenantSummary,
} from "~/composables/api/services/iamService";
import { useIAMStore } from "~/stores/iam";
const { t } = useI18n();

definePageMeta({
  layout: "default",
});

const iam = useIAMService();
const toast = useToast();
const store = useIAMStore();

const loading = ref(false);
const creating = ref(false);
const planSaving = ref(false);
const showCreateModal = ref(false);
const showPlanDrawer = ref(false);
const selectedTenant = ref<TenantSummary | null>(null);

const createForm = reactive({
  key: "",
  name: "",
  plan: "free",
  status: "active",
});

const planForm = reactive({
  plan: "free",
  name: "",
});

const statusOptions = [
  { label: "Active", value: "active" },
  { label: "Suspended", value: "suspended" },
];

const planOptions = [
  { label: "Free", value: "free" },
  { label: "Standard", value: "standard" },
  { label: "Premium", value: "premium" },
];

const columns = [
  { key: "name", label: "Tenant" },
  { key: "key", label: "Key" },
  { key: "status", label: "Status" },
  { key: "plan", label: "Plan" },
  { key: "actions", label: "" },
];

const tenantRows = computed(() =>
  (store.tenants ?? []).map((tenant) => ({
    ...tenant,
  }))
);

const fetchTenants = async () => {
  loading.value = true;
  try {
    const response = await iam.listTenants();
    store.setTenants(response?.data?.items ?? []);
  } catch (error: any) {
    toast.add({
      title: t("iam.notifications.loadTenantsFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    loading.value = false;
  }
};

const formatStatus = (status: string) => {
  if (status === "suspended") {
    return t("iam.overview.status.suspended");
  }
  return t("iam.overview.status.active");
};

const changeTenantStatus = async (tenant: TenantSummary, status: string) => {
  try {
    await iam.updateTenant(tenant.id, { status });
    toast.add({ title: t("iam.overview.statusUpdated") });
    await fetchTenants();
  } catch (error: any) {
    toast.add({
      title: t("iam.overview.updateFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  }
};

const openPlanDrawer = (tenant: TenantSummary) => {
  selectedTenant.value = tenant;
  planForm.plan = tenant.plan ?? "free";
  planForm.name = tenant.name;
  showPlanDrawer.value = true;
};

const submitPlanForm = async () => {
  if (!selectedTenant.value) return;
  planSaving.value = true;
  try {
    await iam.updateTenant(selectedTenant.value.id, {
      plan: planForm.plan,
      name: planForm.name,
    });
    toast.add({ title: t("common.save"), description: t("common.confirm") });
    showPlanDrawer.value = false;
    await fetchTenants();
  } catch (error: any) {
    toast.add({
      title: t("iam.overview.updateFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    planSaving.value = false;
  }
};

const openCreateTenant = () => {
  showCreateModal.value = true;
};

const closeCreate = () => {
  showCreateModal.value = false;
  createForm.key = "";
  createForm.name = "";
  createForm.plan = "free";
  createForm.status = "active";
};

const submitCreate = async () => {
  creating.value = true;
  try {
    await iam.createTenant({
      key: createForm.key,
      name: createForm.name,
      plan: createForm.plan,
      status: createForm.status,
    });
    toast.add({ title: t("iam.overview.createSuccess") });
    closeCreate();
    await fetchTenants();
  } catch (error: any) {
    toast.add({
      title: t("iam.overview.createFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    creating.value = false;
  }
};

const toggleTenantStatus = async (tenant: TenantSummary) => {
  const nextStatus = tenant.status === "active" ? "suspended" : "active";
  await changeTenantStatus(tenant, nextStatus);
};

onMounted(fetchTenants);
</script>
