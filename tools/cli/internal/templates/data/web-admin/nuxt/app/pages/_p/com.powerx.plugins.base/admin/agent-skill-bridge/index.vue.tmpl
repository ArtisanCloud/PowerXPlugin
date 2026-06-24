<template>
  <div class="h-[calc(100vh-7.5rem)] min-h-[680px] overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950">
    <div class="flex h-full min-h-0">
      <aside class="flex w-80 min-w-80 flex-col border-r border-gray-200 bg-gray-50 dark:border-gray-800 dark:bg-gray-900/60">
        <div class="border-b border-gray-200 p-4 dark:border-gray-800">
          <div class="mb-3 flex items-center gap-2">
            <UButton icon="i-heroicons-cog-6-tooth" color="neutral" variant="soft" square />
            <USelectMenu
              v-model="selectedAgentId"
              :items="agentOptions"
              value-key="value"
              label-key="label"
              searchable
              class="min-w-0 flex-1"
              data-testid="agent-chat-agent-select"
              :disabled="loading || agentsLoading || agentOptions.length === 0"
              placeholder="选择智能体"
            />
          </div>
          <div class="flex items-center gap-2">
            <UButton icon="i-heroicons-plus" class="flex-1 justify-center" :loading="sessionLoading" :disabled="loading || sessionLoading || !agentId.trim()" @click="createSession()">
              新建会话
            </UButton>
            <UButton icon="i-heroicons-arrow-path" color="neutral" variant="soft" square :loading="agentsLoading || sessionLoading" :disabled="loading" @click="loadSessions(currentAgent, false)" />
          </div>
        </div>

        <div class="min-h-0 flex-1 overflow-y-auto p-4">
          <div v-if="agentsLoading" class="rounded-md bg-gray-100 p-3 text-sm text-gray-500 dark:bg-gray-900 dark:text-gray-400">
            正在加载底座 Agent...
          </div>
          <div v-else-if="agentsError" class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
            {{ agentsError }}
          </div>
          <template v-else-if="agentOptions.length">
            <div class="mb-2 px-2 py-1 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">最近</div>
            <div
              v-for="item in visibleSessionItems"
              :key="item.id"
              class="group mb-2 flex w-full items-start gap-2 rounded-md border-l-2 px-3 py-3 text-left"
              :class="item.id === sessionId.trim()
                ? 'border-primary bg-primary-50 text-primary-700 dark:bg-primary-950/50 dark:text-primary-300'
                : 'border-transparent bg-white text-gray-700 hover:bg-gray-100 dark:bg-gray-950 dark:text-gray-300 dark:hover:bg-gray-900'"
              data-testid="agent-chat-session-item"
              @click="selectSession(item)"
            >
              <UIcon name="i-heroicons-chat-bubble-left-right" class="mt-0.5 h-4 w-4 shrink-0" />
              <div class="min-w-0 flex-1">
                <div class="truncate text-sm font-medium">{{ item.title }}</div>
                <div class="mt-1 truncate text-xs opacity-80">{{ item.summary }}</div>
              </div>
              <button
                type="button"
                class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-gray-500 opacity-70 hover:bg-gray-200 hover:text-red-600 group-hover:opacity-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-red-300"
                :disabled="loading || sessionLoading || deletingSessionId === item.id"
                aria-label="删除会话"
                title="删除会话"
                data-testid="agent-chat-session-delete"
                @click.stop.prevent="deleteSession(item)"
              >
                <UIcon v-if="deletingSessionId !== item.id" name="i-heroicons-trash" class="h-4 w-4" />
                <UIcon v-else name="i-heroicons-arrow-path" class="h-4 w-4 animate-spin" />
              </button>
            </div>
          </template>
        </div>

        <div class="border-t border-gray-200 p-4 dark:border-gray-800">
          <div class="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
            <span>PowerX Agent Runtime</span>
            <UBadge :color="statusColor" variant="soft" data-testid="agent-chat-status">{{ statusLabel }}</UBadge>
          </div>
        </div>
      </aside>

      <main class="flex min-w-0 flex-1 flex-col bg-white dark:bg-gray-950">
        <header class="border-b border-gray-200 px-5 py-4 dark:border-gray-800">
          <div class="flex items-center justify-between gap-3">
            <div class="flex min-w-0 items-center gap-3">
              <UAvatar :text="currentAgent.avatarText" size="lg" class="bg-primary text-white" />
              <div class="min-w-0">
                <div class="truncate text-lg font-semibold text-gray-900 dark:text-white">{{ currentAgent.label }}</div>
                <div class="mt-1 flex flex-wrap items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                  <span :class="connectionReady ? (loading ? 'text-primary' : 'text-green-500') : 'text-red-500'">{{ connectionLabel }}</span>
                  <span class="hidden sm:inline">/</span>
                  <span class="truncate" data-testid="agent-chat-target">{{ agentProxyPath }}</span>
                </div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <UBadge color="primary" variant="soft">智能会话</UBadge>
              <UButton icon="i-heroicons-arrow-path" color="neutral" variant="soft" square :disabled="loading" @click="resetConversation" />
            </div>
          </div>
        </header>

        <section class="min-h-0 flex-1 overflow-y-auto px-6 py-5">
          <div v-if="chatMessages.length === 0" class="flex h-full items-center justify-center text-center">
            <div class="max-w-md space-y-3 text-gray-500 dark:text-gray-400">
              <UIcon name="i-heroicons-chat-bubble-left-right" class="mx-auto h-10 w-10" />
              <div class="text-xl font-semibold text-gray-800 dark:text-gray-100">欢迎</div>
              <div>{{ welcomeText }}</div>
            </div>
          </div>

          <div v-else class="mx-auto max-w-4xl space-y-5">
            <div
              v-for="item in chatMessages"
              :key="item.id"
              class="flex gap-3"
              :class="item.role === 'user' ? 'justify-end' : 'justify-start'"
            >
              <UAvatar
                v-if="item.role === 'assistant'"
                :text="currentAgent.avatarText"
                size="sm"
                class="mt-1 bg-primary text-white"
              />
              <div
                class="max-w-[78%] rounded-lg px-4 py-3 text-sm leading-6 shadow-sm"
                :class="item.role === 'user'
                  ? 'bg-primary text-white'
                  : 'border border-gray-200 bg-gray-50 text-gray-900 dark:border-gray-800 dark:bg-gray-900 dark:text-gray-100'"
                :data-testid="item.role === 'assistant' ? 'agent-chat-final' : undefined"
              >
                <div class="whitespace-pre-wrap">{{ item.content }}</div>
                <div v-if="item.pending" class="mt-2 flex items-center gap-2 text-xs opacity-70">
                  <UIcon name="i-heroicons-arrow-path" class="h-3 w-3 animate-spin" />
                  <span>streaming</span>
                </div>
              </div>
              <UAvatar
                v-if="item.role === 'user'"
                icon="i-heroicons-user"
                size="sm"
                class="mt-1 bg-gray-200 text-gray-700 dark:bg-gray-800 dark:text-gray-200"
              />
            </div>
          </div>
        </section>

        <footer class="border-t border-gray-200 bg-white px-6 py-4 dark:border-gray-800 dark:bg-gray-950">
          <div class="mx-auto flex max-w-4xl items-end gap-3 rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-gray-800 dark:bg-gray-900">
            <UButton icon="i-heroicons-plus" color="neutral" variant="ghost" square :disabled="loading" />
            <UTextarea
              v-model="message"
              data-testid="agent-chat-input"
              autoresize
              :maxrows="5"
              :disabled="loading"
              class="min-w-0 flex-1"
              :placeholder="messagePlaceholder"
              @keydown="handleInputKeydown"
            />
            <UButton
              icon="i-heroicons-paper-airplane"
              data-testid="agent-chat-send"
              :loading="loading"
              :disabled="!canSend"
              square
              @click="send"
            />
            <UButton
              icon="i-heroicons-stop"
              color="neutral"
              variant="ghost"
              square
              :disabled="!loading"
              @click="abortStream"
            />
          </div>
        </footer>
      </main>

      <aside class="hidden w-[30rem] min-w-[30rem] flex-col border-l border-gray-200 bg-gray-50 dark:border-gray-800 dark:bg-gray-900/60 xl:flex">
        <div class="border-b border-gray-200 p-4 dark:border-gray-800">
          <div class="text-sm font-semibold text-gray-900 dark:text-white">调试设置</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">Agent Session / Skill Bridge</div>
        </div>

        <div class="min-h-0 flex-1 space-y-4 overflow-y-auto p-4">
          <section class="space-y-3">
            <UFormField label="PowerX Agent Proxy">
              <UInput :model-value="agentProxyPath" data-testid="agent-chat-proxy" disabled class="w-full" />
            </UFormField>
            <UFormField label="Agent ID">
              <UInput v-model="agentId" data-testid="agent-chat-agent" class="w-full" />
            </UFormField>
            <UFormField label="Session ID">
              <UInput v-model="sessionId" data-testid="agent-chat-session" class="w-full" />
            </UFormField>
            <UFormField label="Trace ID">
              <UInput v-model="traceId" data-testid="agent-chat-trace" class="w-full" />
            </UFormField>
            <UFormField label="Bearer Token">
              <UInput
                v-model="bearerToken"
                data-testid="agent-chat-token"
                type="password"
                placeholder="默认读取 localStorage access_token"
                class="w-full"
              />
            </UFormField>
          </section>

          <section class="rounded-md border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950">
            <div class="mb-3 flex items-center justify-between">
              <div class="text-sm font-semibold text-gray-900 dark:text-white">执行过程</div>
              <UBadge variant="soft">{{ eventCount }} events</UBadge>
            </div>
            <div v-if="timeline.length" class="space-y-2" data-testid="agent-chat-timeline">
              <button
                v-for="item in timeline"
                :key="item.id"
                class="w-full rounded-md px-3 py-2 text-left text-xs"
                :class="selectedEvent?.id === item.id
                  ? 'bg-primary-50 ring-1 ring-primary-500 dark:bg-primary-950/40'
                  : 'bg-gray-50 hover:bg-gray-100 dark:bg-gray-900 dark:hover:bg-gray-800'"
                @click="selectedEventId = item.id"
              >
                <div class="flex items-center justify-between gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ item.type }}</span>
                  <span class="text-gray-400">{{ item.time }}</span>
                </div>
                <div v-if="item.summary" class="mt-1 text-gray-500 dark:text-gray-400">{{ item.summary }}</div>
              </button>
              <div v-if="selectedEvent" class="rounded-md border border-gray-200 bg-gray-950 p-3 dark:border-gray-800">
                <div class="mb-2 flex items-center justify-between text-xs">
                  <span class="font-medium text-gray-100">{{ selectedEvent.type }} payload</span>
                  <span class="text-gray-500">{{ selectedEvent.time }}</span>
                </div>
                <pre class="max-h-48 overflow-auto whitespace-pre-wrap text-xs text-gray-200">{{ formatPayload(selectedEvent.payload) }}</pre>
              </div>
            </div>
            <div v-else class="rounded-md bg-gray-50 p-3 text-xs text-gray-500 dark:bg-gray-900 dark:text-gray-400">
              等待 Agent Runtime 事件。
            </div>
          </section>

          <section class="rounded-md border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950">
            <div class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">Skill 目标</div>
            <dl class="space-y-2 text-xs">
              <div class="flex items-center justify-between gap-3">
                <dt class="text-gray-500">skill_id</dt>
                <dd class="font-mono text-gray-900 dark:text-white">{{ skillTarget.skill_id }}</dd>
              </div>
              <div class="flex items-center justify-between gap-3">
                <dt class="text-gray-500">capability</dt>
                <dd class="font-mono text-gray-900 dark:text-white">{{ skillTarget.capability }}</dd>
              </div>
              <div class="flex items-center justify-between gap-3">
                <dt class="text-gray-500">executor</dt>
                <dd class="font-mono text-gray-900 dark:text-white">{{ skillTarget.executor }}</dd>
              </div>
            </dl>
          </section>

          <section class="rounded-md border border-gray-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-950">
            <div class="mb-3 flex items-center justify-between">
              <div class="text-sm font-semibold text-gray-900 dark:text-white">Raw SSE</div>
              <UButton icon="i-heroicons-clipboard" size="xs" variant="ghost" @click="copyRaw">复制</UButton>
            </div>
            <pre data-testid="agent-chat-events" class="max-h-64 overflow-auto whitespace-pre-wrap rounded-md bg-gray-950 p-3 text-xs text-gray-100">{{ rawLog || "尚无事件。" }}</pre>
          </section>

          <section v-if="errorText" class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
            <div class="mb-1 font-medium">错误</div>
            <div data-testid="agent-chat-error">{{ errorText }}</div>
          </section>
          <section v-if="agentsError" class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950 dark:text-red-200">
            <div class="mb-1 font-medium">Agent 清单加载失败</div>
            <div data-testid="agent-chat-agents-error">{{ agentsError }}</div>
          </section>
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { createFetchSSE, type SSEStreamEvent } from "~/composables/api/useStream";
import { getAuthToken } from "~/composables/api/_base";
import { useAuth } from "~/composables/useAuth";

