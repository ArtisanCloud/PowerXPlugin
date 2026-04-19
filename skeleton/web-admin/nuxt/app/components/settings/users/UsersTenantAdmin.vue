<!-- /components/settings/users/UsersTenantAdmin.vue -->
<script setup lang="ts">
import {
  ref,
  reactive,
  computed,
  h,
  resolveComponent,
  onMounted,
  watch,
} from "vue";
import { useI18n } from "#imports";
import SelectTree from "~/components/ui/SelectTree.vue";
import { useDepartmentStore } from "~/stores/department";
import {
  useUserService,
  type MemberWithProfile,
} from "~/composables/api/services/userService";
import { useIAMService } from "~/composables/api/services/iamService";
import type { Department } from "~/composables/api/services/departmentService";

// ==== 输入属性（Root 复用时传入 tenantUuid） ====
const props = withDefaults(defineProps<{ tenantUuid: string; readonly?: boolean }>(), {
  readonly: false,
});
const { t, locale } = useI18n();

// ==== 部门store ====
const departmentStore = useDepartmentStore();
const userService = useUserService();
const iamService = useIAMService();

// ===== 类型与数据 =====
type StatusType = "active" | "inactive";
type UserCategory = "all" | "active" | "inactive";

interface RowUser {
  id: number; // Member ID
  userId?: number; // User ID
  sourceType: "federated" | "local";
  name: string;
  username?: string;
  email?: string;
  phone?: string;
  department?: string;
  departmentIds: number[];
  roles: string[];
  status: StatusType | string;
  avatar: string;
  meta?: Record<string, any> | null;
}

// 表格数据和加载状态
const users = ref<RowUser[]>([]);
const loading = ref(false);
const userCategory = ref<UserCategory>("all");
const federatedMemberSource = ref<Record<number, string>>({});

// ====== 过滤/分页（与你现有一致） ======
const searchQuery = ref("");
const filters = reactive({
  department: null as string | null,
  role: null as string | null,
  status: null as string | null,
});

const pagination = reactive({ page: 1, pageSize: 10, total: 0, totalPages: 0 });

// 将部门数据转换为SelectTree需要的TreeNode格式
const departmentTreeItems = computed(() => {
  const convertDepartmentToTreeNode = (dept: Department) => ({
    label: dept.name || "未命名部门",
    value: String(dept.id),
    icon: "i-heroicons-building-office-2",
    children: dept.children?.map(convertDepartmentToTreeNode) || [],
    disabled: dept.status === 0, // 假设status为0表示禁用
  });

  return departmentStore.tree.map(convertDepartmentToTreeNode);
});
const roleOptions = ref<Array<{ label: string; value: number; code: string }>>([]);

const departmentFlatOptions = computed(() => {
  const out: Array<{ label: string; value: number }> = [];
  const walk = (nodes: Department[], prefix = "") => {
    for (const node of nodes) {
      const label = prefix ? `${prefix} / ${node.name}` : node.name;
      out.push({ label, value: Number(node.id) });
      if (Array.isArray(node.children) && node.children.length > 0) {
        walk(node.children as Department[], label);
      }
    }
  };
  walk(departmentStore.tree || []);
  return out;
});

const departmentNameMap = computed(() => {
  const map = new Map<number, string>();
  for (const item of departmentFlatOptions.value) {
    map.set(item.value, item.label);
  }
  return map;
});

const roleCodeToIDMap = computed(() => {
  const map = new Map<string, number>();
  for (const item of roleOptions.value) {
    if (item.code) {
      map.set(String(item.code).toLowerCase(), Number(item.value));
    }
  }
  return map;
});

const roleCodeToLabelMap = computed(() => {
  const map = new Map<string, string>();
  for (const item of roleOptions.value) {
    if (item.code) {
      map.set(String(item.code).toLowerCase(), String(item.label || item.code));
    }
  }
  return map;
});

