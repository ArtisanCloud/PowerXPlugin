<template>
  <div class="space-y-6 p-4 text-gray-900 dark:text-gray-100">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-xl font-semibold">{{ t('localIntelligence.' + kind) }}</h1>
        <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ t('localIntelligence.description') }}</p>
      </div>
      <div class="flex items-center gap-2">
        <UBadge color="success" variant="soft">{{ t('localIntelligence.local') }}</UBadge>
        <UButton variant="soft" icon="i-heroicons-arrow-path" :loading="loading" @click="load">{{ t('localIntelligence.refresh') }}</UButton>
        <UButton icon="i-heroicons-plus" @click="openEditor()">{{ t('localIntelligence.create') }}</UButton>
      </div>
    </header>
    <div class="grid gap-3 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-800/40 sm:grid-cols-3">
      <UInput v-model="search" icon="i-heroicons-magnifying-glass" :placeholder="t('localIntelligence.search')" class="w-full sm:col-span-2" @keyup.enter="reload" />
      <USelect v-model="statusFilter" :items="filterOptions" class="w-full" @update:model-value="reload" />
    </div>
    <UAlert v-if="error" color="error" variant="soft" :title="error" />
    <p v-if="loading" role="status">{{ t('localIntelligence.loading') }}</p>
    <UAlert v-else-if="!items.length && !error" color="neutral" variant="soft" :title="t('localIntelligence.empty')" />
    <div v-if="kind === 'agents'" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <UCard v-for="item in items" :key="item.uuid" :ui="{ body: 'space-y-4' }">
        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-center gap-3">
            <UAvatar :text="item.name.slice(0, 2)" size="lg" />
            <div class="min-w-0"><h2 class="truncate font-semibold">{{ item.name }}</h2><p class="text-xs text-gray-500 dark:text-gray-400">{{ item.key }}</p></div>
          </div>
          <UBadge :color="item.status === 'active' ? 'success' : 'neutral'" variant="soft">{{ statusLabel(item.status) }}</UBadge>
        </div>
        <p class="line-clamp-2 min-h-10 text-sm text-gray-600 dark:text-gray-300">{{ item.description || t('localIntelligence.noDescription') }}</p>
        <div class="flex flex-wrap gap-2 text-xs">
          <UBadge color="neutral" variant="soft">{{ modelLabel(item.model_key) }}</UBadge>
          <UBadge color="neutral" variant="soft">{{ t('localIntelligence.skillCount', { count: item.skill_uuids?.length || 0 }) }}</UBadge>
        </div>
        <div class="flex flex-wrap gap-2">
          <UButton size="sm" variant="soft" icon="i-heroicons-pencil-square" @click="openEditor(item)">{{ t('localIntelligence.edit') }}</UButton>
          <UButton size="sm" variant="soft" icon="i-heroicons-chat-bubble-left-right" :disabled="item.status !== 'active'" @click="openRun(item)">{{ t('localIntelligence.debug') }}</UButton>
          <UButton size="sm" color="neutral" variant="ghost" :disabled="mutating" @click="toggle(item)">{{ t(item.status === 'active' ? 'localIntelligence.disable' : 'localIntelligence.enable') }}</UButton>
        </div>
      </UCard>
    </div>
    <div v-else class="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700">
      <table class="w-full text-left text-sm">
        <thead class="bg-gray-50 text-gray-700 dark:bg-gray-800 dark:text-gray-200"><tr><th v-for="field in ['name', 'version', 'executor', 'status', 'actions']" :key="field" class="p-3">{{ t('localIntelligence.' + field) }}</th></tr></thead>
        <tbody><tr v-for="item in items" :key="item.uuid" class="border-t border-gray-200 dark:border-gray-700">
          <td class="p-3"><p class="font-medium">{{ item.name }}</p><p class="text-xs text-gray-500 dark:text-gray-400">{{ item.key }}</p><p class="mt-1 text-gray-600 dark:text-gray-300">{{ item.description }}</p></td>
          <td class="p-3">{{ item.version }}</td><td class="p-3">{{ t('localIntelligence.executors.' + item.executor) }}</td>
          <td class="p-3"><UBadge :color="item.status === 'active' ? 'success' : 'neutral'" variant="soft">{{ statusLabel(item.status) }}</UBadge></td>
          <td class="p-3"><div class="flex gap-2">
            <UButton size="sm" variant="soft" @click="openEditor(item)">{{ t('localIntelligence.edit') }}</UButton>
            <UButton size="sm" variant="soft" :disabled="item.status !== 'active'" @click="openRun(item)">{{ t('localIntelligence.debug') }}</UButton>
            <UButton size="sm" color="neutral" variant="ghost" :disabled="mutating" @click="toggle(item)">{{ t(item.status === 'active' ? 'localIntelligence.disable' : 'localIntelligence.enable') }}</UButton>
          </div></td>
        </tr></tbody>
      </table>
    </div>
    <div class="flex items-center justify-end gap-3">
      <span class="text-sm">{{ t('localIntelligence.total', { count: total }) }}</span>
      <UButton variant="soft" :disabled="loading || page <= 1" @click="page--; load()">{{ t('localIntelligence.previous') }}</UButton>
      <span>{{ page }} / {{ Math.max(1, Math.ceil(total / 20)) }}</span>
      <UButton variant="soft" :disabled="loading || page * 20 >= total" @click="page++; load()">{{ t('localIntelligence.next') }}</UButton>
    </div>
    <UModal v-model:open="editorOpen" :dismissible="!mutating" :close="!mutating" :title="t(editID ? 'localIntelligence.edit' : 'localIntelligence.create')" :description="t('localIntelligence.editorHint')" :ui="{ content: 'w-full max-w-3xl' }">
      <template #body>
        <form class="space-y-4 text-gray-900 dark:text-gray-100" @submit.prevent="save">
          <UAlert v-if="formError" color="error" variant="soft" :title="formError" />
          <UTabs v-model="editTab" :items="editTabs" :content="false" />
          <div v-show="editTab === 'basic'" class="space-y-4">
            <div class="grid gap-3 sm:grid-cols-2">
              <UFormField :label="t('localIntelligence.name')" required><UInput v-model="form.name" class="w-full" :disabled="mutating" /></UFormField>
              <UFormField :label="t('localIntelligence.key')" required><UInput v-model="form.key" class="w-full" :disabled="!!editID || mutating" /></UFormField>
              <UFormField v-if="kind === 'skills'" :label="t('localIntelligence.version')" required><UInput v-model="form.version" class="w-full" :disabled="!!editID || mutating" /></UFormField>
              <UFormField :label="t('localIntelligence.status')"><USelect v-model="form.status" :items="statusOptions" class="w-full" :disabled="mutating" /></UFormField>
            </div>
            <UFormField :label="t('localIntelligence.summary')"><UTextarea v-model="form.description" class="w-full" :disabled="mutating" /></UFormField>
            <UFormField v-if="kind === 'skills'" :label="t('localIntelligence.executor')"><USelect v-model="form.executor" :items="executorOptions" class="w-full" :disabled="mutating" /></UFormField>
            <p v-if="kind === 'skills'" class="text-sm text-gray-600 dark:text-gray-300">{{ t('localIntelligence.executorHint') }}</p>
          </div>
          <div v-show="editTab === 'prompt'" class="space-y-4">
            <UAlert v-if="modelError" color="warning" variant="soft" :title="modelError" />
            <UFormField :label="t('localIntelligence.model')" :description="t('localIntelligence.modelHint')"><USelect v-model="form.model_key" :items="modelOptions" class="w-full" :disabled="mutating" /></UFormField>
            <UFormField v-if="kind === 'agents'" :label="t('localIntelligence.persona')"><UTextarea v-model="form.persona" :rows="3" class="w-full" :disabled="mutating" /></UFormField>
            <UFormField :label="t('localIntelligence.prompt')"><UTextarea v-model="form.prompt" :rows="8" class="w-full" :disabled="mutating" /></UFormField>
          </div>
          <div v-if="kind === 'agents'" v-show="editTab === 'skills'" class="space-y-3">
            <p class="text-sm">{{ t('localIntelligence.bindingHint') }}</p>
            <UAlert v-if="skillError" color="error" variant="soft" :title="skillError" />
            <p v-else-if="!skillOptions.length">{{ t('localIntelligence.noSkills') }}</p>
            <UCheckbox v-for="skill in skillOptions" :key="skill.uuid" :model-value="form.skill_uuids.includes(skill.uuid)" :label="skill.name + ' · ' + skill.version + ' · ' + statusLabel(skill.status)" :disabled="mutating" @update:model-value="value => selectSkill(skill.uuid, !!value)" />
          </div>
          <div class="flex justify-end gap-2"><UButton color="neutral" variant="soft" :disabled="mutating" @click="editorOpen = false">{{ t('localIntelligence.cancel') }}</UButton><UButton type="submit" :loading="mutating">{{ t('localIntelligence.save') }}</UButton></div>
        </form>
      </template>
    </UModal>
    <UModal v-model:open="runOpen" :dismissible="!running" :close="!running" :title="t('localIntelligence.debug') + ' · ' + (selected?.name || '')" :description="t(kind === 'agents' ? 'localIntelligence.agentDebugHint' : 'localIntelligence.skillDebugHint')" :ui="{ content: 'w-full max-w-3xl' }">
      <template #body>
        <form class="space-y-4 text-gray-900 dark:text-gray-100" @submit.prevent="run">
          <template v-if="kind === 'agents'">
            <UFormField :label="t('localIntelligence.message')" required><UTextarea v-model="message" class="w-full" :rows="4" :disabled="running" /></UFormField>
            <UFormField :label="t('localIntelligence.optionalSkill')"><USelect v-model="runSkill" :items="boundSkillOptions" class="w-full" :disabled="running" /></UFormField>
          </template>
          <UFormField v-if="kind === 'skills' || runSkill !== '__none__'" :label="t('localIntelligence.input')" :description="t('localIntelligence.inputHint')"><UTextarea v-model="inputJSON" class="w-full font-mono" :rows="6" :disabled="running" /></UFormField>
          <UAlert v-if="runError" color="error" variant="soft" :title="runError" />
          <UButton type="submit" :loading="running">{{ t('localIntelligence.run') }}</UButton>
          <div v-if="result" class="space-y-2"><h3 class="font-medium">{{ t('localIntelligence.result') }}</h3><pre class="max-h-96 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-gray-50 p-3 text-sm dark:bg-gray-800">{{ JSON.stringify(result, null, 2) }}</pre></div>
        </form>
      </template>
    </UModal>
  </div>