type AgentEvent = {
  id: string;
  type: string;
  time: string;
  summary: string;
  payload: Record<string, unknown>;
  count?: number;
};

type ChatMessage = {
  id: string;
  role: "user" | "assistant";
  content: string;
  pending?: boolean;
};

type AgentOption = {
  label: string;
  value: string;
  id: number | string;
  uuid: string;
  sessionTitle: string;
  avatarText: string;
  skillId?: string;
  capability?: string;
  executor?: string;
};

type SessionItem = {
  id: string;
  agentId: string;
  title: string;
  summary: string;
  createdAt: number;
};

type AgentSessionRecord = {
  id?: number | string;
  uuid?: string;
  session_id?: string;
  sessionId?: string;
  agentId?: number | string;
  agent_id?: number | string;
  title?: string;
  status?: string;
  latestAt?: string;
  latest_at?: string;
  createdAt?: string;
  created_at?: string;
};

type AgentSessionMessageRecord = {
  id?: number | string;
  role?: string;
  content?: string;
};

type PowerXAgentRecord = {
  id?: number | string;
  uuid?: string;
  powerx_agent_uuid?: string;
  plugin_agent_id?: string;
  agent_key?: string;
  key?: string;
  name?: string;
  status?: string;
  sync_status?: string;
  plugin_skill_ids?: string[] | string;
  powerx_skill_ids?: string[] | string;
};

