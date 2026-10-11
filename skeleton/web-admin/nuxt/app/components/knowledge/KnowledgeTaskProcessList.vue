<template>
  <section class="overflow-hidden rounded-xl border border-slate-200 bg-white dark:border-slate-700 dark:bg-[#0f192a]">
    <header class="space-y-4 border-b border-slate-200 p-4 sm:p-5 dark:border-slate-700">
      <div class="flex flex-wrap items-center justify-between gap-3"><h2 class="font-semibold text-gray-900 dark:text-[#fdfcff]">{{ title || t('knowledgeTaskProcess.title') }}</h2><div class="flex flex-wrap items-center gap-2"><UBadge v-for="mode in modes" :key="mode" color="neutral" variant="soft">{{ modeLabel(mode) }}</UBadge><UButton v-if="refreshable" size="sm" color="neutral" variant="outline" :loading="refreshing" @click="$emit('refresh')">{{ t('knowledgeTaskProcess.refreshStatus') }}</UButton></div></div>
      <dl v-if="!unavailable && (!refreshing || items.length)" class="grid grid-cols-3 gap-3 text-sm sm:grid-cols-6"><div v-for="key in countKeys" :key="key"><dt class="text-gray-600 dark:text-slate-300">{{ t(`knowledgeTaskProcess.counts.${key}`) }}</dt><dd class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ summary[key] }}</dd></div></dl>
      <div v-if="items.length && !unavailable"><div class="mb-2 flex justify-between gap-3 text-xs text-gray-600 dark:text-slate-300"><span>{{ t('knowledgeTaskProcess.ended', {count:summary.ended,total:summary.total}) }}</span><span>{{ summary.endedPercent }}%</span></div><UProgress :model-value="summary.endedPercent" :max="100" :color="summary.failed ? 'error' : summary.ended === summary.total ? 'success' : 'primary'" /></div>
      <p v-if="statusHint" class="text-xs text-gray-600 dark:text-slate-300">{{ statusHint }}</p>
    </header>
    <p v-if="unavailable" class="p-5 text-sm text-gray-600 dark:text-slate-300">{{ t('knowledgeTaskProcess.unavailable') }}</p>
    <p v-else-if="refreshing && !items.length" class="p-5 text-sm text-gray-600 dark:text-slate-300">{{ t('common.loading') }}</p>
    <p v-else-if="!items.length" class="p-5 text-sm text-gray-600 dark:text-slate-300">{{ t('knowledgeTaskProcess.empty') }}</p>
    <div v-else class="divide-y divide-slate-100 dark:divide-slate-800">
      <article v-for="item in items" :key="item.id" class="space-y-3 p-4 sm:p-5">
        <div class="flex flex-wrap items-center justify-between gap-3"><div class="min-w-0 flex-1"><p class="break-words font-medium text-gray-900 dark:text-[#fdfcff]">{{ item.title }}</p><div class="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-600 dark:text-slate-300"><span v-if="item.chunks !== undefined">{{ t('knowledgeTaskProcess.chunks', {count:item.chunks}) }}</span><span v-if="item.durationMs !== undefined">{{ t(`knowledgeTaskProcess.duration.${item.durationKind || 'execution'}`, {value:durationLabel(item.durationMs)}) }}</span></div></div><UBadge :color="phaseColor(item.phase)" variant="soft">{{ t(`knowledgeTaskProcess.phases.${item.phase}`) }}</UBadge><slot name="actions" :item="item" /></div>
        <div v-if="['reading','submitting','running'].includes(item.phase)" class="max-w-xl"><UProgress :model-value="taskProgress(item)" :max="100" size="xs" /><p class="mt-1 text-xs text-gray-600 dark:text-slate-300">{{ taskProgress(item) === null ? t('knowledgeTaskProcess.runningWithoutPercent') : t('knowledgeTaskProcess.progress', {count:taskProgress(item)}) }}</p></div>
        <p v-if="item.error" class="text-xs text-red-600 dark:text-red-300">{{ item.error }}</p><details v-if="item.traceId" class="text-xs text-gray-600 dark:text-slate-300"><summary class="cursor-pointer">{{ t('knowledgeTaskProcess.traceDetails') }}</summary><p class="mt-2 break-all">{{ item.traceId }}</p></details>
      </article>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { taskProgress, taskSummary, type TaskProcessItem, type TaskPhase } from '~/composables/knowledge/taskProcess'
const props = withDefaults(defineProps<{items:TaskProcessItem[];title?:string;statusHint?:string;refreshable?:boolean;refreshing?:boolean;unavailable?:boolean}>(),{refreshable:false,refreshing:false,unavailable:false})
defineEmits<{refresh:[]}>()
const {t,locale}=useI18n()
const summary=computed(()=>taskSummary(props.items))
const modes=computed(()=>[...new Set(props.items.map(item=>item.indexMode).filter((mode):mode is string=>Boolean(mode)))])
const countKeys=['total','ready','waiting','running','succeeded','failed'] as const
function phaseColor(phase:TaskPhase):'success'|'error'|'warning'|'neutral' {return phase==='succeeded'?'success':phase==='failed'?'error':['queued','accepted','blocked'].includes(phase)?'warning':'neutral'}
function modeLabel(mode:string){return ['lexical_chunks','local_vector'].includes(mode)?t(`knowledgeTaskProcess.modes.${mode}`):t('knowledgeTaskProcess.modes.other')}
function durationLabel(ms:number){const fmt=new Intl.NumberFormat(locale.value,{maximumFractionDigits:2});return ms<1000?t('knowledgeTaskProcess.milliseconds',{count:fmt.format(ms)}):t('knowledgeTaskProcess.seconds',{count:fmt.format(ms/1000)})}
</script>
