<template>
  <UModal :open="true" :dismissible="!saving" :close="!saving" :title="t('metadataGovernance.editTag.title')" :description="t('metadataGovernance.editTag.hint')" :ui="{ content: 'w-full max-w-2xl' }" @update:open="value => { if (!value && !saving) emit('close') }">
    <template #body>
      <form class="space-y-4 text-gray-900 dark:text-gray-100" @submit.prevent="save">
        <div class="grid gap-3 sm:grid-cols-3">
          <UFormField v-for="key in ['resource_type', 'namespace', 'code'] as const" :key="key" :label="t('metadataGovernance.editTag.' + key)">
            <UInput :model-value="tag[key]" readonly class="w-full" />
          </UFormField>
        </div>
        <UFormField :label="t('metadataGovernance.editTag.language')">
          <USelect v-model="language" :items="languages" class="w-full" :disabled="saving" />
        </UFormField>
        <UFormField :label="t('metadataGovernance.editTag.name')" :required="language === 'zh-CN'">
          <UInput v-model="names[language]" class="w-full" :disabled="saving" />
        </UFormField>
        <UFormField :label="t('metadataGovernance.editTag.description')">
          <UTextarea v-model="descriptions[language]" class="w-full" :disabled="saving" />
        </UFormField>
        <div class="grid gap-3 sm:grid-cols-2">
          <UFormField :label="t('metadataGovernance.editTag.color')"><UInput v-model="color" class="w-full" :disabled="saving" /></UFormField>
          <UFormField :label="t('metadataGovernance.columns.status')"><USelect v-model="status" :items="statuses" class="w-full" :disabled="saving" /></UFormField>
        </div>
        <UAlert v-if="error" color="error" variant="soft" :title="error" />
        <div class="flex justify-end gap-2">
          <UButton variant="soft" :disabled="saving" @click="emit('close')">{{ t('metadataGovernance.editTag.cancel') }}</UButton>
          <UButton type="submit" :loading="saving">{{ t('metadataGovernance.editTag.save') }}</UButton>
        </div>
      </form>
    </template>
  </UModal>
</template>
<script setup lang="ts">
import { useMetadataGovernanceApi, type MetadataTag } from '~/composables/api/useMetadataGovernance'
const props = defineProps<{ tag: MetadataTag; route: 'local' | 'delegated' }>()
const emit = defineEmits<{ close: []; saved: [] }>()
const { t, te } = useI18n()
const config = useRuntimeConfig()
const api = useMetadataGovernanceApi(() => props.route)
const names = ref({ ...props.tag.label_i18n })
const descriptions = ref({ ...props.tag.description_i18n })
const color = ref(props.tag.color || '')
const status = ref(props.tag.status)
const language = ref('zh-CN')
const saving = ref(false)
const error = ref('')
const languages = computed(() => {
  const configured = String(config.public.availableLanguages || 'zh,en,ja,ko').split(',').map(value => value.trim() === 'zh' ? 'zh-CN' : value.trim())
  return [...new Set(['zh-CN', ...configured, ...Object.keys(names.value), ...Object.keys(descriptions.value)])].filter(Boolean).map(value => {
    const key = 'metadataGovernance.form.locale.' + value.replace('-', '_')
    return { value, label: te(key) ? t(key) : value }
  })
})
const statuses = computed(() => [
  { value: props.route === 'delegated' ? 'enabled' : 'active', label: t('status.active') },
  { value: props.route === 'delegated' ? 'disabled' : 'inactive', label: t('status.inactive') },
  ...(props.route === 'delegated' ? [{ value: 'archived', label: t('metadataGovernance.editTag.archived') }] : []),
])
async function save() {
  if (saving.value) return
  error.value = ''
  if (!names.value['zh-CN']?.trim()) {
    language.value = 'zh-CN'
    error.value = t('metadataGovernance.editTag.requiredName')
    return
  }
  if (!statuses.value.some(item => item.value === status.value)) {
    error.value = t('metadataGovernance.editTag.invalidStatus')
    return
  }
  saving.value = true
  try {
    await api.updateTag(props.tag.uuid, {
      label_i18n: { ...names.value },
      description_i18n: { ...descriptions.value },
      color: color.value,
      status: status.value,
    })
    emit('saved')
  } catch {
    error.value = t('metadataGovernance.editTag.failed')
  } finally { saving.value = false }
}
</script>
