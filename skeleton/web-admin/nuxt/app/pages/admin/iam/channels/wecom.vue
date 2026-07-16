<template>
  <div class="wecom-config-page p-6 space-y-6">
    <div class="flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between">
      <div>
        <p class="text-sm uppercase tracking-wide text-gray-500 dark:text-slate-300">
          Channel Configuration
        </p>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">企业微信配置</h1>
        <p class="text-sm text-gray-600 dark:text-slate-300">
          回调地址会按 Host 与租户、渠道、应用参数动态计算，用于企业微信后台配置。
        </p>
      </div>
      <div class="flex gap-2">
        <UBadge color="neutral" variant="soft">{{ providerModeLabel }}</UBadge>
        <UButton to="/admin/iam/channels/wecom-sync" variant="soft" color="neutral">同步中心</UButton>
        <UButton variant="outline" :loading="refreshing" @click="reloadConfig">刷新</UButton>
        <UButton color="primary" :loading="saving" :disabled="readonlyMode" @click="saveForm">保存配置</UButton>
      </div>
    </div>

    <UCard>
      <template #header>
        <h3 class="text-lg font-semibold">企业微信 OAuth 参数</h3>
      </template>

      <UForm :state="form" class="space-y-4" :disabled="readonlyMode">
        <div class="grid gap-4 md:grid-cols-2">
          <UFormField label="配置状态">
            <USelect v-model="form.status" :items="statusOptions" />
          </UFormField>
          <UFormField label="轮换周期（天）">
            <UInput v-model="form.rotationDays" type="number" min="1" max="365" />
          </UFormField>
        </div>

        <div class="grid gap-4 md:grid-cols-3">
          <div class="md:col-span-2">
            <UFormField label="Host:Port（接收回调）" required>
              <UInput v-model="form.callbackHost" placeholder="https://plugin.example.com" />
            </UFormField>
          </div>
          <div class="md:col-span-1">
            <UFormField label="PowerWechat HttpDebug">
              <div class="flex items-center justify-between rounded-md border border-slate-700/60 px-3 py-2">
                <span class="text-xs text-slate-300">启用后输出 SDK 调试日志</span>
                <USwitch v-model="form.httpDebug" />
              </div>
            </UFormField>
          </div>
        </div>

        <UFormField label="Corp ID" required>
          <UInput v-model="form.corpId" placeholder="wx20306dccb1710bd9" />
        </UFormField>

        <div class="grid gap-4 md:grid-cols-2">
          <UFormField label="App ID（Agent ID）" required>
            <UInput v-model="form.agentId" placeholder="1000018" />
          </UFormField>
          <UFormField label="App Secret" required>
            <UInput v-model="form.secret" :type="showSecret ? 'text' : 'password'" placeholder="企业微信应用 Secret">
              <template #trailing>
                <UButton
                  variant="ghost"
                  color="neutral"
                  size="xs"
                  :label="showSecret ? '隐藏' : '显示'"
                  @click="showSecret = !showSecret"
                />
              </template>
            </UInput>
          </UFormField>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <UFormField label="Token">
            <UInput v-model="form.token" placeholder="用于回调签名校验" />
          </UFormField>
          <UFormField label="Encoding-AESKey">
            <UInput v-model="form.encodingAESKey" placeholder="43 位 EncodingAESKey" />
          </UFormField>
        </div>

        <UFormField label="动态回调 URL 预览">
          <div class="flex flex-col gap-2">
            <div
              class="w-full rounded-md border px-3 py-2 text-sm leading-6 whitespace-pre-wrap break-all font-mono"
              style="border-color:#475569;background:#020617;color:#e2e8f0 !important;"
            >
              {{ callbackPreview }}
            </div>
            <div class="flex justify-end">
              <UButton color="primary" variant="soft" size="sm" @click="copyCallbackUrl">复制回调 URL</UButton>
            </div>
          </div>
        </UFormField>
      </UForm>
    </UCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useToast } from "#imports";
import { storeToRefs } from "pinia";
import { useIAMService } from "~/composables/api/services/iamService";
import { useApiClient } from "~/composables/api/_client";
import { defaultProviderMode, normalizeProviderMode, type ProviderModeDiagnostics } from "~/composables/api/useProviderMode";
import { useUserStore } from "~/stores/user";
import { resolveTenantUUIDForRequest } from "~/utils/tenant-context";

definePageMeta({
  title: "企业微信配置",
  icon: "i-lucide-building-2",
  order: 12,
});

type WecomConfig = {
  status: string;
  rotationDays: number;
  callbackHost: string;
  corpId: string;
  agentId: string;
  secret: string;
  token: string;
  encodingAESKey: string;
  httpDebug: boolean;
};

type WecomConfigResponse = {
  status?: string;
  rotationDays?: number;
  rotation_days?: number;
  callbackHost?: string;
  callback_host?: string;
  corpId?: string;
  corp_id?: string;
  agentId?: string | number;
  agent_id?: string | number;
  secret?: string;
  token?: string;
  encodingAESKey?: string;
  encoding_aes_key?: string;
  aesKey?: string;
  aes_key?: string;
  httpDebug?: boolean;
  http_debug?: boolean;
};

const toast = useToast();
const { t } = useI18n();
const { get, put } = useApiClient();
const iam = useIAMService();
const showSecret = ref(false);
const saving = ref(false);
const refreshing = ref(false);
const providerMode = ref<ProviderModeDiagnostics>(defaultProviderMode());
const readonlyMode = computed(() => Boolean(providerMode.value.read_only));
const providerModeLabel = computed(() => providerMode.value.mode === "delegated" ? t("providerMode.delegated") : t("providerMode.local"));

