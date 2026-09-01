<script setup lang="ts">
import {
  ref,
  reactive,
  computed,
  h,
  resolveComponent,
  watch,
  onMounted,
} from "vue";
import { storeToRefs } from "pinia";
import { useI18n } from "#imports";
import {
  useDepartmentService,
  type Department,
  type DepartmentCreateParams,
  type DepartmentUpdateParams,
} from "~/composables/api/services/departmentService";
import { useDepartmentStore } from "~/stores/department";
import { useOneShotAlert } from "~/composables/useOneShotAlert";
import * as v from "valibot";
import type { FormSubmitEvent } from "@nuxt/ui";

import { normalizeApiError } from "~/composables/api/normalizeApiError";
const { notifyOnce, reset } = useOneShotAlert();

// 字段/表单错误位
const formError = ref<string | null>(null);
const fieldErrors = reactive<Record<string, string>>({});
const clearErrors = () => {
  formError.value = null;
  Object.keys(fieldErrors).forEach((k) => delete fieldErrors[k]);
};

/** ================== UI ================== */
const { t, locale } = useI18n();
const UButton = resolveComponent("UButton");
const props = withDefaults(defineProps<{ readonly?: boolean }>(), { readonly: false });

/** ================== 状态 ================== */
const deptService = useDepartmentService();

const activeNodeUUID = ref<string | null>(null);
const activeNode = computed(
  () => flat.value.find((d) => d.uuid === activeNodeUUID.value) || null
);

onMounted(() => {
  fetchTree();
});

const searchQuery = ref("");

/** 分页 */
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0,
  totalPages: 0,
});
const pageSizeOptions = [
  { label: "10", value: 10 },
  { label: "20", value: 20 },
  { label: "50", value: 50 },
  { label: "100", value: 100 },
];

/** 表单 & 弹窗 */
const showForm = ref(false);
const isEditing = ref(false);
const editingDepartmentUUID = ref<string | null>(null);

const originalEditing = ref<
  | (Department & {
      key?: string | null;
      sort?: number | null;
      status?: number | null;
      meta?: any;
    })
  | null
>(null);

const departmentForm = reactive({
  name: "",
  parent_department_uuid: undefined as string | undefined,
  sort: undefined as number | undefined,
});

const resetForm = () => {
  departmentForm.name = "";
  departmentForm.parent_department_uuid = activeNodeUUID.value ?? undefined;
  departmentForm.sort = undefined;
  isEditing.value = false;
  editingDepartmentUUID.value = null;
  originalEditing.value = null;
};

const openAddForm = () => {
  resetForm();
  showForm.value = true;
};

const openEditForm = (dept: Department & any) => {
  const full = flat.value.find((d) => d.uuid === dept.uuid) || dept;
  departmentForm.name = full.name ?? "";
  departmentForm.parent_department_uuid = full.parent_department_uuid;
  departmentForm.sort = full.sort;

  isEditing.value = true;
  editingDepartmentUUID.value = full.uuid;
  originalEditing.value = JSON.parse(JSON.stringify(full));
  showForm.value = true;
};

/** ================== 数据获取 & 工具 ================== */
// 使用全局Store
const deptStore = useDepartmentStore();
const {
  tree: storeTree,
  flat: storeFlat,
  status,
  error,
} = storeToRefs(deptStore);

// 计算属性来兼容现有代码
const tree = computed(() => storeTree.value);
const flat = computed(() => storeFlat.value);
const isLoadingTree = computed(() => status.value === "loading");
const loadError = computed(() => error.value);

const fetchTree = async (options: { force?: boolean } = {}) => {
  try {
    await deptStore.fetchTree(options);

    // 默认选择第一个根节点
    if (!activeNodeUUID.value) {
      const firstRoot = flat.value.find((d) => !d.parent_department_uuid);
      activeNodeUUID.value = firstRoot?.uuid ?? null;
    }
    selectedValue.value = activeNodeUUID.value
      ? [activeNodeUUID.value]
      : [];
  } catch (e: any) {
    console.error("获取部门树失败:", e);
  }
};

