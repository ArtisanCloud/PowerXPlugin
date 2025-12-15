<template>
  <div class="space-y-6">
    <UCard>
      <template #header>
        <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          <div>
            <h2 class="text-lg font-semibold">{{ t("iam.members.title") }}</h2>
            <p class="text-sm text-gray-500">
              {{ t("iam.members.caption") }}
            </p>
          </div>
          <div class="flex flex-col md:flex-row gap-3 w-full md:w-auto">
            <USelectMenu
              :options="tenantOptions"
              v-model="selectedTenantUuid"
              :placeholder="t('iam.members.selectTenant')"
              class="w-full md:w-64"
              data-testid="tenant-select"
            />
            <div class="flex gap-2">
              <UButton color="primary" icon="i-heroicons-plus" @click="openMemberModal()" data-testid="create-member">
                {{ t("iam.members.createButton") }}
              </UButton>
              <UButton icon="i-heroicons-arrow-up-tray" @click="showImportModal = true">
                {{ t("iam.members.importButton") }}
              </UButton>
            </div>
            <UButton
              icon="i-heroicons-arrow-path"
              variant="soft"
              class="md:ml-2"
              :loading="loading"
              @click="fetchMembers"
            >
              {{ t("common.refresh") }}
            </UButton>
          </div>
        </div>
      </template>

      <div class="flex flex-col md:flex-row gap-4 mb-4">
        <UInput
          v-model="search"
          :placeholder="t('iam.members.searchPlaceholder')"
          icon="i-heroicons-magnifying-glass"
          class="w-full md:w-1/2"
        />
        <USelectMenu
          v-model="statusFilter"
          :options="statusOptions"
          class="w-full md:w-64"
          :placeholder="t('iam.members.statusFilter')"
        />
      </div>

      <UTable :rows="memberRows" :columns="columns" :loading="loading" data-testid="member-table">
        <template #status-data="{ row }">
          <UBadge :color="row.status === 'active' ? 'green' : 'yellow'">
            {{ formatMemberStatus(row.status) }}
          </UBadge>
        </template>

        <template #roles-data="{ row }">
          <div class="flex flex-wrap gap-1">
            <UBadge v-for="role in row.roles" :key="role" variant="soft" color="neutral">
              {{ role }}
            </UBadge>
            <span v-if="!row.roles?.length" class="text-xs text-gray-400">
              {{ t("iam.members.noRoles") }}
            </span>
          </div>
        </template>

        <template #actions-data="{ row }">
          <div class="flex items-center gap-2">
            <UButton size="2xs" variant="ghost" icon="i-heroicons-pencil-square" @click="openMemberModal(row)" />
            <UButton
              size="2xs"
              variant="ghost"
              :color="row.status === 'active' ? 'yellow' : 'green'"
              @click="toggleMemberStatus(row)"
            >
              {{ row.status === "active" ? t("iam.members.disable") : t("iam.members.enable") }}
            </UButton>
          </div>
        </template>
      </UTable>
    </UCard>

    <UModal v-model="showMemberModal">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-base font-semibold">
                {{
                  editingMember
                    ? t("iam.members.editModal.title")
                    : t("iam.members.createButton")
                }}
              </h3>
              <p class="text-sm text-gray-500">
                {{ t("iam.members.editModal.caption") }}
              </p>
            </div>
            <UButton icon="i-heroicons-x-mark" variant="ghost" @click="closeMemberModal" />
          </div>
        </template>

        <UForm :state="memberForm" class="space-y-4">
          <UFormGroup :label="t('iam.members.fields.email')" required>
            <UInput v-model="memberForm.email" type="email" placeholder="user@example.com" />
          </UFormGroup>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <UFormGroup :label="t('iam.members.fields.displayName')">
              <UInput v-model="memberForm.display_name" />
            </UFormGroup>
            <UFormGroup :label="t('iam.members.fields.username')">
              <UInput v-model="memberForm.username" />
            </UFormGroup>
          </div>
          <UFormGroup :label="t('iam.members.fields.phone')">
            <UInput v-model="memberForm.phone" />
          </UFormGroup>
          <UFormGroup :label="t('iam.members.fields.department')">
            <USelectMenu
              v-model="memberForm.department_id"
              :options="departmentOptions"
              option-attribute="label"
              value-attribute="value"
              searchable
            />
          </UFormGroup>
          <UFormGroup :label="t('iam.members.fields.status')">
            <USelectMenu v-model="memberForm.status" :options="statusOptions" />
          </UFormGroup>
        </UForm>

        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton variant="soft" @click="closeMemberModal">
              {{ t("common.cancel") }}
            </UButton>
            <UButton color="primary" :loading="memberSaving" @click="submitMember">
              {{ t("common.save") }}
            </UButton>
          </div>
        </template>
      </UCard>
    </UModal>

    <UModal v-model="showImportModal">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-base font-semibold">
                {{ t("iam.members.importModal.title") }}
              </h3>
              <p class="text-sm text-gray-500">
                {{ t("iam.members.importModal.caption") }}
              </p>
            </div>
            <UButton icon="i-heroicons-x-mark" variant="ghost" @click="closeImportModal" />
          </div>
        </template>

        <div class="space-y-3">
          <UAlert
            color="emerald"
            variant="soft"
            title=""
            :description="t('iam.members.importModal.hint')"
          />
          <UTextarea
            v-model="importText"
            :rows="8"
            :placeholder="t('iam.members.importModal.placeholder')"
          />
        </div>

        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton variant="soft" @click="closeImportModal">
              {{ t("common.cancel") }}
            </UButton>
            <UButton color="primary" :loading="importing" @click="submitImport">
              {{ t("iam.members.importModal.submit") }}
            </UButton>
          </div>
        </template>
      </UCard>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, reactive, watch, onMounted } from "vue";