const roleFilterOptions = computed(() => {
  const items: Array<{ label: string; value: string | null }> = [
    { label: t("organization.user.form.selectRole"), value: null },
  ];
  for (const item of roleOptions.value) {
    items.push({ label: item.label, value: item.code });
  }
  return items;
});

// ====== 导入导出 ======
type ExportFormat = "csv" | "json";

async function exportUsers(format: ExportFormat) {
  try {
    let content: string;
    let filename: string;
    let mimeType: string;

    if (format === "csv") {
      const { default: Papa } = await import("papaparse");
      content = Papa.unparse(
        users.value.map((u) => ({
          用户ID: u.userId || "",
          类型: u.sourceType === "federated" ? "渠道同步" : "本地",
          姓名: u.name,
          用户名: u.username,
          邮箱: u.email,
          部门: u.department || "",
          状态: u.status === "active" ? "激活" : "停用",
        }))
      );
      filename = `users_${new Date().toISOString().split("T")[0]}.csv`;
      mimeType = "text/csv;charset=utf-8;";
    } else {
      content = JSON.stringify(users.value, null, 2);
      filename = `users_${new Date().toISOString().split("T")[0]}.json`;
      mimeType = "application/json;charset=utf-8;";
    }

    const { saveAs } = await import("file-saver");
    const blob = new Blob([content], { type: mimeType });
    saveAs(blob, filename);
  } catch (error) {
    console.error("导出失败:", error);
    alert("导出失败，请重试");
  }
}

function importUsers() {
  const input = document.createElement("input");
  input.type = "file";
  input.accept = ".csv,.json";
  input.onchange = async (e) => {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (!file) return;

    try {
      const text = await file.text();
      let importedData: any[];

      if (file.name.endsWith(".csv")) {
        const { default: Papa } = await import("papaparse");
        const result = Papa.parse(text, { header: true, skipEmptyLines: true });
        importedData = result.data;
      } else {
        importedData = JSON.parse(text);
      }

      // 这里可以添加数据验证和转换逻辑
      console.log("导入的数据:", importedData);
      alert(`成功导入 ${importedData.length} 条记录`);
    } catch (error) {
      console.error("导入失败:", error);
      alert("导入失败，请检查文件格式");
    }
  };
  input.click();
}

const importExportItems = computed(() => [
  [
    {
      label: t("organization.user.export.csv"),
      icon: "i-heroicons-arrow-down-tray",
      click: () => exportUsers("csv"),
    },
    {
      label: t("organization.user.export.json"),
      icon: "i-heroicons-arrow-down-tray",
      click: () => exportUsers("json"),
    },
  ],
  [
    {
      label: t("organization.user.import.button"),
      icon: "i-heroicons-arrow-up-tray",
      click: () => importUsers(),
    },
  ],
]);

// ====== 新增/编辑 ======
const showForm = ref(false);
const isEditing = ref(false);
const editingId = ref<number | null>(null);

// 统一"扁平表单" -> 后端映射 User+Member（我们之前对齐的）
const userForm = reactive({
  name: "",
  username: "",
  email: "",
  phone: "",
  departmentId: null as number | null,
  departmentIds: [] as number[],
  roleIds: [] as number[],
  avatarUrl: "",
  password: "",
  confirmPassword: "",
  status: "active" as "active" | "disabled" | "locked",
  meta: {} as Record<string, any>,
});

function resetForm() {
  userForm.name = "";
  userForm.username = "";
  userForm.email = "";
  userForm.phone = "";
  userForm.departmentId = null;
  userForm.departmentIds = [];
  userForm.roleIds = [];
  userForm.avatarUrl = "";
  userForm.password = "";
  userForm.confirmPassword = "";
  userForm.status = "active";
  userForm.meta = {};
  isEditing.value = false;
  editingId.value = null;
}

function openAddForm() {
  resetForm();
  showForm.value = true;
}

