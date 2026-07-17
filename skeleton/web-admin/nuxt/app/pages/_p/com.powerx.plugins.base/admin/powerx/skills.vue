<template>
  <div class="min-h-screen bg-gray-50 p-6 text-gray-900 dark:bg-gray-950 dark:text-white">
    <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold">Skill 管理</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">插件 Skill，同步后进入 PowerX 底座治理态 Skill。</p>
      </div>
      <div class="flex gap-2">
        <UButton icon="i-heroicons-sparkles" color="neutral" variant="soft" :loading="bootstrapping" @click="initializeBuiltinTemplateAgent">初始化固有 Skill/Agent</UButton>
        <UButton icon="i-heroicons-arrow-path" color="neutral" variant="soft" :loading="loading" @click="loadItems">刷新</UButton>
        <UButton icon="i-heroicons-plus" @click="resetForm">新建 Skill</UButton>
      </div>
    </div>

    <div class="grid gap-4 lg:grid-cols-[360px_minmax(0,1fr)]">
      <section class="rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900">
        <div class="mb-3 text-sm font-semibold">{{ form.id ? '编辑 Skill' : '新建 Skill' }}</div>
        <div class="space-y-3">
          <UFormField label="Plugin Skill ID">
            <UInput v-model="form.plugin_skill_id" placeholder="powerxplugin.template.basic" />
          </UFormField>
          <UFormField label="标题">
            <UInput v-model="form.title" placeholder="模板管理" />
          </UFormField>
          <UFormField label="Capability">
            <UInput v-model="form.capability" placeholder="powerxplugin.template" />
          </UFormField>
          <UFormField label="描述">
            <UTextarea v-model="form.description" :rows="3" />
          </UFormField>
          <UFormField label="Intent Examples JSON">
            <UTextarea v-model="form.intent_examples_json" :rows="4" />
          </UFormField>
          <UFormField label="Input Schema JSON">
            <UTextarea v-model="form.input_schema_json" :rows="6" />
          </UFormField>
          <div class="flex gap-2">
            <UButton icon="i-heroicons-check" :loading="saving" @click="save">保存</UButton>
            <UButton color="neutral" variant="soft" @click="resetForm">重置</UButton>
          </div>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900">
        <div class="border-b border-gray-200 p-4 dark:border-gray-800">
          <div class="text-sm font-semibold">插件 Skill</div>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full min-w-[880px] text-sm">
            <thead class="bg-gray-50 text-left text-xs uppercase text-gray-500 dark:bg-gray-950 dark:text-gray-400">
              <tr>
                <th class="px-4 py-3">Skill</th>
                <th class="px-4 py-3">PowerX ID</th>
                <th class="px-4 py-3">Capability</th>
                <th class="px-4 py-3">状态</th>
                <th class="px-4 py-3">同步时间</th>
                <th class="px-4 py-3 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in items" :key="item.id" class="border-t border-gray-100 dark:border-gray-800">
                <td class="px-4 py-3">
                  <div class="font-medium">{{ item.title }}</div>
                  <div class="font-mono text-xs text-gray-500">{{ item.plugin_skill_id }}</div>
                  <div v-if="item.sync_error_message" class="mt-1 text-xs text-red-600">{{ item.sync_error_message }}</div>
                </td>
                <td class="px-4 py-3 font-mono text-xs">{{ item.powerx_skill_id || '-' }}</td>
                <td class="px-4 py-3 font-mono text-xs">{{ item.capability }}</td>
                <td class="px-4 py-3"><UBadge :color="statusColor(item.sync_status)" variant="soft">{{ item.sync_status }}</UBadge></td>
                <td class="px-4 py-3 text-xs text-gray-500">{{ item.last_sync_at || '-' }}</td>
                <td class="px-4 py-3">
                  <div class="flex justify-end gap-2">
                    <UButton size="xs" color="neutral" variant="soft" icon="i-heroicons-pencil-square" @click="edit(item)">编辑</UButton>
                    <UButton size="xs" icon="i-heroicons-arrow-up-tray" :loading="syncingId === item.id" @click="sync(item)">同步</UButton>
                  </div>
                </td>
              </tr>
              <tr v-if="!items.length && !loading">
                <td colspan="6" class="px-4 py-10 text-center text-gray-500">暂无 Skill</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useApiClient } from "~/composables/api/_client";
