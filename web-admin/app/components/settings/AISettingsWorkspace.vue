<template>
  <UContainer class="py-6 text-default sm:py-8">
    <div ref="workspaceElement" class="ai-settings space-y-5">
      <header class="flex flex-wrap items-start justify-between gap-4">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-3">
            <h1 class="text-2xl font-semibold text-highlighted">{{ t('aiSettings.title') }}</h1>
            <UBadge color="neutral" variant="soft">{{ modeLabel }}</UBadge>
          </div>
          <p class="ai-settings-copy mt-1 text-sm">{{ t('aiSettings.description') }}</p>
        </div>
        <div class="flex items-center gap-2">
          <UButton data-testid="ai-settings-reset" icon="i-heroicons-arrow-path" color="neutral" variant="ghost" @click="reset">{{ t('common.reset') }}</UButton>
          <UButton data-testid="ai-settings-save" :class="primaryButtonClass" icon="i-heroicons-cloud-arrow-up" :loading="saving" :disabled="coreSelectionInvalid || localSelectionInvalid" @click="saveSettings">{{ t('aiSettings.workspace.save') }}</UButton>
        </div>
      </header>

      <div data-testid="ai-settings-context" class="flex flex-wrap items-end gap-4">
        <UFormField :label="t('aiSettings.environment')" class="min-w-0 flex-1 sm:max-w-44">
          <USelect data-testid="ai-settings-environment" v-model="environment" :items="environmentOptions" class="w-full" />
        </UFormField>
        <UFormField :label="t('aiSettings.catalogSource')" class="min-w-0 basis-full sm:basis-auto sm:w-64">
          <template #hint>
              <UPopover>
                <UButton data-testid="ai-settings-source-help" :aria-label="t('aiSettings.workspace.sourceHelp')" icon="i-heroicons-information-circle" color="neutral" variant="ghost" size="xs" class="rounded-full" />
                <template #content>
                  <p class="ai-settings-copy max-w-72 p-4 text-sm">{{ t('aiSettings.catalogSourceDescription') }}</p>
                </template>
              </UPopover>
          </template>
          <USelect data-testid="ai-settings-catalog-source" v-model="catalogSource" :items="catalogSourceOptions" class="w-full" />
        </UFormField>
        <UFormField :label="t('aiSettings.modalityTabsLabel')" class="ai-settings-modality-picker min-w-0 basis-full">
          <USelect data-testid="ai-settings-modality-picker" v-model="modality" :items="modalities" class="w-full" />
        </UFormField>
      </div>

      <UAlert v-if="catalogError" data-testid="ai-settings-catalog-error" role="alert" class="ai-settings-catalog-alert" color="warning" variant="soft" :title="t('aiSettings.catalogUnavailable')" :description="catalogError" :ui="{ description: 'text-inherit opacity-100' }" icon="i-heroicons-exclamation-triangle">
        <template #description>
          {{ catalogError }}
          <details v-if="catalogDiagnostic" class="mt-2">
            <summary class="cursor-pointer">{{ t('aiSettings.testing.diagnostics') }}</summary>
            <pre class="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-all text-xs">{{ catalogDiagnostic }}</pre>
          </details>
        </template>
        <template #actions>
          <UButton data-testid="ai-settings-retry-catalog" color="neutral" variant="outline" size="sm" @click="refreshCatalog">{{ t('common.retry') }}</UButton>
        </template>
      </UAlert>

      <div class="ai-settings-layout">
        <aside class="ai-settings-navigation ai-settings-panel min-w-0 rounded-xl border p-3">
          <h2 class="ai-settings-copy mb-3 px-2 text-xs font-medium">{{ t('aiSettings.modalityTabsLabel') }}</h2>
          <nav role="tablist" aria-orientation="vertical" :aria-label="t('aiSettings.modalityTabsLabel')" class="space-y-1" @keydown="navigateTabs($event, 'modality')">
            <UButton
              v-for="item in modalities" :key="item.value"
              :id="`ai-settings-modality-tab-${item.value}`" :data-testid="`ai-settings-modality-${item.value}`"
              role="tab" :aria-selected="modality === item.value" aria-controls="ai-settings-workspace"
              :tabindex="modality === item.value ? 0 : -1"
              color="neutral" variant="ghost" :icon="item.icon"
              class="ai-settings-modality w-full justify-start px-2 py-2.5"
              @click="modality = item.value"
            >{{ item.label }}</UButton>
          </nav>
        </aside>

        <section id="ai-settings-workspace" :aria-label="currentTitle" class="ai-settings-editor ai-settings-panel min-w-0 overflow-hidden rounded-xl border">
          <header data-testid="ai-settings-summary" class="flex flex-wrap items-start justify-between gap-3 border-b border-default p-4">
              <div class="min-w-0">
                <h2 class="font-semibold text-highlighted">{{ currentTitle }}</h2>
                <p class="ai-settings-copy mt-1 break-words text-sm">{{ selectedModel?.label || t('aiSettings.states.not_configured') }}<template v-if="selectedProvider"> · {{ selectedProvider.name }}</template></p>
              </div>
              <span data-testid="ai-settings-status" role="status" :class="`ai-settings-status ai-settings-status--${statusTone}`" class="inline-flex max-w-full items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium">
                <UIcon :name="statusIcon" class="size-4 shrink-0" />
                {{ statusLabel }}
              </span>
              <UButton v-if="!wideWorkspace" data-testid="ai-settings-open-test" icon="i-heroicons-beaker" color="neutral" variant="outline" size="sm" @click="testDrawerOpen = true">{{ t('aiSettings.workspace.openTest') }}</UButton>
          </header>

          <nav role="tablist" :aria-label="t('aiSettings.sections.label')" class="grid grid-cols-2 border-b border-default px-4" @keydown="navigateTabs($event, 'section')">
            <UButton
              v-for="section in ['model', 'parameters']" :key="section"
              :id="`ai-settings-tab-${section}`" :data-testid="`ai-settings-section-${section}`"
              role="tab" :aria-selected="activeSection === section" :aria-controls="`ai-settings-panel-${section}`"
              :tabindex="activeSection === section ? 0 : -1"
              color="neutral" variant="ghost" class="ai-settings-section justify-center rounded-none px-1 py-3 text-xs sm:px-3 sm:text-sm"
              @click="activeSection = section"
            >{{ t(`aiSettings.sections.${section}`) }}</UButton>
          </nav>

          <div class="p-4">
            <div :id="`ai-settings-panel-${activeSection}`" role="tabpanel" :aria-labelledby="`ai-settings-tab-${activeSection}`" class="min-w-0">
              <fieldset v-if="activeSection === 'model'" :disabled="Boolean(testState.attempt) || Boolean(testing)" class="min-w-0 space-y-5">
                <p class="ai-settings-copy text-sm">{{ t('aiSettings.workspace.modelDescription') }}</p>
                <div class="ai-settings-fields grid gap-4">
                  <UFormField :label="t('aiSettings.provider')"><USelect data-testid="ai-settings-provider" v-model="providerSelection" :items="providerOptions" :ui="{ placeholder: 'ai-settings-selection-placeholder' }" :placeholder="t('aiSettings.notSelected')" :loading="providersLoading" :disabled="providersLoading || (powerxLLMCatalog && Boolean(catalogError))" class="w-full" /></UFormField>
                  <UFormField v-if="appOptions.length" :label="t('aiSettings.app')"><USelect v-model="draft.app" :items="appOptions" class="w-full" /></UFormField>
                  <div class="min-w-0 space-y-2">
                    <div class="flex flex-wrap items-center justify-between gap-2">
                      <span class="text-sm font-medium text-highlighted">{{ t('aiSettings.model') }}</span>
                      <UButton v-if="catalogSource === 'local'" data-testid="ai-settings-online-models" color="neutral" variant="outline" size="xs" icon="i-heroicons-cloud-arrow-down" :disabled="Boolean(onlineBlocked) || modelsLoading || providersLoading" @click="onlineOpen = true">{{ t('aiSettings.online.update') }}</UButton>
                      <UButton v-else data-testid="ai-settings-refresh-models" color="neutral" variant="outline" size="xs" icon="i-heroicons-arrow-path" :loading="modelsLoading" :disabled="!selectedProvider || modelsLoading || providersLoading" @click="loadCatalogModels(true)">{{ t('aiSettings.online.refreshCore') }}</UButton>
                    </div>
                    <UFormField :aria-label="t('aiSettings.model')"><USelectMenu :aria-label="t('aiSettings.model')" data-testid="ai-settings-model" :key="modelSelectorKey" v-model="modelSelection" :items="modelOptions" value-key="value" :placeholder="t('aiSettings.notSelected')" :loading="modelsLoading" :disabled="providersLoading || modelsLoading || !selectedProvider || (powerxLLMCatalog && Boolean(catalogError))" :search-input="{ placeholder: t('aiSettings.searchModel') }" :ui="{ content: 'ai-settings-model-menu', empty: 'text-toned', placeholder: 'ai-settings-selection-placeholder' }" class="w-full"><template #empty="{ searchTerm }">{{ t(searchTerm ? 'aiSettings.noMatchingModels' : 'aiSettings.noModelsInCatalog') }}</template></USelectMenu></UFormField>
                    <p v-if="catalogSource === 'local' && !modelsLoading && !catalogError && loadedCatalogScope === modelCatalogScope() && draft.modelKey && !catalogModels.some(item => item.id === draft.modelKey)" class="ai-settings-copy text-xs">{{ t('aiSettings.online.selectedMissing') }}</p>
                    <p v-if="catalogSource === 'local' && onlineBlocked" class="ai-settings-copy text-xs">{{ t(`aiSettings.online.${onlineBlocked}`) }}</p>
                    <template v-if="catalogSource === 'powerx'"><p class="ai-settings-copy text-xs">{{ t('aiSettings.online.coreHint') }}</p><UButton data-testid="ai-settings-open-core" :to="coreSettingsURL || undefined" target="_blank" rel="noopener noreferrer" color="neutral" variant="link" size="xs" :disabled="!coreSettingsURL">{{ t('aiSettings.online.openCore') }}</UButton><p v-if="!coreSettingsURL" class="ai-settings-copy text-xs">{{ t('aiSettings.online.coreAddressRequired') }}</p></template>
                  </div>
                  <UFormField v-if="catalogSource === 'local'" :label="t('aiSettings.baseURL')" class="ai-settings-field-wide"><UInput data-testid="ai-settings-endpoint" v-model="draft.endpoint" @blur="localConnectionBlur" :placeholder="selectedProvider?.auth?.defaults?.base_url || ''" class="w-full" /></UFormField>
                  <p v-if="powerxLLMCatalog" data-testid="ai-settings-core-managed" class="ai-settings-copy ai-settings-field-wide text-xs">{{ t('aiSettings.coreConnectionManaged') }}</p>
                  <UFormField v-if="requiresCredentialReference" :label="t('aiSettings.credentialReference')" class="ai-settings-field-wide" :description="t('aiSettings.credentialReferenceDescription')"><UInput data-testid="ai-settings-credential-ref" v-model="draft.credentialRef" @blur="localConnectionBlur" autocomplete="off" class="w-full" /></UFormField>
                </div>
              </fieldset>

              <fieldset v-else-if="activeSection === 'parameters'" :disabled="Boolean(testState.attempt) || Boolean(testing)" class="min-w-0 space-y-5">
                <p class="ai-settings-copy text-sm">{{ t('aiSettings.workspace.parametersDescription') }}</p>
                <div v-if="modality === 'llm'" class="ai-settings-fields grid gap-4">
                  <UFormField :label="t('aiSettings.temperature')"><UInput v-model.number="draft.temperature" type="number" min="0" max="2" step="0.1" class="w-full" /></UFormField>
                  <UFormField :label="t('aiSettings.maxTokens')"><UInput v-model.number="draft.maxTokens" type="number" min="1" class="w-full" /></UFormField>
                  <UFormField :label="t('aiSettings.topP')"><UInput v-model.number="draft.topP" type="number" min="0" max="1" step="0.1" class="w-full" /></UFormField>
                  <UCheckbox v-model="draft.stream" :label="t('aiSettings.stream')" class="self-end" />
                </div>
                <div v-else-if="mediaFields.length" class="ai-settings-fields grid gap-4">
                  <UFormField v-for="[key, schema] in mediaFields" :key="key" :label="t('aiSettings.mediaParameter', { parameter: key })">
                    <USelect v-if="schema.enum" :data-testid="`ai-media-param-${key}`" v-model="draft.mediaParams[key]" :items="schema.enum.map(value => ({ label: String(value), value }))" class="w-full" />
                    <UCheckbox v-else-if="schema.type === 'boolean'" :data-testid="`ai-media-param-${key}`" v-model="draft.mediaParams[key]" />
                    <UInput v-else-if="schema.type === 'number' || schema.type === 'integer'" :data-testid="`ai-media-param-${key}`" v-model.number="draft.mediaParams[key]" type="number" :min="schema.minimum" :max="schema.maximum" :step="schema.type === 'integer' ? 1 : 'any'" class="w-full" />
                    <UInput v-else :data-testid="`ai-media-param-${key}`" v-model="draft.mediaParams[key]" class="w-full" />
                  </UFormField>
                </div>
                <p v-else class="ai-settings-copy text-sm">{{ t('aiSettings.modalityParametersPending') }}</p>
              </fieldset>
            </div>
          </div>
        </section>

        <AISettingsTestPanel
          v-if="wideWorkspace" v-bind="testPanelProps" v-on="testPanelActions"
          v-model:prompt="testState.prompt" v-model:image="testState.image" v-model:image-type="testState.imageType"
        />
        <USlideover
          v-else v-model:open="testDrawerOpen"
          :title="t('aiSettings.workspace.testPanelTitle')" :description="t('aiSettings.workspace.testDescription')"
          :close="{ 'aria-label': t('aiSettings.workspace.closeTests') }"
          :ui="{ content: 'ai-settings-test-sheet w-full max-w-md', body: 'p-4' }"
        >
          <template #body>
            <AISettingsTestPanel
              embedded v-bind="testPanelProps" v-on="testPanelActions"
              v-model:prompt="testState.prompt" v-model:image="testState.image" v-model:image-type="testState.imageType"
            />
          </template>
        </USlideover>
      </div>
      <UAlert v-if="onlineDiagnostic" class="ai-settings-catalog-alert" data-testid="ai-online-page-error" role="alert" color="warning" variant="soft" :title="t('aiSettings.online.failed')" :description="t('aiSettings.online.failedHint')" :ui="{ description: 'text-inherit opacity-100' }"><template #description><p>{{ t('aiSettings.online.failedHint') }}</p><details class="mt-2"><summary>{{ t('aiSettings.testing.diagnostic') }}</summary><pre class="mt-2 max-h-32 overflow-auto whitespace-pre-wrap break-all text-xs">{{ onlineDiagnostic }}</pre></details></template></UAlert>
      <AIOnlineModelCatalog v-if="onlineOpen" :key="onlineScope" :input="onlineInput" :provider-name="selectedProvider?.name || draft.provider" @close="onlineOpen = false" @updated="loadCatalogModels(true)" @failure="onlineDiagnostic = $event" @started="onlineDiagnostic = ''" />
      <p class="ai-settings-copy text-xs">{{ t('aiSettings.workspace.saveScope') }}</p>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import AISettingsTestPanel from './AISettingsTestPanel.vue';