function openEditForm(row: RowUser) {
  resetForm();
  isEditing.value = true;
  editingId.value = row.id; // 这里使用的是Member的ID

  // 将行数据映射回表单
  userForm.name = row.name;
  userForm.username = row.username || "";
  userForm.email = row.email || "";
  userForm.phone = row.phone || "";
  userForm.avatarUrl = row.avatar;
  userForm.status = row.status === "active" ? "active" : "disabled";
  userForm.meta = row.meta || {};
  userForm.departmentIds = Array.isArray(row.departmentIds) ? [...row.departmentIds] : [];
  userForm.departmentId = userForm.departmentIds.length > 0 ? userForm.departmentIds[0] : null;
  userForm.roleIds = (row.roles || [])
    .map((code) => roleCodeToIDMap.value.get(String(code || "").toLowerCase()) || 0)
    .filter((id) => id > 0);
  showForm.value = true;
}

async function saveUser() {
  // 基础校验
  if (!userForm.name || !userForm.email) {
    return alert(t("organization.user.validation.requiredFields"));
  }
  if (!isEditing.value && !userForm.username) {
    return alert("用户名为必填项");
  }
  if (!isEditing.value && userForm.password !== userForm.confirmPassword) {
    return alert(t("organization.user.validation.passwordMismatch"));
  }

  try {
    if (isEditing.value && editingId.value) {
      // 更新用户
      const updatePayload = {
        display_name: userForm.name,
        email: userForm.email,
        phone: userForm.phone,
        avatar_url: userForm.avatarUrl,
        status: userForm.status === "active" ? "active" : "disabled",
        department_id: userForm.departmentIds?.[0] || null,
        department_ids: userForm.departmentIds || [],
        roleIds: userForm.roleIds || [],
        replaceRoles: true,
      };
      await userService.updateUser(editingId.value, updatePayload);
    } else {
      // 创建系统用户
      const createPayload = {
        tenant_uuid: props.tenantUuid,
        display_name: userForm.name,
        email: userForm.email,
        phone: userForm.phone,
        avatar_url: userForm.avatarUrl,
        status: userForm.status === "active" ? "active" : "disabled",
        meta: userForm.meta ?? {},
        username: userForm.username || userForm.email.split("@")[0],
        initial_password: userForm.password,
        department_id: userForm.departmentIds?.[0] || null,
        department_ids: userForm.departmentIds || [],
        roleIds: userForm.roleIds || [],
      };
      await userService.createSystemUser(createPayload);
    }
    showForm.value = false;
    await loadUsers(); // 重新加载数据
  } catch (e: any) {
    alert(e?.message || "保存失败");
  }
}

async function deleteUser(id: number) {
  if (!confirm(t("organization.user.confirmDelete"))) return;
  try {
    // 注意：这里的id是Member的ID，但API可能需要User的ID
    // 根据后端实现调整
    await userService.deleteUser(id);
    await loadUsers(); // 重新加载数据
  } catch (e: any) {
    alert(e?.message || "删除失败");
  }
}

async function toggleUserStatus(row: RowUser) {
  try {
    const newStatus = row.status === "active" ? 0 : 1;
    // 注意：这里的row.id是Member的ID，但API可能需要User的ID
    // 根据后端实现调整
    await userService.setUserStatus(row.id, { status: newStatus });
    await loadUsers(); // 重新加载数据
  } catch (e: any) {
    alert(e?.message || "状态更新失败");
  }
}

// ===== 过滤/分页逻辑 =====
const filteredUsers = computed(() => {
  return users.value.filter((item) => {
    if (filters.department) {
      const departmentID = Number(filters.department);
      if (!item.departmentIds.includes(departmentID)) {
        return false;
      }
    }
    if (filters.role) {
      const roleCode = String(filters.role).toLowerCase();
      if (!(item.roles || []).some((code) => String(code || "").toLowerCase() === roleCode)) {
        return false;
      }
    }
    return true;
  });
});

