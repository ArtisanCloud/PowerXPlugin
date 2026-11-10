<template>
  <UModal :open="true" :title="t('aiSettings.online.title')" :description="t('aiSettings.online.scope', { provider: providerName, environment: t(`aiSettings.environments.${input.environment}`), modality: modalityName(input.modality) })" :dismissible="!busy" :close="!busy" :ui="{ content: 'ai-settings-online-dialog w-[calc(100vw-2rem)] max-w-4xl text-highlighted', body: 'min-w-0' }" @update:open="value => { if (!value) emit('close'); }">
    <template #body>
      <div class="min-w-0 space-y-4 text-highlighted" data-testid="ai-online-catalog">
        <p class="text-sm text-toned">{{ t('aiSettings.online.description') }}</p>
        <p v-if="busy" role="status" class="flex items-center gap-2 text-sm"><UIcon name="i-heroicons-arrow-path" class="animate-spin" />{{ t('aiSettings.online.loading') }}</p>
        <div v-if="error" role="alert" data-testid="ai-online-error" class="rounded-lg border border-default bg-elevated p-3 text-highlighted">
          <p>{{ error }}</p>
          <details class="mt-2 text-sm"><summary>{{ t('aiSettings.testing.diagnostic') }}</summary><pre class="mt-2 max-h-40 overflow-auto whitespace-pre-wrap break-all text-xs">{{ diagnostic }}</pre></details>
        </div>
        <template v-if="catalog">
          <div role="status" data-testid="ai-online-summary" class="space-y-1 rounded-lg border border-default bg-elevated p-3 text-highlighted">
            <p class="text-sm font-semibold">{{ t('aiSettings.online.summary', { total: catalog.models.length, matched: matched.length, pending: pending.length }) }}</p>
            <p class="break-all text-xs text-toned">{{ t('aiSettings.online.source', { source: catalog.source, protocol: catalog.protocol }) }}</p>
            <p class="text-xs text-toned">{{ t('aiSettings.online.updatedAt', { time: new Date(catalog.updated_at).toLocaleString() }) }}</p>
          </div>
          <div class="flex min-w-0 flex-col gap-4 sm:flex-row">
            <div role="tablist" :aria-label="t('aiSettings.online.results')" class="flex shrink-0 gap-2 sm:w-36 sm:flex-col" @keydown="navigateTabs">
              <UButton v-for="item in tabs" :id="`ai-online-tab-${item.value}`" :key="item.value" :data-testid="`ai-online-tab-${item.value}`" role="tab" :aria-selected="tab === item.value" :aria-controls="`ai-online-panel-${item.value}`" :tabindex="tab === item.value ? 0 : -1" color="neutral" :variant="tab === item.value ? 'soft' : 'ghost'" class="justify-start text-highlighted" @click="tab = item.value">{{ item.label }}</UButton>
            </div>
            <section :id="`ai-online-panel-${tab}`" role="tabpanel" :aria-labelledby="`ai-online-tab-${tab}`" class="min-w-0 flex-1 space-y-3">
              <p class="text-sm text-toned">{{ t(tab === 'matched' ? 'aiSettings.online.matchedHint' : 'aiSettings.online.pendingHint') }}</p>
              <UInput v-model="query" data-testid="ai-online-search" icon="i-heroicons-magnifying-glass" :placeholder="t('aiSettings.online.search')" class="w-full" />
              <div v-if="tab === 'pending' && canAssign && pending.length" class="flex flex-wrap items-center gap-2">
                <UButton color="neutral" variant="outline" size="xs" :disabled="busy" @click="selectVisible">{{ t('aiSettings.online.selectVisible') }}</UButton>
                <UButton data-testid="ai-online-assign" color="neutral" variant="outline" size="sm" :disabled="busy || !selected.length" @click="run(true)">{{ t('aiSettings.online.assign', { count: selected.length, modality: modalityName(input.modality) }) }}</UButton>
              </div>
              <p v-if="tab === 'pending' && !canAssign" class="text-sm text-toned">{{ t('aiSettings.online.mediaContractRequired') }}</p>
              <div class="max-h-80 divide-y divide-default overflow-y-auto rounded-lg border border-default" data-testid="ai-online-list">
                <p v-if="!visible.length" class="p-4 text-sm text-toned">{{ t('aiSettings.online.empty') }}</p>
                <label v-for="item in visible" :key="item.id" class="flex min-w-0 items-start gap-3 p-3">
                  <UCheckbox v-if="tab === 'pending' && canAssign" :model-value="selected.includes(item.id)" :disabled="busy || (selected.length >= 1000 && !selected.includes(item.id))" :aria-label="item.label" @update:model-value="value => select(item.id, value === true)" />
                  <div class="min-w-0"><p class="break-all text-sm text-highlighted">{{ item.label }}</p><p class="break-all font-mono text-xs text-toned">{{ item.id }}</p><p v-if="tab === 'matched'" class="text-xs text-toned">{{ t(`aiSettings.online.match.${item.match}`) }} · {{ modelModalities(item) }}</p></div>
                </label>
              </div>
            </section>
          </div>
        </template>
      </div>
    </template>
    <template #footer><div class="flex w-full flex-wrap justify-end gap-2"><UButton color="neutral" variant="outline" :disabled="busy" @click="emit('close')">{{ t('aiSettings.online.close') }}</UButton><UButton data-testid="ai-online-refresh" color="neutral" variant="outline" icon="i-heroicons-cloud-arrow-down" :loading="busy" :disabled="busy" @click="run(false)">{{ t('aiSettings.online.update') }}</UButton></div></template>
  </UModal>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { createLocalModelCatalogClient, type LocalOnlineCatalog, type LocalOnlineCatalogInput } from '@artisan-cloud/plugin-framework-client';