const agentProxyPath = "plugin/agent/stream/sse";

const agentOptions = ref<AgentOption[]>([]);
const agentsLoading = ref(false);
const sessionLoading = ref(false);
const deletingSessionId = ref("");
const agentsError = ref("");
const selectedAgentId = ref("");
const agentId = ref("");
const sessionId = ref("");
const traceId = ref(`trace_${Date.now()}`);
const message = ref("");
const bearerToken = ref("");
const rawLog = ref("");
const finalMessage = ref("");
const timeline = ref<AgentEvent[]>([]);
const selectedEventId = ref("");
const eventCount = ref(0);
const chatMessages = ref<ChatMessage[]>([]);
const sessionItems = ref<SessionItem[]>([]);
const loading = ref(false);
const status = ref<"idle" | "streaming" | "completed" | "error" | "aborted">("idle");
const errorText = ref("");
const auth = useAuth();
let abortController: AbortController | null = null;
let activeAssistantMessageID = "";
let assistantBuffer = "";
let suppressAgentWatch = false;

const currentAgent = computed<AgentOption>(() => {
  const matched = agentOptions.value.find((item) => item.value === agentId.value.trim());
  if (matched) return matched;
  return {
    label: agentId.value.trim() || "未选择 Agent",
    value: agentId.value.trim(),
    id: agentId.value.trim(),
    uuid: "",
    sessionTitle: "Agent 会话",
    avatarText: "A",
  };
});