const paginatedUsers = computed(() => filteredUsers.value);

const categoryStats = computed(() => {
  const active = users.value.filter((item) => item.status === "active").length;
  const inactive = users.value.length - active;
  return {
    all: users.value.length,
    active,
    inactive,
  };
});

function applyCategory(category: UserCategory) {
  userCategory.value = category;
  filters.status = category === "all" ? null : category;
}

const normalizedTotalPages = computed(() => {
  if (pagination.totalPages && pagination.totalPages > 0) {
    return pagination.totalPages;
  }
  const calculated = Math.ceil(pagination.total / Math.max(1, pagination.pageSize));
  return Math.max(1, calculated);
});
const hasNextPage = computed(() => pagination.page < normalizedTotalPages.value);
const hasPrevPage = computed(() => pagination.page > 1);
const pageNumbers = computed(() => {
  const total = normalizedTotalPages.value;
  const current = pagination.page;
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1);
  }
  if (current <= 4) {
    return [1, 2, 3, 4, 5, total];
  }
  if (current >= total - 3) {
    return [1, total - 4, total - 3, total - 2, total - 1, total];
  }
  return [1, current - 1, current, current + 1, total];
});

async function changePage(p: number) {
  if (p >= 1 && p <= normalizedTotalPages.value) {
    pagination.page = p;
    await loadUsers();
  }
}

async function changePageSize(size: number) {
  pagination.pageSize = size;
  pagination.page = 1;
  await loadUsers();
}

function resetFilters() {
  filters.department = filters.role = filters.status = null;
  searchQuery.value = "";
  pagination.page = 1;
  loadUsers();
}

// 监听搜索和过滤条件变化
watch(
  [
    searchQuery,
    () => filters.status,
    () => filters.department,
    () => filters.role,
  ],
  () => {
    pagination.page = 1;
    loadUsers();
  }
);

watch(
  () => filters.status,
  (value) => {
    userCategory.value = value === "active" ? "active" : value === "inactive" ? "inactive" : "all";
  }
);

watch(
  () => props.tenantUuid,
  async () => {
    pagination.page = 1;
    await loadRoleOptions();
    await loadUsers();
  }
);

// ===== 列定义：含"编辑/禁用/删除"操作 =====
const UButton = resolveComponent("UButton");
const UAvatar = resolveComponent("UAvatar");
const UBadge = resolveComponent("UBadge");