import { resolveTenantUUIDForRequest } from "~/utils/tenant-context";

type PluginSkill = {
  id: number;
  plugin_skill_id: string;
  powerx_skill_id?: string;
  title: string;
  description?: string;
  capability: string;
  intent_examples?: unknown;
  input_schema?: unknown;
  sync_status: string;
  sync_error_message?: string;
  last_sync_at?: string;
};

const items = ref<PluginSkill[]>([]);
const loading = ref(false);
const saving = ref(false);
const syncingId = ref<number | null>(null);
const bootstrapping = ref(false);
const { client } = useApiClient();
const toast = useToast();

const form = reactive({
  id: 0,
  plugin_skill_id: "powerxplugin.template.basic",
  title: "模板管理",
  capability: "powerxplugin.template",
  description: "创建、查询、更新和删除插件模板",
  intent_examples_json: JSON.stringify(["帮我创建一个测试模板", "查询当前模板"], null, 2),
  input_schema_json: JSON.stringify({
    type: "object",
    required: ["action"],
    properties: {
      action: { type: "string", enum: ["create", "get", "update", "delete", "list"] },
      template_id: { type: "string" },
      template: { type: "object" }
    }
  }, null, 2)
});

onMounted(loadItems);

function requestHeaders() {
  const tenantUUID = resolveTenantUUIDForRequest();
  return tenantUUID ? { tenant_uuid: tenantUUID } : undefined;
}

async function loadItems() {
  loading.value = true;
  try {
    const payload = await client<any>("/plugin/agent-registry/skills", { headers: requestHeaders() });
    items.value = payload?.data?.items || [];
  } catch (error) {
    notifyError("加载 Skill 失败", error);
  } finally {
    loading.value = false;
  }
}

async function save() {
  saving.value = true;
  try {
    const body = {
      plugin_skill_id: form.plugin_skill_id,
      title: form.title,
      capability: form.capability,
      description: form.description,
      intent_examples: JSON.parse(form.intent_examples_json || "[]"),
      input_schema: JSON.parse(form.input_schema_json || "{}")
    };
    await client("/plugin/agent-registry/skills", { method: "POST", headers: requestHeaders(), body });
    await loadItems();
  } catch (error) {
    notifyError("保存 Skill 失败", error);
  } finally {
    saving.value = false;
  }
}

async function sync(item: PluginSkill) {
  syncingId.value = item.id;
  try {
    await client(`/plugin/agent-registry/skills/${encodeURIComponent(String(item.id))}/sync`, { method: "POST", headers: requestHeaders() });
    await loadItems();
  } catch (error) {
    notifyError("同步 Skill 失败", error);
    await loadItems();
  } finally {
    syncingId.value = null;
  }
}

async function initializeBuiltinTemplateAgent() {
  bootstrapping.value = true;
  try {
    await client("/plugin/agent-registry/builtin/template/initialize", { method: "POST", headers: requestHeaders() });
    toast.add({
      title: "固有 Skill/Agent 已初始化",
      description: "固有 Skill/Agent 已入库，并已 upsert 同步到 PowerX。",
      color: "success",
    });
    await loadItems();
  } catch (error) {
    notifyError("初始化固有 Skill/Agent 失败", error);
  } finally {
    bootstrapping.value = false;
  }
}

function edit(item: PluginSkill) {
  form.id = item.id;
  form.plugin_skill_id = item.plugin_skill_id;
  form.title = item.title;
  form.capability = item.capability;
  form.description = item.description || "";
  form.intent_examples_json = JSON.stringify(item.intent_examples || [], null, 2);
  form.input_schema_json = JSON.stringify(item.input_schema || {}, null, 2);
}

function resetForm() {
  form.id = 0;
  form.plugin_skill_id = "powerxplugin.template.basic";
  form.title = "模板管理";
  form.capability = "powerxplugin.template";
  form.description = "创建、查询、更新和删除插件模板";
}

function statusColor(status: string) {
  if (status === "synced") return "success";
  if (status === "failed") return "error";
  if (status === "pending" || status === "drifted") return "warning";
  return "neutral";
}

function notifyError(title: string, error: unknown) {
  toast.add({
    title,
    description: error instanceof Error ? error.message : String(error),
    color: "error",
  });
}
</script>
