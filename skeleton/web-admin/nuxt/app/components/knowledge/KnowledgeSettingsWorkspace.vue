<template>
  <div class="mx-auto w-full max-w-[1600px] space-y-6 px-5 py-6 lg:px-8">
    <section class="rounded-xl border border-gray-200 bg-white p-6 dark:border-gray-800 dark:bg-slate-900">
      <UBadge color="primary" variant="soft">{{ t('knowledgeSpaces.settings.badge') }}</UBadge>
      <div class="mt-3 flex flex-wrap items-start justify-between gap-4"><div><h1 class="text-2xl font-semibold">{{ t('knowledgeSpaces.settings.title') }}</h1><p class="mt-2 max-w-3xl text-sm text-gray-500">{{ t('knowledgeSpaces.settings.description') }}</p></div><UButton color="neutral" variant="soft" icon="i-heroicons-arrow-path" :loading="loading" @click="load">{{ t('common.refresh') }}</UButton></div>
      <UAlert class="mt-5" color="info" variant="soft" :description="t('knowledgeSpaces.ui.profileHelp')" />
    </section>
    <section class="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-gray-800 dark:bg-slate-900">
      <div class="border-b border-gray-200 p-5 dark:border-gray-800"><h2 class="font-semibold">{{ t('knowledgeSpaces.ui.strategyPackage') }}</h2><p class="mt-1 text-sm text-gray-500">{{ t('knowledgeSpaces.settings.localDescription') }}</p></div>
      <div v-if="loading" class="p-12 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
      <div v-else class="overflow-x-auto"><table class="min-w-full text-left text-sm"><thead class="border-b border-gray-200 text-gray-500 dark:border-gray-800"><tr><th class="px-5 py-3">{{ t('knowledgeSpaces.ui.strategyPackage') }}</th><th class="px-5 py-3">{{ t('knowledgeSpaces.ui.ingestionProfile') }}</th><th class="px-5 py-3">{{ t('knowledgeSpaces.ui.indexChannels') }}</th><th class="px-5 py-3">{{ t('knowledgeSpaces.ui.documentStatus') }}</th></tr></thead><tbody class="divide-y divide-gray-100 dark:divide-gray-800"><tr v-for="item in strategies" :key="item.key"><td class="px-5 py-4 font-mono font-medium">{{ item.key }}</td><td class="px-5 py-4 font-mono">{{ item.profile_key }}</td><td class="px-5 py-4"><span v-for="dependency in item.dependencies.index" :key="dependency" class="mr-2 inline-flex rounded bg-gray-100 px-2 py-1 font-mono text-xs dark:bg-slate-800">{{ dependency }}</span><span v-if="!item.dependencies.index.length" class="text-gray-500">-</span></td><td class="px-5 py-4"><UBadge :color="item.ready ? 'success' : 'warning'" variant="soft">{{ item.ready ? t('knowledgeSpaces.ui.strategyReady') : t('knowledgeSpaces.ui.strategyUnavailable', { requirements: item.missing_requirements.join(', ') }) }}</UBadge></td></tr></tbody></table></div>
    </section>
  </div>
</template>
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useLocalKnowledgeApi, type KnowledgeStrategyPackage } from '~/composables/api/useLocalKnowledge'
const { t } = useI18n(); const toast = useToast(); const api = useLocalKnowledgeApi(); const loading = ref(false); const strategies = ref<KnowledgeStrategyPackage[]>([])
async function load() { loading.value = true; try { strategies.value = (await api.strategyPackages()).items || [] } catch { toast.add({ title: t('common.error'), description: t('knowledgeSpaces.ui.loadFailed'), color: 'error' }) } finally { loading.value = false } }
onMounted(load)
</script>
