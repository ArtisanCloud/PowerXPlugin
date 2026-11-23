import { afterEach, expect, it, vi } from 'vitest'
import { ref, nextTick } from 'vue'
import { createReviewWorkspace } from '../../../../../framework/frontend/nuxt/framework-admin/publisher/review-workspace'
import {
  PublisherClientError,
  type PublisherClient
} from '../../../../../framework/frontend/nuxt/framework-admin/publisher/client'
import type {
  PublisherMarket,
  PublisherSubmission,
  PublisherSubmissionPreparation,
  PublisherSubmissionPreview
} from '../../../../../framework/frontend/nuxt/framework-admin/publisher/types'
const uuid = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const m = () =>
  ({
    connection_id: uuid(1),
    name: 'fixture',
    api_root: 'https://api.example',
    console_base: 'https://market.example',
    features: ['publication-submit', 'idempotent-writes'],
    verified_at: new Date().toISOString(),
    identity: {
      user: { user_id: uuid(2), name: 'fixture' },
      developer: { developer_id: uuid(3), name: 'fixture' },
      credential: {
        token_id: uuid(4),
        plugin_ids: ['com.example.publisher'],
        scopes: ['plugins:read', 'publications:read', 'publications:submit'],
        expires_at: new Date(Date.now() + 3600000).toISOString()
      }
    }
  }) satisfies PublisherMarket
const meta = {
  description: 'fixture_description',
  license: 'MIT',
  min_core: '1.0.0',
  support_url: 'https://support.example',
  icon_url: '',
  security_statement: 'fixture_security',
  privacy_statement: 'fixture_privacy'
}
const prepared: PublisherSubmissionPreparation = {
  version: {
    version_id: uuid(5),
    plugin_id: 'com.example.publisher',
    version: '1.2.0',
    channel: 'stable',
    availability_status: 'draft',
    artifacts: [
      { artifact_id: uuid(6), os: 'linux', arch: 'amd64', size: 10, sha256: 'a'.repeat(64) },
      { artifact_id: uuid(7), os: 'darwin', arch: 'arm64', size: 20, sha256: 'b'.repeat(64) }
    ],
    market_version_url: 'https://market.example/version'
  },
  options: {
    versions: ['1.0.0'],
    licenses: ['MIT'],
    source: 'https://github.com/example/releases',
    version_source: 'verified_snapshot'
  }
}
const preview: PublisherSubmissionPreview = {
  input: {
    connection_id: uuid(1),
    version_id: uuid(5),
    version: '1.2.0',
    metadata: meta,
    expected_artifacts: prepared.version.artifacts
  },
  request_sha256: 'c'.repeat(64)
}
const task = (stage = 'sending') =>
  ({
    schema: 'powerx.plugin-publisher-submission/v1',
    submission_id: uuid(8),
    operation_id: uuid(9),
    plugin_id: 'com.example.publisher',
    ...preview.input,
    request_sha256: preview.request_sha256,
    stage,
    attempts: 1,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString()
  }) satisfies PublisherSubmission