import AIOnlineModelCatalog from './AIOnlineModelCatalog.vue';
import { useUserStore } from '~/stores/user';
import type { LocalOnlineCatalogInput } from '@artisan-cloud/plugin-framework-client';
import { createAIConnectionClient, type AIConnectionTestResult, type AIConnectionTestInput, type AIConnectionModality, isMediaTaskTerminal, validateMediaTask, type MediaTask, type MediaModelOperation } from '@artisan-cloud/plugin-framework-client';
import { settingsMediaCapabilities, settingsTestBlocked, settingsConnectionBlocked, settingsTestAttempt, type SettingsAttempt } from '~/utils/aiSettingsTesting';
import { useAIMediaApi } from '~/composables/api/useAIMedia';
import { useApiClient } from '~/composables/api/_client';
import { defaultProviderMode, normalizeProviderMode } from '~/composables/api/useProviderMode';

type Modality = 'llm' | 'vlm' | 'image' | 'embedding' | 'audio_tts' | 'audio_asr' | 'video' | 'model3d' | 'rerank';
type ModelProfile = { name?: string; model_key?: string; model?: string; provider?: string; endpoint?: string; modalities?: string[] };
type CatalogAuth = { scheme?: string; fields?: string[]; defaults?: Record<string, string> };
type CatalogProvider = { id: string; name: string; apps?: { id: string; name: string }[]; auth?: CatalogAuth };
type CatalogModel = { id: string; label: string; available?: boolean; operations?: Record<string, MediaModelOperation> };
// Pair accent backgrounds and text explicitly so light-mode labels and interaction states stay readable.
const primaryButtonClass = [
  'bg-primary-700 text-white hover:bg-primary-800 active:bg-primary-800 focus-visible:outline-primary-700',
  'disabled:bg-primary-700 aria-disabled:bg-primary-700 disabled:opacity-100 aria-disabled:opacity-100',
  'dark:bg-primary-400 dark:text-gray-950 dark:hover:bg-primary-300 dark:active:bg-primary-300',
  'dark:disabled:bg-primary-400 dark:aria-disabled:bg-primary-400 dark:focus-visible:outline-primary-400',
].join(' ');
const { t } = useI18n();
const toast = useToast();
const { client } = useApiClient();
const userStore = useUserStore();
const runtimeConfig = useRuntimeConfig();
const onlineOpen = ref(false), onlineDiagnostic = ref('');
const saving = ref(false);
const workspaceElement = ref<HTMLElement | null>(null);
const wideWorkspace = ref(false);
const testDrawerOpen = ref(false);
let workspaceObserver: ResizeObserver | undefined;
onMounted(() => {
  workspaceObserver = new ResizeObserver(([entry]) => {
    const rootFontSize = Number.parseFloat(getComputedStyle(document.documentElement).fontSize);
    // Match the 56rem container query, including customized browser font sizes.
    wideWorkspace.value = entry.contentRect.width >= 56 * rootFontSize;
    if (wideWorkspace.value) testDrawerOpen.value = false;
  });
  if (workspaceElement.value) workspaceObserver.observe(workspaceElement.value);
});
onBeforeUnmount(() => workspaceObserver?.disconnect());
const activeSection = ref('model');
function navigateTabs(event: KeyboardEvent, group: 'modality' | 'section') {
  const previous = group === 'modality' ? 'ArrowUp' : 'ArrowLeft';
  const next = group === 'modality' ? 'ArrowDown' : 'ArrowRight';
  if (![previous, next, 'Home', 'End'].includes(event.key)) return;
  event.preventDefault();
  const tabs = Array.from((event.currentTarget as HTMLElement).querySelectorAll<HTMLButtonElement>('[role="tab"]'));
  const current = tabs.findIndex(tab => tab.getAttribute('aria-selected') === 'true');
  const index = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (current + (event.key === next ? 1 : -1) + tabs.length) % tabs.length;
  tabs[index]?.click();
  tabs[index]?.focus();
}
const testing = ref<'connection' | 'quick' | ''>('');
const runtimeReady = ref(false);
const connectionProbes = ref({ local: false, powerx: false });
const connectionApi = createAIConnectionClient({ request: (path, options) => client(path, options) });
type ConnectionState = { result?: AIConnectionTestResult; failed: boolean; diagnostic: string };
const connectionStates = reactive<Record<string, ConnectionState>>({});
const connectionState = computed(() => connectionStates[JSON.stringify([contextKey(), draft.provider, catalogSource.value === 'local' ? draft.endpoint : ''])] ||= { failed: false, diagnostic: '' });
const mediaApi = useAIMediaApi();
type TestState = { prompt: string; image: string; imageType: string; text: string; passed: boolean; diagnostic: string; task?: MediaTask; attempt?: SettingsAttempt<Record<string, unknown>> };
const testStates = reactive<Record<string, TestState>>({});
const testState = computed(() => testStates[contextKey()] ||= { prompt: t('aiSettings.testing.defaultPrompt'), image: '', imageType: 'image_url', text: '', passed: false, diagnostic: '' });
const generation = computed(() => ['image', 'video', 'audio_tts'].includes(modality.value));
const imageTypeOptions = computed(() => ['image_url', 'image_base64'].map(value => ({ value, label: t(`aiSettings.testing.imageTypes.${value}`) })));