</template>
<script setup lang="ts">
import { useLocalIntelligence, type LocalKind, type LocalDefinition, type LocalModel } from '~/composables/api/useLocalIntelligence'
const props = defineProps<{ kind: LocalKind }>()
const { t, te } = useI18n()
const api = useLocalIntelligence()
const items = ref<LocalDefinition[]>([]), total = ref(0), page = ref(1), search = ref(''), statusFilter = ref('__all__')
const loading = ref(false), error = ref(''), mutating = ref(false), formError = ref('')
const models = ref<LocalModel[]>([]), modelError = ref(''), skillError = ref('')
const skillOptions = ref<LocalDefinition[]>([])
const editorOpen = ref(false), editID = ref(''), editTab = ref('basic')
const fresh = () => ({ key: '', name: '', description: '', status: 'inactive' as 'active' | 'inactive', model_key: '', prompt: '', persona: '', skill_uuids: [] as string[], version: '1.0.0', executor: 'prompt' })
const form = reactive(fresh())
const runOpen = ref(false), running = ref(false), runError = ref(''), selected = ref<LocalDefinition>()
const message = ref(''), inputJSON = ref('{}'), runSkill = ref('__none__'), result = ref<Record<string, unknown> | null>(null)
const statusLabel = (value: string) => t('localIntelligence.' + value)
const statusOptions = computed(() => ['active', 'inactive'].map(value => ({ value, label: statusLabel(value) })))
const filterOptions = computed(() => [{ value: '__all__', label: t('localIntelligence.all') }, ...statusOptions.value])
const executorOptions = computed(() => ['prompt', 'template'].map(value => ({ value, label: t('localIntelligence.executors.' + value) })))
const modelOptions = computed(() => models.value.filter(m => m.configured).map(m => ({ value: m.model_key, label: m.label || m.model_key })))
const modelLabel = (key: string) => modelOptions.value.find(m => m.value === key)?.label || key || t('localIntelligence.noModel')
const editTabs = computed(() => [
  { value: 'basic', label: t('localIntelligence.basic') },
  ...(props.kind === 'agents' || form.executor === 'prompt' ? [{ value: 'prompt', label: t('localIntelligence.modelPrompt') }] : []),
  ...(props.kind === 'agents' ? [{ value: 'skills', label: t('localIntelligence.skills') }] : []),
])
const boundSkillOptions = computed(() => [{ value: '__none__', label: t('localIntelligence.noSkill') }, ...skillOptions.value.filter(s => selected.value?.skill_uuids?.includes(s.uuid)).map(s => ({ value: s.uuid, label: s.name + ' · ' + s.version }))])
function errorText(cause: any) {
  const code = cause?.data?.error?.code || cause?.data?.code || ''
  const key = 'localIntelligence.errors.' + code
  return te(key) ? t(key) : t('localIntelligence.failed', { code: code || cause?.statusCode || cause?.status || 'NETWORK_ERROR' })
}
let request = 0
async function load() {
  const current = ++request
  loading.value = true; error.value = ''
  try {
    const response = await api.list(props.kind, { q: search.value, status: statusFilter.value === '__all__' ? undefined : statusFilter.value, page: page.value, page_size: 20 })
    if (current !== request) return
    items.value = response.data?.items || []; total.value = response.data?.total || 0
  } catch (cause) { if (current === request) { items.value = []; error.value = errorText(cause) } }
  finally { if (current === request) loading.value = false }
}
function reload() { page.value = 1; void load() }
async function loadChoices() {
  modelError.value = ''; skillError.value = ''
  await Promise.all([
    api.models().then(response => { models.value = response.data?.items || [] }).catch(cause => { modelError.value = errorText(cause) }),
    (async () => {
      const all: LocalDefinition[] = []; let next = 1
      for (;;) {
        const response = await api.list('skills', { page: next++, page_size: 100 })
        all.push(...(response.data?.items || []))
        if (all.length >= (response.data?.total || 0) || !response.data?.items?.length) break
      }
      skillOptions.value = all
    })().catch(cause => { skillOptions.value = []; skillError.value = errorText(cause) }),
  ])
}
function openEditor(item?: LocalDefinition) {
  Object.assign(form, fresh(), item ? { key: item.key, name: item.name, description: item.description, status: item.status, model_key: item.model_key, prompt: item.prompt, persona: item.persona || '', skill_uuids: [...(item.skill_uuids || [])], version: item.version || '1.0.0', executor: item.executor || 'prompt' } : {})
  editID.value = item?.uuid || ''; editTab.value = 'basic'; formError.value = ''; editorOpen.value = true
  void loadChoices()
}
function selectSkill(id: string, value: boolean) { form.skill_uuids = value ? [...new Set([...form.skill_uuids, id])] : form.skill_uuids.filter(v => v !== id) }
function payload(item: typeof form) {
  const base = { key: item.key, name: item.name, description: item.description, status: item.status, model_key: item.model_key, prompt: item.prompt }
  return props.kind === 'agents' ? { ...base, persona: item.persona, skill_uuids: item.skill_uuids } : { ...base, version: item.version, executor: item.executor }
}
async function save() {
  if (mutating.value) return
  if (!form.name.trim() || !form.key.trim()) { formError.value = t('localIntelligence.required'); editTab.value = 'basic'; return }
  mutating.value = true; formError.value = ''
  try { await api.save(props.kind, editID.value, payload(form)); editorOpen.value = false; await load() }
  catch (cause) { formError.value = errorText(cause) } finally { mutating.value = false }
}
async function toggle(item: LocalDefinition) {
  mutating.value = true; error.value = ''
  try { await api.save(props.kind, item.uuid, payload({ ...fresh(), ...item, skill_uuids: [...(item.skill_uuids || [])], status: item.status === 'active' ? 'inactive' : 'active' } as typeof form)); await load() }
  catch (cause) { error.value = errorText(cause) } finally { mutating.value = false }
}
function openRun(item: LocalDefinition) {
  selected.value = item; message.value = ''; inputJSON.value = item.executor === 'template' ? JSON.stringify({ action: 'list' }, null, 2) : '{}'
  runSkill.value = '__none__'; result.value = null; runError.value = ''; runOpen.value = true; void loadChoices()
}
watch(runSkill, value => { inputJSON.value = skillOptions.value.find(s => s.uuid === value)?.executor === 'template' ? JSON.stringify({ action: 'list' }, null, 2) : '{}' })
async function run() {
  if (!selected.value || running.value) return
  let input: Record<string, unknown>
  try { input = JSON.parse(inputJSON.value); if (!input || Array.isArray(input) || typeof input !== 'object') throw new Error() }
  catch { runError.value = t('localIntelligence.invalidJSON'); return }
  running.value = true; runError.value = ''; result.value = null
  try {
    const body = props.kind === 'skills' ? { input } : { message: message.value, ...(runSkill.value !== '__none__' ? { skill_uuid: runSkill.value, skill_input: input } : {}) }
    const response = await api.run(props.kind, selected.value.uuid, body); result.value = response.data || null
  } catch (cause) { runError.value = errorText(cause) } finally { running.value = false }
}
onMounted(() => { void load(); void loadChoices() })
</script>
