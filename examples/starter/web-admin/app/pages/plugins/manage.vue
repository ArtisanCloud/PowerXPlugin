<template>
  <section class="plugins-manage">
    <header class="plugins-manage__header">
      <h1>插件版本管理</h1>
      <button type="button" @click="refresh">刷新</button>
    </header>

    <table class="plugins-manage__table">
      <thead>
        <tr>
          <th>插件</th>
          <th>版本</th>
          <th>状态</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="entry in entries" :key="entry.plugin">
          <td>{{ entry.plugin }}</td>
          <td>{{ entry.version }}</td>
          <td>{{ entry.status }}</td>
          <td>
            <button type="button" @click="install(entry)">安装</button>
            <button type="button" @click="rollback(entry)" class="warn">回滚</button>
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'

interface Entry {
  plugin: string
  version: string
  status: string
}

const entries = ref<Entry[]>([])

function refresh() {
  entries.value = [
    { plugin: 'plugin.demo', version: '1.5.0', status: '可安装' },
    { plugin: 'plugin.beta', version: '1.6.0-beta', status: '灰度中' },
  ]
}

function install(entry: Entry) {
  console.log('install', entry)
}

function rollback(entry: Entry) {
  console.log('rollback', entry)
}
</script>

<style scoped>
.plugins-manage {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 0.75rem;
  padding: 1.25rem;
}
.plugins-manage__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}
.plugins-manage__table {
  width: 100%;
  border-collapse: collapse;
}
.plugins-manage__table th,
.plugins-manage__table td {
  padding: 0.75rem 0.5rem;
  border-bottom: 1px solid #e5e7eb;
}
button.warn {
  color: #b91c1c;
}
</style>
