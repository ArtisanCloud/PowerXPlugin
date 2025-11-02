<template>
  <UContainer class="py-10 space-y-8">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
          {{ $t('templates.crud.title') }}
        </h1>
        <p class="text-gray-600 dark:text-gray-300">
          {{ $t('templates.crud.description') }}
        </p>
      </div>
      <UButton icon="i-heroicons-plus" color="primary" @click="startCreate">
        {{ $t('templates.crud.create') }}
      </UButton>
    </div>

    <TemplateFormModal
      v-model="showFormModal"
      :title="modalTitle"
      :submit-label="submitLabel"
      :initial-value="formSnapshot"
      :loading="saving"
      @submit="handleSubmit"
    />

    <UCard>
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium">{{ $t('templates.crud.listTitle') }}</span>
          <UBadge variant="soft" color="primary">{{ templates.length }}</UBadge>
        </div>
      </template>

      <UTable
        :columns="columns"
        :data="templates"
        :loading="loading"
        :ui="{ table: 'min-w-full table-fixed divide-y divide-gray-200 dark:divide-gray-700' }"
      >
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
            <UButton size="xs" variant="soft" icon="i-heroicons-pencil" @click="startEdit(row.original)">
              {{ $t('common.edit') }}
            </UButton>
            <UButton size="xs" variant="soft" color="error" icon="i-heroicons-trash" @click="confirmDelete(row.original)">
              {{ $t('common.delete') }}
            </UButton>
          </div>
        </template>
      </UTable>
    </UCard>

    <ConfirmDialog
      v-model="deleteDialog"
      :title="$t('templates.crud.deleteTitle')"
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
      :duration="toast.duration"
    />
  </UContainer>
</template>

<script setup lang="ts">
import TemplateFormModal from '~/components/templates/TemplateFormModal.vue'
import ConfirmDialog from '~/components/ConfirmDialog.vue'
import ToastAlert from '~/components/ToastAlert.vue'
import { useTemplateApi, type Template } from '~/composables/api/useTemplate'

const columns = [
  { accessorKey: 'name', header: 'Name' },
  { accessorKey: 'description', header: 'Description' },
  { accessorKey: 'content', header: 'Content' },
  { id: 'actions', header: '' }
] as const

const api = useTemplateApi()
const templates = ref<Template[]>([])
const loading = ref(false)
const saving = ref(false)
const deleting = ref(false)
const deleteDialog = ref(false)
const showFormModal = ref(false)
const editingId = ref<number | null>(null)
const selectedTemplate = ref<Template | null>(null)

const toast = reactive({
  visible: false,
  title: '',
  message: '',
  duration: 3000
})

type TemplateFormState = {
  name: string
  description: string
  content: string
}

const defaultFormValue = (): TemplateFormState => ({
  name: '',
  description: '',
  content: ''
})

const formSnapshot = computed<TemplateFormState>(() => {
  if (!showFormModal.value) return defaultFormValue()
  if (editingId.value && selectedTemplate.value) {
    return {
      name: selectedTemplate.value.name,
      description: selectedTemplate.value.description,
      content: selectedTemplate.value.content
    }
  }
  return defaultFormValue()
})

const modalTitle = computed(() =>
  editingId.value ? $t('templates.crud.actions.update') : $t('templates.crud.create')
)

const submitLabel = computed(() =>
  editingId.value ? $t('templates.crud.actions.update') : $t('templates.crud.actions.save')
)

onMounted(fetchTemplates)

async function fetchTemplates() {
  loading.value = true
  try {
    const { data } = await api.listTemplates()
    templates.value = data.list
  } finally {
    loading.value = false
  }
}

function startCreate() {
  editingId.value = null
  selectedTemplate.value = null
  showFormModal.value = true
}

function startEdit(template: Template) {
  editingId.value = template.id
  selectedTemplate.value = template
  showFormModal.value = true
}

async function handleSubmit(payload: TemplateFormState) {
  saving.value = true
  try {
    if (editingId.value) {
      await api.updateTemplate(editingId.value, payload)
      showToast($t('templates.crud.toast.updated'))
    } else {
      await api.createTemplate(payload)
      showToast($t('templates.crud.toast.created'))
    }
    await fetchTemplates()
    showFormModal.value = false
  } finally {
    saving.value = false
  }
}

function confirmDelete(template: Template) {
  selectedTemplate.value = template
  deleteDialog.value = true
}

async function performDelete() {
  if (!selectedTemplate.value) return
  deleting.value = true
  try {
    await api.deleteTemplate(selectedTemplate.value.id)
    showToast($t('templates.crud.toast.deleted'))
    await fetchTemplates()
  } finally {
    deleting.value = false
    deleteDialog.value = false
  }
}

function handleDeleteCancel() {
  selectedTemplate.value = null
}

function showToast(message: string) {
  Object.assign(toast, {
    visible: true,
    title: $t('templates.crud.toast.title'),
    message
  })
}
</script>

<style scoped>
.description-cell,
.content-cell {
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
</style>
