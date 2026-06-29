<template>
  <UContainer class="py-8 space-y-6">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="space-y-2">
        <div class="flex flex-wrap items-center gap-2">
          <UBadge color="neutral" variant="soft">基础能力</UBadge>
          <UBadge color="info" variant="soft">非 SCRM</UBadge>
        </div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">客户基础管理</h1>
        <p class="max-w-3xl text-sm text-gray-600 dark:text-gray-300">
          查看 C 端客户账号、外部身份、租户关系、登录事件与 MiniApp 入口解析结果。
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UInput
          v-model="search"
          icon="i-heroicons-magnifying-glass"
          placeholder="搜索 UUID、邮箱、手机、入口码"
          class="w-72"
          @keyup.enter="reloadCurrentTab"
        />
        <UButton icon="i-heroicons-arrow-path" variant="soft" :loading="loading" @click="reloadAll">
          刷新
        </UButton>
        <UButton icon="i-heroicons-plus" color="primary" @click="openCreateForm">
          新增客户
        </UButton>
      </div>
    </div>

    <UAlert color="info" variant="soft" icon="i-heroicons-information-circle">
      <template #title>边界说明</template>
      <template #description>
        本页面只管理客户身份、认证来源和租户归属。客户画像、标签、分群、跟进、销售归属和营销生命周期由 SCRM 插件实现。
      </template>
    </UAlert>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <UCard v-for="item in stats" :key="item.label">
        <div class="flex items-center justify-between gap-3">
          <div>
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ item.label }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ item.value }}</p>
          </div>
          <UIcon :name="item.icon" class="h-8 w-8 text-primary-600 dark:text-primary-300" />
        </div>
      </UCard>
    </div>

    <UCard>
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <UTabs v-model="activeTab" :items="tabs" />
          <div class="flex items-center gap-2">
            <USelect v-model="status" :items="statusOptions" class="w-36" />
            <USelect v-model="pageSize" :items="pageSizeOptions" class="w-28" />
          </div>
        </div>
      </template>

      <UTable
        :columns="activeColumns"
        :data="activeItems"
        :loading="loading"
        :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
      >
        <template #status-cell="{ row }">
          <UBadge :color="statusColor(row.original.status)" variant="soft">
            {{ row.original.status || "-" }}
          </UBadge>
        </template>
        <template #ok-cell="{ row }">
          <UBadge :color="row.original.ok ? 'success' : 'error'" variant="soft">
            {{ row.original.ok ? "成功" : "失败" }}
          </UBadge>
        </template>
        <template #uuid-cell="{ row }">
          <span class="block truncate font-mono text-xs">{{ row.original.uuid || "-" }}</span>
        </template>
        <template #customer-cell="{ row }">
          <span class="block truncate font-mono text-xs">{{ row.original.customer_uuid || "-" }}</span>
        </template>
        <template #tenant-cell="{ row }">
          <span class="block truncate font-mono text-xs">{{ row.original.tenant_uuid || "-" }}</span>
        </template>
        <template #updated-cell="{ row }">
          <span class="text-sm text-gray-600 dark:text-gray-300">{{ formatDate(row.original.updated_at || row.original.created_at) }}</span>
        </template>
      </UTable>

      <div class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-200 px-4 py-3 dark:border-gray-700">
        <span class="text-sm text-gray-500 dark:text-gray-400">
          共 {{ activePage.total }} 条，当前第 {{ activePage.page }} / {{ Math.max(activePage.total_pages, 1) }} 页
        </span>
        <UPagination
          v-if="activePage.total_pages > 1"
          v-model:page="page"
          :total="activePage.total"
          :items-per-page="pageSize"
          show-edges
        />
      </div>
    </UCard>

    <UModal v-model:open="showCreateModal">
      <template #content>
        <div class="space-y-5 p-6">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">新增客户账号</h2>
            <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
              创建基础客户身份、密码登录身份和当前租户关系。
            </p>
          </div>

          <UAlert color="warning" variant="soft" icon="i-heroicons-exclamation-triangle">
            <template #title>仅限基础身份</template>
            <template #description>
              这里不会创建客户画像、标签、分群或跟进记录。
            </template>
          </UAlert>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField v-if="isRoot" label="租户" required>
              <USelect
                v-model="createForm.tenant_uuid"
                :items="tenantOptions"
                :loading="loadingTenants"
                placeholder="选择租户"
              />
            </UFormField>
            <UFormField v-else label="当前租户">
              <UInput :model-value="currentTenantLabel" readonly />
            </UFormField>
            <UFormField label="显示名">
              <UInput v-model="createForm.display_name" autocomplete="off" placeholder="可选" />
            </UFormField>
            <UFormField label="邮箱">
              <UInput
                v-model="createForm.email"
                autocomplete="off"
                name="customer_email"
                placeholder="customer@example.com"
              />
            </UFormField>
            <UFormField label="手机号">
              <UInput
                v-model="createForm.phone"
                autocomplete="off"
                name="customer_phone"
                placeholder="可选，与邮箱二选一"
              />
            </UFormField>
            <UFormField label="客户登录密码" required help="用于 C 端客户本地登录，不是后台员工/member 密码。">
              <UInput
                v-model="createForm.password"
                type="password"
                autocomplete="new-password"
                name="customer_login_password"
                placeholder="至少 8 位"
              />
            </UFormField>
            <UFormField label="备注">
              <UInput v-model="createForm.note" placeholder="写入 metadata.note" />
            </UFormField>
          </div>

          <p v-if="createError" class="text-sm text-error-600 dark:text-error-400">{{ createError }}</p>

          <div class="flex justify-end gap-2">
            <UButton variant="ghost" color="neutral" :disabled="creating" @click="showCreateModal = false">
              取消
            </UButton>
            <UButton color="primary" icon="i-heroicons-check" :loading="creating" @click="submitCreate">
              创建
            </UButton>
          </div>
        </div>
      </template>
    </UModal>
  </UContainer>
