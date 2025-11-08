<template>
  <UContainer class="py-10 space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold">Release Pipelines</h1>
        <p class="text-sm text-gray-500">
          查看 `px-plugin publish create/deploy` 创建的计划，监控灰度批次与回滚状态。
        </p>
      </div>
      <div class="space-x-2">
        <UButton icon="i-heroicons-arrow-path" @click="fetchPlans">刷新</UButton>
        <UButton color="primary" icon="i-heroicons-plus-circle" @click="openDrawer">新建计划</UButton>
      </div>
    </div>

    <UTable :rows="plans" :columns="columns" />

    <USlideover v-model="drawerOpen">
      <UCard>
        <template #header>
          <div class="flex items-center justify-between">
            <h3 class="text-lg font-semibold">创建发布计划</h3>
            <UButton icon="i-heroicons-x-mark" color="gray" variant="ghost" @click="drawerOpen = false" />
          </div>
        </template>
        <div class="space-y-4">
          <UFormGroup label="Channel">
            <USelect v-model="form.channel" :options="['stable', 'beta']" />
          </UFormGroup>
          <UFormGroup label="Strategy">
            <USelect v-model="form.strategy" :options="['canary', 'direct']" />
          </UFormGroup>
          <UFormGroup label="Notes">
            <UTextarea v-model="form.notes" placeholder="说明发布内容、风险、回滚方案" />
          </UFormGroup>
        </div>
        <template #footer>
          <div class="flex justify-end space-x-2">
            <UButton color="gray" variant="ghost" @click="drawerOpen = false">取消</UButton>
            <UButton color="primary" @click="submitPlan">提交</UButton>
          </div>
        </template>
      </UCard>
    </USlideover>
  </UContainer>
</template>

<script setup lang="ts">
import { ref } from "vue";

interface PlanRow {
  planId: string;
  publishId: string;
  channel: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

const columns = [
  { key: "planId", label: "Plan ID" },
  { key: "publishId", label: "Publish ID" },
  { key: "channel", label: "Channel" },
  { key: "status", label: "Status" },
  { key: "createdAt", label: "Created" },
  { key: "updatedAt", label: "Updated" },
];

const plans = ref<PlanRow[]>([]);
const drawerOpen = ref(false);
const form = ref({
  channel: "stable",
  strategy: "canary",
  notes: "",
});

async function fetchPlans() {
  try {
    const response = await fetch("/api/internal/publish/plans");
    if (!response.ok) {
      throw new Error("failed to load plans");
    }
    plans.value = await response.json();
  } catch (error) {
    console.warn("[publish pipelines] fetchPlans error", error);
  }
}

async function submitPlan() {
  try {
    await fetch("/api/internal/publish/create", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        channel: form.value.channel,
        notes: form.value.notes,
        rollout: {
          strategy: form.value.strategy,
          batches: [{ percentage: 20 }, { percentage: 80, wait: "20m" }],
        },
      }),
    });
    drawerOpen.value = false;
    await fetchPlans();
  } catch (error) {
    console.error("[publish pipelines] submit error", error);
  }
}

function openDrawer() {
  drawerOpen.value = true;
}

onMounted(() => {
  fetchPlans();
});
</script>
