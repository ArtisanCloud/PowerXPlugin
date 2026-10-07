<template>
  <main class="mx-auto w-full max-w-5xl space-y-6 px-5 py-6 lg:px-8">
    <header class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-[#172536]">
      <p class="text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.overview.eyebrow') }}</p>
      <h1 class="mt-1 text-2xl font-semibold text-gray-900 dark:text-[#fdfcff]">{{ t('knowledgeSpaces.create.title') }}</h1>
      <p class="mt-2 text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t(descriptionKey || 'knowledgeSpaces.create.description') }}</p>
    </header>

    <slot name="notices" />
    <form class="space-y-6" @submit.prevent="submit">
    <section class="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-[#0f192a]">
      <header class="flex items-center justify-between border-b border-slate-200 p-5 dark:border-slate-700"><div><h2 class="font-semibold text-gray-900 dark:text-[#fdfcff]">{{ t('knowledgeSpaces.create.basicTitle') }}</h2><p class="mt-1 text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t(basicDescriptionKey || 'knowledgeSpaces.create.basicDescription') }}</p></div><UButton color="neutral" variant="outline" :to="overviewPath">{{ t('knowledgeSpaces.create.back') }}</UButton></header>
      <div class="grid gap-5 p-5 md:grid-cols-2">
        <UFormField :label="t('knowledgeSpaces.ui.spaceName')" required><UInput v-model="name" class="w-full" :placeholder="t('knowledgeSpaces.create.namePlaceholder')" icon="i-heroicons-rectangle-stack" :disabled="saving" /></UFormField>
        <UFormField :label="t('knowledgeSpaces.ui.department')" required><USelectMenu v-model="department" :items="departmentOptions" value-key="value" label-key="label" :loading="departmentsLoading" :disabled="saving || !!departmentsError || (!departmentsLoading && !departmentOptions.length)" :placeholder="t('knowledgeSpaces.create.departmentPlaceholder')" :portal="true" :ui="{ content: 'z-50' }" class="w-full" /><template #help><span v-if="departmentsError" class="text-error-600 dark:text-error-300">{{ departmentsError }}</span><span v-else-if="!departmentsLoading && !departmentOptions.length" class="text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t(departmentsEmptyKey || 'knowledgeSpaces.create.noDepartments') }}</span></template></UFormField>
      </div>
      <footer class="flex justify-end gap-2 border-t border-slate-200 p-5 dark:border-slate-700"><UButton color="neutral" variant="subtle" :to="overviewPath">{{ t('common.cancel') }}</UButton><UButton :loading="saving" :disabled="!canSave || !!departmentsError || !!strategyError || saving" type="submit">{{ t('knowledgeSpaces.create.submit') }}</UButton></footer>
    </section>

    <section class="rounded-2xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-[#0f192a]">
      <header class="border-b border-slate-200 p-5 dark:border-slate-700"><h2 class="font-semibold text-gray-900 dark:text-[#fdfcff]">{{ t('knowledgeSpaces.ui.strategyPackage') }}</h2><p class="mt-1 text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.create.strategyDescription') }}</p></header>
      <div class="grid gap-5 p-5 md:grid-cols-2">
        <UFormField :label="t('knowledgeSpaces.ui.strategyPackage')" required><USelectMenu v-model="strategy" :items="strategyOptions" value-key="value" label-key="label" :disabled="saving || !!strategyError || !strategyOptions.length" :search-input="{ placeholder: t('knowledgeSpaces.create.searchStrategy') }" :portal="true" :ui="{ content: 'z-50' }" class="w-full" /><template #help><span :class="strategyError ? 'text-error-600 dark:text-error-300' : 'text-gray-600 dark:text-[#d6e2ff]'" class="text-sm">{{ strategyError || strategySummary }}</span></template></UFormField>
        <div class="rounded-xl border border-slate-200 p-4 dark:border-slate-700"><h3 class="font-medium text-gray-900 dark:text-[#fdfcff]">{{ t('knowledgeSpaces.ui.strategyAutoMapping') }}</h3><p class="mt-3 text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t('knowledgeSpaces.create.recommendedProfile') }}：<span class="font-medium text-gray-900 dark:text-[#fdfcff]">{{ mappedProfile || '-' }}</span></p><p class="mt-1 text-sm text-gray-600 dark:text-[#d6e2ff]">{{ t(executionStatusKey || 'knowledgeSpaces.create.executionStatus') }}：<span class="font-medium" :class="statusColor === 'success' ? 'text-success-600 dark:text-success-400' : 'text-warning-700 dark:text-warning-300'">{{ executionStatus }}</span></p></div>
      </div>
      <div class="px-5 pb-5"><slot name="strategy-notice" /></div>
    </section>
    </form>
    <slot name="technical-details" />
  </main>
</template>

<script setup lang="ts">
type SelectOption = { label: string; value: string; disabled?: boolean }
const props = defineProps<{
  overviewPath: string
  departmentOptions: SelectOption[]
  strategyOptions: SelectOption[]
  departmentsEmptyKey?: string
  departmentsLoading?: boolean
  departmentsError?: string
  strategyError?: string
  strategySummary?: string
  mappedProfile?: string
  executionStatus: string
  statusColor?: 'success' | 'warning' | 'neutral'
  canSave: boolean
  saving?: boolean
  descriptionKey?: string
  basicDescriptionKey?: string
  executionStatusKey?: string
}>()
const name = defineModel<string>('name', { required: true })
const department = defineModel<string>('department', { required: true })
const strategy = defineModel<string>('strategy', { required: true })
const emit = defineEmits<{ save: [] }>()
const { t } = useI18n()
function submit() {
  if (props.canSave && !props.saving && !props.departmentsError && !props.strategyError) emit('save')
}
</script>
