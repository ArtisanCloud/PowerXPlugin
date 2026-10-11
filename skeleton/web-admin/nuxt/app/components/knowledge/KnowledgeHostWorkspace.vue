<template>
  <KnowledgeIngestionWorkspace v-if="section === 'ingestion'" />
  <KnowledgeSpaceRecordsWorkspace v-else-if="section === 'records'" />
  <KnowledgeSpaceStrategyWorkspace v-else-if="section === 'strategy'" />
  <KnowledgeSpaceSourcesWorkspace v-else-if="section === 'sources'" />
  <KnowledgeSearchWorkspace v-else-if="section === 'playground'" />
</template>
<script setup lang="ts">
import { provide } from 'vue'
import KnowledgeIngestionWorkspace from './KnowledgeIngestionWorkspace.vue'
import KnowledgeSpaceRecordsWorkspace from './KnowledgeSpaceRecordsWorkspace.vue'
import KnowledgeSpaceStrategyWorkspace from './KnowledgeSpaceStrategyWorkspace.vue'
import KnowledgeSpaceSourcesWorkspace from './KnowledgeSpaceSourcesWorkspace.vue'
import KnowledgeSearchWorkspace from './KnowledgeSearchWorkspace.vue'
import { knowledgeWorkspaceKey } from '~/composables/api/useKnowledgeWorkspace'
import { usePowerXKnowledgeWorkspace } from '~/composables/api/usePowerXKnowledgeWorkspace'
defineProps<{ section: 'ingestion' | 'records' | 'strategy' | 'sources' | 'playground' }>()
const route = useRoute()
const basePath = route.path.split('/knowledge-lab')[0] + '/knowledge-lab'
provide(knowledgeWorkspaceKey, usePowerXKnowledgeWorkspace(basePath))
</script>