const profiles = ref<ModelProfile[]>([]);
const catalogProviders = ref<CatalogProvider[]>([]);
const catalogModels = ref<CatalogModel[]>([]);
const catalogError = ref('');
const catalogDiagnostic = ref('');
const catalogReason = ref('');
const providersLoading = ref(false);
const modelsLoading = ref(false);
const loadedProvidersScope = ref('');
const loadedCatalogScope = ref('');
const coreCatalogEnvironment = ref('');
let providersRequest = 0;
let modelsRequest = 0;
function modelCatalogScope() { return JSON.stringify([contextKey(), draft.provider, draft.app, ...(catalogSource.value === 'local' ? [draft.endpoint, draft.credentialRef] : [])]); }
const providerMode = ref(defaultProviderMode());
const environment = ref('development');
const catalogSource = ref<'local' | 'powerx'>('local');
const modality = ref<Modality>('llm');
const powerxLLMCatalog = computed(() => catalogSource.value === 'powerx' && modality.value === 'llm');
const draft = reactive({ provider: '', app: '', modelKey: '', endpoint: '', credentialRef: '', temperature: 0.7, maxTokens: 4096, topP: 1, stream: true, mediaParams: {} as Record<string, any> });
const onlineInput = computed<LocalOnlineCatalogInput>(() => ({ environment: environment.value as 'development' | 'production', modality: modality.value, provider: draft.provider, app: draft.app, endpoint: draft.endpoint, credential_ref: draft.credentialRef }));
const onlineScope = computed(() => JSON.stringify([userStore.currentTenantUuid, catalogSource.value, onlineInput.value]));
const onlineSupported = computed(() => ['ollama', 'openai', 'openrouter', 'deepseek', 'moonshot', 'huggingface', 'openai_compatible', 'qwen-cn', 'qwen-intl', 'anthropic', 'google'].includes(draft.provider));
const onlineBlocked = computed(() => catalogReason.value === 'AI_LOCAL_CATALOG_MIGRATION_REQUIRED' ? 'errors.AI_LOCAL_CATALOG_MIGRATION_REQUIRED' : !userStore.isCurrentTenantAdmin ? 'adminRequired' : !onlineSupported.value ? 'unsupported' : !draft.endpoint.trim() ? 'endpointRequired' : '');
const coreSettingsURL = computed(() => {
  try { const base = String(runtimeConfig.public.powerxConsoleBase || ''); if (!base) return ''; const url = new URL(base); if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || url.search || url.hash) return ''; return new URL('settings/ai', url.href.replace(/\/$/, '') + '/').href; } catch { return ''; }
});
const environmentOptions = computed(() => [{ label: t('aiSettings.environments.development'), value: 'development' }, { label: t('aiSettings.environments.production'), value: 'production' }]);
const catalogSourceOptions = computed(() => [{ label: t('aiSettings.catalogSources.local'), value: 'local' }, { label: t('aiSettings.catalogSources.powerx'), value: 'powerx' }]);
const modalities = computed(() => [{ value: 'llm' as const, label: t('aiSettings.modalities.llm'), icon: 'i-heroicons-bars-3-bottom-left' }, { value: 'vlm' as const, label: t('aiSettings.modalities.vlm'), icon: 'i-heroicons-eye' }, { value: 'image' as const, label: t('aiSettings.modalities.image'), icon: 'i-heroicons-photo' }, { value: 'embedding' as const, label: t('aiSettings.modalities.embedding'), icon: 'i-heroicons-square-3-stack-3d' }, { value: 'audio_tts' as const, label: t('aiSettings.modalities.audioTTS'), icon: 'i-heroicons-speaker-wave' }, { value: 'audio_asr' as const, label: t('aiSettings.modalities.audioASR'), icon: 'i-heroicons-microphone' }, { value: 'video' as const, label: t('aiSettings.modalities.video'), icon: 'i-heroicons-video-camera' }, { value: 'model3d' as const, label: t('aiSettings.modalities.model3d'), icon: 'i-heroicons-cube' }, { value: 'rerank' as const, label: t('aiSettings.modalities.rerank'), icon: 'i-heroicons-arrows-up-down' }]);
const currentTitle = computed(() => modalities.value.find((item) => item.value === modality.value)?.label || '');
const modeLabel = computed(() => providerMode.value.mode === 'delegated' ? t('providerMode.delegated') : t('providerMode.local'));
const providerOptions = computed(() => {
  if (powerxLLMCatalog.value && loadedProvidersScope.value !== contextKey()) return [];
  const options = catalogProviders.value.map((item) => ({ label: item.name, value: item.id }));
  if (!powerxLLMCatalog.value && draft.provider && !options.some((item) => item.value === draft.provider)) options.unshift({ label: draft.provider, value: draft.provider });
  return options;
});
const selectedProvider = computed(() => catalogProviders.value.find((item) => item.id === draft.provider));
const appOptions = computed(() => (selectedProvider.value?.apps || []).map((item) => ({ label: item.name, value: item.id })));
// Labels are presentation only; opaque model keys remain the persisted values.
const mediaCapability = computed(() => settingsMediaCapabilities[modality.value]);
const modelOptions = computed(() => {
  if (powerxLLMCatalog.value && loadedCatalogScope.value !== modelCatalogScope()) return [];
  const options = catalogModels.value.map((item) => ({ label: item.label, value: item.id, disabled: item.available === false || (catalogSource.value === 'powerx' && Boolean(mediaCapability.value) && item.operations?.[mediaCapability.value!]?.available !== true) }));
  if (!powerxLLMCatalog.value && draft.modelKey && !options.some((item) => item.value === draft.modelKey)) options.unshift({ label: draft.modelKey, value: draft.modelKey, disabled: true });
  return options;
});
// USelect caches its option collection while it is open. Recreate it when the
// catalog scope changes so a local profile value never narrows the Core list.
const modelSelectorKey = computed(() => JSON.stringify([modelCatalogScope(), catalogModels.value.map(item => [item.id, item.label])]));
const selectedModel = computed(() => {
  if (powerxLLMCatalog.value && loadedCatalogScope.value !== modelCatalogScope()) return undefined;
  const catalogModel = catalogModels.value.find((item) => item.id === draft.modelKey);
  if (catalogModel) return catalogModel;
  if (powerxLLMCatalog.value) return undefined;
  const runtimeModel = profiles.value.find((item) => (item.model_key || item.name) === draft.modelKey);
  if (runtimeModel) return { id: draft.modelKey, label: runtimeModel.model || runtimeModel.name || draft.modelKey };
  return draft.modelKey ? { id: draft.modelKey, label: draft.modelKey } : undefined;
});
const statusLabel = computed(() => testing.value ? t('aiSettings.testing.inProgress')
  : connectionState.value.failed ? t('aiSettings.testing.connectionFailed')
  : testState.value.diagnostic ? t('aiSettings.workspace.trialFailed')
  : testState.value.task ? t(`aiSettings.testing.taskStatus.${testState.value.task.status}`)
  : connectionState.value.result ? t(connectionState.value.result.verification.authentication === 'not_required' ? 'aiSettings.testing.connectionNoAuth' : 'aiSettings.testing.connectionVerified')
  : testState.value.passed ? t('aiSettings.workspace.trialPassed') : t('aiSettings.states.not_verified'));
