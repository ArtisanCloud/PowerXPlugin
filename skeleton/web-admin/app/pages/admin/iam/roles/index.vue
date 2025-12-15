<template>
  <div class="space-y-6">
    <UCard>
      <template #header>
        <div class="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div>
            <h2 class="text-lg font-semibold">{{ t("iam.roles.title") }}</h2>
            <p class="text-sm text-gray-500">
              {{ t("iam.roles.caption") }}
            </p>
          </div>
          <div class="flex flex-col gap-3 w-full md:w-auto">
            <div class="flex flex-col md:flex-row gap-2">
              <USelectMenu
                v-model="selectedTenant"
                :options="tenantOptions"
                class="w-full md:w-64"
                :placeholder="t('iam.roles.selectTenant')"
              />
              <UInput
                v-model="search"
                icon="i-heroicons-magnifying-glass"
                :placeholder="t('iam.roles.searchPlaceholder')"
                class="w-full md:w-64"
              />
              <USelectMenu
                v-model="scopeFilter"
                :options="scopeOptions"
                class="w-full md:w-52"
                :placeholder="t('iam.roles.scopeFilter')"
              />
            </div>
            <div class="flex gap-2">
              <UButton color="primary" icon="i-heroicons-plus" @click="openCreateRole" :disabled="!store.activeTenantUuid">
                {{ t("iam.roles.create") }}
              </UButton>
              <UButton
                icon="i-heroicons-arrow-path"
                variant="soft"
                :loading="rolesLoading"
                @click="fetchRoles"
              >
                {{ t("common.refresh") }}
              </UButton>
            </div>
          </div>
        </div>
      </template>

      <UTable :rows="roleRows" :columns="columns" :loading="rolesLoading" data-testid="roles-table">
        <template #scope_type-data="{ row }">
          <UBadge :color="row.scope_type === 'system' ? 'orange' : 'blue'" variant="soft">
            {{ formatScope(row.scope_type) }}
          </UBadge>
        </template>
        <template #created_at-data="{ row }">
          <span class="text-xs text-gray-500">{{ formatDate(row.created_at) }}</span>
        </template>
        <template #actions-data="{ row }">
          <div class="flex flex-wrap gap-2">
            <UButton size="2xs" variant="soft" color="neutral" @click="openEditRole(row)">
              {{ t("common.edit") }}
            </UButton>
            <UButton size="2xs" variant="soft" color="primary" @click="openPermissionDrawer(row)">
              {{ t("iam.roles.managePermissions") }}
            </UButton>
            <UButton size="2xs" variant="soft" @click="openMembersDrawer(row)">
              {{ t("iam.roles.manageMembers") }}
            </UButton>
            <UButton
              size="2xs"
              variant="ghost"
              color="red"
              icon="i-heroicons-trash"
              @click="deleteRole(row)"
            />
          </div>
        </template>
      </UTable>
    </UCard>

    <UModal v-model="showRoleModal">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-base font-semibold">
                {{ editingRole ? t("iam.roles.editTitle") : t("iam.roles.create") }}
              </h3>
              <p class="text-sm text-gray-500">{{ t("iam.roles.formCaption") }}</p>
            </div>
            <UButton icon="i-heroicons-x-mark" variant="ghost" @click="closeRoleModal" />
          </div>
        </template>

        <UForm :state="roleForm" class="space-y-4">
          <UFormGroup :label="t('iam.roles.fields.code')" required>
            <UInput v-model="roleForm.code" :disabled="!!editingRole" placeholder="system.viewer" />
          </UFormGroup>
          <UFormGroup :label="t('iam.roles.fields.name')" required>
            <UInput v-model="roleForm.name" />
          </UFormGroup>
          <UFormGroup :label="t('iam.roles.fields.description')">
            <UTextarea v-model="roleForm.description" :rows="3" />
          </UFormGroup>
          <UFormGroup :label="t('iam.roles.fields.scope')">
            <USelectMenu v-model="roleForm.scope_type" :options="scopeOptions" value-attribute="value" />
          </UFormGroup>
          <UFormGroup v-if="!editingRole" :label="t('iam.roles.fields.cloneRole')">
            <USelectMenu
              v-model="roleForm.clone_role_id"
              :options="cloneOptions"
              :placeholder="t('iam.roles.clonePlaceholder')"
            />
          </UFormGroup>
        </UForm>

        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton variant="soft" @click="closeRoleModal">{{ t("common.cancel") }}</UButton>
            <UButton color="primary" :loading="roleSaving" @click="submitRole">
              {{ t("common.save") }}
            </UButton>
          </div>
        </template>
      </UCard>
    </UModal>

    <USlideover v-model="showPermissionDrawer">
      <UCard class="w-full max-w-xl">
        <template #header>
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-base font-semibold">
                {{ t("iam.roles.permissionsTitle", { role: roleDetail?.name ?? "" }) }}
              </h3>
              <p class="text-sm text-gray-500">
                {{ t("iam.roles.permissionsCaption") }}
              </p>
            </div>
            <UButton icon="i-heroicons-x-mark" variant="ghost" @click="closePermissionDrawer" />
          </div>
        </template>
        <PermissionTree
          v-model="permissionSelection"
          :permissions="permissionList"
          :loading="permissionLoading"
        />
        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton variant="soft" @click="closePermissionDrawer">{{ t("common.cancel") }}</UButton>
            <UButton color="primary" :loading="permissionSaving" @click="savePermissions">
              {{ t("common.save") }}
            </UButton>
          </div>
        </template>
      </UCard>
    </USlideover>

    <USlideover v-model="showMembersDrawer">
      <UCard class="w-full max-w-lg">
        <template #header>
          <div class="flex items-center justify-between">
            <div>
              <h3 class="text-base font-semibold">
                {{ t("iam.roles.membersTitle", { role: roleDetail?.name ?? "" }) }}
              </h3>
              <p class="text-sm text-gray-500">
                {{ t("iam.roles.membersCaption") }}
              </p>
            </div>
            <UButton icon="i-heroicons-x-mark" variant="ghost" @click="closeMembersDrawer" />
          </div>
        </template>
        <div class="space-y-3">
          <UInput
            v-model="memberSearch"
            icon="i-heroicons-magnifying-glass"
            :placeholder="t('iam.roles.membersSearch')"
          />
          <div class="max-h-96 overflow-y-auto space-y-2 pr-2">
            <div
              v-for="member in filteredMembers"
              :key="member.member_id"
              class="flex items-start justify-between rounded border border-gray-100 dark:border-gray-800 px-3 py-2"
            >
              <div>
                <p class="text-sm font-medium">{{ member.display_name || member.email }}</p>
                <p class="text-xs text-gray-500">{{ member.email }}</p>
              </div>
              <UCheckbox
                :model-value="memberSelectionSet.has(member.member_id)"
                @update:modelValue="() => toggleMember(member.member_id)"
              />
            </div>
            <p v-if="!filteredMembers.length" class="text-sm text-gray-500 text-center py-6">
              {{ t("iam.roles.noMembers") }}
            </p>
          </div>
        </div>
        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton variant="soft" @click="closeMembersDrawer">{{ t("common.cancel") }}</UButton>
            <UButton color="primary" :loading="membersSaving" @click="saveMembers">
              {{ t("common.save") }}
            </UButton>
          </div>
        </template>
      </UCard>
    </USlideover>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, onMounted, watch } from "vue";