import { useDebounceFn } from "@vueuse/core";
import { useIAMService } from "~/composables/api/services/iamService";
import { useIAMStore } from "~/stores/iam";
import { useToast } from "#imports";

definePageMeta({
  layout: "default",
});

const iam = useIAMService();
const store = useIAMStore();
const toast = useToast();
const { t } = useI18n();

const loading = ref(false);
const memberSaving = ref(false);
const importing = ref(false);
const showMemberModal = ref(false);
const showImportModal = ref(false);
const editingMember = ref<any | null>(null);
const search = ref("");
const statusFilter = ref("");
const importText = ref("");

const memberForm = reactive({
  email: "",
  display_name: "",
  username: "",
  phone: "",
  department_id: undefined as number | undefined,
  status: "active",
});

const selectedTenantUuid = computed({
  get: () => store.activeTenantUuid,
  set: (value: string | null) => {
    if (value) {
      store.setActiveTenant(value);
    }
  },
});

const tenantOptions = computed(() =>
  store.tenants.map((tenant) => ({
    label: `${tenant.name} (${tenant.key})`,
    value: tenant.key,
  }))
);

const departmentOptions = computed(() => [
  { label: t("iam.members.fields.noDepartment"), value: undefined },
  ...store.departments.map((dept) => ({
    label: `${dept.name} (${dept.code})`,
    value: dept.id,
  })),
]);

const statusOptions = [
  { label: t("iam.members.status.active"), value: "active" },
  { label: t("iam.members.status.disabled"), value: "disabled" },
  { label: t("iam.members.status.locked"), value: "locked" },
];

const columns = [
  { key: "display_name", label: t("iam.members.table.name") },
  { key: "email", label: "Email" },
  { key: "status", label: t("iam.members.table.status") },
  { key: "roles", label: t("iam.members.table.roles") },
  { key: "department", label: t("iam.members.table.department") },
  { key: "actions", label: "" },
];

const memberRows = computed(() =>
  store.members.map((member) => ({
    ...member,
    department: store.departments.find(
      (dept) => dept.id === member.department_id
    )?.name,
  }))
);