const connectionReady = computed(() => Boolean(agentId.value.trim() && sessionId.value.trim() && !agentsError.value));

const connectionLabel = computed(() => {
  if (agentsLoading.value) return "加载中";
  if (loading.value) return "处理中";
  if (connectionReady.value) return "已连接";
  return "未就绪";
});

const welcomeText = computed(() => {
  if (agentsError.value) return agentsError.value;
  if (!connectionReady.value) return "请先在 Agent 管理中同步插件 Agent。";
  return "当前会话暂无消息";
});

const messagePlaceholder = computed(() => {
  const skillID = String(currentAgent.value.skillId || "").trim();
  if (skillID.includes("template")) {
    return "例如：帮我创建一个标题为测试模板的模板";
  }
  if (skillID) {
    return `输入消息，测试 ${skillID}`;
  }
  return "输入消息";
});

const skillTarget = computed(() => ({
  skill_id: currentAgent.value.skillId || "powerxplugin.template.basic",
  capability: currentAgent.value.capability || "powerxplugin.template",
  executor: currentAgent.value.executor || "capability",
}));

const currentSessionSummary = computed(() => {
  const lastUserMessage = [...chatMessages.value].reverse().find((item) => item.role === "user");
  if (lastUserMessage?.content) return lastUserMessage.content;
  return sessionId.value.trim() || "本地调试会话";
});

const currentSessionItem = computed(() => {
  const id = sessionId.value.trim();
  if (!id) return null;
  return sessionItems.value.find((item) => item.id === id) || null;
});

const visibleSessionItems = computed(() => {
  const currentAgentId = agentId.value.trim();
  if (!currentAgentId) return [];
  return sessionItems.value.filter((item) => item.agentId === currentAgentId);
});

const selectedEvent = computed(() => {
  if (!selectedEventId.value) return null;
  return timeline.value.find((item) => item.id === selectedEventId.value) || null;
});

watch(selectedAgentId, (next) => {
  if (suppressAgentWatch) return;
  const selected = agentOptions.value.find((item) => item.value === next);
  if (!selected) return;
  agentId.value = selected.value;
  void loadSessions(selected, true);
});

watch(agentId, (next) => {
  if (next === selectedAgentId.value) return;
  const selected = agentOptions.value.find((item) => item.value === next);
  if (selected) {
    selectedAgentId.value = selected.value;
  }
});

onMounted(async () => {
  await loadPowerXAgents();
});

const canSend = computed(() => {
  return Boolean(
    message.value.trim() &&
      agentId.value.trim() &&
      sessionId.value.trim() &&
      traceId.value.trim() &&
      !sessionLoading.value &&
      !loading.value
  );
});

const statusLabel = computed(() => {
  switch (status.value) {
    case "streaming":
      return "streaming";
    case "completed":
      return "completed";
    case "error":
      return "error";
    case "aborted":
      return "aborted";
    default:
      return "idle";
  }
});

const statusColor = computed(() => {
  switch (status.value) {
    case "streaming":
      return "warning";
    case "completed":
      return "success";
    case "error":
      return "error";
    case "aborted":
      return "neutral";
    default:
      return "primary";
  }
});

function authToken() {
  const explicit = bearerToken.value.trim();
  if (explicit) return explicit;
  if (typeof window === "undefined") return "";
  return String(
    auth.getToken?.() ||
      getAuthToken() ||
      window.localStorage.getItem("access_token") ||
      window.localStorage.getItem("__px_access_token") ||
      window.localStorage.getItem("auth_token") ||
      window.localStorage.getItem("token") ||
      ""
  ).trim();
}

function requestHeaders(contentType = false) {
  const headers: Record<string, string> = {
    Accept: "application/json",
  };
  if (contentType) {
    headers["Content-Type"] = "application/json";
  }
  const token = authToken();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}

