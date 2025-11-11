<template>
  <section class="review-console">
    <header class="review-console__header">
      <h1>Marketplace 审核队列</h1>
      <div class="review-console__actions">
        <button type="button" @click="refresh">刷新</button>
      </div>
    </header>

    <table class="review-console__table">
      <thead>
        <tr>
          <th>Publish ID</th>
          <th>插件</th>
          <th>渠道</th>
          <th>状态</th>
          <th>提交时间</th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="!records.length">
          <td colspan="5" class="text-muted">暂无记录</td>
        </tr>
        <tr v-for="record in records" :key="record.publishId">
          <td>{{ record.publishId }}</td>
          <td>{{ record.versionId }}</td>
          <td>{{ record.channel }}</td>
          <td>{{ record.status }}</td>
          <td>{{ new Date(record.submittedAt).toLocaleString() }}</td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'

interface ReviewRecord {
  publishId: string
  versionId: string
  channel: string
  status: string
  submittedAt: string
}

const records = ref<ReviewRecord[]>([])

async function refresh() {
  // TODO: fetch real API once backend ready
  records.value = [
    {
      publishId: 'demo-1',
      versionId: 'plugin-1.3.0',
      channel: 'stable',
      status: 'pending',
      submittedAt: new Date().toISOString(),
    },
  ]
}
</script>

<style scoped>
.review-console {
  background: white;
  padding: 1.25rem;
  border-radius: 0.75rem;
  border: 1px solid #e5e7eb;
}
.review-console__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}
.review-console__table {
  width: 100%;
  border-collapse: collapse;
}
.review-console__table th,
.review-console__table td {
  padding: 0.75rem 0.5rem;
  border-bottom: 1px solid #e5e7eb;
}
.text-muted {
  color: #6b7280;
}
</style>
