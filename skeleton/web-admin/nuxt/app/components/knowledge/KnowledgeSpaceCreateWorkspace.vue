<template>
  <KnowledgeSpaceCreatePanel
    v-model:name="draft.name" v-model:department="draft.department_code" v-model:strategy="draft.strategy_package_key"
    :overview-path="overviewPath" :department-options="departmentOptions" :strategy-options="strategyOptions"
    :departments-loading="departmentsLoading" :departments-error="departmentsError"
    :strategy-summary="selectedStrategy ? strategySummary(selectedStrategy.key) : ''"
    :mapped-profile="profileLabel(selectedStrategy?.profile_key)"
    :execution-status="selectedStrategy?.ready ? t('knowledgeSpaces.ui.strategyReady') : t('knowledgeSpaces.create.unavailableLocally')"
    :status-color="selectedStrategy?.ready ? 'success' : 'warning'" :can-save="canSave" :saving="saving" @save="save">
    <template #strategy-notice>
      <UAlert v-if="selectedStrategy && !selectedStrategy.ready" color="warning" variant="soft" :title="t('knowledgeSpaces.create.strategyUnavailableTitle')" :description="t('knowledgeSpaces.create.strategyUnavailableDescription')" />
      <UAlert v-else-if="selectedStrategy" color="success" variant="soft" :title="t('knowledgeSpaces.ui.strategyReady')" :description="t('knowledgeSpaces.create.strategyReadyDescription')" />
    </template>
  </KnowledgeSpaceCreatePanel>
</template>
<script setup lang="ts">
import KnowledgeSpaceCreatePanel from './KnowledgeSpaceCreatePanel.vue'
import { computed, onMounted, reactive, ref } from 'vue'
import { useLocalKnowledgeApi, type KnowledgeStrategyPackage } from '~/composables/api/useLocalKnowledge'
import { useDepartmentService, type Department } from '~/composables/api/services/departmentService'
const { t } = useI18n(); const toast = useToast(); const api = useLocalKnowledgeApi(); const departmentsApi = useDepartmentService(); const runtimeConfig = useRuntimeConfig(); const router = useRouter()
const draft = reactive({ name: '', department_code: '', strategy_package_key: '' }); const departments = ref<Department[]>([]); const strategies = ref<KnowledgeStrategyPackage[]>([]); const departmentsLoading = ref(false); const departmentsError = ref(''); const saving = ref(false)
const adminBase = computed(() => String(runtimeConfig.public?.pluginAdminBase || '/_p/com.powerx.plugins.base/admin/').replace(/\/+$/, '')); const overviewPath = computed(() => `${adminBase.value}/knowledge`)
function flatten(nodes: Department[], out: Department[] = []) { for (const item of nodes) { out.push(item); if (item.children?.length) flatten(item.children, out) }; return out }
const departmentOptions = computed(() => flatten(departments.value).map(item => ({ label: item.name, value: item.code }))); const strategyOptions = computed(() => strategies.value.map(item => ({ label: strategyLabel(item.key), value: item.key, disabled: !item.ready }))); const selectedStrategy = computed(() => strategies.value.find(item => item.key === draft.strategy_package_key)); const canSave = computed(() => Boolean(draft.name.trim() && draft.department_code && selectedStrategy.value?.ready))
function strategyLabel(key: string) { return t(`knowledgeSpaces.strategies.${key}.label`) } function strategySummary(key: string) { return t(`knowledgeSpaces.strategies.${key}.summary`) } function profileLabel(key?: string) { return key ? t(`knowledgeSpaces.profiles.${key}`) : '-' }
async function load() { departmentsLoading.value = true; departmentsError.value = ''; try { const [tree, catalog] = await Promise.all([departmentsApi.getDepartmentTree(), api.strategyPackages()]); departments.value = tree; strategies.value = catalog.items || []; draft.department_code = departmentOptions.value[0]?.value || ''; draft.strategy_package_key = strategies.value.find(item => item.ready)?.key || '' } catch { departmentsError.value = t('knowledgeSpaces.create.loadFailed'); toast.add({ title: t('common.error'), description: t('knowledgeSpaces.create.loadFailed'), color: 'error' }) } finally { departmentsLoading.value = false } }
async function save() { if (!canSave.value) return; saving.value = true; let created = false; try { const space = await api.saveSpace({ ...draft, name: draft.name.trim() }); created = true; await api.activateVectorIndex(space.uuid); toast.add({ title: t('knowledgeSpaces.ui.spaceSaved'), description: t('knowledgeSpaces.ui.spaceSavedAndVectorActivated'), color: 'success' }); await router.push(overviewPath.value) } catch { toast.add({ title: t('common.error'), description: created ? t('knowledgeSpaces.ui.spaceSavedVectorActivationFailed') : t('knowledgeSpaces.ui.spaceSaveFailed'), color: 'error' }); if (created) await router.push(overviewPath.value) } finally { saving.value = false } }
onMounted(load)
</script>