</template>

<script setup lang="ts">
import {
  useCustomerBaseApi,
  type CustomerAccount,
  type CustomerIdentity,
  type CustomerLoginEvent,
  type CustomerMembership,
  type CustomerOverview,
  type CustomerPage,
  type MiniAppEntry,
} from "~/composables/api/useCustomerBase";
import { useIAMService } from "~/composables/api/services/iamService";
import { useUserStore } from "~/stores/user";

type TabKey = "accounts" | "identities" | "memberships" | "loginEvents" | "entries";
type TableRow = Record<string, any>;

const api = useCustomerBaseApi();
const iam = useIAMService();
const userStore = useUserStore();
const { isRoot, currentTenantUuid, currentTenant } = storeToRefs(userStore);
const activeTab = ref<TabKey>("accounts");
const search = ref("");
const status = ref("all");
const page = ref(1);
const pageSize = ref(20);
const loading = ref(false);
const loadingTenants = ref(false);
const showCreateModal = ref(false);
const creating = ref(false);
const createError = ref("");
const tenantOptions = ref<{ label: string; value: string }[]>([]);
const createForm = reactive({
  tenant_uuid: "",
  email: "",
  phone: "",
  password: "",
  display_name: "",
  note: "",
});
const overview = ref<CustomerOverview>({
  accounts_total: 0,
  accounts_active: 0,
  memberships_total: 0,
  memberships_active: 0,
  mini_app_entries_active: 0,
  login_events_24h: 0,
});

const emptyPage = <T,>(): CustomerPage<T> => ({
  items: [],
  page: 1,
  page_size: pageSize.value,
  total: 0,
  total_pages: 0,
});

const accounts = ref<CustomerPage<CustomerAccount>>(emptyPage());
const identities = ref<CustomerPage<CustomerIdentity>>(emptyPage());
const memberships = ref<CustomerPage<CustomerMembership>>(emptyPage());
const loginEvents = ref<CustomerPage<CustomerLoginEvent>>(emptyPage());
const entries = ref<CustomerPage<MiniAppEntry>>(emptyPage());

const tabs = [
  { label: "客户账号", value: "accounts" },
  { label: "外部身份", value: "identities" },
  { label: "租户关系", value: "memberships" },
  { label: "登录事件", value: "loginEvents" },
  { label: "MiniApp 入口", value: "entries" },
];