const ensureTenants = async () => {
  if (store.tenants.length) return;
  try {
    const response = await iam.listTenants();
    store.setTenants(response?.data?.items ?? []);
  } catch (error: any) {
    toast.add({
      title: t("iam.notifications.loadTenantsFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  }
};

const loadDepartments = async () => {
  if (!store.activeTenantUuid) return;
  try {
    const response = await iam.listDepartments(store.activeTenantUuid);
    store.setDepartments(response?.data?.items ?? []);
  } catch {
    // silent
  }
};

const fetchMembers = async () => {
  if (!store.activeTenantUuid) {
    store.setMembers([]);
    return;
  }
  loading.value = true;
  try {
    const response = await iam.listMembers({
      tenantUuid: store.activeTenantUuid,
      status: statusFilter.value,
      query: search.value,
    });
    store.setMembers(response?.data?.items ?? []);
  } catch (error: any) {
    toast.add({
      title: t("iam.members.loadFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    loading.value = false;
  }
};

const debouncedFetch = useDebounceFn(fetchMembers, 400);

watch(search, () => {
  debouncedFetch();
});

watch(statusFilter, () => {
  fetchMembers();
});

watch(
  () => store.activeTenantUuid,
  async (next) => {
    if (next) {
      await loadDepartments();
      await fetchMembers();
    }
  }
);

const openMemberModal = (member?: any) => {
  if (member) {
    editingMember.value = member;
    memberForm.email = member.email;
    memberForm.display_name = member.display_name;
    memberForm.username = member.username;
    memberForm.phone = member.phone || "";
    memberForm.department_id = member.department_id;
    memberForm.status = member.status;
  } else {
    editingMember.value = null;
    memberForm.email = "";
    memberForm.display_name = "";
    memberForm.username = "";
    memberForm.phone = "";
    memberForm.department_id = undefined;
    memberForm.status = "active";
  }
  showMemberModal.value = true;
};

const closeMemberModal = () => {
  showMemberModal.value = false;
};

const submitMember = async () => {
  if (!store.activeTenantUuid) {
    toast.add({
      title: t("iam.notifications.selectTenant"),
      color: "red",
    });
    return;
  }
  memberSaving.value = true;
  try {
    if (editingMember.value) {
      await iam.updateMember(editingMember.value.member_id, {
        display_name: memberForm.display_name,
        status: memberForm.status,
        department_id: memberForm.department_id ?? null,
      });
    } else {
      await iam.createMember({
        tenant_uuid: store.activeTenantUuid,
        email: memberForm.email,
        display_name: memberForm.display_name,
        username: memberForm.username,
        phone: memberForm.phone,
        department_id: memberForm.department_id ?? null,
        status: memberForm.status,
      });
    }
    toast.add({ title: t("iam.members.saveSuccess"), color: "green" });
    showMemberModal.value = false;
    await fetchMembers();
  } catch (error: any) {
    toast.add({
      title: t("iam.members.saveFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    memberSaving.value = false;
  }
};

const toggleMemberStatus = async (member: any) => {
  const nextStatus = member.status === "active" ? "disabled" : "active";
  try {
    await iam.updateMember(member.member_id, { status: nextStatus });
    toast.add({ title: t("iam.members.statusUpdated") });
    await fetchMembers();
  } catch (error: any) {
    toast.add({
      title: t("iam.members.updateFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  }
};

const closeImportModal = () => {
  showImportModal.value = false;
  importText.value = "";
};

const submitImport = async () => {
  if (!store.activeTenantUuid) {
    toast.add({ title: t("iam.notifications.selectTenant"), color: "red" });
    return;
  }
  const rows = importText.value
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);
  if (!rows.length) {
    toast.add({ title: t("iam.members.importModal.empty"), color: "red" });
    return;
  }
  const members = rows.map((row) => {
    const [email, display] = row.split(",").map((value) => value?.trim());
    return {
      email,
      display_name: display,
      status: "active",
    };
  });
  importing.value = true;
  try {
    const response = await iam.bulkImportMembers({
      tenant_uuid: store.activeTenantUuid,
      members,
    });
    toast.add({
      title: t("iam.members.importModal.success"),
      description: t("iam.members.importModal.summary", {
        success: response?.data?.created?.length ?? 0,
        failed: response?.data?.failed?.length ?? 0,
      }),
      color: "green",
    });
    closeImportModal();
    await fetchMembers();
  } catch (error: any) {
    toast.add({
      title: t("iam.members.importModal.failed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    importing.value = false;
  }
};

const formatMemberStatus = (status: string) => {
  switch (status) {
    case "disabled":
      return t("iam.members.status.disabled");
    case "locked":
      return t("iam.members.status.locked");
    default:
      return t("iam.members.status.active");
  }
};

onMounted(async () => {
  await ensureTenants();
  if (store.activeTenantUuid) {
    await loadDepartments();
    await fetchMembers();
  }
});
</script>