/** UTree 数据 */
const treeItems = computed(() => tree.value.map((n) => toTreeItem(n)));

function toTreeItem(n: Department): any {
  const hasChildren = !!(n.children && n.children.length);
  return {
    // ✅ UTree 用 value 作为唯一标识（或 label）
    value: n.uuid,
    label: n.name,
    id: n.uuid,
    hasChildren,
    children: hasChildren ? n.children!.map(toTreeItem) : undefined,
  };
}

const activeNodeActivePath = ref<string[]>([]);

watch(
  () => activeNodeUUID.value,
  (uuid) => {
    activeNodeActivePath.value = uuid ? [uuid] : [];
  },
  { immediate: true }
);

// 新增：选中值 & 展开集合（字符串数组）
const selectedValue = ref<string[]>([]);
const expandedValues = ref<string[]>([]);

watch(selectedValue, (vals) => {
  const first = Array.isArray(vals) && vals.length ? vals[0] : null;
  activeNodeUUID.value = first || null;
  pagination.page = 1;
});

onMounted(fetchTree);

const onFormSubmit = async (
  _e: FormSubmitEvent<v.InferOutput<typeof schema>>
) => {
  reset();
  await saveDepartment(); // 仍然走你已经改造过的 saveDepartment（带 notifyOnce）
};

function flattenDepartments(nodes: Department[], result: Department[] = []) {
  for (const n of nodes) {
    result.push(n);
    if (n.children?.length) flattenDepartments(n.children, result);
  }
  return result;
}

/** 右侧表格：显示当前选中节点的“直接子部门”，并支持搜索+分页 */
const childrenOfActive = computed<Department[]>(() => {
  if (!activeNodeUUID.value) return [];
  const parent = flat.value.find((d) => d.uuid === activeNodeUUID.value);
  return parent?.children ?? [];
});

const filteredDepartments = computed(() => {
  const q = searchQuery.value.trim().toLowerCase();
  const list = childrenOfActive.value;
  return q
    ? list.filter((d) => (d.name ?? "").toLowerCase().includes(q))
    : list;
});

watch(
  [filteredDepartments, () => pagination.pageSize],
  () => {
    pagination.total = filteredDepartments.value.length;
    pagination.totalPages = Math.ceil(pagination.total / pagination.pageSize);
    if (pagination.page > pagination.totalPages)
      pagination.page = pagination.totalPages || 1;
  },
  { immediate: true }
);

const paginatedDepartments = computed(() => {
  const start = (pagination.page - 1) * pagination.pageSize;
  return filteredDepartments.value.slice(start, start + pagination.pageSize);
});

const paginationInfo = computed(() => {
  const start = (pagination.page - 1) * pagination.pageSize + 1;
  const end = Math.min(pagination.page * pagination.pageSize, pagination.total);
  return {
    start: pagination.total > 0 ? start : 0,
    end,
    total: pagination.total,
    page: pagination.page,
    totalPages: pagination.totalPages,
  };
});

const changePage = (page: number) => {
  if (page >= 1 && page <= pagination.totalPages) pagination.page = page;
};
const changePageSize = (v: number | string) => {
  pagination.pageSize = Number(v);
  pagination.page = 1;
};
const hasNextPage = computed(() => pagination.page < pagination.totalPages);
const hasPrevPage = computed(() => pagination.page > 1);

/** 选择上级部门（下拉用） */
const parentOptions = computed(() => {
  const selfUUID = editingDepartmentUUID.value;
  return [
    {
      label: t("organization.department.form.noParent") as string,
      value: null as any,
    },
    ...flat.value
      .filter((d) => d.uuid !== selfUUID)
      .map((d) => ({ label: d.name, value: d.uuid })),
  ];
});