const columns = computed(() => {
  const _ = locale.value;
  return [
    {
      id: "avatar",
      accessorKey: "avatar",
      header: "",
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        return h(UAvatar, { src: u.avatar, alt: u.name, size: "sm" });
      },
    },
    {
      id: "userId",
      accessorKey: "userId",
      header: "用户ID",
    },
    {
      id: "sourceType",
      accessorKey: "sourceType",
      header: "类型",
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        return u.sourceType === "federated" ? "渠道同步" : "本地";
      },
    },
    {
      id: "name",
      accessorKey: "name",
      header: t("organization.user.table.name").toString(),
    },
    {
      id: "username",
      accessorKey: "username",
      header: t("organization.user.table.username").toString(),
    },
    ...(props.readonly
      ? []
      : [{
      id: "email",
      accessorKey: "email",
      header: t("organization.user.table.email").toString(),
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        return u.email || "-";
      },
    }]),
    ...(props.readonly
      ? []
      : [{
      id: "phone",
      accessorKey: "phone",
      header: t("organization.user.table.phone").toString(),
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        return maskPhone(u.phone || "");
      },
    }]),
    {
      id: "department",
      accessorKey: "department",
      header: t("organization.user.form.department").toString(),
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        return u.department || "-";
      },
    },
    ...(props.readonly
      ? []
      : [{
      id: "roles",
      accessorKey: "roles",
      header: t("organization.user.form.role").toString(),
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        const labels = (u.roles || [])
          .map((code) => roleCodeToLabelMap.value.get(String(code || "").toLowerCase()) || code)
          .filter(Boolean);
        return labels.length > 0 ? labels.join(" / ") : "-";
      },
    }]),
    {
      id: "status",
      accessorKey: "status",
      header: t("organization.user.table.status").toString(),
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        return h(
          UBadge,
          {
            color: u.status === "active" ? "success" : "neutral",
            variant: "subtle",
            size: "sm",
          },
          () =>
            u.status === "active"
              ? t("organization.user.form.active")
              : t("organization.user.form.inactive")
        );
      },
    },
    ...(props.readonly
      ? []
      : [{
      id: "actions",
      header: t("organization.user.table.actions").toString(),
      cell: ({ row }: any) => {
        const u = row.original as RowUser;
        return h("div", { class: "flex gap-2" }, [
          h(
            UButton,
            {
              size: "xs",
              variant: "ghost",
              icon: "i-heroicons-pencil-square",
              disabled: props.readonly,
              onClick: () => openEditForm(u),
            },
            () => t("organization.common.edit")
          ),
          h(
            UButton,
            {
              size: "xs",
              color: u.status === "active" ? "warning" : "success",
              variant: "ghost",
              icon:
                u.status === "active"
                  ? "i-heroicons-lock-closed"
                  : "i-heroicons-lock-open",
              disabled: props.readonly,
              onClick: () => toggleUserStatus(u),
            },
            () =>
              u.status === "active"
                ? t("organization.user.disable")
                : t("organization.user.enable")
          ),
          h(
            UButton,
            {
              size: "xs",
              color: "error",
              variant: "ghost",
              icon: "i-heroicons-trash",
              disabled: props.readonly,
              onClick: () => deleteUser(u.id),
            },
            () => t("organization.common.delete")
          ),
        ]);
      },
    }]),
  ];
});

// 手机号脱敏函数
function maskPhone(phone: string): string {
  if (!phone) return "";
  if (phone.length <= 7) return phone;
  return phone.slice(0, 3) + "****" + phone.slice(-4);
}

function isSyntheticWecomEmail(email?: string): boolean {
  const value = String(email || "").trim().toLowerCase();
  return value.endsWith(".wecom.local");
}

// 转换API数据为组件需要的格式
function transformUserData(memberWithProfile: MemberWithProfile): RowUser {
  const { Member, User, DeptIDs, RoleCodes } = memberWithProfile;
  const mergedMeta = { ...User.meta, ...Member.meta } as Record<string, any>;
  const rawEmail = String(User.email || "").trim();
  const provider = String(federatedMemberSource.value[Member.id] || "").toLowerCase();
  const departmentIDs = Array.isArray(DeptIDs)
    ? DeptIDs.map((id) => Number(id || 0)).filter((id) => Number.isFinite(id) && id > 0)
    : [];
  const departmentLabels = departmentIDs
    .map((id) => departmentNameMap.value.get(id) || `#${id}`)
    .filter(Boolean);
  return {
    id: Member.id, // 使用Member的ID作为主要ID
    userId: User.id, // 保存User的ID以备后用
    sourceType: provider ? "federated" : "local",
    name: Member.display_name || User.display_name,
    username: Member.username,
    email: props.readonly ? "" : (isSyntheticWecomEmail(rawEmail) ? "" : rawEmail),
    phone: props.readonly ? "" : (User.phone || ""),
    department: departmentLabels.join(" / "),
    departmentIds: departmentIDs,
    roles: props.readonly ? [] : (Array.isArray(RoleCodes) ? RoleCodes : []),
    status: Member.status === 1 ? "active" : "inactive",
    avatar:
      Member.avatar_url ||
      User.avatar_url ||
      `https://i.pravatar.cc/150?u=${encodeURIComponent(User.email || Member.display_name)}`,
    meta: mergedMeta, // 合并User和Member的meta
  };
}