function normalizeAgentRecord(item: PowerXAgentRecord): AgentOption {
  const uuid = String(item.powerx_agent_uuid || item.uuid || "").trim();
  const label = String(item.name || item.agent_key || item.key || item.plugin_agent_id || uuid || item.id || "").trim();
  const value = String(uuid || item.plugin_agent_id || item.key || item.id || "").trim();
  const localSkillIDs = parseStringList(item.plugin_skill_ids);
  const powerxSkillIDs = parseStringList(item.powerx_skill_ids);
  if (!label || !value) {
    throw new Error("Runnable plugin agent missing name or powerx_agent_uuid");
  }
  return {
    label,
    value,
    id: item.id || value,
    uuid,
    sessionTitle: `${label} 会话`,
    avatarText: label.slice(0, 1).toUpperCase() || "A",
    skillId: powerxSkillIDs[0] || localSkillIDs[0] || "powerxplugin.template.basic",
    capability: "powerxplugin.template",
    executor: "capability",
  };
}

function parseStringList(value: unknown) {
  if (Array.isArray(value)) {
    return value.map((item) => String(item || "").trim()).filter(Boolean);
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return [];
    try {
      const parsed = JSON.parse(trimmed);
      if (Array.isArray(parsed)) {
        return parsed.map((item) => String(item || "").trim()).filter(Boolean);
      }
    } catch {
      return trimmed.split(",").map((item) => item.trim()).filter(Boolean);
    }
  }
  return [];
}

function extractAgentItems(payload: unknown): PowerXAgentRecord[] {
  const root = payload as Record<string, any>;
  const candidates = [
    root?.data?.items,
    root?.items,
    root?.data,
  ];
  for (const candidate of candidates) {
    if (Array.isArray(candidate)) {
      return candidate as PowerXAgentRecord[];
    }
  }
  throw new Error("PowerX Agent list response missing data.items");
}

async function loadPowerXAgents() {
  if (typeof window === "undefined") return;
  agentsLoading.value = true;
  agentsError.value = "";
  try {
    const url = new URL("/api/v1/plugin/agent-registry/agents/runnable", window.location.origin);
    const headers = requestHeaders();
    const response = await fetch(url.toString(), { headers });
    if (!response.ok) {
      throw new Error(`PowerX Agent proxy failed: HTTP ${response.status}`);
    }
    const payload = await response.json();
    const items = extractAgentItems(payload)
      .filter((item) => String(item.sync_status || "synced") === "synced")
      .map(normalizeAgentRecord);
    if (!items.length) {
      throw new Error("没有已同步的插件 Agent，请先在 Agent 管理中同步 Agent");
    }
    const defaultAgent = items[0];
    agentOptions.value = items;
    suppressAgentWatch = true;
    selectedAgentId.value = defaultAgent.value;
    agentId.value = defaultAgent.value;
    await loadSessions(defaultAgent, true);
    suppressAgentWatch = false;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    agentsError.value = message;
    agentOptions.value = [];
    selectedAgentId.value = "";
    agentId.value = "";
    sessionId.value = "";
    sessionItems.value = [];
    status.value = "error";
    errorText.value = message;
  } finally {
    suppressAgentWatch = false;
    agentsLoading.value = false;
  }
}

function extractSessionID(payload: unknown) {
  const root = payload as Record<string, any>;
  const data = root?.data ?? root;
  const candidates = [
    data?.uuid,
    data?.session_uuid,
    data?.session_id,
    data?.sessionId,
    data?.id,
  ];
  for (const item of candidates) {
    const text = String(item || "").trim();
    if (text) return text;
  }
  throw new Error("PowerX Agent session response missing session uuid");
}

async function createSession(agent = currentAgent.value) {
  if (typeof window === "undefined") return;
  if (sessionLoading.value) return;
  const targetAgent = agent || currentAgent.value;
  if (!String(targetAgent.value || "").trim()) {
    errorText.value = "agent_id is required";
    return;
  }
  abortStream();
  sessionLoading.value = true;
  errorText.value = "";
  try {
    const body: Record<string, unknown> = {
      title: targetAgent.sessionTitle,
      env: "dev",
      meta: {
        source: "powerxplugin.local_chat",
      },
    };
    if (targetAgent.uuid) {
      body.agent_uuid = targetAgent.uuid;
    } else {
      body.agent_id = String(targetAgent.id || targetAgent.value).trim();
    }
    const headers = requestHeaders(true);
    const response = await fetch("/api/v1/plugin/agent/sessions", {
      method: "POST",
      headers,
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(`PowerX Agent session proxy failed: HTTP ${response.status}`);
    }
    const payload = await response.json();
    const createdSessionId = extractSessionID(payload);
    sessionId.value = createdSessionId;
    upsertSessionItem({
      id: createdSessionId,
      agentId: targetAgent.value,
      title: targetAgent.sessionTitle,
      summary: createdSessionId,
      createdAt: Date.now(),
    });
    resetRuntimeState();
    chatMessages.value = [];
    await loadSessions(targetAgent, false);
    await loadSessionMessages(createdSessionId);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    errorText.value = message;
    status.value = "error";
  } finally {
    sessionLoading.value = false;
  }
}

function upsertSessionItem(item: SessionItem) {
  const existing = sessionItems.value.filter((session) => session.id !== item.id);
  sessionItems.value = [item, ...existing].slice(0, 20);
}

async function loadSessions(agent = currentAgent.value, selectFirst = false) {
  if (typeof window === "undefined") return;
  const targetAgent = agent || currentAgent.value;
  const agentUUID = String(targetAgent.uuid || targetAgent.value || "").trim();
  if (!agentUUID) return;
  try {
    const url = new URL("/api/v1/plugin/agent/sessions", window.location.origin);
    url.searchParams.set("agent_uuid", agentUUID);
    url.searchParams.set("env", "dev");
    url.searchParams.set("status", "active");
    url.searchParams.set("limit", "50");
    const response = await fetch(url.toString(), { headers: requestHeaders() });
    if (!response.ok) {
      throw new Error(`PowerX Agent sessions proxy failed: HTTP ${response.status}`);
    }
    const payload = await response.json();
    const items = extractSessionItems(payload).map((record) => normalizeSessionRecord(record, targetAgent));
    const otherAgentItems = sessionItems.value.filter((item) => item.agentId !== targetAgent.value);
    sessionItems.value = [...items, ...otherAgentItems].slice(0, 50);
    const currentStillExists = items.some((item) => item.id === sessionId.value);
    if (selectFirst && items.length > 0) {
      await selectSession(items[0]);
    } else if (!currentStillExists && items.length > 0 && targetAgent.value === agentId.value) {
      await selectSession(items[0]);
    } else if (!currentStillExists && targetAgent.value === agentId.value) {
      sessionId.value = "";
      resetConversation();
    }
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error);
  }
}