import { useDebounceFn } from "@vueuse/core";
import { useToast } from "#imports";
import {
  useIAMService,
  type RoleRecord,
  type PermissionRecord,
  type MemberRecord,
} from "~/composables/api/services/iamService";
import { useIAMStore } from "~/stores/iam";

definePageMeta({
  layout: "default",
});

const { t } = useI18n();
const toast = useToast();
const iam = useIAMService();
const store = useIAMStore();

const roles = ref<RoleRecord[]>([]);
const rolesLoading = ref(false);
const search = ref("");
const scopeFilter = ref("");
const scopeOptions = [
  { label: t("iam.roles.scopeTenant"), value: "tenant" },
  { label: t("iam.roles.scopeSystem"), value: "system" },
];

const roleRows = computed(() => roles.value ?? []);
const columns = computed(() => [
  { key: "name", label: t("iam.roles.table.name") },
  { key: "code", label: "Code" },
  { key: "scope_type", label: t("iam.roles.table.scope") },
  { key: "member_count", label: t("iam.roles.table.members") },
  { key: "created_at", label: t("iam.roles.table.createdAt") },
  { key: "actions", label: "" },
]);

const tenantOptions = computed(() =>
  store.tenants.map((tenant) => ({
    label: `${tenant.name} (${tenant.key})`,
    value: tenant.key,
  }))
);

