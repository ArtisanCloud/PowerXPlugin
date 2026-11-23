import { computed, reactive, ref, watch, type Ref } from 'vue'
import { PublisherClientError, type PublisherClient } from './client'
import { activeSubmissionStages } from './types'
import type {
  PublisherMarket,
  PublisherMetadata,
  PublisherSubmission,
  PublisherSubmissionPreparation,
  PublisherSubmissionPreview,
  PublisherPublication,
  PublisherPage
} from './types'
const emptyMetadata = (): PublisherMetadata => ({
  description: '',
  license: '',
  min_core: '',
  support_url: '',
  icon_url: '',
  security_statement: '',
  privacy_statement: ''
})
export function createReviewWorkspace(
  client: PublisherClient,
  market: Ref<PublisherMarket | undefined>
) {
  const versionID = ref(''),
    tasks = ref<PublisherSubmission[]>([]),
    selectedID = ref(''),
    preparation = ref<PublisherSubmissionPreparation | null>(null),
    preview = ref<PublisherSubmissionPreview | null>(null)
  const drafts = reactive<Record<string, PublisherMetadata>>({}),
    error = ref(''),
    errorRequestID = ref(''),
    busy = ref(''),
    needsReconcile = ref(false),
    pollPaused = ref(false)
  const history = ref<PublisherPage<PublisherPublication>>({
      items: [],
      page: 1,
      size: 20,
      total: 0
    }),
    historyLoaded = ref(false)
  const key = computed(
    () => `${market.value?.connection_id || ''}/${versionID.value}`
  )
  const draft = computed(() => (drafts[key.value] ||= emptyMetadata()))
  const selected = computed(() =>
    tasks.value.find((t) => t.submission_id === selectedID.value)
  )
  const ongoing = computed(() =>
    tasks.value.filter((t) => activeSubmissionStages.includes(t.stage))
  )
  const attention = computed(() =>
    tasks.value.filter(
      (t) => t.error || t.current_publication?.status === 'rejected'
    )
  )
  const capable = computed(
    () =>
      !!market.value?.verified_at &&
      ['publication-submit', 'idempotent-writes'].every((f) =>
        market.value!.features.includes(f)
      ) &&
      ['publications:read', 'publications:submit'].every((s) =>
        market.value!.identity?.credential.scopes.includes(s)
      ) &&
      Date.parse(market.value!.identity!.credential.expires_at) > Date.now()
  )
  const existing = computed(() =>
    tasks.value.find(
      (t) =>
        t.connection_id === market.value?.connection_id &&
        t.version_id === versionID.value &&
        t.stage !== 'failed' &&
        !(
          t.stage === 'succeeded' &&
          t.current_publication?.status === 'rejected'
        )
    )
  )
  // Preparation freezes the input set; its version status is only a pre-submit
  // observation. Follow the latest task for this connection/version, even when
  // the user is inspecting an older rejected application below.
  const collectionState = computed(() => {
    const prepared = preparation.value
    if (!prepared || !market.value) return null
    const latest = tasks.value
      .filter(
        (task) =>
          task.connection_id === market.value!.connection_id &&
          task.version_id === prepared.version.version_id &&
          task.plugin_id === prepared.version.plugin_id
      )
      .sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at))[0]
    if (!latest)
      return {
        source: 'version' as const,
        status: prepared.version.availability_status
      }
    const publication = latest.current_publication
    if (
      latest.stage === 'succeeded' &&
      !latest.error &&
      publication?.version_id === prepared.version.version_id &&
      publication.plugin_id === prepared.version.plugin_id
    )
      return { source: 'publication' as const, status: publication.status }
    return { source: 'unconfirmed' as const }
  })
  let timer: ReturnType<typeof setTimeout> | undefined,
    epoch = 0,
    contextEpoch = 0,
    disposed = false,
    deadline = 0
  watch(key, () => {
    contextEpoch++
    preparation.value = null
    preview.value = null
  })
  watch(
    () => market.value?.connection_id,
    () => {
      versionID.value = ''
      historyLoaded.value = false
      history.value = { items: [], page: 1, size: 20, total: 0 }
    }
  )
  watch(
    draft,
    () => {
      preview.value = null
    },
    { deep: true }
  )
  function stop() {
    epoch++
    clearTimeout(timer)
    timer = undefined
  }
  function merge(task: PublisherSubmission) {
    const i = tasks.value.findIndex(
      (t) => t.submission_id === task.submission_id
    )
    if (i < 0) tasks.value.unshift(task)
    else tasks.value[i] = task
  }
  function fail(cause: unknown) {
    errorRequestID.value =
      cause instanceof PublisherClientError ? cause.requestID : ''
    error.value =
      cause instanceof PublisherClientError
        ? cause.code
        : 'PUBLISHER_PROXY_RESPONSE_INVALID'
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
        const result = await Promise.all(
          ongoing.value.map((t) => client.submission(t.submission_id))
        )
        if (disposed || captured !== epoch) return
        result.forEach(merge)
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
  async function prepare() {
    if (!market.value || !versionID.value) return
    const captured = contextEpoch
    const p = await client.prepareSubmission(
      market.value.connection_id,
      versionID.value
    )
    if (captured !== contextEpoch || disposed) return
    preparation.value = p
    preview.value = null
  }
  return {
    versionID,
    tasks,
    selectedID,
    selected,
    preparation,
    preview,
    draft,
    error,
    errorRequestID,
    busy,
    needsReconcile,
    pollPaused,
    ongoing,
    attention,
    capable,
    existing,
    collectionState,
    history,
    historyLoaded,
    refresh: () =>
      act('refresh', async () => {
        tasks.value = await client.submissions()
        if (!selectedID.value)
          selectedID.value = tasks.value[0]?.submission_id || ''
        needsReconcile.value = false
        pollPaused.value = false
        poll()
      }),
    prepare: () => act('prepare', prepare),
    previewInput: () =>
      act('preview', async () => {
        if (!preparation.value || !market.value) return
        const captured = contextEpoch
        const result = await client.previewSubmission({
          connection_id: market.value.connection_id,
          version_id: preparation.value.version.version_id,
          version: preparation.value.version.version,
          metadata: { ...draft.value },
          expected_artifacts: preparation.value.version.artifacts.map((a) => ({
            ...a
          }))
        })
        if (captured === contextEpoch && !disposed) preview.value = result
      }),
    submit: () =>
      act(
        'submit',
        async () => {
          if (!preview.value) return
          const task = await client.startSubmission(preview.value.input)
          merge(task)
          selectedID.value = task.submission_id
          preview.value = null
          deadline = 0
          poll()
        },
        true
      ),
    resume: () =>
      act(
        'resume',
        async () => {
          if (!selected.value) return
          merge(await client.resumeSubmission(selected.value.submission_id))
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
          merge(await client.cancelSubmission(id))
          poll()
        },
        true
      ),
    revise: () =>
      act('revise', async () => {
        const t = selected.value
        if (
          !t ||
          t.connection_id !== market.value?.connection_id ||
          t.current_publication?.status !== 'rejected'
        )
          return
        versionID.value = t.version_id
        await Promise.resolve()
        if (!draft.value.description) Object.assign(draft.value, t.metadata)
        await prepare()
      }),
    readHistory: (page = 1) =>
      act('history', async () => {
        if (!market.value) return
        const captured = market.value.connection_id
        const result = await client.publications(captured, page)
        if (captured === market.value?.connection_id) {
          history.value = result
          historyLoaded.value = true
        }
      }),
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
