<template>
  <UContainer>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-semibold">Support Playbook</h1>
      <div class="space-x-2">
        <UButton color="secondary" @click="refresh" :loading="store.loading">刷新</UButton>
        <UButton color="primary" @click="save" :loading="store.loading">保存</UButton>
      </div>
    </div>

    <UAlert v-if="store.error" color="error" :title="store.error" class="mb-4" />

    <div class="grid gap-4 md:grid-cols-2">
      <UCard>
        <template #header>渠道配置</template>
        <div v-if="form.channels.length === 0" class="text-sm text-gray-500">暂未配置渠道</div>
        <div v-for="(channel, index) in form.channels" :key="index" class="space-y-2 border rounded-md p-3 mb-3">
          <UInput v-model="channel.channel" label="渠道类型" placeholder="marketplace_ticket" />
          <UInput v-model="channel.address" label="地址" placeholder="https://support.example.com" />
          <UTag v-for="role in channel.escalates" :key="role" class="mr-1">{{ role }}</UTag>
        </div>
        <UButton icon="i-heroicons-plus" variant="ghost" @click="addChannel">新增渠道</UButton>
      </UCard>

      <UCard>
        <template #header>知识库</template>
        <div v-if="form.knowledge_base.length === 0" class="text-sm text-gray-500">请补充 README / FAQ 等文档链接</div>
        <div v-for="(doc, index) in form.knowledge_base" :key="index" class="space-y-2 border rounded-md p-3 mb-3">
          <UInput v-model="doc.label" label="标题" placeholder="README" />
          <UInput v-model="doc.url" label="URL" placeholder="https://docs.example.com/readme" />
        </div>
        <UButton icon="i-heroicons-plus" variant="ghost" @click="addDoc">新增文档</UButton>
      </UCard>
    </div>

    <UCard class="mt-6">
      <template #header>就绪检查</template>
      <div class="flex flex-wrap gap-2">
        <UBadge v-for="item in store.playbook?.readiness || []" :key="item.key" :color="item.completed ? 'success' : 'warning'">
          {{ item.key }}
        </UBadge>
      </div>
    </UCard>
  </UContainer>
</template>

<script setup lang="ts">
import { useOperationsStore } from '~/stores/operations/useOperationsStore'

const toast = useToast()
const store = useOperationsStore()
const form = reactive({
  channels: [] as any[],
  knowledge_base: [] as any[],
})

const refresh = async () => {
  await store.fetchPlaybook()
  syncForm()
}

const syncForm = () => {
  form.channels = (store.playbook?.channels || []).map((channel) => ({ ...channel }))
  form.knowledge_base = (store.playbook?.knowledge_base || []).map((doc) => ({ ...doc }))
}

const addChannel = () => {
  form.channels.push({ channel: '', address: '', escalates: [] })
}

const addDoc = () => {
  form.knowledge_base.push({ label: '', url: '' })
}

const save = async () => {
  await store.savePlaybook(form)
  syncForm()
  toast.add({ title: '保存成功', color: 'success' })
}

await refresh()
</script>