async function loadRoleOptions() {
  if (!props.tenantUuid) {
    roleOptions.value = [];
    return;
  }
  try {
    const response = await iamService.listRoles({ tenantUuid: props.tenantUuid });
    const payload = (response as any)?.data ?? response ?? {};
    const items = payload?.data?.items ?? payload?.items ?? payload?.data ?? [];
    roleOptions.value = Array.isArray(items)
      ? items
          .map((item: any) => ({
            label: String(item?.name || item?.code || "").trim(),
            value: Number(item?.id || 0),
            code: String(item?.code || "").trim(),
          }))
          .filter((item: any) => item.value > 0 && item.code)
      : [];
  } catch (error) {
    console.error("加载角色数据失败:", error);
    roleOptions.value = [];
  }
}

// 加载用户数据
async function loadUsers() {
  if (!props.tenantUuid) {
    return;
  }
  try {
    loading.value = true;
    const params: any = {
      tenant_uuid: props.tenantUuid,
      page: pagination.page,
      page_size: pagination.pageSize,
      status: filters.status
        ? filters.status === "active"
          ? 1
          : 0
        : undefined, // 不传status则显示所有状态
    };

    // 添加搜索参数
    if (searchQuery.value.trim()) {
      params.q = searchQuery.value.trim(); // 后端使用q参数
    }

    const [response, bindingResp] = await Promise.all([
      userService.getUsers(params),
      iamService.listFederatedBindings({
        tenantUuid: props.tenantUuid,
      }),
    ]);

    const bindingPayload = (bindingResp as any)?.data ?? bindingResp ?? {};
    const bindingItems =
      bindingPayload?.data?.items ??
      bindingPayload?.items ??
      bindingPayload?.data ??
      [];
    const sourceMap: Record<number, string> = {};
    if (Array.isArray(bindingItems)) {
      for (const item of bindingItems) {
        const memberID = Number((item as any)?.member_id || 0);
        const provider = String((item as any)?.provider || "").trim();
        if (memberID > 0 && provider) {
          sourceMap[memberID] = provider;
        }
      }
    }
    federatedMemberSource.value = sourceMap;

    if (response.data) {
      users.value = response.data.items.map(transformUserData);
      pagination.total = response.data.pagination.total;
      pagination.totalPages = response.data.pagination.pages;
    }
  } catch (error) {
    console.error("加载用户数据失败:", error);
  } finally {
    loading.value = false;
  }
}

// 初始化数据
onMounted(async () => {
  // 初始化部门数据
  try {
    await departmentStore.fetchTree();
  } catch (error) {
    console.error("加载部门数据失败:", error);
  }

  // 加载角色和用户数据
  await loadRoleOptions();
  await loadUsers();
});
</script>

