<template>
  <UContainer class="py-10 space-y-8">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ $t("templates.crud.title") }}
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          {{ $t("templates.crud.description") }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <UButton
          icon="i-heroicons-plus"
          color="primary"
          :disabled="isDelegatedReadOnly"
          @click="startCreate"
        >
          {{ $t("templates.crud.create") }}
        </UButton>
      </div>
    </div>

    <UAlert
      v-if="isDelegatedReadOnly"
      color="info"
      variant="soft"
      icon="i-heroicons-information-circle"
    >
      <template #title>{{ $t("templates.crud.readonlyTitle") }}</template>
      <template #description>
        {{ $t("templates.crud.readonlyDescription") }}
      </template>
    </UAlert>

    <TemplateFormModal
      v-if="showFormModal"
      v-model="showFormModal"
      :title="modalTitle"
      :submit-label="submitLabel"
      :initial-value="formSnapshot"
      :loading="saving"
      @submit="handleSubmit"
    />

    <UCard>
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span class="font-medium text-gray-900 dark:text-gray-100">{{ $t("templates.crud.listTitle") }}</span>
          <div class="flex items-center gap-3">
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ $t("templates.crud.pagination.total", { total: pagination.total }) }}
            </span>
            <USelect
              v-model="pagination.pageSize"
              :items="pageSizeOptions"
              class="w-32"
            />
          </div>
        </div>
      </template>
      <UTable
        :columns="tableColumns"
        :data="templates"
        :loading="loading"
        :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
      >
        <!-- 注意：v3 是 -cell，不是 -data；row.original 才是你的对象 -->
        <template #description-header="{ column }">
          <span class="block w-64">
            {{ column.columnDef.header }}
          </span>
        </template>
        <template #content-header="{ column }">
          <span class="block w-80">
            {{ column.columnDef.header }}
          </span>
        </template>
        <template #description-cell="{ row }">
          <div class="description-cell">
            {{ row.original.description }}
          </div>
        </template>
        <template #content-cell="{ row }">
          <div class="content-cell">
            {{ row.original.content }}
          </div>
        </template>
        <template #actions-cell="{ row }">
          <div class="flex gap-2">
            <UButton
              size="xs"
              variant="soft"
              icon="i-heroicons-pencil"
              :disabled="isDelegatedReadOnly"
              @click="startEdit(row.original)"
            >
              {{ $t('common.edit') }}
            </UButton>
            <UButton
              size="xs"
              variant="soft"
              color="error"
              icon="i-heroicons-trash"
              :disabled="isDelegatedReadOnly"
              @click="confirmDelete(row.original)"
            >
              {{ $t('common.delete') }}
            </UButton>
          </div>
        </template>
      </UTable>

      <div class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-200 px-4 py-3 dark:border-gray-700">
        <span class="text-sm text-gray-500 dark:text-gray-400">
          {{ paginationSummary }}
        </span>
        <UPagination
          v-if="pagination.totalPages > 1"
          v-model:page="pagination.page"
          :total="pagination.total"
          :items-per-page="pagination.pageSize"
          show-edges
        />
      </div>
    </UCard>

    <ConfirmDialog
      v-model="deleteDialog"
      :title="$t('templates.crud.deleteTitle')"
      :description="$t('templates.crud.deleteDescription')"
      :message="$t('templates.crud.deleteConfirm', { name: selectedTemplate?.name || '' })"
      confirm-color="error"
      :confirm-text="$t('common.delete')"
      :loading="deleting"
      @confirm="performDelete"
      @cancel="handleDeleteCancel"
    />

    <ToastAlert
      v-model="toast.visible"
      :title="toast.title"
      :message="toast.message"
      :color="toast.color"
      :duration="toast.duration"
    />
  </UContainer>
</template>

<script setup lang="ts">
import ConfirmDialog from "~/components/ConfirmDialog.vue"
import ToastAlert from "~/components/ToastAlert.vue"
import { useTemplateApi } from "~/composables/api/useTemplate"
import type { Template } from "~/composables/api/useTemplate"
import TemplateFormModal from "~/components/templates/TemplateFormModal.vue"
import { nextTick } from "vue"
import { useI18n } from "vue-i18n"
import { storeToRefs } from "pinia"
import { useUserStore } from "~/stores/user"

type TemplateFormState = {
  name: string
  description: string
  content: string
}

const {
  listTemplates,
  createTemplate: createTemplateApi,
  updateTemplate: updateTemplateApi,
  deleteTemplate: deleteTemplateApi,
} = useTemplateApi()

const templates = ref<Template[]>([])
const loading = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const showFormModal = ref(false)
const deleteDialog = ref(false)
const deleting = ref(false)
const selectedTemplate = ref<Template | null>(null)
const pageSizeOptions = [
  { label: "10", value: 10 },
  { label: "20", value: 20 },
  { label: "50", value: 50 },
]
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0,
  totalPages: 0,
})

type ToastColor = "primary" | "secondary" | "success" | "info" | "warning" | "error" | "neutral"