/** ================== CRUD（走后端） ================== */
const deleteDepartment = async (departmentUUID: string) => {
  if (!confirm(t("organization.department.confirmDelete") as string)) return;
  try {
    await deptStore.deleteDepartment(departmentUUID);
    if (activeNodeUUID.value === departmentUUID) {
      const deleted = flat.value.find((d) => d.uuid === departmentUUID);
      activeNodeUUID.value =
        deleted?.parent_department_uuid ?? flat.value.find((d) => !d.parent_department_uuid)?.uuid ?? null;
    }
    notifyOnce("部门删除成功", "", "success", "solid");
  } catch (e: any) {
    const { title, description } = normalizeApiError(e, { meta: "metaText" }); // ✨ 统一解析
    reset(); // ✨ 先重置一次 one-shot
    notifyOnce(title || "删除失败", description, "error", "solid"); // ✨ 弹全局 Alert（会在 Modal 之上）
  }
};

/** ============ TanStack 列定义（右侧“子部门列表”） ============ */
const columns = computed(() => {
  const _ = locale.value; // 语言切换依赖
  return [
    {
      id: "name",
      accessorKey: "name",
      header: t("organization.department.table.name"),
      cell: ({ row }: any) => {
        const d: Department = row.original;
        // 高亮当前选择的节点的直接子项名称
        return h("div", { class: "flex items-center gap-2" }, [
          h("span", d.name),
        ]);
      },
    },
    {
      id: "parent",
      header: t("organization.department.form.parent") || "上级部门",
      cell: ({ row }: any) => {
        const d: Department = row.original;
        const parentName = d.parent_department_uuid
          ? (flat.value.find((x) => x.uuid === d.parent_department_uuid)?.name ?? "-")
          : "-";
        return h("span", parentName);
      },
    },
    {
      id: "sort",
      accessorKey: "sort",
      header: t("organization.department.form.sort") || "排序",
    },
    ...(props.readonly
      ? []
      : [{
      id: "actions",
      header: t("organization.department.table.actions"),
      enableSorting: false,
      cell: ({ row }: any) => {
        const d: Department = row.original;
        return h("div", { class: "flex gap-2" }, [
          h(
            UButton,
            {
              size: "xs",
              variant: "ghost",
              icon: "i-heroicons-chevron-up",
              onClick: async () => {
                const cur = (d.sort ?? 0) - 1;
				await deptService.updateDepartment(d.uuid, { sort: cur });
                await fetchTree({ force: true });
              },
            },
            { default: () => t("organization.common.up") }
          ),
          h(
            UButton,
            {
              size: "xs",
              variant: "ghost",
              icon: "i-heroicons-chevron-down",
              onClick: async () => {
                const cur = (d.sort ?? 0) + 1;
				await deptService.updateDepartment(d.uuid, { sort: cur });
                await fetchTree({ force: true });
              },
            },
            { default: () => t("organization.common.down") }
          ),
          h(
            UButton,
            {
              size: "xs",
              variant: "ghost",
              icon: "i-heroicons-pencil-square",
              onClick: () => openEditForm(d),
            },
            { default: () => t("organization.common.edit") }
          ),
          h(
            UButton,
            {
              size: "xs",
              color: "error",
              variant: "ghost",
              icon: "i-heroicons-trash",
				onClick: () => deleteDepartment(d.uuid),
            },
            { default: () => t("organization.common.delete") }
          ),
        ]);
      },
    }]),
  ];
});

/** UTree 选择 */
function onSelectNode(payload: any) {
  // 统一把各种形态归一到 string[]
  let arr: string[] = [];

  if (Array.isArray(payload)) {
    arr = payload.map(String);
  } else if (payload && typeof payload === "object") {
    if ("id" in payload) arr = [String((payload as any).id)];
    else if ("value" in payload) arr = [String((payload as any).value)];
  } else if (payload != null) {
    arr = [String(payload)];
  }

  selectedValue.value = arr;
  activeNodeUUID.value = arr[0] || null;
  activeNodeActivePath.value = selectedValue.value.slice(0, 1);
  pagination.page = 1;
}

