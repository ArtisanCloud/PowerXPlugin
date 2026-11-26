import { computed, reactive, ref, watch } from 'vue'
import { PublisherClientError, type PublisherClient } from './client'
import { activeUploadStages } from './types'
import type {
  PublisherMarket,
  PublisherUpload,
  PublisherMarketVersion
} from './types'

export function createMarketWorkspace(client: PublisherClient) {
  const draft = reactive({ name: '', api_root: '', console_base: '' })
  const markets = ref<PublisherMarket[]>([]),
    uploads = ref<PublisherUpload[]>([])
  const selectedMarketID = ref(''),
    selectedUploadID = ref('')
  const versions = ref<PublisherMarketVersion[]>([])
  const versionsLoaded = ref(false)
  const versionsPage = ref(1),
    versionsTotal = ref(0)
  watch(selectedMarketID, () => {
    versions.value = []
    versionsLoaded.value = false
    versionsPage.value = 1
    versionsTotal.value = 0
  })
  const error = ref(''),
    errorRequestID = ref(''),
    busy = ref(''),
    needsReconcile = ref(false),
    pollPaused = ref(false)
  const selectedMarket = computed(() =>
    markets.value.find((m) => m.connection_id === selectedMarketID.value)
  )
  const selectedUpload = computed(() =>
    uploads.value.find((u) => u.upload_id === selectedUploadID.value)
  )
  const ongoing = computed(() =>
    uploads.value.filter((u) => activeUploadStages.includes(u.stage))
  )
  const canUpload = computed(
    () =>
      !!selectedMarket.value?.verified_at &&
      !!selectedMarket.value.identity?.credential.scopes.includes(
        'packages:write'
      ) &&
      Date.parse(selectedMarket.value.identity.credential.expires_at) >
        Date.now() &&
      ['package-upload', 'idempotent-writes'].every((f) =>
        selectedMarket.value!.features.includes(f)
      )
  )
  let timer: ReturnType<typeof setTimeout> | undefined,
    epoch = 0,
    disposed = false,
    deadline = 0
  function stop() {
    epoch++
    clearTimeout(timer)
    timer = undefined
  }
  function fail(cause: unknown) {
    errorRequestID.value =
      cause instanceof PublisherClientError ? cause.requestID : ''
    error.value =
      cause instanceof PublisherClientError
        ? cause.code
        : 'PUBLISHER_PROXY_RESPONSE_INVALID'
  }
  function merge(u: PublisherUpload) {
    const i = uploads.value.findIndex((v) => v.upload_id === u.upload_id)
    if (i < 0) uploads.value.unshift(u)
    else uploads.value[i] = u
  }
  function poll() {
    stop()
    if (disposed || !ongoing.value.length) return
    if (!deadline) deadline = Date.now() + 7200000
    const captured = epoch
    const tick = async () => {
      if (disposed || captured !== epoch) return
      if (Date.now() >= deadline) {
        pollPaused.value = true
        return
      }
      try {
        const tasks = await Promise.all(
          ongoing.value.map((u) => client.upload(u.upload_id))
        )
        if (disposed || captured !== epoch) return
        tasks.forEach(merge)
        if (ongoing.value.length) timer = setTimeout(tick, 2000)
      } catch (cause) {
        if (captured === epoch && !disposed) {
          fail(cause)
          pollPaused.value = true
        }
      }
    }
    timer = setTimeout(tick, 2000)
  }
  async function act(
    name: string,
    work: () => Promise<void>,
    mutation = false
  ) {
    if (disposed || busy.value || (mutation && needsReconcile.value))
      return false
    busy.value = name
    error.value = ''
    errorRequestID.value = ''
    try {
      await work()
      return true
    } catch (cause) {
      fail(cause)
      if (
        mutation &&
        [
          'PUBLISHER_PROXY_UNAVAILABLE',
          'PUBLISHER_PROXY_RESPONSE_INVALID'
        ].includes(error.value)
      )
        needsReconcile.value = true
      return false
    } finally {
      busy.value = ''
    }
  }
  async function refresh() {
    const [m, u] = await Promise.all([client.markets(), client.uploads()])
    if (disposed) return
    markets.value = m
    uploads.value = u
    if (!m.some(connection => connection.connection_id === selectedMarketID.value))
      selectedMarketID.value = m[0]?.connection_id || ''
    if (!selectedUploadID.value) selectedUploadID.value = u[0]?.upload_id || ''
    needsReconcile.value = false
    pollPaused.value = false
    poll()
  }
  function acceptMarket(m: PublisherMarket) {
    const i = markets.value.findIndex(
      (v) => v.connection_id === m.connection_id
    )
    if (i < 0) markets.value.push(m)
    else markets.value[i] = m
    selectedMarketID.value = m.connection_id
  }
  return {
    draft,
    markets,
    uploads,
    selectedMarketID,
    selectedUploadID,
    selectedMarket,
    selectedUpload,
    versions,
    versionsLoaded,
    versionsPage,
    versionsTotal,
    error,
    errorRequestID,
    busy,
    needsReconcile,
    pollPaused,
    ongoing,
    canUpload,
    refresh: () => act('refresh', refresh),
    add: (input: Pick<PublisherMarket, 'name' | 'api_root' | 'console_base'>) =>
      act('add', async () => acceptMarket(await client.addMarket(input)), true),
    bind: (token: string) =>
      act('bind', async () => {
        if (selectedMarketID.value)
          acceptMarket(await client.bindMarket(selectedMarketID.value, token))
      }, true),
    verify: () =>
      act('verify', async () => {
        if (selectedMarketID.value)
          acceptMarket(await client.verifyMarket(selectedMarketID.value))
      }),
    disconnect: () =>
      act(
        'disconnect',
        async () => {
          if (selectedMarketID.value)
            acceptMarket(await client.disconnectMarket(selectedMarketID.value))
        },
        true
      ),
    readVersions: (page = 1) =>
      act('versions', async () => {
        if (selectedMarketID.value) {
          const result = await client.marketVersions(
            selectedMarketID.value,
            page
          )
          versions.value = result.items
          versionsPage.value = result.page
          versionsTotal.value = result.total
          versionsLoaded.value = true
        }
      }),
    start: (buildID: string) =>
      act(
        'upload',
        async () => {
          const u = await client.startUpload(selectedMarketID.value, buildID)
          merge(u)
          selectedUploadID.value = u.upload_id
          deadline = 0
          poll()
        },
        true
      ),
    resume: () =>
      act(
        'resume',
        async () => {
          if (!selectedUpload.value) return
          merge(await client.resumeUpload(selectedUpload.value.upload_id))
          deadline = 0
          poll()
        },
        true
      ),
    cancel: (id: string) =>
      act(
        'cancel',
        async () => {
          stop()
          merge(await client.cancelUpload(id))
          poll()
        },
        true
      ),
    resumePolling: () => {
      deadline = 0
      pollPaused.value = false
      poll()
    },
    dispose: () => {
      disposed = true
      stop()
    }
  }
}