function extractSessionItems(payload: unknown): AgentSessionRecord[] {
  const root = payload as Record<string, any>;
  const candidates = [root?.data?.items, root?.items, root?.data];
  for (const candidate of candidates) {
    if (Array.isArray(candidate)) return candidate as AgentSessionRecord[];
  }
  return [];
}

function normalizeSessionRecord(record: AgentSessionRecord, agent: AgentOption): SessionItem {
  const id = String(record.uuid || record.session_id || record.sessionId || record.id || "").trim();
  const title = String(record.title || agent.sessionTitle || "Agent 会话").trim();
  const createdText = String(record.latestAt || record.latest_at || record.createdAt || record.created_at || "").trim();
  const createdAt = createdText ? Date.parse(createdText) || Date.now() : Date.now();
  return {
    id,
    agentId: agent.value,
    title,
    summary: id,
    createdAt,
  };
}

async function selectSession(item: SessionItem) {
  if (!item) return;
  abortStream();
  const selectedAgent = agentOptions.value.find((agent) => agent.value === item.agentId);
  if (selectedAgent) {
    suppressAgentWatch = true;
    selectedAgentId.value = selectedAgent.value;
    agentId.value = selectedAgent.value;
    suppressAgentWatch = false;
  }
  sessionId.value = item.id;
  resetRuntimeState();
  await loadSessionMessages(item.id);
}

async function loadSessionMessages(id: string) {
  const sessionUUID = String(id || "").trim();
  if (!sessionUUID) return;
  try {
    const url = new URL(`/api/v1/plugin/agent/sessions/${encodeURIComponent(sessionUUID)}/messages`, window.location.origin);
    url.searchParams.set("env", "dev");
    url.searchParams.set("limit", "200");
    const response = await fetch(url.toString(), { headers: requestHeaders() });
    if (!response.ok) {
      throw new Error(`PowerX Agent messages proxy failed: HTTP ${response.status}`);
    }
    const payload = await response.json();
    chatMessages.value = extractMessageItems(payload)
      .map(normalizeChatMessage)
      .filter((item): item is ChatMessage => Boolean(item));
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error);
  }
}

function extractMessageItems(payload: unknown): AgentSessionMessageRecord[] {
  const root = payload as Record<string, any>;
  const candidates = [root?.data?.items, root?.items, root?.data];
  for (const candidate of candidates) {
    if (Array.isArray(candidate)) return candidate as AgentSessionMessageRecord[];
  }
  return [];
}

function normalizeChatMessage(record: AgentSessionMessageRecord): ChatMessage | null {
  const role = String(record.role || "").trim();
  if (role !== "user" && role !== "assistant") return null;
  return {
    id: `${role}_${String(record.id || Date.now())}`,
    role,
    content: String(record.content || ""),
    pending: false,
  };
}

async function responseErrorMessage(response: Response, fallback: string): Promise<string> {
  try {
    const payload = await response.clone().json();
    const message = String(payload?.error?.message || payload?.message || "").trim();
    if (message) return message;
  } catch {
    // Ignore non-JSON error bodies and fall back to the caller message.
  }
  return fallback;
}