const selectedTenant = computed({
  get: () => store.activeTenantUuid,
  set: (value: string | null) => {
    if (value) {
      store.setActiveTenant(value);
    }
  },
});

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

const fetchRoles = async () => {
  if (!store.activeTenantUuid) {
    roles.value = [];
    return;
  }
  rolesLoading.value = true;
  try {
    const response = await iam.listRoles({
      tenantUuid: store.activeTenantUuid,
      query: search.value,
      scopeType: scopeFilter.value,
    });
    roles.value = response?.data?.items ?? [];
  } catch (error: any) {
    toast.add({
      title: t("iam.roles.loadFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    rolesLoading.value = false;
  }
};

const debouncedFetchRoles = useDebounceFn(fetchRoles, 400);
watch(search, () => debouncedFetchRoles());
watch(scopeFilter, () => fetchRoles());

watch(
  () => store.activeTenantUuid,
  async (next) => {
    if (next) {
      await fetchRoles();
      await fetchMembers();
    } else {
      roles.value = [];
    }
  }
);

onMounted(async () => {
  await ensureTenants();
  if (store.activeTenantUuid) {
    await fetchRoles();
    await fetchMembers();
  }
});

const formatScope = (value: string) => {
  if (value === "system") return t("iam.roles.scopeSystem");
  return t("iam.roles.scopeTenant");
};

const formatDate = (value?: string) => {
  if (!value) return "-";
  try {
    return new Intl.DateTimeFormat(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    }).format(new Date(value));
  } catch {
    return value;
  }
};

const editingRole = ref<RoleRecord | null>(null);
const showRoleModal = ref(false);
const roleSaving = ref(false);
const roleForm = reactive({
  code: "",
  name: "",
  description: "",
  scope_type: "tenant",
  clone_role_id: undefined as number | undefined,
});

const cloneOptions = computed(() =>
  roles.value.map((role) => ({
    label: `${role.name} (${role.code})`,
    value: role.id,
  }))
);

const openCreateRole = () => {
  editingRole.value = null;
  roleForm.code = "";
  roleForm.name = "";
  roleForm.description = "";
  roleForm.scope_type = "tenant";
  roleForm.clone_role_id = undefined;
  showRoleModal.value = true;
};

const openEditRole = (role: RoleRecord) => {
  editingRole.value = role;
  roleForm.code = role.code;
  roleForm.name = role.name;
  roleForm.description = role.description || "";
  roleForm.scope_type = role.scope_type || "tenant";
  roleForm.clone_role_id = undefined;
  showRoleModal.value = true;
};

const closeRoleModal = () => {
  showRoleModal.value = false;
};

const submitRole = async () => {
  if (!store.activeTenantUuid) {
    toast.add({ title: t("iam.notifications.selectTenant"), color: "red" });
    return;
  }
  if (!roleForm.name || (!editingRole.value && !roleForm.code)) {
    toast.add({ title: t("iam.roles.validationRequired"), color: "red" });
    return;
  }
  roleSaving.value = true;
  try {
    if (editingRole.value) {
      await iam.updateRole(editingRole.value.id, {
        name: roleForm.name,
        description: roleForm.description,
        scope_type: roleForm.scope_type,
      });
    } else {
      await iam.createRole({
        tenant_uuid: store.activeTenantUuid,
        code: roleForm.code,
        name: roleForm.name,
        description: roleForm.description,
        scope_type: roleForm.scope_type,
        clone_role_id: roleForm.clone_role_id,
      });
    }
    toast.add({ title: t("iam.roles.saveSuccess"), color: "green" });
    showRoleModal.value = false;
    await fetchRoles();
  } catch (error: any) {
    toast.add({
      title: t("iam.roles.saveFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    roleSaving.value = false;
  }
};

const deleteRole = async (role: RoleRecord) => {
  if (!window.confirm(t("iam.roles.deleteConfirm", { role: role.name }))) {
    return;
  }
  try {
    await iam.deleteRole(role.id);
    toast.add({ title: t("iam.roles.deleteSuccess"), color: "green" });
    await fetchRoles();
  } catch (error: any) {
    toast.add({
      title: t("iam.roles.deleteFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  }
};

// Permissions drawer state
const permissionList = ref<PermissionRecord[]>([]);
const permissionLoading = ref(false);
const permissionSaving = ref(false);
const permissionSelection = ref<number[]>([]);
const showPermissionDrawer = ref(false);
const roleDetail = ref<RoleRecord | null>(null);

const ensurePermissionsLoaded = async () => {
  if (permissionList.value.length) return;
  permissionLoading.value = true;
  try {
    const response = await iam.listPermissions();
    permissionList.value = response?.data?.items ?? [];
  } catch (error: any) {
    toast.add({
      title: t("iam.roles.permissionsLoadFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    permissionLoading.value = false;
  }
};

const loadRoleDetail = async (role: RoleRecord) => {
  try {
    const response = await iam.getRole(role.id);
    roleDetail.value = response?.data ?? null;
  } catch (error: any) {
    toast.add({
      title: t("iam.roles.loadFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  }
};

const openPermissionDrawer = async (role: RoleRecord) => {
  await ensurePermissionsLoaded();
  await loadRoleDetail(role);
  permissionSelection.value = [...(roleDetail.value?.permission_ids ?? [])];
  showPermissionDrawer.value = true;
};

const closePermissionDrawer = () => {
  showPermissionDrawer.value = false;
};

const savePermissions = async () => {
  if (!roleDetail.value || !store.activeTenantUuid) {
    return;
  }
  permissionSaving.value = true;
  try {
    await iam.replaceRolePermissions(roleDetail.value.id, {
      tenant_uuid: store.activeTenantUuid,
      permission_ids: permissionSelection.value,
    });
    toast.add({ title: t("iam.roles.permissionsSaved"), color: "green" });
    showPermissionDrawer.value = false;
    await fetchRoles();
  } catch (error: any) {
    toast.add({
      title: t("iam.roles.permissionsFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    permissionSaving.value = false;
  }
};

// Members drawer state
const showMembersDrawer = ref(false);
const membersSaving = ref(false);
const memberOptions = ref<MemberRecord[]>([]);
const memberSearch = ref("");
const memberSelection = ref<number[]>([]);

const fetchMembers = async () => {
  if (!store.activeTenantUuid) {
    memberOptions.value = [];
    return;
  }
  try {
    const response = await iam.listMembers({
      tenantUuid: store.activeTenantUuid,
      pageSize: 200,
    });
    memberOptions.value = response?.data?.items ?? [];
  } catch {
    memberOptions.value = [];
  }
};

const memberSelectionSet = computed(() => new Set(memberSelection.value));

const filteredMembers = computed(() => {
  if (!memberSearch.value.trim()) {
    return memberOptions.value;
  }
  const term = memberSearch.value.trim().toLowerCase();
  return memberOptions.value.filter(
    (member) =>
      member.email.toLowerCase().includes(term) ||
      (member.display_name ?? "").toLowerCase().includes(term)
  );
});

const toggleMember = (memberId: number) => {
  const next = new Set(memberSelectionSet.value);
  if (next.has(memberId)) {
    next.delete(memberId);
  } else {
    next.add(memberId);
  }
  memberSelection.value = Array.from(next.values());
};

const openMembersDrawer = async (role: RoleRecord) => {
  await loadRoleDetail(role);
  await fetchMembers();
  memberSelection.value = [...(roleDetail.value?.member_ids ?? [])];
  showMembersDrawer.value = true;
};

const closeMembersDrawer = () => {
  showMembersDrawer.value = false;
};

const saveMembers = async () => {
  if (!roleDetail.value || !store.activeTenantUuid) {
    return;
  }
  const current = new Set(roleDetail.value.member_ids ?? []);
  const next = new Set(memberSelection.value);
  const toAdd = Array.from(next).filter((id) => !current.has(id));
  const toRemove = Array.from(current).filter((id) => !next.has(id));
  if (!toAdd.length && !toRemove.length) {
    showMembersDrawer.value = false;
    return;
  }
  membersSaving.value = true;
  try {
    if (toAdd.length) {
      await iam.addRoleMembers(roleDetail.value.id, {
        tenant_uuid: store.activeTenantUuid,
        member_ids: toAdd,
      });
    }
    if (toRemove.length) {
      await iam.removeRoleMembers(roleDetail.value.id, {
        tenant_uuid: store.activeTenantUuid,
        member_ids: toRemove,
      });
    }
    toast.add({ title: t("iam.roles.membersSaved"), color: "green" });
    showMembersDrawer.value = false;
    await fetchRoles();
  } catch (error: any) {
    toast.add({
      title: t("iam.roles.membersFailed"),
      description: error?.data?.message || error?.message,
      color: "red",
    });
  } finally {
    membersSaving.value = false;
  }
};
</script>