const statusTone = computed(() => {
  if (connectionState.value.failed || testState.value.diagnostic || testState.value.task?.status === 'failed') return 'error';
  if (testState.value.task?.status === 'cancelled') return 'neutral';
  if (testState.value.task && !isMediaTaskTerminal(testState.value.task)) return 'info';
  return connectionState.value.result || testState.value.passed || testState.value.task?.status === 'succeeded' ? 'success' : testing.value ? 'info' : 'warning';
});
const statusIcon = computed(() => ({ error: 'i-heroicons-exclamation-circle', success: 'i-heroicons-check-circle', info: 'i-heroicons-clock', warning: 'i-heroicons-exclamation-circle', neutral: 'i-heroicons-minus-circle' })[statusTone.value]);
const coreSelectionInvalid = computed(() => powerxLLMCatalog.value && (providersLoading.value || modelsLoading.value || Boolean(catalogError.value) || loadedProvidersScope.value !== contextKey() || loadedCatalogScope.value !== modelCatalogScope() || !selectedProvider.value || !selectedModel.value));
const localSelectionInvalid = computed(() => catalogSource.value === 'local' && (modelsLoading.value || Boolean(catalogError.value) || loadedCatalogScope.value !== modelCatalogScope() || !catalogModels.value.some(item => item.id === draft.modelKey)));
const providerSelection = computed<string | undefined>({
  get: () => powerxLLMCatalog.value && (loadedProvidersScope.value !== contextKey() || !selectedProvider.value) ? undefined : draft.provider || undefined,
  set: value => { if (draft.provider !== (value ?? '')) draft.credentialRef = ''; draft.provider = value ?? ''; },
});
const modelSelection = computed<string | undefined>({
  get: () => powerxLLMCatalog.value && !selectedModel.value ? undefined : draft.modelKey || undefined,
  set: value => { draft.modelKey = value ?? ''; },
});
const requiresCredentialReference = computed(() => catalogSource.value === 'local' && Boolean(selectedProvider.value?.auth?.fields?.some((field) => field !== 'base_url')));
function reset() { const selected = profiles.value[0]; draft.provider = selected?.provider || ''; draft.app = ''; draft.modelKey = selected?.model_key || selected?.name || ''; draft.endpoint = selected?.endpoint || ''; draft.credentialRef = ''; draft.temperature = 0.7; draft.maxTokens = 4096; draft.topP = 1; draft.stream = true;  }
function requestBody() { return { source: catalogSource.value, environment: environment.value, modality: modality.value, provider: draft.provider, app: draft.app, model_key: draft.modelKey, endpoint: powerxLLMCatalog.value ? '' : draft.endpoint, credential_ref: powerxLLMCatalog.value ? '' : draft.credentialRef, parameters: modality.value === 'llm' ? { temperature: draft.temperature, max_tokens: draft.maxTokens, top_p: draft.topP, stream: draft.stream } : mediaDefaults() }; }
const mediaOperation = computed(() => {
  const capability = mediaCapability.value;
  const model = catalogModels.value.find(item => item.id === draft.modelKey);
  return capability ? model?.operations?.[capability] : undefined;
});
const mediaFields = computed(() => Object.entries(mediaOperation.value?.parameter_schema.properties || {}));
let mediaParameterModel = '';
watch([() => draft.modelKey, mediaOperation] as const, ([key, operation]) => {
  draft.mediaParams = key === mediaParameterModel ? { ...operation?.defaults, ...draft.mediaParams } : { ...operation?.defaults };
  mediaParameterModel = key;
});
function mediaDefaults() { return { ...draft.mediaParams }; }
async function saveSettings() { if (coreSelectionInvalid.value || localSelectionInvalid.value) return; saving.value = true; try { await client('admin/ai-settings/local-setting', { method: 'PUT', body: requestBody() }); toast.add({ title: t('common.success'), description: t('aiSettings.saveSucceeded'), color: 'success' }); } catch { toast.add({ title: t('common.error'), description: t('aiSettings.saveFailed'), color: 'error' }); } finally { saving.value = false; } }
const blockedReason = computed(() => settingsTestBlocked({
  ready: runtimeReady.value, modality: modality.value, delegated: providerMode.value.mode === 'delegated', source: catalogSource.value,
  modelKey: draft.modelKey, provider: draft.provider, endpoint: draft.endpoint, prompt: testState.value.prompt,
  image: testState.value.image, imageType: testState.value.imageType, catalogFailed: Boolean(catalogError.value) || loadedCatalogScope.value !== modelCatalogScope(),
  available: catalogModels.value.some(item => item.id === draft.modelKey && item.available !== false), operation: mediaOperation.value,
  localVision: profiles.value.some(item => (item.model_key || item.name) === draft.modelKey && item.modalities?.includes('vlm')),
}));
const connectionBlockedReason = computed(() => settingsConnectionBlocked({
  ready: runtimeReady.value, source: catalogSource.value, provider: draft.provider, endpoint: draft.endpoint,
  adapterAvailable: connectionProbes.value[catalogSource.value],
  providersReady: loadedProvidersScope.value === contextKey(),
}));
const testPanelProps = computed(() => ({
  currentTitle: currentTitle.value,
  providerLabel: selectedProvider.value?.name || t('aiSettings.notSelected'),
  modelLabel: selectedModel.value?.label || t('aiSettings.notSelected'),
  modality: modality.value, generation: generation.value, blockedReason: blockedReason.value,
  connectionBlockedReason: connectionBlockedReason.value, connection: connectionState.value, testing: testing.value, imageTypeOptions: imageTypeOptions.value, state: testState.value,
}));
const testPanelActions = {
  connection: testConnection, quick: quickCall,
  refresh: () => updateTask(false), cancel: () => updateTask(true), 'new-attempt': newTestAttempt,
};
function diagnostics(error: any) {
  const data = error?.data || error?.response?._data || {};
  return JSON.stringify({ status: error?.statusCode || error?.status || error?.response?.status, error_code: data.error_code,
    reason_code: data.reason_code || error?.reasonCode, trace_id: data.trace_id, request_id: data.request_id, details: data.details });
}
async function runTest() {
  if (blockedReason.value || testing.value || testState.value.task) return;
  const state = testState.value;
  const selectedModality = modality.value;
  const inputs: Array<Record<string, string>> = [{ role: 'user', type: 'text', content: state.prompt }];
  if (selectedModality === 'vlm') inputs.push(state.imageType === 'image_url' ? { role: 'user', type: state.imageType, url: state.image } : { role: 'user', type: state.imageType, content: state.image });
  const body = { ...requestBody(), inputs };
  testing.value = 'quick'; state.text = ''; state.diagnostic = ''; state.passed = false;
  try {
    let submission: Record<string, unknown> = body;
    if (generation.value) {
      state.attempt = settingsTestAttempt(state.attempt, body, () => crypto.randomUUID());
      submission = state.attempt.body;
    }
    const out: any = await client('admin/ai-settings/quick-call', { method: 'POST', body: submission });
    const data = out?.data;
    if (['image', 'video', 'audio_tts'].includes(selectedModality)) {
      const task = validateMediaTask(data);
      const capability = settingsMediaCapabilities[selectedModality];
      if (task.model_key !== body.model_key || task.capability !== capability) throw { reasonCode: 'AI_TASK_RESPONSE_INVALID' };
      state.task = task; state.text = t('aiSettings.testing.taskAccepted');
    } else {
      if (!data || typeof data.text !== 'string' || (selectedModality === 'vlm' && (data.type !== 'analysis' || data.contract_version !== 2))) throw { reasonCode: 'AI_RESPONSE_INVALID' };
      state.text = data.text; state.passed = true;
    }
  } catch (error) { state.text = t('aiSettings.testing.failed'); state.diagnostic = diagnostics(error); }
  finally { testing.value = ''; }
}
async function testConnection() {
  if (connectionBlockedReason.value || testing.value) return;
  const state = connectionState.value;
  const input: AIConnectionTestInput = catalogSource.value === 'powerx'
    ? { source: 'powerx', modality: modality.value as AIConnectionModality, provider: draft.provider }
    : { source: 'local', modality: modality.value as AIConnectionModality, provider: draft.provider, endpoint: draft.endpoint, environment: environment.value as 'development' | 'production' };
  testing.value = 'connection'; state.failed = false; state.result = undefined; state.diagnostic = '';
  try { state.result = await connectionApi.testConnection(input); }
  catch (error) { state.failed = true; state.diagnostic = diagnostics(error); }
  finally { testing.value = ''; }
}
function quickCall() { return runTest(); }
function newTestAttempt() { if (!testing.value && (!testState.value.task || isMediaTaskTerminal(testState.value.task))) { testState.value.attempt = undefined; testState.value.task = undefined; testState.value.text = ''; testState.value.diagnostic = ''; testState.value.passed = false; } }
async function updateTask(cancel: boolean) {
  const state = testState.value, task = state.task;
  if (!task || testing.value) return;
  testing.value = 'quick';
  try {
    const next = cancel ? await mediaApi.cancelTask(task.task_uuid) : await mediaApi.getTask(task.task_uuid);
    if (next.model_key !== task.model_key || next.capability !== task.capability) throw { reasonCode: 'AI_TASK_RESPONSE_INVALID' };
    if (next.revision > task.revision) state.task = next;
    state.diagnostic = '';
  } catch (error) { state.diagnostic = diagnostics(error); state.text = t('aiSettings.testing.failed'); }
  finally { testing.value = ''; }
}
async function loadSavedSetting() { const scope = contextKey(); try { const out: any = await client('admin/ai-settings/local-setting', { method: 'GET', query: { environment: environment.value, modality: modality.value, source: catalogSource.value }, silentAuthError: true }); if (scope !== contextKey()) return; const item = out?.data ?? out; if (!item?.model_key) { draft.provider = ''; draft.app = ''; draft.modelKey = ''; draft.endpoint = ''; draft.credentialRef = ''; draft.temperature = 0.7; draft.maxTokens = 4096; draft.topP = 1; draft.stream = true; return; } mediaParameterModel = item.model_key; draft.mediaParams = { ...item.parameters }; draft.provider = item.provider || ''; draft.app = item.app || ''; draft.modelKey = item.model_key; draft.endpoint = item.endpoint || ''; draft.credentialRef = item.credential_ref || ''; const params = item.parameters || {}; if (modality.value === 'llm') { draft.temperature = Number(params.temperature ?? 0.7); draft.maxTokens = Number(params.max_tokens ?? 4096); draft.topP = Number(params.top_p ?? 1); draft.stream = params.stream !== false; } } catch { /* profile is optional until first save */ } }
function settingsCatalogEnvelope(data: any) {
  return data?.contract_version === 1 && data?.catalog_kind === 'ai_settings' && data?.modality === 'llm' && typeof data?.env === 'string' && data.env.trim();
}
function catalogFailure(error: unknown) {
  const failure = error as any, data = failure?.data || failure?.response?._data || {};
  catalogReason.value = data.reason_code || data.error_code || failure?.reasonCode || '';
  catalogError.value = t(powerxLLMCatalog.value ? 'aiSettings.coreCatalogUnavailableDescription' : 'aiSettings.catalogUnavailableDescription');
  if (catalogSource.value === 'local' && catalogReason.value === 'AI_LOCAL_CATALOG_MIGRATION_REQUIRED') catalogError.value = t('aiSettings.online.errors.AI_LOCAL_CATALOG_MIGRATION_REQUIRED');
  catalogDiagnostic.value = diagnostics(error);
}
async function loadCatalog() {
  const scope = contextKey(), request = ++providersRequest;
  ++modelsRequest;
  providersLoading.value = true;
  modelsLoading.value = false;
  loadedProvidersScope.value = '';
  loadedCatalogScope.value = '';
  catalogProviders.value = [];
  catalogModels.value = [];
  coreCatalogEnvironment.value = '';
  try {
    const out: any = await client('admin/ai-settings/catalog/providers', { method: 'GET', query: { modality: modality.value, source: catalogSource.value }, silentAuthError: true });
    if (scope !== contextKey() || request !== providersRequest) return false;
    const data = out?.data ?? out;
    const items = powerxLLMCatalog.value ? data?.providers : data?.items;
    if (!Array.isArray(items) || items.some(item => typeof item?.id !== 'string' || !item.id.trim() || typeof item?.name !== 'string' || !item.name.trim()) || (powerxLLMCatalog.value && !settingsCatalogEnvelope(data))) throw { reasonCode: 'AI_SETTINGS_CATALOG_RESPONSE_INVALID' };
    catalogProviders.value = items;
    if (powerxLLMCatalog.value) coreCatalogEnvironment.value = data.env;
    loadedProvidersScope.value = scope;
    if (!items.some(item => item.id === draft.provider)) draft.provider = items[0]?.id || '';
    return true;
  } catch (error) {
    if (scope !== contextKey() || request !== providersRequest) return false;
    catalogFailure(error);
    return false;
  } finally {
    if (request === providersRequest) providersLoading.value = false;
  }
}
function localConnectionBlur(event: FocusEvent) {
  const target = event.relatedTarget as HTMLElement | null;
  if (catalogSource.value !== 'local' || loadedCatalogScope.value === modelCatalogScope() || target?.closest('[data-testid="ai-settings-online-models"]')) return;
  void loadCatalogModels(true);
}
async function loadCatalogModels(preserve = false) {
  if (catalogSource.value === 'local' && !draft.endpoint) draft.endpoint = selectedProvider.value?.auth?.defaults?.base_url || '';
  const scope = modelCatalogScope(), request = ++modelsRequest;
  if (!preserve) { loadedCatalogScope.value = ''; catalogModels.value = []; }
  modelsLoading.value = false;
  if (!draft.provider || loadedProvidersScope.value !== contextKey()) return;
  modelsLoading.value = true;
  try {
    const out: any = await client('admin/ai-settings/catalog/models', { method: 'GET', query: { modality: modality.value, provider: draft.provider, app: draft.app, source: catalogSource.value, ...(catalogSource.value === 'local' ? { environment: environment.value, endpoint: draft.endpoint, credential_ref: draft.credentialRef } : {}) }, silentAuthError: true });
    if (scope !== modelCatalogScope() || request !== modelsRequest) return;
    const data = out?.data ?? out;
    if (!Array.isArray(data?.items)) throw { reasonCode: 'AI_SETTINGS_CATALOG_RESPONSE_INVALID' };
    if (powerxLLMCatalog.value) {
      if (!settingsCatalogEnvelope(data) || data.env !== coreCatalogEnvironment.value || data.provider !== draft.provider || data.app !== draft.app || data.items.some((item: any) => !item.model_key || !item.label || item.provider !== draft.provider)) throw { reasonCode: 'AI_SETTINGS_CATALOG_RESPONSE_INVALID' };
      catalogModels.value = data.items.map((item: any) => ({ ...item, id: item.model_key }));
    } else if (catalogSource.value === 'powerx' && modality.value !== 'llm') {
      if (data.contract_version !== 2) throw { reasonCode: 'AI_CATALOG_RESPONSE_INVALID' };
      catalogModels.value = data.items.map((item: any) => ({ ...item, id: item.model_key }));
    } else catalogModels.value = data.items;
    if (!preserve && !catalogModels.value.some(item => item.id === draft.modelKey)) draft.modelKey = powerxLLMCatalog.value ? '' : catalogModels.value.find(item => item.available !== false)?.id || '';
    catalogError.value = '';
    catalogDiagnostic.value = '';
    catalogReason.value = '';
    loadedCatalogScope.value = scope;
    if (catalogSource.value === 'local' && !draft.endpoint) draft.endpoint = selectedProvider.value?.auth?.defaults?.base_url || '';
  } catch (error) {
    if (scope !== modelCatalogScope() || request !== modelsRequest) return;
    catalogFailure(error);
  } finally {
    if (request === modelsRequest) modelsLoading.value = false;
  }
}
async function refreshCatalog() {
  catalogReason.value = '';
  catalogError.value = '';
  catalogDiagnostic.value = '';
  if (await loadCatalog()) await loadCatalogModels();
}
const contexts = new Map<string, typeof draft>();
let activeContext = '';
function contextKey() { return `${userStore.currentTenantUuid || ''}:${catalogSource.value}:${environment.value}:${modality.value}`; }
async function refreshProfile() {
  const key = contextKey();
  if (activeContext && activeContext !== key) contexts.set(activeContext, { ...draft, mediaParams: { ...draft.mediaParams } });
  activeContext = key;
  const saved = contexts.get(key);
  if (saved) { mediaParameterModel = saved.modelKey; Object.assign(draft, { ...saved, mediaParams: { ...saved.mediaParams } }); }
  else await loadSavedSetting();
  if (key === contextKey()) await refreshCatalog();
}
watch(onlineScope, () => { onlineOpen.value = false; onlineDiagnostic.value = ''; });
watch([modality, catalogSource, environment, () => userStore.currentTenantUuid], () => { void refreshProfile(); });
watch(() => draft.provider, () => { draft.app = ''; const endpoint = selectedProvider.value?.auth?.defaults?.base_url; if (endpoint) draft.endpoint = endpoint; void loadCatalogModels(); });
watch(() => draft.app, () => { void loadCatalogModels(); });