const statusOptions = [
  { label: "全部状态", value: "all" },
  { label: "Active", value: "active" },
  { label: "Pending", value: "pending" },
  { label: "Suspended", value: "suspended" },
  { label: "Disabled", value: "disabled" },
];

const pageSizeOptions = [
  { label: "20 / 页", value: 20 },
  { label: "50 / 页", value: 50 },
  { label: "100 / 页", value: 100 },
];

const stats = computed(() => [
  { label: "客户账号", value: overview.value.accounts_total, icon: "i-heroicons-user-circle" },
  { label: "活跃租户关系", value: overview.value.memberships_active, icon: "i-heroicons-building-office-2" },
  { label: "可用入口", value: overview.value.mini_app_entries_active, icon: "i-heroicons-qr-code" },
  { label: "24h 登录事件", value: overview.value.login_events_24h, icon: "i-heroicons-clock" },
]);

const accountColumns = [
  { accessorKey: "display_name", header: "显示名" },
  { accessorKey: "customer_uuid", header: "Customer UUID", id: "customer" },
  { accessorKey: "primary_email", header: "邮箱" },
  { accessorKey: "primary_phone", header: "手机" },
  { accessorKey: "tenant_uuid", header: "Tenant UUID", id: "tenant" },
  { accessorKey: "status", header: "状态" },
  { accessorKey: "updated_at", header: "更新时间", id: "updated" },
];
const identityColumns = [
  { accessorKey: "provider", header: "来源" },
  { accessorKey: "provider_subject", header: "外部主体" },
  { accessorKey: "customer_uuid", header: "Customer UUID", id: "customer" },
  { accessorKey: "email", header: "邮箱" },
  { accessorKey: "phone", header: "手机" },
  { accessorKey: "status", header: "状态" },
  { accessorKey: "updated_at", header: "更新时间", id: "updated" },
];
const membershipColumns = [
  { accessorKey: "membership_uuid", header: "Membership UUID", id: "uuid" },
  { accessorKey: "customer_uuid", header: "Customer UUID", id: "customer" },
  { accessorKey: "tenant_uuid", header: "Tenant UUID", id: "tenant" },
  { accessorKey: "source", header: "来源" },
  { accessorKey: "status", header: "状态" },
  { accessorKey: "updated_at", header: "更新时间", id: "updated" },
];
const loginEventColumns = [
  { accessorKey: "identity_provider", header: "来源" },
  { accessorKey: "event_type", header: "事件" },
  { accessorKey: "customer_uuid", header: "Customer UUID", id: "customer" },
  { accessorKey: "tenant_uuid", header: "Tenant UUID", id: "tenant" },
  { accessorKey: "ok", header: "结果" },
  { accessorKey: "error_code", header: "错误码" },
  { accessorKey: "created_at", header: "时间", id: "updated" },
];
const entryColumns = [
  { accessorKey: "entry_code", header: "入口码" },
  { accessorKey: "entry_type", header: "类型" },
  { accessorKey: "tenant_uuid", header: "Tenant UUID", id: "tenant" },
  { accessorKey: "channel", header: "渠道" },
  { accessorKey: "campaign", header: "活动" },
  { accessorKey: "status", header: "状态" },
  { accessorKey: "updated_at", header: "更新时间", id: "updated" },
];

const activeColumns = computed(() => {
  switch (activeTab.value) {
    case "identities":
      return identityColumns;
    case "memberships":
      return membershipColumns;
    case "loginEvents":
      return loginEventColumns;
    case "entries":
      return entryColumns;
    default:
      return accountColumns;
  }
});

const activePage = computed<CustomerPage<any>>(() => {
  switch (activeTab.value) {
    case "identities":
      return identities.value;
    case "memberships":
      return memberships.value;
    case "loginEvents":
      return loginEvents.value;
    case "entries":
      return entries.value;
    default:
      return accounts.value;
  }
});