<template>
  <div>
    <!-- 顶部：导入导出 + 新增 -->
    <div class="flex justify-between items-center mb-6">
      <div>
        <h2 class="text-xl font-semibold text-gray-900 dark:text-white">
          {{ $t("organization.user.title") }}
        </h2>
        <p class="text-sm text-gray-500 mt-1 dark:text-slate-200">
          {{ $t("organization.user.description") }}
        </p>
        <p v-if="props.readonly" class="text-xs text-amber-600 mt-1 dark:text-amber-300">当前为只读视图，隐藏联系方式与角色详情</p>
      </div>
      <div class="flex space-x-2">
        <UDropdownMenu :items="importExportItems">
          <UButton
            color="neutral"
            variant="outline"
            icon="i-heroicons-arrow-up-tray"
          >
            {{ $t("organization.user.importExport") }}
          </UButton>
        </UDropdownMenu>
        <UButton v-if="!props.readonly" color="primary" icon="i-heroicons-plus" @click="openAddForm">
          {{ $t("organization.user.add") }}
        </UButton>
      </div>
    </div>

    <!-- 搜索与筛选（与你现有一致） -->
    <div class="mb-4 flex flex-wrap gap-2">
      <UButton
        :variant="userCategory === 'all' ? 'solid' : 'soft'"
        :color="userCategory === 'all' ? 'primary' : 'neutral'"
        size="sm"
        @click="applyCategory('all')"
      >
        全部 ({{ categoryStats.all }})
      </UButton>
      <UButton
        :variant="userCategory === 'active' ? 'solid' : 'soft'"
        :color="userCategory === 'active' ? 'primary' : 'neutral'"
        size="sm"
        @click="applyCategory('active')"
      >
        启用 ({{ categoryStats.active }})
      </UButton>
      <UButton
        :variant="userCategory === 'inactive' ? 'solid' : 'soft'"
        :color="userCategory === 'inactive' ? 'primary' : 'neutral'"
        size="sm"
        @click="applyCategory('inactive')"
      >
        停用 ({{ categoryStats.inactive }})
      </UButton>
    </div>

    <div class="mb-6 rounded-lg bg-white p-4 shadow-sm dark:bg-slate-950/70 dark:border dark:border-slate-800/60">
      <div class="flex flex-wrap gap-4 items-end">
        <div class="flex-grow min-w-[200px]">
          <UInput
            v-model="searchQuery"
            icon="i-heroicons-magnifying-glass"
        :placeholder="$t('organization.user.search')"
        />
      </div>
        <UFormField :label="$t('organization.user.form.department')">
          <SelectTree
            v-model="filters.department"
            :items="departmentTreeItems"
            :placeholder="$t('organization.user.form.selectDepartment')"
            searchable
            clearable
            class="w-full sm:min-w-[12rem]"
          />
        </UFormField>
        <UFormField v-if="!props.readonly" :label="$t('organization.user.form.role')">
          <USelect
            v-model="filters.role"
            :items="roleFilterOptions"
            class="w-full sm:min-w-[12rem]"
            :placeholder="$t('organization.user.form.selectRole')"
            option-attribute="label"
          />
        </UFormField>
        <UFormField :label="$t('organization.user.form.status')" class="mb-0">
          <USelect
            v-model="filters.status"
            :items="[
              { label: $t('organization.user.filter.allStatus'), value: null },
              { label: $t('organization.user.filter.active'), value: 'active' },
              {
                label: $t('organization.user.filter.inactive'),
                value: 'inactive',
              },
            ]"
            class="w-full sm:w-40"
            :placeholder="$t('organization.user.filter.allStatus')"
          />
        </UFormField>
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-heroicons-arrow-path"
          @click="resetFilters"
        >
          {{ $t("organization.user.filter.reset") }}
        </UButton>
      </div>
    </div>

    <!-- 表格 + 分页 -->
    <div
      class="rounded-lg bg-white shadow-sm dark:bg-slate-950/70 dark:border dark:border-slate-800/60"
    >
      <UTable
        :data="paginatedUsers"
        :columns="columns"
        :loading="loading"
        :empty-state="{
          icon: 'i-heroicons-circle-stack-20-solid',
          label: '暂无用户数据',
          description: '当前没有找到任何用户信息',
        }"
      />
      <div
        v-if="normalizedTotalPages > 1"
        class="px-6 py-4 border-t border-gray-200 dark:border-slate-800/60 flex justify-between items-center"
      >
        <div class="text-sm text-gray-600 dark:text-slate-200 flex items-center gap-3">
          <span>
            第 {{ pagination.page }} / {{ normalizedTotalPages }} 页，共 {{ pagination.total }} 条
          </span>
          <USelect
            :model-value="String(pagination.pageSize)"
            :items="[
              { label: '10 / 页', value: '10' },
              { label: '20 / 页', value: '20' },
              { label: '50 / 页', value: '50' },
            ]"
            class="w-28"
            @update:model-value="(value) => changePageSize(Number(value))"
          />
        </div>
        <div class="flex gap-2 items-center">
          <UButton
            :disabled="!hasPrevPage || loading"
            variant="outline"
            size="sm"
            icon="i-heroicons-chevron-left"
            @click="changePage(pagination.page - 1)"
            >上一页</UButton
          >
          <UButton
            v-for="p in pageNumbers"
            :key="p"
            :variant="p === pagination.page ? 'solid' : 'outline'"
            size="sm"
            :disabled="loading"
            @click="changePage(p)"
          >
            {{ p }}
          </UButton>
          <UButton
            :disabled="!hasNextPage || loading"
            variant="outline"
            size="sm"
            icon="i-heroicons-chevron-right"
            @click="changePage(pagination.page + 1)"
            >下一页</UButton
          >
        </div>
      </div>
    </div>

    <!-- 表单弹窗（新增/编辑） -->
    <UModal
      v-if="!props.readonly"
      v-model:open="showForm"
      :title="
        isEditing
          ? t('organization.user.form.editUser')
          : t('organization.user.form.addUser')
      "
      :description="
        isEditing
          ? t('organization.user.form.editUserDesc')
          : t('organization.user.form.addUserDesc')
      "
    >
      <template #content>
        <div class="py-8 px-8">
          <form
            @submit.prevent="saveUser"
            class="grid grid-cols-1 md:grid-cols-2 gap-4"
          >
            <UFormField :label="$t('organization.user.form.name')" required>
              <UInput v-model="userForm.name" />
            </UFormField>
            <UFormField
              :label="$t('organization.user.form.username')"
              :required="!isEditing"
            >
              <UInput
                v-model="userForm.username"
                :placeholder="isEditing ? '编辑时可选' : '必填，用于租户内登录'"
              />
            </UFormField>
            <UFormField
              :label="$t('organization.user.form.email')"
              required
              class="md:col-span-2"
            >
              <UInput v-model="userForm.email" type="email" />
            </UFormField>
            <UFormField :label="$t('organization.user.form.phone')">
              <UInput
                v-model="userForm.phone"
                type="tel"
                :placeholder="$t('organization.user.form.phonePlaceholder')"
              />
            </UFormField>
            <UFormField :label="$t('organization.user.form.department')" class="md:col-span-2">
              <USelectMenu
                v-model="userForm.departmentIds"
                :items="departmentFlatOptions"
                value-key="value"
                searchable
                multiple
                class="w-full"
                :placeholder="$t('organization.user.form.selectDepartment')"
                @update:model-value="(values:any) => { userForm.departmentIds = (values || []).map((v:any) => Number(v)); userForm.departmentId = userForm.departmentIds[0] || null; }"
              />
            </UFormField>
            <UFormField :label="$t('organization.user.form.role')" class="md:col-span-2">
              <USelectMenu
                v-model="userForm.roleIds"
                :items="roleOptions"
                value-key="value"
                searchable
                multiple
                class="w-full"
                :placeholder="$t('organization.user.form.selectRole')"
                @update:model-value="(values:any) => { userForm.roleIds = (values || []).map((v:any) => Number(v)); }"
              />
            </UFormField>
            <UFormField
              :label="$t('organization.user.form.password')"
              :required="!isEditing"
              ><UInput v-model="userForm.password" type="password"
            /></UFormField>
            <UFormField
              :label="$t('organization.user.form.confirmPassword')"
              :required="!isEditing"
              ><UInput v-model="userForm.confirmPassword" type="password"
            /></UFormField>
            <div class="md:col-span-2 flex justify-end gap-3 mt-2">
              <UButton
                color="neutral"
                variant="outline"
                @click="showForm = false"
                >{{ $t("organization.common.cancel") }}</UButton
              >
              <UButton type="submit" color="primary">{{
                $t("organization.common.save")
              }}</UButton>
            </div>
          </form>
        </div>
      </template>
    </UModal>
  </div>
</template>