const schema = v.object({
  name: v.pipe(v.string(), v.minLength(1, "部门名称为必填项")),
  // 允许为空/不选
	parent_department_uuid: v.nullable(v.optional(v.string())),
  // 可选；如果填了必须是 >=0 的整数
  sort: v.optional(
    v.pipe(
      v.number(),
      v.integer("排序必须是整数"),
      v.minValue(0, "排序不能为负数")
    )
  ),
  // 仅编辑时会用到；这里统一允许 null/不传
});

const saveDepartment = async () => {
  let success = false; // 标记是否成功
  try {
		if (isEditing.value && editingDepartmentUUID.value) {
      const payload = buildUpdatePayload();

      // 没有任何变化：不调接口，直接提示并返回
      if (Object.keys(payload).length === 0) {
        notifyOnce("无变更", "没有检测到修改内容", "warning", "solid");
        return;
      }

      // 如果你的 deptService 要求 meta 为对象，保持不变；若后端要字符串，可在这里 JSON.stringify
			const ok = await deptService.updateDepartment(editingDepartmentUUID.value, payload);
      success = !!ok;
    } else {
      // 创建
      const created = await deptService.createDepartment({
        name: departmentForm.name,
			parent_department_uuid: departmentForm.parent_department_uuid,
        sort: departmentForm.sort,
      } as DepartmentCreateParams);
      success = !!created;
    }
  } catch (e: any) {
    const { title, description } = normalizeApiError(e, { meta: "metaText" }); // ✨ 统一解析
    reset(); // ✨ 先重置一次 one-shot
    notifyOnce(title || "保存失败", description, "error", "solid"); // ✨ 弹全局 Alert（会在 Modal 之上）
  } finally {
    if (success) {
      reset(); // 允许成功提示出现
      notifyOnce("保存成功", "部门信息已成功保存", "success", "solid");
      showForm.value = false;
      await fetchTree({ force: true });
      resetForm();
    }
  }
};

function buildUpdatePayload(): DepartmentUpdateParams {
  const orig = originalEditing.value || ({} as any);
  const payload: DepartmentUpdateParams = {};

  // name
  if (departmentForm.name !== orig.name) payload.name = departmentForm.name;

	if (departmentForm.parent_department_uuid !== orig.parent_department_uuid) {
		payload.parent_department_uuid = departmentForm.parent_department_uuid;
	}

  // sort
  if (departmentForm.sort !== orig.sort) payload.sort = departmentForm.sort;

  return payload;
}
</script>