function fixture() {
  const market = ref<PublisherMarket | undefined>(m())
  const api = {
    submissions: vi.fn().mockResolvedValue([]),
    submission: vi.fn().mockResolvedValue(task('succeeded')),
    prepareSubmission: vi.fn().mockResolvedValue(prepared),
    previewSubmission: vi.fn().mockResolvedValue(preview),
    startSubmission: vi.fn().mockResolvedValue(task()),
    resumeSubmission: vi.fn().mockResolvedValue(task()),
    cancelSubmission: vi.fn().mockResolvedValue(task('pending')),
    publications: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, size: 20 })
  }
  return { api, market, state: createReviewWorkspace(api as unknown as PublisherClient, market) }
}
afterEach(() => vi.useRealTimers())
it('requires advertised capability and dedicated submit scope', () => {
  const { state, market } = fixture()
  expect(state.capable.value).toBe(true)
  market.value!.features = []
  expect(state.capable.value).toBe(false)
  market.value!.features = m().features
  market.value!.identity!.credential.scopes = ['plugins:read', 'packages:write']
  expect(state.capable.value).toBe(false)
  state.dispose()
})
it('preserves drafts by connection and version and discards stale preparation', async () => {
  const { state, api, market } = fixture()
  state.versionID.value = uuid(5)
  await nextTick()
  state.draft.value.description = meta.description
  state.versionID.value = uuid(10)
  await nextTick()
  expect(state.draft.value.description).toBe('')
  state.versionID.value = uuid(5)
  await nextTick()
  expect(state.draft.value.description).toBe(meta.description)
  let resolve!: (p: PublisherSubmissionPreparation) => void
  api.prepareSubmission.mockImplementationOnce(
    () =>
      new Promise((r) => {
        resolve = r
      })
  )
  const work = state.prepare()
  market.value = { ...m(), connection_id: uuid(11) }
  await nextTick()
  resolve(prepared)
  await work
  expect(state.preparation.value).toBeNull()
  state.dispose()
})
it('previews the entire prepared set and never invents default choices', async () => {
  const { state, api } = fixture()
  state.versionID.value = uuid(5)
  await nextTick()
  await state.prepare()
  expect(state.draft.value.license).toBe('')
  Object.assign(state.draft.value, meta)
  await nextTick()
  await state.previewInput()
  expect(api.previewSubmission.mock.calls[0]![0].expected_artifacts).toHaveLength(2)
  expect(state.preview.value?.request_sha256).toBe(preview.request_sha256)
  state.dispose()
})
it('blocks ambiguous duplicate submissions until explicit reconciliation', async () => {
  const { state, api } = fixture()
  state.preview.value = preview
  api.startSubmission.mockRejectedValueOnce(new PublisherClientError('PUBLISHER_PROXY_UNAVAILABLE'))
  await state.submit()
  await state.submit()
  expect(api.startSubmission).toHaveBeenCalledTimes(1)
  expect(state.needsReconcile.value).toBe(true)
  await state.refresh()
  state.preview.value = preview
  await state.submit()
  expect(api.startSubmission).toHaveBeenCalledTimes(2)
  state.dispose()
})
it('keeps active tasks visible across target changes and stops queries on completion', async () => {
  vi.useFakeTimers()
  const { state, api, market } = fixture()
  state.preview.value = preview
  await state.submit()
  market.value = { ...m(), connection_id: uuid(20) }
  expect(state.ongoing.value).toHaveLength(1)
  await vi.advanceTimersByTimeAsync(2000)
  expect(state.selected.value?.stage).toBe('succeeded')
  await vi.advanceTimersByTimeAsync(10000)
  expect(api.submission).toHaveBeenCalledTimes(1)
  state.dispose()
})
it('discards a late query after cancellation and keeps the original operation', async () => {
  vi.useFakeTimers()
  const { state, api } = fixture()
  state.preview.value = preview
  await state.submit()
  let resolve!: (p: PublisherSubmission) => void
  api.submission.mockImplementationOnce(
    () =>
      new Promise((r) => {
        resolve = r
      })
  )
  await vi.advanceTimersByTimeAsync(2000)
  await state.cancel(uuid(8))
  resolve(task())
  await Promise.resolve()
  await Promise.resolve()
  expect(state.selected.value?.stage).toBe('pending')
  expect(state.selected.value?.operation_id).toBe(uuid(9))
  state.dispose()
})
it('keeps rejected snapshot immutable while preparing revised details', async () => {
  const { state } = fixture()
  const rejected: PublisherSubmission = {
    ...task('succeeded'),
    current_publication: {
      publication_id: uuid(30),
      plugin_id: 'com.example.publisher',
      version_id: uuid(5),
      version: '1.2.0',
      status: 'rejected',
      metadata: meta,
      artifacts: prepared.version.artifacts,
      submitted_at: new Date().toISOString(),
      reviewed_at: null,
      review_reason: 'fixture_reason',
      published_at: null,
      publish_reason: '',
      market_publication_url: 'https://market.example/publication'
    }
  }
  state.tasks.value = [rejected]
  state.selectedID.value = rejected.submission_id
  await state.revise()
  state.draft.value.description = 'fixture_changed'
  expect(state.selected.value?.metadata.description).toBe(meta.description)
  expect(state.existing.value).toBeUndefined()
  expect(state.attention.value).toHaveLength(1)
  state.dispose()
})