const toast = reactive({
  visible: false,
  title: "",
  message: "",
  color: "primary" as ToastColor,
  duration: 3000,
})

const { t } = useI18n()

const auth = useAuth()
const userStore = useUserStore()
const { isRoot, isCurrentTenantAdmin, canReadTemplates, canWriteTemplates } = storeToRefs(userStore)
const isDelegatedReadOnly = computed(() => {
  const delegated = auth.delegatedIAM?.value ?? false
  if (!delegated) {
    return false
  }
  if (Boolean(canWriteTemplates.value)) {
    return false
  }
  return !(Boolean(isRoot.value) || Boolean(isCurrentTenantAdmin.value))
})
const canReadTemplateList = computed(() => {
  const delegated = auth.delegatedIAM?.value ?? false
  if (!delegated) {
    return true
  }
  return Boolean(isRoot.value) || Boolean(isCurrentTenantAdmin.value) || Boolean(canReadTemplates.value)
})

const tableColumns = computed(() => [
  { accessorKey: 'name', header: t('templates.crud.fields.name') },
  { accessorKey: 'description', header: t('templates.crud.fields.description') },
  { accessorKey: 'content', header: t('templates.crud.fields.content') },
  { id: 'actions', header: '' },
] satisfies any)

const defaultFormValue = (): TemplateFormState => ({
  name: "",
  description: "",
  content: "",
})

const form = reactive<TemplateFormState>(defaultFormValue())

const makeLogHandlers = (action: string, context: Record<string, any> = {}) => ({
  onRequest({ request: _request, options: _options }: any) {
    // console.debug(`[templates/crud] ${action} request`, {
    //   baseURL: templateApiBase,
    //   request,
    //   options,
    //   context,
    // })
  },
  onResponse({ response: _response }: any) {
    // console.debug(`[templates/crud] ${action} response`, {
    //   status: response.status,
    //   data: response._data,
    //   headers: typeof response.headers?.get === "function"
    //     ? {
    //         "x-request-id": response.headers.get("x-request-id"),
    //       }
    //     : undefined,
    //   context,
    // })
  },
  onResponseError({ response }: any) {
    console.error(`[templates/crud] ${action} response error`, {
      status: response?.status,
      data: response?._data,
      context,
    })
  },
})

const modalTitle = computed(() =>
  editingId.value ? t("templates.crud.actions.update") : t("templates.crud.create")
)

const submitLabel = computed(() =>
  editingId.value ? t("templates.crud.actions.update") : t("templates.crud.actions.save")
)

const formSnapshot = computed(() => ({
  name: form.name,
  description: form.description,
  content: form.content,
}))

const paginationSummary = computed(() => {
  if (pagination.total <= 0) {
    return t("templates.crud.pagination.empty")
  }
  const start = (pagination.page - 1) * pagination.pageSize + 1
  const end = Math.min(pagination.page * pagination.pageSize, pagination.total)
  return t("templates.crud.pagination.range", {
    start,
    end,
    total: pagination.total,
    page: pagination.page,
    totalPages: Math.max(pagination.totalPages, 1),
  })
})