const activeItems = computed<TableRow[]>(() => activePage.value.items || []);
const resolvedCurrentTenantUUID = computed(() => currentTenantUuid.value?.trim() || "");
const currentTenantLabel = computed(() => {
  const tenant = currentTenant.value as any;
  const name = String(tenant?.tenant_name || tenant?.name || "").trim();
  const uuid = resolvedCurrentTenantUUID.value;
  if (name && uuid) return `${name} / ${uuid}`;
  return uuid || "未解析到当前租户";
});

const query = computed(() => ({
  page: page.value,
  page_size: pageSize.value,
  q: search.value.trim() || undefined,
  status: status.value === "all" ? undefined : status.value,
}));

async function openCreateForm() {
  createError.value = "";
  createForm.email = "";
  createForm.phone = "";
  createForm.password = "";
  createForm.display_name = "";
  createForm.note = "";
  if (isRoot.value) {
    await loadTenantOptions();
    createForm.tenant_uuid =
      createForm.tenant_uuid ||
      overview.value.tenant_uuid ||
      tenantOptions.value[0]?.value ||
      "";
  } else {
    createForm.tenant_uuid = resolvedCurrentTenantUUID.value || overview.value.tenant_uuid || "";
  }
  showCreateModal.value = true;
}

async function loadTenantOptions() {
  if (!isRoot.value || tenantOptions.value.length > 0 || loadingTenants.value) return;
  loadingTenants.value = true;
  try {
    const response = await iam.listTenants({ page: 1, pageSize: 100, status: "active" });
    const items = response.data?.items || [];
    tenantOptions.value = items
      .map((tenant: any) => {
        const value = String(tenant.uuid || tenant.key || "").trim();
        if (!value) return null;
        const label = `${tenant.name || tenant.key || value} / ${value}`;
        return { label, value };
      })
      .filter(Boolean) as { label: string; value: string }[];
  } finally {
    loadingTenants.value = false;
  }
}

function validateCreateForm() {
  if (!createForm.tenant_uuid.trim()) return "Tenant UUID 必填";
  if (!createForm.email.trim() && !createForm.phone.trim()) return "邮箱或手机号至少填写一个";
  if (createForm.password.length < 8) return "密码至少 8 位";
  return "";
}

async function submitCreate() {
  createError.value = validateCreateForm();
  if (createError.value) return;
  creating.value = true;
  try {
    await api.createAccount({
      tenant_uuid: createForm.tenant_uuid.trim(),
      email: createForm.email.trim() || undefined,
      phone: createForm.phone.trim() || undefined,
      password: createForm.password,
      display_name: createForm.display_name.trim() || undefined,
      metadata: createForm.note.trim() ? { note: createForm.note.trim() } : undefined,
    });
    showCreateModal.value = false;
    activeTab.value = "accounts";
    page.value = 1;
    await reloadAll();
  } catch (error: any) {
    createError.value =
      error?.data?.error?.message ||
      error?.response?._data?.error?.message ||
      error?.message ||
      "创建失败";
  } finally {
    creating.value = false;
  }
}

async function reloadOverview() {
  const response = await api.overview();
  overview.value = response.data;
}

async function reloadCurrentTab() {
  loading.value = true;
  try {
    if (activeTab.value === "accounts") {
      accounts.value = (await api.listAccounts(query.value)).data;
    } else if (activeTab.value === "identities") {
      identities.value = (await api.listIdentities(query.value)).data;
    } else if (activeTab.value === "memberships") {
      memberships.value = (await api.listMemberships(query.value)).data;
    } else if (activeTab.value === "loginEvents") {
      loginEvents.value = (await api.listLoginEvents(query.value)).data;
    } else {
      entries.value = (await api.listMiniAppEntries(query.value)).data;
    }
  } finally {
    loading.value = false;
  }
}

async function reloadAll() {
  loading.value = true;
  try {
    await reloadOverview();
    await reloadCurrentTab();
  } finally {
    loading.value = false;
  }
}

function statusColor(value?: string) {
  if (value === "active") return "success";
  if (value === "pending") return "warning";
  if (value === "suspended" || value === "disabled") return "error";
  return "neutral";
}

function formatDate(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

watch([activeTab, status, pageSize], () => {
  page.value = 1;
  reloadCurrentTab();
});
watch(page, () => reloadCurrentTab());

onMounted(() => {
  reloadAll();
});
</script>
