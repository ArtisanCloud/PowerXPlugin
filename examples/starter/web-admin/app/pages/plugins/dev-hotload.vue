<template>
  <section class="dev-hotload">
    <header class="dev-hotload__header">
      <h1>Dev 热加载监控</h1>
      <div class="dev-hotload__status">
        <span :class="['badge', isConnected ? 'badge--ok' : 'badge--warn']">
          {{ isConnected ? 'LIVE' : 'DISCONNECTED' }}
        </span>
        <button class="dev-hotload__refresh" type="button" @click="restartStream">
          重新连接
        </button>
      </div>
    </header>

    <div class="dev-hotload__grid">
      <div class="card">
        <h2>当前 Session</h2>
        <p><strong>Session ID:</strong> {{ sessionInfo?.sessionId || '未连接' }}</p>
        <p><strong>Tenant:</strong> {{ sessionInfo?.tenant || '—' }}</p>
        <p><strong>Reload Token:</strong> {{ sessionInfo?.reloadToken || '—' }}</p>
        <p><strong>最近更新:</strong> {{ formatRelativeTime(sessionInfo?.updatedAt) }}</p>
      </div>

      <div class="card">
        <h2>错误 / 警告</h2>
        <p v-if="!errors.length" class="text-muted">暂无</p>
        <ul v-else class="dev-hotload__errors">
          <li v-for="err in errors" :key="err.timestamp">
            <span>{{ formatTimestamp(err.timestamp) }}</span>
            <p>{{ err.message }}</p>
          </li>
        </ul>
      </div>
    </div>

    <div class="card">
      <h2>Reload 日志（保留 7 天）</h2>
      <table class="dev-hotload__table">
        <thead>
          <tr>
            <th>时间</th>
            <th>文件</th>
            <th>状态</th>
            <th>耗时</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!reloads.length">
            <td colspan="4" class="text-muted">暂无记录</td>
          </tr>
          <tr v-for="reload in reloads" :key="reload.id">
            <td>{{ formatTimestamp(reload.timestamp) }}</td>
            <td>{{ reload.files.join(', ') }}</td>
            <td>
              <span :class="['badge', reload.status === 'ok' ? 'badge--ok' : 'badge--warn']">
                {{ reload.status }}
              </span>
            </td>
            <td>{{ reload.latency }}ms</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

interface SessionEvent {
  sessionId: string
  tenant?: string
  reloadToken: string
  updatedAt: string
}

interface ReloadEvent {
  id: string
  timestamp: string
  files: string[]
  latency: number
  status: 'ok' | 'error'
}

interface ErrorEvent {
  timestamp: string
  message: string
}

const eventSource = ref<EventSource | null>(null)
const sessionInfo = ref<SessionEvent | null>(null)
const reloads = ref<ReloadEvent[]>([])
const errors = ref<ErrorEvent[]>([])
const isConnected = ref(false)

function startStream() {
  stopStream()
  const source = new EventSource('/api/dev-hotload/events')
  eventSource.value = source
  isConnected.value = true

  source.addEventListener('session', (event) => {
    const data = JSON.parse((event as MessageEvent).data) as SessionEvent
    sessionInfo.value = data
  })

  source.addEventListener('reload', (event) => {
    const data = JSON.parse((event as MessageEvent).data) as ReloadEvent
    reloads.value = [{ ...data }, ...reloads.value].slice(0, 100)
  })

  source.addEventListener('error', () => {
    errors.value.unshift({ timestamp: new Date().toISOString(), message: 'SSE 连接中断' })
    isConnected.value = false
    stopStream()
  })
}

function stopStream() {
  if (eventSource.value) {
    eventSource.value.close()
    eventSource.value = null
  }
}

function restartStream() {
  startStream()
}

function formatTimestamp(value?: string) {
  if (!value) return '—'
  return new Date(value).toLocaleString()
}

function formatRelativeTime(value?: string) {
  if (!value) return '—'
  const delta = Date.now() - new Date(value).getTime()
  return `${Math.round(delta / 1000)}s 前`
}

onMounted(() => startStream())
onBeforeUnmount(() => stopStream())
</script>

<style scoped>
.dev-hotload {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}
.dev-hotload__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.dev-hotload__status {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.badge {
  padding: 0.25rem 0.75rem;
  border-radius: 999px;
  font-weight: 600;
  text-transform: uppercase;
  font-size: 0.8rem;
}
.badge--ok {
  background: #d1fae5;
  color: #065f46;
}
.badge--warn {
  background: #fee2e2;
  color: #991b1b;
}
.card {
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  padding: 1.25rem;
  background: white;
}
.dev-hotload__grid {
  display: grid;
  gap: 1.5rem;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
}
.dev-hotload__table {
  width: 100%;
  border-collapse: collapse;
}
.dev-hotload__table th,
.dev-hotload__table td {
  padding: 0.75rem 0.5rem;
  border-bottom: 1px solid #e5e7eb;
}
.dev-hotload__errors {
  list-style: none;
  padding: 0;
  margin: 0;
}
.dev-hotload__errors li {
  border-bottom: 1px solid #f3f4f6;
  padding: 0.5rem 0;
}
.text-muted {
  color: #6b7280;
}
</style>
