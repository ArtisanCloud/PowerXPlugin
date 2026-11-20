<template>
  <div class="lark-config-page p-6 space-y-6">
    <div class="flex flex-col gap-2 lg:flex-row lg:items-start lg:justify-between">
      <div>
        <p class="text-sm uppercase tracking-wide text-gray-500 dark:text-slate-300">Channel Configuration</p>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">飞书配置</h1>
        <p class="text-sm text-gray-600 dark:text-slate-300">配置完成后可用于扫码挑战重写与同步任务鉴权。</p>
      </div>
      <div class="flex gap-2">
        <UBadge color="neutral" variant="soft">{{ providerModeLabel }}</UBadge>
        <UButton variant="outline" :loading="refreshing" @click="reloadConfig">刷新</UButton>
        <UButton color="primary" :loading="saving" :disabled="readonlyMode" @click="saveForm">保存配置</UButton>
      </div>
    </div>

    <UCard>
      <template #header>
        <h3 class="text-lg font-semibold">飞书 OAuth 参数</h3>
      </template>

      <div class="mb-4 rounded-lg border border-cyan-500/40 bg-cyan-500/10 px-4 py-3 text-sm text-cyan-100">
        <p>联调建议：优先使用可公网访问的 HTTPS 域名作为回调 Host，避免 `localhost/127.0.0.1` 导致授权回调失败。</p>
        <p class="mt-1 text-cyan-200/90">字段映射：`Tenant Key -> tenant_key`，`App ID -> app_id`，`App Secret -> app_secret`。</p>
      </div>

      <UForm :state="form" class="space-y-4" :disabled="readonlyMode">
        <div class="grid gap-4 md:grid-cols-2">
          <UFormField label="配置状态">
            <USelect v-model="form.status" :items="statusOptions" />
          </UFormField>
          <UFormField label="轮换周期（天）">
            <UInput v-model="form.rotationDays" type="number" min="1" max="365" />
          </UFormField>
        </div>

        <UFormField label="Host:Port（接收回调）" required description="示例：https://debug.artisan-cloud.com">
          <UInput v-model="form.callbackHost" placeholder="https://plugin.example.com" />
        </UFormField>

        <div class="grid gap-4 md:grid-cols-2">
          <UFormField label="Tenant Key" required>
            <UInput v-model="form.tenantKey" placeholder="tenant-xxxxxxxx" />
          </UFormField>
          <UFormField label="App ID" required>
            <UInput v-model="form.appId" placeholder="cli_xxxxxxxxxx" />
          </UFormField>
        </div>

        <UFormField label="App Secret" required>
          <UInput v-model="form.appSecret" :type="showSecret ? 'text' : 'password'" placeholder="飞书应用 App Secret">
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

        <UFormField label="SDK HttpDebug">
          <div class="flex items-center justify-between rounded-md border border-slate-700/60 px-3 py-2">
            <span class="text-xs text-slate-300">启用后输出渠道 SDK 调试日志</span>
            <USwitch v-model="form.httpDebug" />
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
import { useApiClient } from "~/composables/api/_client";
import { useIAMService } from "~/composables/api/services/iamService";
import { defaultProviderMode, normalizeProviderMode, type ProviderModeDiagnostics } from "~/composables/api/useProviderMode";
import { useUserStore } from "~/stores/user";
import { resolveTenantUUIDForRequest } from "~/utils/tenant-context";

definePageMeta({
  title: "飞书配置",
  icon: "i-lucide-bird",
  order: 14,
});

type LarkConfig = {
  status: string;
  rotationDays: number;
  callbackHost: string;
  tenantKey: string;
  appId: string;
  appSecret: string;
  httpDebug: boolean;
};

type LarkConfigResponse = {
  status?: string;
  rotationDays?: number;
  rotation_days?: number;
  callbackHost?: string;
  callback_host?: string;
  tenantKey?: string;
  tenant_key?: string;
  appId?: string;
  app_id?: string;
  appSecret?: string;
  app_secret?: string;
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

const defaultForm = (): LarkConfig => ({
  status: "active",
  rotationDays: 30,
  callbackHost: "",
  tenantKey: "",
  appId: "",
  appSecret: "",
  httpDebug: false,
});

const form = reactive(defaultForm());
const userStore = useUserStore();
const { currentTenantUuid } = storeToRefs(userStore);

const tenantUUID = computed(() => {
  const fromStore = currentTenantUuid.value?.trim();
  if (fromStore) return fromStore;
  return resolveTenantUUIDForRequest() || "00000000-0000-0000-0000-000000000001";
});

const normalizeStatus = (raw?: string) => ((raw || "").trim().toUpperCase() === "REVOKED" ? "inactive" : "active");

const applyServerConfig = (data?: LarkConfigResponse | null) => {
  if (!data) {
    Object.assign(form, defaultForm());
    return;
  }
  Object.assign(form, defaultForm(), {
    status: normalizeStatus(data.status),
    rotationDays: Number(data.rotation_days ?? data.rotationDays ?? 30) || 30,
    callbackHost: String(data.callback_host ?? data.callbackHost ?? ""),
    tenantKey: String(data.tenant_key ?? data.tenantKey ?? ""),
    appId: String(data.app_id ?? data.appId ?? ""),
    appSecret: String(data.app_secret ?? data.appSecret ?? ""),
    httpDebug: Boolean(data.http_debug ?? data.httpDebug ?? false),
  });
};

const validateRequired = () => {
  if (!tenantUUID.value.trim()) return "tenant_uuid 为空，请先切换租户";
  const host = form.callbackHost.trim();
  if (!host) return "Host:Port 必填";
  if (!/^https:\/\//i.test(host)) return "Host:Port 必须以 https:// 开头";
  if (/localhost|127\.0\.0\.1/i.test(host)) return "Host:Port 不建议使用 localhost/127.0.0.1";
  if (!form.tenantKey.trim()) return "Tenant Key 必填";
  if (!form.appId.trim()) return "App ID 必填";
  if (!form.appSecret.trim()) return "App Secret 必填";
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
    const response = await put<any>("/admin/iam/channels/lark/config", {
      status: form.status,
      rotation_days: Number(form.rotationDays) || 30,
      callback_host: form.callbackHost.trim(),
      tenant_key: form.tenantKey.trim(),
      app_id: form.appId.trim(),
      app_secret: form.appSecret.trim(),
      http_debug: form.httpDebug,
    });
    applyServerConfig((response?.data || response) as LarkConfigResponse);
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

const reloadConfig = async () => {
  refreshing.value = true;
  try {
    const response = await get<any>("/admin/iam/channels/lark/config");
    applyServerConfig((response?.data || response) as LarkConfigResponse);
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
.dark .lark-config-page :deep(label) {
  color: #e2e8f0;
}

.dark .lark-config-page :deep(input::placeholder),
.dark .lark-config-page :deep(textarea::placeholder) {
  color: #94a3b8;
}
</style>