function reviewedTask(status: 'submitted' | 'rejected' | 'approved' | 'published', n = 40): PublisherSubmission {
  const input = task('succeeded')
  const publication = {
    publication_id: uuid(n + 2), plugin_id: input.plugin_id,
    version_id: input.version_id, version: input.version, status,
    metadata: { ...meta }, artifacts: structuredClone(prepared.version.artifacts),
    submitted_at: `2026-11-12T00:00:${n === 20 ? '01' : '02'}Z`,
    reviewed_at: null, review_reason: '', published_at: null, publish_reason: '',
    market_publication_url: 'https://market.example/publication/' + uuid(n + 2)
  }
  return {
    ...input, submission_id: uuid(n), operation_id: uuid(n + 1),
    created_at: publication.submitted_at, updated_at: publication.submitted_at,
    receipt: { publication_id: publication.publication_id, version_id: input.version_id,
      market_publication_url: publication.market_publication_url },
    current_publication: publication
  }
}

it.each(['submitted', 'rejected', 'approved', 'published'] as const)(
  'updates the collection to %s after task refresh without changing the preparation or old application',
  async (status) => {
    const { state, api } = fixture()
    state.versionID.value = prepared.version.version_id
    await nextTick()
    const snapshot = structuredClone(prepared)
    snapshot.version.availability_status = 'rejected'
    state.preparation.value = snapshot
    const old = reviewedTask('rejected', 20)
    const original = JSON.stringify(old)
    const latest = reviewedTask('submitted')
    state.tasks.value = [old, latest]
    state.selectedID.value = latest.submission_id
    api.resumeSubmission.mockResolvedValueOnce(reviewedTask(status))
    await state.resume()
    expect(state.collectionState.value).toEqual({ source: 'publication', status })
    expect(state.preparation.value?.version.availability_status).toBe('rejected')
    expect(JSON.stringify(state.tasks.value.find(t => t.submission_id === old.submission_id))).toBe(original)
    state.selectedID.value = old.submission_id
    expect(state.selected.value?.current_publication?.status).toBe('rejected')
    expect(state.collectionState.value).toEqual({ source: 'publication', status })
    state.dispose()
  }
)
it('does not reuse a stale rejected or approved observation while the latest task is unresolved', async () => {
  const { state } = fixture()
  state.versionID.value = prepared.version.version_id
  await nextTick()
  state.preparation.value = structuredClone(prepared)
  state.preparation.value.version.availability_status = 'rejected'
  const latest = reviewedTask('approved')
  state.tasks.value = [reviewedTask('rejected', 20), { ...latest, stage: 'reconciling' }]
  expect(state.collectionState.value).toEqual({ source: 'unconfirmed' })
  state.tasks.value[1] = { ...latest, stage: 'pending', error: { code: 'PUBLISHER_MARKET_CREDENTIAL', message_key: 'PUBLISHER_MARKET_CREDENTIAL' } }
  expect(state.collectionState.value).toEqual({ source: 'unconfirmed' })
  state.dispose()
})
it('keeps collection status scoped to the prepared connection and version', async () => {
  const { state } = fixture()
  state.versionID.value = prepared.version.version_id
  await nextTick()
  state.preparation.value = structuredClone(prepared)
  state.tasks.value = [
    { ...reviewedTask('published'), connection_id: uuid(90) },
    { ...reviewedTask('approved'), version_id: uuid(91) }
  ]
  expect(state.collectionState.value).toEqual({ source: 'version', status: 'draft' })
  state.dispose()
})
