<script setup lang="ts">
import UButton from '@nuxt/ui/components/Button.vue'
import { onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { PublisherClientError, type PublisherClient } from '../client'
import type { PublisherPage, PublisherRequestTrace } from '../types'
import en from '../locales/en.json'
import zh from '../locales/zh.json'
const props = defineProps<{
  client: PublisherClient
  kind: 'uploads' | 'submissions'
  taskId: string
}>()
const { t, te, locale } = useI18n({ useScope: 'local', messages: { en, zh } })
const data = ref<PublisherPage<PublisherRequestTrace> | null>(null),
  busy = ref(false),
  error = ref(''),
  requestID = ref('')
let epoch = 0
watch(
  () => `${props.kind}/${props.taskId}`,
  () => {
    epoch++
    data.value = null
    error.value = ''
    requestID.value = ''
    busy.value = false
  }
)
onUnmounted(() => {
  epoch++
})
async function read(page = 1) {
  if (busy.value) return
  const current = ++epoch
  busy.value = true
  error.value = ''
  requestID.value = ''
  try {
    const result = await props.client.requestTraces(
      props.kind,
      props.taskId,
      page
    )
    if (current === epoch) data.value = result
  } catch (cause) {
    if (current === epoch) {
      error.value =
        cause instanceof PublisherClientError
          ? cause.code
          : 'PUBLISHER_PROXY_RESPONSE_INVALID'
      requestID.value =
        cause instanceof PublisherClientError ? cause.requestID : ''
    }
  } finally {
    if (current === epoch) busy.value = false
  }
}
function date(raw: string) {
  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'short',
    timeStyle: 'medium'
  }).format(new Date(raw))
}
</script>
<template>
  <section class="request-history" :aria-label="t('trace.title')">
    <p>{{ t('trace.boundary') }}</p>
    <UButton
      color="neutral"
      variant="outline"
      :disabled="busy"
      @click="void read(data?.page || 1)"
      >{{ t('trace.read') }}</UButton
    >
    <p v-if="error" role="alert">
      {{
        t(
          te(`errors.${error}`)
            ? `errors.${error}`
            : 'errors.PUBLISHER_MARKET_PROTOCOL'
        )
      }}
    </p>
    <p v-if="requestID">
      {{ t('trace.serverID') }} <code>{{ requestID }}</code>
    </p>
    <template v-if="data">
      <p>{{ t('trace.page', { page: data.page, total: data.total }) }}</p>
      <p v-if="!data.items.length">{{ t('trace.empty') }}</p>
      <ol>
        <li v-for="item in data.items" :key="item.client_request_id">
          <strong
            >{{ t(`trace.types.${item.response_type}`) }} ·
            {{ item.method }}</strong
          >
          <p>
            {{ date(item.started_at) }} ·
            {{ item.http_status || t('trace.noResponse') }}
          </p>
          <p v-if="!item.finished_at">{{ t('trace.unconfirmed') }}</p>
          <dl>
            <dt>{{ t('trace.clientID') }}</dt>
            <dd>
              <code>{{ item.client_request_id }}</code>
            </dd>
            <dt>{{ t('trace.serverID') }}</dt>
            <dd>
              <code v-if="item.server_request_id">{{
                item.server_request_id
              }}</code
              ><span v-else>{{ t('trace.noResponse') }}</span>
            </dd>
          </dl>
        </li>
      </ol>
      <div class="request-pages">
        <UButton
          color="neutral"
          variant="outline"
          :disabled="busy || data.page === 1"
          @click="void read(data.page - 1)"
          >{{ t('market.previous') }}</UButton
        >
        <UButton
          color="neutral"
          variant="outline"
          :disabled="busy || data.page * data.size >= data.total"
          @click="void read(data.page + 1)"
          >{{ t('market.next') }}</UButton
        >
      </div>
    </template>
  </section>
</template>
<style scoped>
.request-history {
  color: var(--ui-text);
  margin-top: 1rem;
  overflow-wrap: anywhere;
}
.request-history p {
  margin: 0.75rem 0;
}
strong,
dt {
  color: var(--ui-text-highlighted);
  font-weight: 600;
}
ol {
  list-style: none;
  padding: 0;
  max-height: 26rem;
  overflow: auto;
}
li {
  padding: 1rem;
  border: 1px solid var(--ui-border);
  border-radius: 0.5rem;
  margin-bottom: 0.75rem;
  background: var(--ui-bg);
}
.request-pages {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}
</style>