const fetchTemplates = async () => {
  loading.value = true
  try {
    const query = { page: pagination.page, page_size: pagination.pageSize }
    // console.debug('[templates/crud] fetching templates', {
    //   baseURL: templateApiBase,
    //   path: 'templates',
    //   query,
    // })

    const res = await listTemplates(query.page, query.page_size, "", makeLogHandlers("templates:list", { query }))
    if (res?.success && res.data && Array.isArray(res.data.list)) {
      const total = Number(res.data.total) || 0
      const pageSize = Number(res.data.page_size) || query.page_size
      const totalPages = Number(res.data.total_pages) || (total > 0 ? Math.ceil(total / pageSize) : 0)
      templates.value = res.data.list
      pagination.total = total
      pagination.pageSize = pageSize
      pagination.totalPages = totalPages
      pagination.page = Math.min(Math.max(Number(res.data.page) || query.page, 1), Math.max(totalPages, 1))
      // console.debug('[templates/crud] templates loaded', {
      //   count: templates.value.length,
      // })
    } else {
      templates.value = []
      pagination.total = 0
      pagination.totalPages = 0
      console.warn('[templates/crud] templates response unexpected', res)
    }
  } catch (error) {
    console.error("[templates/crud] Failed to load templates", error)
    templates.value = []
    pagination.total = 0
    pagination.totalPages = 0
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  editingId.value = null
  Object.assign(form, defaultFormValue())
}

const openFormModal = () => {
  showFormModal.value = true
}

const closeFormModal = () => {
  showFormModal.value = false
}

const ensureWritable = () => {
  if (!isDelegatedReadOnly.value) {
    return true
  }
  showToast({
    title: t("templates.crud.readonlyTitle"),
    message: t("templates.crud.readonlyToast"),
    color: "warning",
    duration: 4500,
  })
  return false
}

const startCreate = () => {
  if (!ensureWritable()) return
  resetForm()
  openFormModal()
}

const startEdit = (tpl: Template) => {
  if (!ensureWritable()) return
  editingId.value = tpl.id
  Object.assign(form, {
    name: tpl.name,
    description: tpl.description,
    content: tpl.content,
  })
  openFormModal()
}

const handleSubmit = async (payload: { name: string; description: string; content: string }) => {
  if (isDelegatedReadOnly.value) {
    ensureWritable()
    return
  }
  if (!payload.name || !payload.description || !payload.content) {
    return
  }
  saving.value = true
  const isUpdate = Boolean(editingId.value)
  try {
    if (editingId.value) {
      const res = await updateTemplateApi(
        editingId.value,
        payload,
        makeLogHandlers("templates:update", { id: editingId.value, payload })
      )
      if (!res?.success) {
        throw new Error(res?.message || "Update template failed")
      }
    } else {
      const res = await createTemplateApi(
        payload,
        makeLogHandlers("templates:create", { payload })
      )
      if (!res?.success) {
        throw new Error(res?.message || "Create template failed")
      }
    }
    if (!isUpdate) {
      pagination.page = 1
    }
    await fetchTemplates()
    closeFormModal()
    resetForm()
    showToast({
      title: isUpdate ? t("templates.crud.actions.update") : t("templates.crud.create"),
      message: isUpdate ? t("message.saveSuccess") : t("message.templateCreated"),
      color: "success",
    })
  } catch (error: any) {
    console.error("[templates/crud] Failed to save template", error)
    showToast({
      title: t("message.error"),
      message: error?.message || t("message.error"),
      color: "error",
      duration: 5000,
    })
  } finally {
    saving.value = false
  }
}

const confirmDelete = (tpl: Template) => {
  if (!ensureWritable()) return
  selectedTemplate.value = tpl
  deleteDialog.value = true
}

const performDelete = async () => {
  if (!selectedTemplate.value || deleting.value) return
  if (isDelegatedReadOnly.value) {
    ensureWritable()
    return
  }
  deleting.value = true
  try {
    const res = await deleteTemplateApi(
      selectedTemplate.value.id,
      makeLogHandlers("templates:delete", { id: selectedTemplate.value.id })
    )
    if (!res?.success) {
      throw new Error(res?.message || "Delete template failed")
    }
    const shouldMoveToPreviousPage = templates.value.length === 1 && pagination.page > 1
    if (shouldMoveToPreviousPage) {
      pagination.page -= 1
      deleteDialog.value = false
      showToast({
        title: t("templates.crud.deleteTitle"),
        message: t("message.deleteSuccess"),
        color: "success",
      })
      return
    }
    await fetchTemplates()
    deleteDialog.value = false
    showToast({
      title: t("templates.crud.deleteTitle"),
      message: t("message.deleteSuccess"),
      color: "success",
    })
  } catch (error: any) {
    console.error("[templates/crud] Failed to delete template", error)
    showToast({
      title: t("message.error"),
      message: error?.message || t("message.error"),
      color: "error",
      duration: 5000,
    })
  } finally {
    deleting.value = false
  }
}

const handleDeleteCancel = () => {
  deleteDialog.value = false
}

watch(deleteDialog, (isOpen) => {
  if (!isOpen) {
    selectedTemplate.value = null
    deleting.value = false
  }
})

watch(isDelegatedReadOnly, (readonlyMode) => {
  if (!readonlyMode) {
    return
  }
  showFormModal.value = false
  deleteDialog.value = false
})

watch(
  () => pagination.page,
  async () => {
    if (canReadTemplateList.value) {
      await fetchTemplates()
    }
  }
)

watch(
  () => pagination.pageSize,
  async () => {
    if (pagination.page !== 1) {
      pagination.page = 1
      return
    }
    if (canReadTemplateList.value) {
      await fetchTemplates()
    }
  }
)

const normalizeToString = (value?: string | number | null) => {
  if (value === null || value === undefined) {
    return ""
  }
  return typeof value === "string" ? value : String(value)
}

const showToast = ({
  title,
  message,
  color = "primary",
  duration = 3000,
}: {
  title?: string
  message: string | number
  color?: ToastColor
  duration?: number
}) => {
  const normalizedTitle = normalizeToString(title)
  const normalizedMessage = normalizeToString(message)
  toast.title = normalizedTitle
  toast.message = normalizedMessage
  toast.color = color
  toast.duration = duration
  if (!normalizedTitle && !normalizedMessage) {
    toast.visible = false
    return
  }
  toast.visible = false
  nextTick(() => {
    toast.visible = true
  })
}

onMounted(async () => {
  if (!userStore.context && !userStore.isLoading) {
    try {
      await userStore.fetchUserContext()
    } catch {
      // ignore user context fetch errors
    }
  }
  if (canReadTemplateList.value) {
    await fetchTemplates()
  }
})
</script>

<style scoped>
.description-cell,
.content-cell {
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
}

.description-cell {
  max-width: 16rem;
}

.content-cell {
  max-width: 24rem;
}

</style>