async function deleteSession(item: SessionItem) {
  const id = String(item?.id || "").trim();
  if (!id) return;
  abortStream();
  deletingSessionId.value = id;
  try {
    const url = new URL(`/api/v1/plugin/agent/sessions/${encodeURIComponent(id)}`, window.location.origin);
    url.searchParams.set("env", "dev");
    const response = await fetch(url.toString(), {
      method: "DELETE",
      headers: requestHeaders(),
    });
    if (!response.ok) {
      throw new Error(await responseErrorMessage(response, `PowerX Agent session delete failed: HTTP ${response.status}`));
    }
    sessionItems.value = sessionItems.value.filter((item) => item.id !== id);
    await loadSessions(currentAgent.value, false);
    const next = visibleSessionItems.value[0];
    if (id === sessionId.value.trim() && next) {
      await selectSession(next);
    } else if (id === sessionId.value.trim()) {
      sessionId.value = "";
      resetConversation();
    }
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : String(error);
  } finally {
    deletingSessionId.value = "";
  }
}

function resetConversation() {
  resetRuntimeState();
  chatMessages.value = [];
}

function resetRuntimeState() {
  abortStream();
  rawLog.value = "";
  finalMessage.value = "";
  timeline.value = [];
  selectedEventId.value = "";
  eventCount.value = 0;
  errorText.value = "";
  status.value = "idle";
  activeAssistantMessageID = "";
  traceId.value = `trace_${Date.now()}`;
  if (currentSessionItem.value) {
    currentSessionItem.value.summary = currentSessionSummary.value;
  }
}

function abortStream() {
  if (!abortController) return;
  abortController.abort();
  abortController = null;
  markAssistantPending(false);
  if (loading.value) {
    status.value = "aborted";
    loading.value = false;
  }
}

function appendChatMessage(role: "user" | "assistant", content: string, pending = false) {
  const id = `${role}_${Date.now()}_${chatMessages.value.length}`;
  chatMessages.value.push({ id, role, content, pending });
  return id;
}

function updateAssistantMessage(content: string) {
  if (!activeAssistantMessageID) return;
  const target = chatMessages.value.find((item) => item.id === activeAssistantMessageID);
  if (!target) return;
  target.content = content;
}

function markAssistantPending(pending: boolean) {
  if (!activeAssistantMessageID) return;
  const target = chatMessages.value.find((item) => item.id === activeAssistantMessageID);
  if (!target) return;
  target.pending = pending;
}

function appendAssistantText(delta: string) {
  if (!delta) return;
  assistantBuffer = `${assistantBuffer}${delta}`;
  updateAssistantMessage(assistantBuffer);
}

function setAssistantText(text: string) {
  const normalized = String(text || "").trim();
  if (!normalized) return;
  assistantBuffer = normalized;
  updateAssistantMessage(assistantBuffer);
}

function handleInputKeydown(event: KeyboardEvent) {
  if (event.key !== "Enter") return;
  if (!event.metaKey && !event.ctrlKey) return;
  event.preventDefault();
  send();
}

async function send() {
  if (!canSend.value) return;
  const userMessage = message.value.trim();
  abortStream();
  loading.value = true;
  status.value = "streaming";
  errorText.value = "";
  rawLog.value = "";
  finalMessage.value = "";
  timeline.value = [];
  selectedEventId.value = "";
  eventCount.value = 0;
  assistantBuffer = "";
  appendChatMessage("user", userMessage);
  if (currentSessionItem.value) {
    currentSessionItem.value.summary = userMessage;
  }
  activeAssistantMessageID = appendChatMessage("assistant", "正在连接 PowerX Agent Runtime...", true);
  message.value = "";

  abortController = new AbortController();
  try {
    const headers: Record<string, string> = {
    };
    const token = authToken();
    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
    await createFetchSSE({
      path: agentProxyPath,
      params: {
        agent_uuid: currentAgent.value.uuid,
        session_uuid: sessionId.value.trim(),
        trace_id: traceId.value.trim(),
        q: userMessage,
        source: "powerxplugin.local_chat",
        env: "dev",
      },
      headers,
      signal: abortController.signal,
      onEvent: consumeSSEEvent,
    });
    if (status.value === "streaming") {
      status.value = "completed";
    }
    if (!finalMessage.value) {
      finalMessage.value = "PowerX Agent Runtime 已结束本次会话。";
      updateAssistantMessage(finalMessage.value);
    }
    markAssistantPending(false);
    await loadSessionMessages(sessionId.value.trim());
  } catch (error) {
    markAssistantPending(false);
    if ((error as Error)?.name === "AbortError") {
      status.value = "aborted";
      updateAssistantMessage("已停止本次会话。");
      return;
    }
    status.value = "error";
    errorText.value = error instanceof Error ? error.message : String(error);
    updateAssistantMessage(errorText.value);
  } finally {
    loading.value = false;
    abortController = null;
    if (status.value === "completed") {
      await loadSessions(currentAgent.value, false);
    }
  }
}

function consumeSSEEvent(event: SSEStreamEvent) {
  rawLog.value += `${event.raw}\n\n`;
  const payload = normalizeEventPayload(event.payload);
  const type = String(payload.type || event.event || "message");
  const summary = summarizeEvent(type, payload);
  recordRuntimeEvent(type, summary, payload);
  const delta = extractVisibleDelta(type, payload);
  if (delta) appendAssistantText(delta);
  if (type === "final") {
    finalMessage.value = extractFinalMessage(payload);
    setAssistantText(finalMessage.value);
  }
  if (type === "error") {
    errorText.value = summary || "PowerX Agent returned error";
    status.value = "error";
    updateAssistantMessage(errorText.value);
  }
}

