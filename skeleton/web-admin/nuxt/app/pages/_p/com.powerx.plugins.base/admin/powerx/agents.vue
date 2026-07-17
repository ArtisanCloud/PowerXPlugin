<template>
  <div class="min-h-screen bg-gray-50 p-6 text-gray-900 dark:bg-gray-950 dark:text-white">
    <div class="mb-6 flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-semibold">Agent 管理</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">管理插件 Agent，并同步到 PowerX 运行态 Agent 与 Skill 绑定。</p>
      </div>
      <div class="flex flex-wrap gap-2">
        <UButton icon="i-heroicons-sparkles" color="neutral" variant="soft" :loading="bootstrapping" @click="initializeBuiltinTemplateAgent">初始化固有 Agent</UButton>
        <UButton icon="i-heroicons-arrow-path" color="neutral" variant="soft" :loading="loading" @click="loadAll">刷新</UButton>
        <UButton icon="i-heroicons-plus" @click="openCreate">新建 Agent</UButton>
      </div>
    </div>

    <section class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900">
      <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 p-4 dark:border-gray-800">
        <div>
          <div class="text-sm font-semibold">插件 Agent</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">共 {{ items.length }} 个 Agent，已同步 {{ syncedCount }} 个。</div>
        </div>
        <div class="flex gap-2">
          <UBadge color="neutral" variant="soft">draft {{ draftCount }}</UBadge>
          <UBadge color="success" variant="soft">synced {{ syncedCount }}</UBadge>
          <UBadge color="error" variant="soft">failed {{ failedCount }}</UBadge>
        </div>
      </div>

      <div v-if="loading" class="space-y-3 p-4">
        <USkeleton v-for="idx in 3" :key="idx" class="h-24 w-full" />
      </div>

      <div v-else-if="items.length">
        <div class="hidden border-b border-gray-100 bg-gray-50 px-4 py-3 text-xs font-semibold uppercase text-gray-500 dark:border-gray-800 dark:bg-gray-950/60 dark:text-gray-400 md:grid md:grid-cols-[minmax(220px,1fr)_120px_minmax(260px,1.2fr)_110px_90px] md:items-center md:gap-4">
          <div>名称</div>
          <div>类型</div>
          <div>Agent Key</div>
          <div>状态</div>
          <div class="text-right">操作</div>
        </div>
        <div class="divide-y divide-gray-100 dark:divide-gray-800">
        <article
          v-for="item in items"
          :key="item.id"
          class="grid items-center gap-4 p-4 transition hover:bg-gray-50 dark:hover:bg-gray-950/40 md:grid-cols-[minmax(220px,1fr)_120px_minmax(260px,1.2fr)_110px_90px]"
        >
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <div class="truncate text-sm font-semibold">{{ item.name }}</div>
            </div>
          </div>

          <div>
            <UBadge :color="agentTypeColor(item)" variant="soft">{{ agentTypeLabel(item) }}</UBadge>
          </div>

          <div class="min-w-0 font-mono text-xs text-gray-600 dark:text-gray-300">
            {{ item.agent_key }}
          </div>

          <div>
            <UBadge :color="statusColor(item.sync_status)" variant="soft">{{ item.sync_status }}</UBadge>
          </div>

          <div class="flex justify-start md:justify-end">
            <UButton size="xs" color="neutral" variant="soft" icon="i-heroicons-pencil-square" @click="openEdit(item)">编辑</UButton>
          </div>
        </article>
        </div>
      </div>

      <div v-else class="flex flex-col items-center justify-center px-4 py-16 text-center">
        <UIcon name="i-heroicons-user-circle" class="h-10 w-10 text-gray-400" />
        <div class="mt-3 text-sm font-medium">暂无 Agent</div>
        <p class="mt-1 text-sm text-gray-500">点击“新建 Agent”创建自定义智能体，或初始化固有 Agent。</p>
        <UButton class="mt-4" icon="i-heroicons-plus" @click="openCreate">新建 Agent</UButton>
      </div>
    </section>

    <UModal
      v-model:open="formOpen"
      :title="form.id ? '编辑 Agent' : '新建 Agent'"
      description="配置插件 Agent 的源定义，同步后由 PowerX 创建运行态 Agent。"
      :ui="{ content: 'w-[min(94vw,960px)] max-w-none' }"
    >
      <template #content>
        <div class="space-y-5 p-5">
          <div class="flex items-start justify-between gap-4">
            <div>
              <h2 class="text-lg font-semibold">{{ form.id ? "编辑 Agent" : "新建 Agent" }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">配置插件 Agent 的源定义，同步后由 PowerX 创建运行态 Agent。</p>
            </div>
            <UButton icon="i-heroicons-x-mark" color="neutral" variant="ghost" square @click="formOpen = false" />
          </div>

          <div v-if="editingAgent" class="grid gap-3 rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm text-gray-700 dark:border-gray-800 dark:bg-gray-900/70 dark:text-gray-200 md:grid-cols-3">
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">类型</div>
              <div class="mt-1"><UBadge :color="agentTypeColor(editingAgent)" variant="soft">{{ agentTypeLabel(editingAgent) }}</UBadge></div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">PowerX UUID</div>
              <div class="mt-1 break-all font-mono text-xs text-gray-700 dark:text-gray-200">{{ editingAgent.powerx_agent_uuid || "-" }}</div>
            </div>
            <div>
              <div class="text-xs text-gray-500 dark:text-gray-400">同步时间</div>
              <div class="mt-1 text-xs text-gray-700 dark:text-gray-200">{{ editingAgent.last_sync_at || "-" }}</div>
            </div>
            <div v-if="editingAgent.sync_error_message" class="md:col-span-3 rounded-md bg-red-50 px-2 py-1 text-xs text-red-700 dark:bg-red-950/40 dark:text-red-200">
              {{ editingAgent.sync_error_message }}
            </div>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <UFormField label="Agent Key" class="md:col-span-2">
              <UInput v-model="form.agent_key" class="w-full" placeholder="powerxplugin.template.agent" />
            </UFormField>
            <UFormField label="名称" class="md:col-span-2">
              <UInput v-model="form.name" class="w-full" placeholder="模板智能体" />
            </UFormField>
            <UFormField label="描述" class="md:col-span-2">
              <UTextarea v-model="form.description" class="w-full" :rows="3" />
            </UFormField>
            <UFormField label="Persona" class="md:col-span-2">
              <UTextarea v-model="form.persona" class="w-full" :rows="4" />
            </UFormField>
            <UFormField label="Prompt Seed" class="md:col-span-2">
              <UTextarea v-model="form.prompt_seed" class="w-full" :rows="5" />
            </UFormField>
            <UFormField label="绑定 Skill" class="md:col-span-2">
              <USelectMenu
                v-model="form.plugin_skill_ids"
                class="w-full"
                :items="syncedSkillOptions"
                multiple
                value-key="value"
                placeholder="选择已同步 Skill"
              />
            </UFormField>
          </div>

          <div class="flex flex-wrap justify-between gap-2 border-t border-gray-200 pt-4 dark:border-gray-800">
            <div class="flex flex-wrap gap-2">
              <UButton
                v-if="editingAgent"
                size="sm"
                color="primary"
                variant="soft"
                icon="i-heroicons-chat-bubble-left-right"
                :disabled="editingAgent.sync_status !== 'synced'"
                to="/agent-skill-bridge"
              >
                调试
              </UButton>
            </div>
            <div class="flex gap-2">
            <UButton color="neutral" variant="soft" @click="formOpen = false">取消</UButton>
            <UButton icon="i-heroicons-check" :loading="saving" @click="save">保存</UButton>
            </div>
          </div>
        </div>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { useApiClient } from "~/composables/api/_client";
import { resolveTenantUUIDForRequest } from "~/utils/tenant-context";

type PluginSkill = {
  id: number;
  plugin_skill_id: string;
  title: string;
  sync_status: string;
};

type PluginAgent = {
  id: number;
  plugin_agent_id: string;
  powerx_agent_uuid?: string;
  agent_key: string;
  name: string;
  description?: string;
  persona?: string;
  prompt_seed?: string;
  plugin_skill_ids?: unknown;
  sync_status: string;
  sync_error_message?: string;
  last_sync_at?: string;
};

const items = ref<PluginAgent[]>([]);
const skills = ref<PluginSkill[]>([]);
const loading = ref(false);
const saving = ref(false);
const bootstrapping = ref(false);
const formOpen = ref(false);
const editingAgent = computed(() => items.value.find((item) => item.id === form.id) || null);
const { client } = useApiClient();
const toast = useToast();

const form = reactive({
  id: 0,
  plugin_agent_id: "template-agent",
  agent_key: "powerxplugin.template.agent",
  name: "模板智能体",
  description: "面向插件开发者和管理员的 PowerXPlugin 模板对象管理智能体，负责解释并执行模板对象的创建、查询、更新、删除和列表等任务。",
  persona: "你是 PowerXPlugin 模板对象管理助手，服务对象是插件开发者和插件管理员。你负责围绕模板对象进行自然语言对话、能力解释、参数澄清和任务执行。回答时应先理解用户当前问题，再基于当前绑定 Skill 的真实 metadata 说明能力或发起执行；不要编造未绑定能力，不要暴露内部 skill_id、executor path、schema 字段名。",
  prompt_seed: "当用户询问你是谁或能做什么时，请以模板对象管理助手身份回答，并只基于当前已绑定 Skill 的真实 metadata 介绍能力。能力介绍应先说明服务对象，再概括模板创建、查询、更新、删除和列表；推荐先测试创建模板。用户要求执行时，按 Skill response_guidance 和 input_schema 判断缺参并追问，参数足够后调用绑定 Skill。",
  plugin_skill_ids: [] as string[]
});

const syncedSkillOptions = computed(() =>
  skills.value
    .filter((item) => item.sync_status === "synced")
    .map((item) => ({ label: `${item.title} (${item.plugin_skill_id})`, value: item.plugin_skill_id }))
);
const syncedCount = computed(() => items.value.filter((item) => item.sync_status === "synced").length);
const draftCount = computed(() => items.value.filter((item) => item.sync_status === "draft").length);
const failedCount = computed(() => items.value.filter((item) => item.sync_status === "failed").length);

onMounted(loadAll);

function requestHeaders() {
  const tenantUUID = resolveTenantUUIDForRequest();
  return tenantUUID ? { tenant_uuid: tenantUUID } : undefined;
}

function pluginAgentIDForSave() {
  const current = String(form.plugin_agent_id || "").trim();
  if (form.id && current) {
    return current;
  }
  return String(form.agent_key || "").trim();
}

async function loadAll() {
  loading.value = true;
  try {
    const [agentPayload, skillPayload] = await Promise.all([
      client<any>("/plugin/agent-registry/agents", { headers: requestHeaders() }),
      client<any>("/plugin/agent-registry/skills", { headers: requestHeaders() })
    ]);
    items.value = agentPayload?.data?.items || [];
    skills.value = skillPayload?.data?.items || [];
  } catch (error) {
    notifyError("加载 Agent 失败", error);
  } finally {
    loading.value = false;
  }
}

async function save() {
  saving.value = true;
  try {
    await client("/plugin/agent-registry/agents", {
      method: "POST",
      headers: requestHeaders(),
      body: {
        plugin_agent_id: pluginAgentIDForSave(),
        agent_key: form.agent_key,
        name: form.name,
        description: form.description,
        persona: form.persona,
        prompt_seed: form.prompt_seed,
        plugin_skill_ids: form.plugin_skill_ids
      }
    });
    formOpen.value = false;
    await loadAll();
    toast.add({ title: "Agent 已保存", color: "success" });
  } catch (error) {
    notifyError("保存 Agent 失败", error);
  } finally {
    saving.value = false;
  }
}

async function initializeBuiltinTemplateAgent() {
  bootstrapping.value = true;
  try {
    await client<any>("/plugin/agent-registry/builtin/template/initialize", { method: "POST", headers: requestHeaders() });
    await loadAll();
    toast.add({
      title: "固有 Agent 已初始化",
      description: "固有 Skill/Agent 已入库，并已 upsert 同步到 PowerX。",
      color: "success",
    });
  } catch (error) {
    notifyError("初始化固有 Agent 失败", error);
  } finally {
    bootstrapping.value = false;
  }
}

function openCreate() {
  resetForm();
  formOpen.value = true;
}

function openEdit(item: PluginAgent) {
  edit(item);
  formOpen.value = true;
}

function edit(item: PluginAgent) {
  form.id = item.id;
  form.plugin_agent_id = item.plugin_agent_id;
  form.agent_key = item.agent_key;
  form.name = item.name;
  form.description = item.description || "";
  form.persona = item.persona || "";
  form.prompt_seed = item.prompt_seed || "";
  form.plugin_skill_ids = parseList(item.plugin_skill_ids);
}

function resetForm() {
  form.id = 0;
  form.plugin_agent_id = "template-agent";
  form.agent_key = "powerxplugin.template.agent";
  form.name = "模板智能体";
  form.description = "面向插件开发者和管理员的 PowerXPlugin 模板对象管理智能体，负责解释并执行模板对象的创建、查询、更新、删除和列表等任务。";
  form.persona = "你是 PowerXPlugin 模板对象管理助手，服务对象是插件开发者和插件管理员。你负责围绕模板对象进行自然语言对话、能力解释、参数澄清和任务执行。回答时应先理解用户当前问题，再基于当前绑定 Skill 的真实 metadata 说明能力或发起执行；不要编造未绑定能力，不要暴露内部 skill_id、executor path、schema 字段名。";
  form.prompt_seed = "当用户询问你是谁或能做什么时，请以模板对象管理助手身份回答，并只基于当前已绑定 Skill 的真实 metadata 介绍能力。能力介绍应先说明服务对象，再概括模板创建、查询、更新、删除和列表；推荐先测试创建模板。用户要求执行时，按 Skill response_guidance 和 input_schema 判断缺参并追问，参数足够后调用绑定 Skill。";
  form.plugin_skill_ids = [];
}

function parseList(value: unknown): string[] {
  if (Array.isArray(value)) return value.map(String).filter(Boolean);
  if (typeof value === "string") {
    try {
      const parsed = JSON.parse(value);
      if (Array.isArray(parsed)) return parsed.map(String).filter(Boolean);
    } catch {
      return value.split(",").map((item) => item.trim()).filter(Boolean);
    }
  }
  return [];
}

function statusColor(status: string) {
  if (status === "synced") return "success";
  if (status === "failed" || status === "disabled") return "error";
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


function agentTypeLabel(item: PluginAgent) {
  return item.plugin_agent_id === "template-agent" ? "固有" : "自定义";
}

function agentTypeColor(item: PluginAgent) {
  return item.plugin_agent_id === "template-agent" ? "primary" : "neutral";
}
</script>