<template>
  <div class="p-4">
    <div class="flex justify-between items-center mb-6">
      <div>
        <h2 class="text-xl font-semibold text-gray-800 dark:text-white">
          {{ $t("organization.department.title") }}
        </h2>
        <p class="text-sm text-gray-500 mt-1 dark:text-gray-300">
          {{ $t("organization.department.description") }}
          <span v-if="props.readonly" class="block text-xs text-amber-600 mt-1 dark:text-amber-300">当前为只读视图，仅可浏览组织架构</span>
        </p>
      </div>
      <UButton v-if="!props.readonly" color="primary" icon="i-heroicons-plus" @click="openAddForm">
        {{ $t("organization.department.add") }}
      </UButton>
    </div>

    <!-- 搜索 -->
    <UInput
      v-model="searchQuery"
      icon="i-heroicons-magnifying-glass"
      :placeholder="$t('organization.department.search')"
      class="w-full md:w-80 mb-4"
    />

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <!-- 左侧：组织树 -->
      <UCard class="col-span-1">
        <template #header>
          <div class="flex justify-between items-center">
            <h3 class="text-lg font-medium">
              {{ $t("organization.department.title") }}
            </h3>
            <UButton
              v-if="!props.readonly"
              icon="i-heroicons-plus-circle"
              size="sm"
              color="gray"
              variant="ghost"
              @click="openAddForm"
            >
              {{ $t("organization.department.add") }}
            </UButton>
          </div>
        </template>

        <div v-if="isLoadingTree" class="flex justify-center py-4">
          <UIcon name="i-heroicons-arrow-path" class="animate-spin h-6 w-6" />
        </div>
        <div v-else-if="loadError" class="text-center text-red-500 py-4">
          {{ loadError }}
        </div>
        <div
          v-else-if="treeItems.length === 0"
          class="text-center py-4 text-gray-500 dark:text-gray-400"
        >
          {{ $t("organization.department.empty.title") }}
        </div>
        <div v-else class="department-tree">
          <UTree
            :items="treeItems"
            v-model="selectedValue"
            v-model:expanded="expandedValues"
            expanded-icon="i-heroicons-folder-open"
            collapsed-icon="i-heroicons-folder"
            @update:model-value="onSelectNode"
          >
            <!-- 左侧图标：三态明确区分 -->
            <template #item-leading="{ item, expanded }">
              <UIcon
                :name="
                  item.hasChildren
                    ? expanded
                      ? 'i-heroicons-folder-open'
                      : 'i-heroicons-folder'
                    : 'i-heroicons-document'
                "
                :class="[
                  'h-4 w-4',
                  item.hasChildren
                    ? 'text-amber-500 dark:text-amber-300'
                    : 'text-gray-400 dark:text-slate-200',
                ]"
              />
            </template>

            <template #item-label="{ item }">
              <span>{{ item.label }}</span>
            </template>

            <template v-if="!props.readonly" #item-trailing="{ item }">
              <div class="flex space-x-1">
                <UButton
                  icon="i-heroicons-pencil"
                  size="xs"
                  color="gray"
                  variant="ghost"
                  @click.stop="
                    openEditForm({
					  uuid: String(item.id),
					  name: item.label,
					  parent_department_uuid: flat.find((x) => x.uuid === String(item.id))
						?.parent_department_uuid,
                    } as any)
                  "
                />
                <UButton
                  icon="i-heroicons-trash"
                  size="xs"
                  color="error"
                  variant="ghost"
				@click.stop="deleteDepartment(String(item.id))"
                />
              </div>
            </template>
          </UTree>
        </div>
      </UCard>

      <!-- 右侧：选中节点的直接子部门列表（表格） -->
      <UCard class="col-span-1 md:col-span-2">
        <template #header>
          <div class="flex justify-between items-center">
            <h3 class="text-lg font-medium">
              {{
                activeNode
                  ? activeNode.name + " - " + $t("organization.department.title")
                  : $t("organization.department.title")
              }}
            </h3>
            <div class="flex items-center gap-2">
              <span class="text-sm text-gray-600 dark:text-gray-300">{{
                $t("organization.department.table.name")
              }}</span>
            </div>
          </div>
        </template>

        <div class="bg-white rounded-lg dark:bg-slate-950/70 dark:border dark:border-slate-800/60">
          <UTable
            :data="paginatedDepartments"
            :columns="columns"
			:row-key="(row) => row.uuid"
          />

          <div
            v-if="pagination.totalPages > 1"
            class="px-6 py-4 border-t border-gray-200"
          >
            <div class="flex justify-between items-center">
              <div class="text-sm text-gray-600 dark:text-gray-300">
                第 {{ pagination.page }} /
                {{ pagination.totalPages }} 页；本级子部门
                {{ pagination.total }} 个
              </div>
              <div class="flex items-center gap-4">
                <div class="flex items-center gap-2">
                  <span class="text-sm text-gray-600 dark:text-gray-300">每页：</span>
                  <USelect
                    :model-value="pagination.pageSize"
                    :items="pageSizeOptions"
                    option-attribute="label"
                    value-attribute="value"
                    @update:model-value="changePageSize"
                    class="w-20"
                  />
                </div>
                <div class="flex gap-2">
                  <UButton
                    :disabled="!hasPrevPage"
                    variant="outline"
                    size="sm"
                    icon="i-heroicons-chevron-left"
                    @click="changePage(pagination.page - 1)"
                    >上一页</UButton
                  >
                  <UButton
                    :disabled="!hasNextPage"
                    variant="outline"
                    size="sm"
                    icon="i-heroicons-chevron-right"
                    @click="changePage(pagination.page + 1)"
                    >下一页</UButton
                  >
                </div>
              </div>
            </div>
          </div>
        </div>

        <template #footer>
          <div class="flex justify-between items-center text-sm text-gray-500 dark:text-gray-300">
            <span>
              {{
                t("organization.department.pagination.showing", {
                  start: paginationInfo.start,
                  end: paginationInfo.end,
                  total: paginationInfo.total,
                })
              }}
            </span>
          </div>
        </template>
      </UCard>
    </div>

    <!-- 空状态（针对右侧列表） -->
    <div
      v-if="!isLoadingTree && filteredDepartments.length === 0"
      class="text-center py-10 text-gray-500 dark:text-gray-400"
    >
      {{ $t("organization.department.empty.noResults") }}
    </div>

    <!-- 表单 -->
    <UModal
      v-if="!props.readonly"
      v-model:open="showForm"
      title="department - title"
      description="department - description"
      :ui="{ content: 'sm:max-w-3xl' }"
    >
      <template #content>
        <UCard>
          <template #header>
            <h3 class="text-lg font-medium text-gray-900 dark:text-white">
              {{
                isEditing
                  ? $t("organization.department.edit")
                  : $t("organization.department.add")
              }}
            </h3>
          </template>

          <UForm
            :schema="schema"
            :state="departmentForm"
            @submit="onFormSubmit"
          >
            <div class="grid grid-cols-2 gap-4">
              <UFormField
                name="name"
                :label="$t('organization.department.form.name')"
                required
              >
                <UInput v-model="departmentForm.name" />
              </UFormField>

              <UFormField
				name="parent_department_uuid"
                :label="$t('organization.department.form.parent')"
              >
                <USelect
					:model-value="departmentForm.parent_department_uuid"
                  :items="parentOptions"
                  option-attribute="label"
                  value-attribute="value"
                  :placeholder="$t('organization.department.form.noParent')"
                  @update:model-value="
                    (v) =>
					  (departmentForm.parent_department_uuid =
						v === undefined || v === null || v === ''
						  ? undefined
						  : String(v))
                  "
                />
              </UFormField>

              <UFormField
                name="sort"
                :label="$t('organization.department.form.sort') || '排序'"
              >
                <!-- 用 v-model.number 确保是 number，配合 schema 的 number 校验 -->
                <UInput
                  type="number"
                  :min="0"
                  v-model.number="departmentForm.sort"
                  placeholder="数字越小越靠前"
                />
              </UFormField>

            </div>

            <div class="mt-6 flex justify-end space-x-3">
              <UButton
                color="neutral"
                variant="outline"
                @click="showForm = false"
              >
                {{ $t("organization.common.cancel") }}
              </UButton>
              <UButton type="submit" color="primary">
                {{ $t("organization.common.save") }}
              </UButton>
            </div>
          </UForm>
        </UCard>
      </template>
    </UModal>
  </div>
</template>

<style>
.department-tree :deep(.u-tree-node) {
  padding-top: 0.25rem;
  padding-bottom: 0.25rem;
}
.department-tree :deep(.u-tree-node-content) {
  padding: 0.25rem 0.5rem;
  border-radius: 0.375rem;
}
.department-tree :deep(.u-tree-node-content:hover) {
  background-color: #f3f4f6;
}
.dark .department-tree :deep(.u-tree-node-content:hover) {
  background-color: #1f2937;
}
.department-tree :deep(.u-tree-node-selected) {
  background-color: rgba(var(--color-primary-500), 0.1);
}
.dark .department-tree :deep(.u-tree-node-selected) {
  background-color: rgba(var(--color-primary-500), 0.05);
}
</style>