function recordRuntimeEvent(type: string, summary: string, payload: Record<string, unknown>) {
  eventCount.value += 1;
  const now = new Date().toLocaleTimeString();
  if (type === "token") {
    const existing = timeline.value.find((item) => item.type === "token stream");
    const count = (existing?.count || 0) + 1;
    const tokenText = extractVisibleDelta(type, payload);
    const nextPayload = {
      chunks: count,
      latest_delta: tokenText,
      latest_payload: payload,
    };
    if (existing) {
      existing.count = count;
      existing.time = now;
      existing.summary = `${count} chunks${tokenText ? ` / latest: ${trimForSummary(tokenText, 32)}` : ""}`;
      existing.payload = nextPayload;
      return;
    }
    const item = {
      id: `token_stream_${Date.now()}`,
      type: "token stream",
      time: now,
      summary: `${count} chunk${tokenText ? ` / latest: ${trimForSummary(tokenText, 32)}` : ""}`,
      payload: nextPayload,
      count,
    };
    timeline.value.push(item);
    if (!selectedEventId.value) selectedEventId.value = item.id;
    return;
  }
  const item = {
    id: `${Date.now()}_${timeline.value.length}`,
    type,
    time: now,
    summary,
    payload,
  };
  timeline.value.push(item);
  if (!selectedEventId.value) selectedEventId.value = item.id;
}

function trimForSummary(text: string, maxLength: number) {
  const value = String(text || "").replace(/\s+/g, " ").trim();
  if (value.length <= maxLength) return value;
  return `${value.slice(0, maxLength)}...`;
}

function formatPayload(payload: Record<string, unknown>) {
  return JSON.stringify(payload || {}, null, 2);
}

function normalizeEventPayload(payload: unknown): Record<string, unknown> {
  if (payload && typeof payload === "object" && !Array.isArray(payload)) {
    return payload as Record<string, unknown>;
  }
  return { raw: payload };
}

function summarizeEvent(type: string, payload: Record<string, unknown>) {
  if (type === "intent") {
    const tasks = Array.isArray(payload.tasks) ? payload.tasks.length : "";
    return tasks ? `识别到 ${tasks} 个候选任务` : "意图识别完成";
  }
  if (type === "plan") {
    const plan = payload.plan as Record<string, unknown> | undefined;
    const tasks = Array.isArray(plan?.tasks) ? plan?.tasks.length : "";
    return tasks ? `计划包含 ${tasks} 个节点` : "计划生成完成";
  }
  if (type === "node_start" || type === "node_end") {
    const kind = String(payload.node_kind || payload.kind || "node");
    const ref = String(payload.node_ref || payload.skill_id || payload.node_id || "");
    const statusText = String(payload.status || (type === "node_start" ? "running" : "done"));
    return [kind, ref, statusText].filter(Boolean).join(" / ");
  }
  if (type === "final") {
    return extractFinalMessage(payload);
  }
  if (type === "error") {
    return String(payload.message || payload.error || "error");
  }
  return String(payload.message || payload.status || "");
}

function extractVisibleDelta(type: string, payload: Record<string, unknown>) {
  if (!["token", "chunk", "data", "message"].includes(type)) return "";
  const candidates = [
    payload.delta,
    payload.text,
    payload.content,
    (payload.data as Record<string, unknown> | undefined)?.delta,
    (payload.data as Record<string, unknown> | undefined)?.text,
    (payload.data as Record<string, unknown> | undefined)?.content,
    (payload.payload as Record<string, unknown> | undefined)?.delta,
    (payload.payload as Record<string, unknown> | undefined)?.text,
    (payload.payload as Record<string, unknown> | undefined)?.content,
  ];
  for (const item of candidates) {
    const text = String(item || "");
    if (text) return text;
  }
  return "";
}

function extractFinalMessage(payload: Record<string, unknown>) {
  const candidates = [
    payload.text,
    payload.message,
    payload.content,
    (payload.data as Record<string, unknown> | undefined)?.content,
    ((payload.data as Record<string, unknown> | undefined)?.result as Record<string, unknown> | undefined)?.content,
    ((payload.data as Record<string, unknown> | undefined)?.result as Record<string, unknown> | undefined)?.message,
    (payload.result as Record<string, unknown> | undefined)?.content,
    (payload.payload as Record<string, unknown> | undefined)?.message,
    (payload.payload as Record<string, unknown> | undefined)?.content,
    (payload.data as Record<string, unknown> | undefined)?.message,
    (payload.result as Record<string, unknown> | undefined)?.message,
  ];
  for (const item of candidates) {
    const text = String(item || "").trim();
    if (text) return text;
  }
  return JSON.stringify(payload, null, 2);
}

async function copyRaw() {
  if (!rawLog.value || typeof navigator === "undefined") return;
  await navigator.clipboard?.writeText(rawLog.value);
}
</script>
