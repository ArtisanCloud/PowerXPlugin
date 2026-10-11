<template>
  <section class="space-y-4 rounded-lg border border-gray-200 bg-white p-4 text-gray-900 dark:border-gray-700 dark:bg-gray-900/30 dark:text-gray-100">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div><h3 class="font-medium text-gray-900 dark:text-white">{{ t('identityLab.title') }}</h3><p class="text-sm text-gray-600 dark:text-gray-300">{{ t('identityLab.description') }}</p></div>
      <div class="flex flex-wrap gap-2">
        <UButton :disabled="busy || !customerUUID" @click="openDialog('bind')">{{ t('identityLab.add') }}</UButton>
        <UButton variant="soft" :disabled="busy" @click="openDialog('create')">{{ t('identityLab.create') }}</UButton>
      </div>
    </div>
    <p class="text-sm">{{ t('identityLab.selected', { name: customerName || t('identityLab.noCustomer') }) }}</p>
    <UAlert v-if="error" color="error" variant="soft" :title="error" />
    <p v-if="loadingRecords" role="status" class="text-sm text-gray-600 dark:text-gray-300">{{ t('common.loading') }}</p>
    <p v-else-if="!customerUUID" class="text-sm text-gray-600 dark:text-gray-300">{{ t('frameworkLab.selectCustomerForRecords') }}</p>
    <div v-else class="overflow-x-auto rounded-md border border-gray-200 dark:border-gray-800"><table class="w-full text-left text-sm"><thead class="bg-gray-50 text-gray-700 dark:bg-gray-800 dark:text-gray-200"><tr><th class="p-3">{{ t('identityLab.subject') }}</th><th class="p-3">{{ t('frameworkLab.contactStatusLabel') }}</th></tr></thead><tbody><tr v-for="item in items" :key="item.identity_uuid" class="border-t border-gray-200 dark:border-gray-800"><td class="p-3 break-all">{{ item.provider_subject }}</td><td class="p-3">{{ t(`frameworkLab.customerStatus.${item.status}`) }}</td></tr><tr v-if="!items.length"><td colspan="2" class="p-3 text-gray-600 dark:text-gray-300">{{ t('identityLab.empty') }}</td></tr></tbody></table></div>
    <div v-if="total > size" class="flex justify-end gap-3"><UButton variant="soft" :disabled="busy || page === 1" @click="changePage(-1)">{{ t('frameworkLab.previousPage') }}</UButton><span>{{ page }} / {{ Math.ceil(total / size) }}</span><UButton variant="soft" :disabled="busy || page * size >= total" @click="changePage(1)">{{ t('frameworkLab.nextPage') }}</UButton></div>
    <UModal v-model:open="open" :title="t(dialogMode === 'bind' ? 'identityLab.add' : 'identityLab.create')" :description="t(dialogMode === 'bind' ? 'identityLab.bindHint' : 'identityLab.createHint')" :ui="{ content: 'w-full max-w-2xl' }">
      <template #body><div class="space-y-5 text-gray-900 dark:text-gray-100">
        <p v-if="dialogMode === 'bind'" class="font-medium">{{ t('identityLab.selected', { name: customerName }) }}</p>
        <div class="flex items-end gap-3">
          <UFormField :label="t('identityLab.mockPlatform')" class="flex-1"><USelect v-model="platform" :items="platformOptions" class="w-full" :disabled="busy" /></UFormField>
          <UButton variant="soft" :disabled="busy" @click="applyMock(true)">{{ t('identityLab.newMock') }}</UButton>
        </div>
        <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('identityLab.mockHint') }}</p>
        <UFormField :label="t('identityLab.subject')" :description="t('identityLab.subjectHelp')" required><UInput v-model="subject" class="w-full" :disabled="busy" /></UFormField>
        <div class="flex flex-wrap gap-2"><UButton variant="soft" :loading="busy" @click="lookup">{{ t('identityLab.lookup') }}</UButton><UButton v-if="dialogMode === 'bind'" :disabled="!customerUUID || busy" @click="bind">{{ t('identityLab.bind', { name: customerName }) }}</UButton></div>
        <UAlert v-if="result" color="info" variant="soft" :title="result" />
        <div v-if="dialogMode === 'create'" class="space-y-3 border-t border-gray-200 pt-4 dark:border-gray-800">
          <h4 class="font-medium text-gray-900 dark:text-white">{{ t('identityLab.create') }}</h4>
          <p class="text-sm text-gray-600 dark:text-gray-300">{{ t('identityLab.createHint') }}</p>
          <UFormField :label="t('frameworkLab.customerType')"><USelect v-model="profile.type" :items="types" class="w-full" :disabled="busy" /></UFormField>
          <div class="grid gap-3 sm:grid-cols-2"><UFormField v-for="field in profileFields" :key="field.key" :label="t(field.label)"><UInput v-model="profile[field.key]" class="w-full" :disabled="busy" :type="field.key === 'primary_email' ? 'email' : 'text'" /></UFormField></div>
          <UCheckbox v-if="profile.type === 'person'" v-model="explicitContact" :label="t('frameworkLab.explicitPrimaryContact')" :disabled="busy" />
          <div v-if="profile.type === 'company' || explicitContact" class="grid gap-3 sm:grid-cols-2"><UFormField v-for="field in contactFields" :key="field.key" :label="t(field.label)"><UInput v-model="contact[field.key]" class="w-full" :disabled="busy" /></UFormField></div>
          <div class="flex justify-end"><UButton :loading="busy" @click="create">{{ t('identityLab.create') }}</UButton></div>
        </div>
        <UAlert v-if="error" color="error" variant="soft" :title="error" />
      </div></template>
    </UModal>
  </section>