import { useApiClient } from '~/composables/api/_client';
const props = defineProps<{ input: LocalOnlineCatalogInput; providerName: string }>();
const emit = defineEmits<{ close: []; updated: []; failure: [diagnostic: string]; started: [] }>();
const { t } = useI18n();
const modalityKeys: Record<string, string> = { llm: 'llm', vlm: 'vlm', embedding: 'embedding', image: 'image', video: 'video', audio_tts: 'audioTTS', audio_asr: 'audioASR', model3d: 'model3d', rerank: 'rerank' };
function modalityName(value: string) { return t(modalityKeys[value] ? `aiSettings.modalities.${modalityKeys[value]}` : 'aiSettings.online.unknownCapability'); }
function modelModalities(model: { modalities: string[]; manual_modalities: string[] }) { const values = [...new Set([...model.modalities, ...model.manual_modalities])]; return values.length ? values.map(modalityName).join(' · ') : t('aiSettings.online.declaredCatalog'); }
const { client } = useApiClient();
const api = createLocalModelCatalogClient({ request: (path, options) => client(path, options) });
const catalog = ref<LocalOnlineCatalog>();
const busy = ref(false), error = ref(''), diagnostic = ref(''), query = ref('');
const tab = ref<'matched' | 'pending'>('matched'), selected = ref<string[]>([]);
let active = true;
onBeforeUnmount(() => { active = false; });
const matched = computed(() => catalog.value?.models.filter(item => item.match !== 'unmatched') || []);
const pending = computed(() => catalog.value?.models.filter(item => item.match === 'unmatched') || []);
const canAssign = computed(() => ['llm', 'embedding'].includes(props.input.modality));
const tabs = computed(() => [{ value: 'matched' as const, label: t('aiSettings.online.matched', { count: matched.value.length }) }, { value: 'pending' as const, label: t('aiSettings.online.pending', { count: pending.value.length }) }]);
const visible = computed(() => (tab.value === 'matched' ? matched.value : pending.value).filter(item => `${item.label} ${item.id}`.toLowerCase().includes(query.value.toLowerCase())));
function select(id: string, checked: boolean) { selected.value = checked ? [...new Set([...selected.value, id])].slice(0, 1000) : selected.value.filter(value => value !== id); }
function selectVisible() { selected.value = [...new Set([...selected.value, ...visible.value.map(item => item.id)])].slice(0, 1000); }
function navigateTabs(event: KeyboardEvent) {
  if (!['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
  event.preventDefault(); tab.value = event.key === 'Home' ? 'matched' : event.key === 'End' ? 'pending' : tab.value === 'matched' ? 'pending' : 'matched';
  document.getElementById(`ai-online-tab-${tab.value}`)?.focus();
}
async function run(assign: boolean) {
  if (busy.value || (assign && !catalog.value)) return;
  busy.value = true; error.value = ''; diagnostic.value = ''; emit('started');
  try {
    const result = await api.sync({ ...props.input }, assign ? { revision: catalog.value!.revision, models: [...selected.value] } : undefined);
    if (!active) return;
    catalog.value = result; selected.value = []; emit('updated');
  } catch (failure: any) {
    if (!active) return;
    const data = failure?.data || failure?.response?._data || {};
    const code = data.reason_code || data.error_code || failure?.reasonCode;
    const keys = ['AI_CREDENTIAL_REQUIRED', 'AI_CREDENTIAL_UNAVAILABLE', 'AI_CREDENTIAL_SCOPE_MISMATCH', 'AI_MODEL_SYNC_AUTH_FAILED', 'AI_MODEL_SYNC_STALE', 'AI_MODEL_SYNC_CONNECTION_CHANGED', 'AI_MODEL_SYNC_ADMIN_REQUIRED', 'AI_MODEL_SYNC_UNSUPPORTED', 'AI_LOCAL_CATALOG_MIGRATION_REQUIRED'];
    error.value = t(`aiSettings.online.errors.${keys.includes(code) ? code : 'generic'}`);
    diagnostic.value = JSON.stringify({ reason_code: code, status: failure?.status || failure?.statusCode || failure?.response?.status, trace_id: data.trace_id, request_id: data.request_id, details: data.details }, null, 2);
    emit('failure', diagnostic.value);
  } finally { if (active) busy.value = false; }
}
onMounted(() => { void run(false); });
</script>

<style>
.ai-settings-online-dialog { color: var(--ui-text); }
.ai-settings-online-dialog input::placeholder { color: var(--ui-text-toned); opacity: 1; }
.ai-settings-online-dialog :disabled { color: var(--ui-text-toned); opacity: 1; }
</style>