const statusOptions = [
  { label: "active", value: "active" },
  { label: "inactive", value: "inactive" },
];

const defaultForm = (): WecomConfig => ({
  status: "active",
  rotationDays: 30,
  callbackHost: "",
  corpId: "",
  agentId: "",
  secret: "",
  token: "",
  encodingAESKey: "",
  httpDebug: true,
});

const form = reactive(defaultForm());

const userStore = useUserStore();
const { currentTenantUuid } = storeToRefs(userStore);

const tenantUUID = computed(() => {
  const fromStore = currentTenantUuid.value?.trim();
  if (fromStore) return fromStore;
  return resolveTenantUUIDForRequest() || "00000000-0000-0000-0000-000000000001";
});

const callbackPreview = computed(() => {
  if (!form.callbackHost.trim()) return "请先输入 Host:Port";
  const corp = encodeURIComponent(form.corpId.trim() || "corp_id");
  const agent = encodeURIComponent(form.agentId.trim() || "agent_id");
  const tenant = encodeURIComponent(tenantUUID.value);
  const host = form.callbackHost.trim().replace(/\/+$/, "");
  return `${host}/api/v1/webhooks/wecom/tenant/${tenant}/corp/${corp}/app/${agent}`;
});

const normalizeStatus = (raw?: string) => {
  const value = (raw || "").trim().toUpperCase();
  return value === "REVOKED" ? "inactive" : "active";
};

const applyServerConfig = (data?: WecomConfigResponse | null) => {
  if (!data) {
    Object.assign(form, defaultForm());
    return;
  }
  Object.assign(form, defaultForm(), {
    status: normalizeStatus(data.status),
    rotationDays: Number(data.rotation_days ?? data.rotationDays ?? 30) || 30,
    callbackHost: String(data.callback_host ?? data.callbackHost ?? ""),
    corpId: String(data.corp_id ?? data.corpId ?? ""),
    agentId: String(data.agent_id ?? data.agentId ?? ""),
    secret: String(data.secret ?? ""),
    token: String(data.token ?? ""),
    encodingAESKey: String(data.encoding_aes_key ?? data.encodingAESKey ?? data.aes_key ?? data.aesKey ?? ""),
    httpDebug: Boolean(data.http_debug ?? data.httpDebug ?? true),
  });
};

const validateRequired = () => {
  if (!form.callbackHost.trim()) return "Host:Port 必填";
  if (!form.corpId.trim()) return "Corp ID 必填";
  if (!form.agentId.trim()) return "App ID 必填";
  if (!form.secret.trim()) return "App Secret 必填";
  return "";
};

const saveForm = async () => {
  if (readonlyMode.value) {
    toast.add({ title: t("providerMode.readOnly"), description: t("providerMode.readOnlyDescription"), color: "warning" });
    return;
  }
  const error = validateRequired();
  if (error) {
    toast.add({ title: "保存失败", description: error, color: "error" });
    return;
  }
  saving.value = true;
  try {
    const response = await put<any>("/admin/iam/channels/wecom/config", {
      status: form.status,
      rotation_days: Number(form.rotationDays) || 30,
      callback_host: form.callbackHost.trim(),
      corp_id: form.corpId.trim(),
      agent_id: Number(form.agentId),
      secret: form.secret.trim(),
      token: form.token.trim(),
      aes_key: form.encodingAESKey.trim(),
      http_debug: form.httpDebug,
    });
    applyServerConfig((response?.data || response) as WecomConfigResponse);
    toast.add({ title: "配置已保存", description: "已保存到租户数据库配置。", color: "success" });
  } catch (err: any) {
    toast.add({
      title: "保存失败",
      description: err?.data?.error?.message || err?.message || "请求失败",
      color: "error",
    });
  } finally {
    saving.value = false;
  }
};

const copyCallbackUrl = async () => {
  try {
    if (process.client && navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(callbackPreview.value);
      toast.add({ title: "已复制回调 URL", color: "success" });
      return;
    }
    toast.add({ title: "复制失败", description: "浏览器不支持剪贴板接口", color: "warning" });
  } catch {
    toast.add({ title: "复制失败", description: "请手动复制", color: "error" });
  }
};

const reloadConfig = async () => {
  refreshing.value = true;
  try {
    const response = await get<any>("/admin/iam/channels/wecom/config");
    applyServerConfig((response?.data || response) as WecomConfigResponse);
  } catch (err: any) {
    Object.assign(form, defaultForm());
    const statusCode = Number(err?.status || err?.response?.status || 0);
    if (statusCode !== 404) {
      toast.add({
        title: "加载失败",
        description: err?.data?.error?.message || err?.message || "请求失败",
        color: "error",
      });
    }
  } finally {
    refreshing.value = false;
  }
};

onMounted(async () => {
  try {
    const response = await iam.mode();
    providerMode.value = normalizeProviderMode((response as any)?.data);
  } catch {
    providerMode.value = defaultProviderMode();
  }
  reloadConfig();
});
</script>

<style scoped>
.dark .wecom-config-page :deep(label) {
  color: #e2e8f0;
}

.dark .wecom-config-page :deep(input::placeholder),
.dark .wecom-config-page :deep(textarea::placeholder) {
  color: #94a3b8;
}
</style>