</template>
<script setup lang="ts">
import { useCustomerBaseApi, type ExternalIdentityItem } from '~/composables/api/useCustomerBase'
const props = defineProps<{ active: boolean; route: 'local' | 'delegated'; customerUUID: string; customerName: string }>()
const emit = defineEmits<{ created: [customerUUID: string] }>()
const dialogMode = ref<'bind' | 'create'>('bind')
const { t, te } = useI18n()
const api = useCustomerBaseApi()
const open = ref(false), busy = ref(false), error = ref(''), subject = ref(''), result = ref('')
const items = ref<ExternalIdentityItem[]>([]), page = ref(1), total = ref(0), size = 20
const explicitContact = ref(false)
const loadingRecords = ref(false)
let listRequest = 0
const profile = reactive({ type: 'person', display_name: '', nickname: '', given_name: '', family_name: '', primary_email: '', primary_phone: '' })
const contact = reactive({ display_name: '', given_name: '', family_name: '', email: '', phone: '' })
const profileFields = [{ key: 'display_name', label: 'frameworkLab.customerName' }, { key: 'nickname', label: 'frameworkLab.customerNickname' }, { key: 'given_name', label: 'frameworkLab.contactGivenName' }, { key: 'family_name', label: 'frameworkLab.contactFamilyName' }, { key: 'primary_email', label: 'frameworkLab.customerEmail' }, { key: 'primary_phone', label: 'frameworkLab.customerPhone' }] as const
const contactFields = [{ key: 'display_name', label: 'frameworkLab.primaryContactName' }, { key: 'given_name', label: 'frameworkLab.contactGivenName' }, { key: 'family_name', label: 'frameworkLab.contactFamilyName' }, { key: 'email', label: 'frameworkLab.customerEmail' }, { key: 'phone', label: 'frameworkLab.customerPhone' }] as const
const types = computed(() => [{ value: 'person', label: t('frameworkLab.customerTypePerson') }, { value: 'company', label: t('frameworkLab.customerTypeCompany') }])
type MockPlatform = 'shopify' | 'wechat' | 'woocommerce'
const platform = ref<MockPlatform>('shopify')
const platformOptions = computed(() => (['shopify', 'wechat', 'woocommerce'] as const).map(value => ({ value, label: t(`identityLab.platforms.${value}`) })))
const samples: Record<'bind' | 'create', Partial<Record<MockPlatform, string>>> = { bind: {}, create: {} }
function openDialog(mode: 'bind' | 'create') {
  if (busy.value || (mode === 'bind' && !props.customerUUID)) return
  dialogMode.value = mode
  applyMock()
  open.value = true
}
function applyMock(fresh = false) {
  const mockIDs = samples[dialogMode.value]
  const key = platform.value
  if (fresh || !mockIDs[key]) mockIDs[key] = `${Date.now()}${Math.floor(Math.random() * 1000000).toString().padStart(6, '0')}`
  const id = mockIDs[key]!
  const subjects: Record<MockPlatform, string> = {
    shopify: `shop:framework-lab-demo.myshopify.com:customer:gid://shopify/Customer/${id}`,
    wechat: `wechat:framework-lab-demo:openid:mock_${id}`,
    woocommerce: `woocommerce:framework-lab-demo.example:customer:${id}`,
  }
  subject.value = subjects[key]
  const name = t('identityLab.mockCustomerName', { platform: t(`identityLab.platforms.${key}`), id: id.slice(-6) })
  const email = `mock-${key}-${id.slice(-6)}@example.com`
  const given = t('identityLab.mockGivenName')
  const family = t('identityLab.mockFamilyName')
  const nickname = t('identityLab.mockNickname', { id: id.slice(-6) })
  const phone = `+120255501${id.slice(-2)}`
  Object.assign(profile, { type: 'person', display_name: name, nickname, given_name: given, family_name: family, primary_email: email, primary_phone: phone })
  Object.assign(contact, { display_name: name, given_name: given, family_name: family, email, phone })
  explicitContact.value = false
  error.value = ''; result.value = ''
}
watch(platform, () => applyMock())
watch(open, value => { if (value && !subject.value) applyMock() })
let generation = 0
// Keep the sample when changing customers so the same identity can exercise conflicts.
watch(() => [props.route, props.customerUUID], () => { generation++; open.value = false; items.value = []; total.value = 0; page.value = 1; result.value = ''; error.value = '' })
async function invoke(operation: string, body: Record<string, unknown> = {}) {
  if (busy.value) return
  const run = generation
  busy.value = true; error.value = ''; result.value = ''
  try {
    const response = await api.externalIdentities({ operation, ...body }, props.route)
    if (run !== generation) return
    return response.data
  } catch (cause: any) {
    if (run !== generation) return
    const code = cause?.data?.error?.code || cause?.data?.code || ''
    error.value = te(`identityLab.errors.${code}`) ? t(`identityLab.errors.${code}`) : t('identityLab.failed')
  } finally { busy.value = false }
}
async function lookup() { const data = await invoke('lookup', { provider_subject: subject.value.trim() }); if (data) result.value = t(data.found ? 'identityLab.found' : 'identityLab.notFound') }
async function list() {
  const request = ++listRequest
  const customer = props.customerUUID, route = props.route
  items.value = []; total.value = 0
  if (!customer || !props.active) { loadingRecords.value = false; return }
  loadingRecords.value = true; error.value = ''
  try {
    const response = await api.externalIdentities({ operation: 'list_by_customer', customer_uuid: customer, page: page.value, page_size: size }, route)
    if (request !== listRequest || customer !== props.customerUUID || route !== props.route) return
    items.value = response.data?.items || []; total.value = response.data?.total || 0
  } catch (cause: any) {
    if (request !== listRequest) return
    const code = cause?.data?.error?.code || ''
    error.value = te(`identityLab.errors.${code}`) ? t(`identityLab.errors.${code}`) : t('identityLab.failed')
  } finally { if (request === listRequest) loadingRecords.value = false }
}
watch(() => [props.customerUUID, props.route, props.active], () => { page.value = 1; void list() }, { immediate: true, flush: 'post' })

async function changePage(delta: number) { page.value += delta; await list() }
async function bind() {
  if (dialogMode.value !== 'bind' || !props.customerUUID) return
  const data = await invoke('bind', { customer_uuid: props.customerUUID, provider_subject: subject.value.trim() })
  if (data) {
    page.value = 1
    await list()
    result.value = t('identityLab.bound')
  }
}
async function create() {
  if (dialogMode.value !== 'create') return
  const data = await invoke('create_and_bind', { provider_subject: subject.value.trim(), customer: { ...profile, ...((profile.type === 'company' || explicitContact.value) ? { primary_contact: { ...contact } } : {}) } })
  if (data) {
    if (!data.customer_uuid) { error.value = t('identityLab.failed'); return }
    open.value = false
    emit('created', data.customer_uuid)
  }
}
</script>