onMounted(async () => { try { const [mode, modelResponse] = await Promise.all([client<any>('admin/ai-settings/mode', { method: 'GET', silentAuthError: true }), client<any>('admin/ai-settings/model-profiles', { method: 'GET', silentAuthError: true })]); providerMode.value = normalizeProviderMode(mode?.data); connectionProbes.value = { local: mode?.data?.connection_probes?.local === true, powerx: mode?.data?.connection_probes?.powerx === true }; const data = modelResponse?.data ?? modelResponse ?? {}; profiles.value = Array.isArray(data.items) ? data.items : []; runtimeReady.value = true; reset(); await refreshProfile(); } catch { toast.add({ title: t('common.error'), description: t('aiSettings.loadFailed'), color: 'error' }); } });
</script>

<style>
.ai-settings,
.ai-settings-test-sheet {
  --ai-settings-panel-bg: var(--ui-bg);
  --ai-settings-field-bg: var(--ui-bg-muted);
  --ui-border: var(--ui-border-accented);
}

.dark .ai-settings,
.dark .ai-settings-test-sheet {
  --ai-settings-panel-bg: var(--ui-bg-elevated);
  --ai-settings-field-bg: var(--ui-bg);
}

.ai-settings {
  container: ai-settings / inline-size;
}

.ai-settings-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  align-items: start;
  gap: 1.25rem;
}

