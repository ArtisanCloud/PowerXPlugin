<template>
  <section class="offline-review">
    <header class="offline-review__header">
      <h1>离线上传队列</h1>
      <div class="offline-review__actions">
        <button type="button" @click="refresh">刷新</button>
      </div>
    </header>

    <div class="card">
      <h2>白名单配置</h2>
      <p>选择允许导入的租户：</p>
      <div class="tenant-grid">
        <label v-for="tenant in allTenants" :key="tenant" class="tenant-chip">
          <input type="checkbox" :value="tenant" v-model="selectedTenants" />
          <span>{{ tenant }}</span>
        </label>
      </div>
    </div>

    <table class="offline-review__table">
      <thead>
        <tr>
          <th>Publish ID</th>
          <th>插件</th>
          <th>状态</th>
          <th>白名单租户</th>
          <th>SLA</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!entries.length">
          <td colspan="5" class="text-muted">暂无离线上传</td>
        </tr>
        <tr v-for="entry in entries" :key="entry.publishId">
          <td>{{ entry.publishId }}</td>
          <td>{{ entry.versionId }}</td>
          <td>{{ entry.status }}</td>
          <td>{{ entry.allowedTenants.join(', ') || '未配置' }}</td>
          <td>
            <span :class="['badge', slaClass(entry.slaMinutes)]">
              {{ entry.slaMinutes <= 1440 ? `${entry.slaMinutes}m` : '超时' }}
            </span>
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'

interface OfflineEntry {
  publishId: string
  versionId: string
  status: string
  allowedTenants: string[]
  slaMinutes: number
}

const allTenants = ['tenant-a', 'tenant-b', 'tenant-c']
const selectedTenants = ref<string[]>([])
const entries = ref<OfflineEntry[]>([])

function slaClass(value: number) {
  return value > 1440 ? 'badge--warn' : 'badge--ok'
}

function refresh() {
  entries.value = [
    {
      publishId: 'offline-1',
      versionId: 'plugin-1.5.0',
      status: 'pending',
      allowedTenants: selectedTenants.value,
      slaMinutes: 120,
    },
  ]
}
</script>

<style scoped>
.offline-review {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}
.offline-review__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  padding: 1.25rem;
}
.tenant-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
.tenant-chip {
  border: 1px solid #e5e7eb;
  border-radius: 999px;
  padding: 0.25rem 0.75rem;
  display: flex;
  align-items: center;
  gap: 0.25rem;
}
.offline-review__table {
  width: 100%;
  border-collapse: collapse;
}
.offline-review__table th,
.offline-review__table td {
  padding: 0.75rem 0.5rem;
  border-bottom: 1px solid #e5e7eb;
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
.text-muted {
  color: #6b7280;
}
</style>