.ai-settings-navigation {
  display: none;
}

.ai-settings-editor {
  container: ai-settings-editor / inline-size;
}

.ai-settings-panel,
.ai-settings-test-sheet {
  background: var(--ai-settings-panel-bg);
  border-color: var(--ui-border);
  color: var(--ui-text);
}

.ai-settings :is(input:not([type='checkbox']), textarea, button[role='combobox']),
.ai-settings-test-panel :is(input:not([type='checkbox']), textarea, button[role='combobox']),
.ai-settings-test-result {
  background: var(--ai-settings-field-bg);
}

@container ai-settings (min-width: 42rem) {
  .ai-settings-layout {
    grid-template-columns: 10rem minmax(0, 1fr);
  }

  .ai-settings-navigation {
    display: block;
  }

  .ai-settings-modality-picker {
    display: none;
  }
}

@container ai-settings (min-width: 56rem) {
  .ai-settings-layout {
    grid-template-columns: 10rem minmax(0, 1fr) 17rem;
  }
}

@container ai-settings-editor (min-width: 28rem) {
  .ai-settings-fields {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .ai-settings-field-wide {
    grid-column: 1 / -1;
  }
}

.ai-settings-model-menu input::placeholder,
.ai-settings-selection-placeholder {
  color: var(--ui-text-toned);
  opacity: 1;
}

.ai-settings-copy {
  color: var(--ui-text-toned);
}

.ai-settings-modality[aria-selected='true'] {
  background: var(--ui-bg-elevated);
  color: var(--ui-text-highlighted);
  box-shadow: inset 3px 0 0 var(--ui-primary);
  font-weight: 600;
}

.ai-settings-section {
  border-bottom: 2px solid transparent;
}

.ai-settings-section[aria-selected='true'] {
  border-bottom-color: var(--ui-primary);
  color: var(--ui-text-highlighted);
  font-weight: 600;
}

.ai-settings-modality:focus-visible,
.ai-settings-section:focus-visible {
  outline: 2px solid var(--ui-text-highlighted);
  outline-offset: -3px;
}

.ai-settings-status--warning,
.ai-settings-catalog-alert {
  background: var(--ui-color-warning-50);
  color: var(--ui-color-warning-800);
}

.ai-settings-caution {
  color: var(--ui-color-warning-800);
}

.ai-settings-status--error {
  background: var(--ui-color-error-50);
  color: var(--ui-color-error-800);
}

.ai-settings-status--success {
  background: var(--ui-color-success-50);
  color: var(--ui-color-success-800);
}

.ai-settings-status--info {
  background: var(--ui-color-info-50);
  color: var(--ui-color-info-800);
}

.ai-settings-status--neutral {
  background: var(--ui-bg-elevated);
  color: var(--ui-text);
}

.dark .ai-settings-status--warning,
.dark .ai-settings-catalog-alert {
  background: var(--ui-color-warning-950);
  color: var(--ui-color-warning-200);
}

.dark .ai-settings-caution {
  color: var(--ui-color-warning-200);
}

.dark .ai-settings-status--error {
  background: var(--ui-color-error-950);
  color: var(--ui-color-error-200);
}

.dark .ai-settings-status--success {
  background: var(--ui-color-success-950);
  color: var(--ui-color-success-200);
}

.dark .ai-settings-status--info {
  background: var(--ui-color-info-950);
  color: var(--ui-color-info-200);
}

.ai-settings :disabled,
.ai-settings-test-panel :disabled {
  opacity: 1;
}
</style>
