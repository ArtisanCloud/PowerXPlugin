import {
  activeBuildStages,
  publisherAPIPath,
  publisherIntent,
  type PublisherConnection,
  type PublisherBuildEnvironment,
  type PublisherPage,
  type PublisherPlan,
  type PublisherPlanInput,
  type PublisherBuild,
  type PublisherSubmission,
  type PublisherSubmissionInput,
  type PublisherSubmissionPreview,
  type PublisherSubmissionPreparation,
  type PublisherPublication,
  type PublisherMarket,
  type PublisherUpload,
  type PublisherMarketVersion,
  type PublisherArtifact,
  type PublisherRequestTrace
} from './types'

const uuid =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/
const date = (value: unknown) =>
  typeof value === 'string' && Number.isFinite(Date.parse(value))
const artifact = (a: any) =>
  a &&
  typeof a.path === 'string' &&
  /^[0-9a-f]{64}$/.test(a.sha256) &&
  Number.isSafeInteger(a.size) &&
  a.size > 0 &&
  a.size <= 2147483648 &&
  Number.isSafeInteger(a.file_count) &&
  a.file_count > 0 &&
  a.file_count <= 50000
const plan = (p: any) =>
  p &&
  p.schema === 'powerx.plugin-publisher-plan/v1' &&
  uuid.test(p.release_plan_id) &&
  typeof p.plugin_id === 'string' &&
  typeof p.version === 'string' &&
  p.channel === 'stable' &&
  p.target?.os === 'linux' &&
  p.target?.arch === 'amd64' &&
  typeof p.release_notes === 'string' &&
  date(p.created_at)
const build = (b: any) =>
  b &&
  b.schema === 'powerx.plugin-publisher-build/v1' &&
  uuid.test(b.build_id) &&
  uuid.test(b.release_plan_id) &&
  typeof b.plugin_id === 'string' &&
  typeof b.version === 'string' &&
  b.target?.os === 'linux' &&
  b.target?.arch === 'amd64' &&
  [
    ...activeBuildStages,
    'ready',
    'failed',
    'cancelled',
    'interrupted'
  ].includes(b.stage) &&
  date(b.created_at) &&
  date(b.updated_at) &&
  (b.stage !== 'ready' || artifact(b.artifact))
const market = (m: any) =>
  m &&
  uuid.test(m.connection_id) &&
  typeof m.name === 'string' &&
  typeof m.api_root === 'string' &&
  typeof m.console_base === 'string' &&
  Array.isArray(m.features) &&
  m.features.every((v: unknown) => typeof v === 'string') &&
  (!m.identity ||
    (typeof m.identity.user?.name === 'string' &&
      uuid.test(m.identity.user.user_id) &&
      typeof m.identity.developer?.name === 'string' &&
      uuid.test(m.identity.developer.developer_id) &&
      date(m.identity.credential?.expires_at) &&
      Array.isArray(m.identity.credential.scopes)))
const version = (v: any) =>
  v &&
  uuid.test(v.version_id) &&
  typeof v.version === 'string' &&
  typeof v.availability_status === 'string' &&
  typeof v.market_version_url === 'string'
const diagnostic = (e: any) => !e || (typeof e.code === 'string' && (!e.params || e.params.request_id === undefined || typeof e.params.request_id === 'string' && uuid.test(e.params.request_id)))
const upload = (u: any) =>
  u &&
  u.schema === 'powerx.plugin-publisher-upload/v1' &&
  diagnostic(u.error) &&
  uuid.test(u.upload_id) &&
  uuid.test(u.operation_id) &&
  uuid.test(u.connection_id) &&
  uuid.test(u.build_id) &&
  typeof u.version === 'string' &&
  [
    'queued',
    'uploading',
    'reconciling',
    'processing',
    'uploaded',
    'pending',
    'succeeded',
    'failed'
  ].includes(u.stage) &&
  artifact(u.artifact) &&
  Number.isSafeInteger(u.bytes_sent) &&
  u.bytes_sent >= 0 &&
  Number.isSafeInteger(u.attempts) &&
  u.attempts >= 0 &&
  u.attempts <= 3 &&
  date(u.created_at) &&
  (!u.current_version || version(u.current_version))
const metadata = (m: any) =>
  m &&
  [
    'description',
    'license',
    'min_core',
    'support_url',
    'icon_url',
    'security_statement',
    'privacy_statement'
  ].every((k) => typeof m[k] === 'string')
const collection = (a: any) =>
  Array.isArray(a) &&
  a.length >= 1 &&
  a.length <= 6 &&
  a.every(
    (v: any) =>
      uuid.test(v.artifact_id) &&
      typeof v.os === 'string' &&
      typeof v.arch === 'string' &&
      /^[a-f0-9]{64}$/.test(v.sha256) &&
      Number.isSafeInteger(v.size) &&
      v.size > 0
  )
const submissionInput = (s: any) =>
  s &&
  uuid.test(s.connection_id) &&
  uuid.test(s.version_id) &&
  typeof s.version === 'string' &&
  metadata(s.metadata) &&
  collection(s.expected_artifacts)
const publication = (p: any) =>
  p &&
  uuid.test(p.publication_id) &&
  uuid.test(p.version_id) &&
  typeof p.version === 'string' &&
  ['submitted', 'rejected', 'approved', 'published'].includes(p.status) &&
  metadata(p.metadata) &&
  collection(p.artifacts) &&
  date(p.submitted_at) &&
  typeof p.review_reason === 'string' &&
  typeof p.market_publication_url === 'string'
const submission = (s: any) =>
  submissionInput(s) &&
  s.schema === 'powerx.plugin-publisher-submission/v1' &&
  diagnostic(s.error) &&
  uuid.test(s.submission_id) &&
  uuid.test(s.operation_id) &&
  /^[a-f0-9]{64}$/.test(s.request_sha256) &&
  [
    'queued',
    'reconciling',
    'sending',
    'processing',
    'submitted',
    'pending',
    'failed',
    'succeeded'
  ].includes(s.stage) &&
  Number.isInteger(s.attempts) &&
  s.attempts >= 0 &&
  s.attempts <= 3 &&
  date(s.created_at) &&
  (!s.current_publication || publication(s.current_publication))
function validPayload(path: string, value: any): boolean {
  if (path === '/build-environment/start') return ['native_available', 'container_available', 'container_ready', 'can_start'].every(key => typeof value?.[key] === 'boolean')
  if (/^\/(uploads|submissions)\/[^/]+\/requests(?:\?.*)?$/.test(path)) {
    const [, kind, taskID] = path.split('/')
    return (
      value &&
      Array.isArray(value.items) &&
      value.items.every(
        (x: any) =>
          x &&
          x.task_kind === kind &&
          x.task_id === taskID &&
          [
            'DiscoveryResponse',
            'MeResponse',
            'PluginResponse',
            'VersionsPageResponse',
            'MarketVersionResponse',
            'UploadReceiptResponse',
            'OperationResponse',
            'PublicationOptionsResponse',
            'PublicationsPageResponse',
            'SubmissionReceiptResponse'
          ].includes(x.response_type) &&
          x.schema === 'powerx.plugin-publisher-request/v1' &&
          ['uploads', 'submissions'].includes(x.task_kind) &&
          uuid.test(x.task_id) &&
          uuid.test(x.operation_id) &&
          uuid.test(x.client_request_id) &&
          (!x.server_request_id || uuid.test(x.server_request_id)) &&
          ['GET', 'POST'].includes(x.method) &&
          typeof x.response_type === 'string' &&
          Number.isInteger(x.http_status) &&
          (x.http_status === 0 ||
            (x.http_status >= 100 && x.http_status <= 599)) &&
          date(x.started_at) &&
          (!x.finished_at || date(x.finished_at))
      ) &&
      Number.isInteger(value.total) &&
      value.total >= value.items.length &&
      Number.isInteger(value.page) &&
      value.page >= 1 &&
      Number.isInteger(value.size) &&
      value.size >= 1 &&
      value.size <= 100 &&
      value.items.length <= value.size
    )
  }

  if (path === '/submissions/preview')
    return (
      submissionInput(value?.input) &&
      /^[a-f0-9]{64}$/.test(value.request_sha256)
    )
  if (path === '/submissions/prepare')
    return (
      version(value?.version) &&
      collection(value.version.artifacts) &&
      typeof value.options?.source === 'string' &&
      ['verified_snapshot', 'official_releases'].includes(
        value.options.version_source
      ) &&
      Array.isArray(value.options?.licenses) &&
      value.options.licenses.length > 0 &&
      value.options.licenses.every((v: unknown) => typeof v === 'string') &&
      Array.isArray(value.options.versions) &&
      value.options.versions.length > 0 &&
      value.options.versions.every((v: unknown) => typeof v === 'string')
    )
  if (path === '/submissions')
    return Array.isArray(value) ? value.every(submission) : !!submission(value)
  if (path.startsWith('/submissions/')) return !!submission(value)
  if (/^\/markets\/[^/]+\/publications(?:\?.*)?$/.test(path))
    return (
      value &&
      Array.isArray(value.items) &&
      value.items.every(publication) &&
      Number.isInteger(value.total) &&
      value.total >= value.items.length &&
      Number.isInteger(value.page) &&
      value.page >= 1 &&
      Number.isInteger(value.size) &&
      value.size >= 1 &&
      value.size <= 100 &&
      value.items.length <= value.size
    )
  if (/^\/markets\/[^/]+\/versions(?:\?.*)?$/.test(path))
    return (
      value &&
      Array.isArray(value.items) &&
      value.items.every(version) &&
      Number.isSafeInteger(value.total) &&
      value.total >= value.items.length &&
      Number.isInteger(value.page) &&
      value.page >= 1 &&
      Number.isInteger(value.size) &&
      value.size >= 1 &&
      value.size <= 100 &&
      value.items.length <= value.size
    )
  if (path === '/markets')
    return Array.isArray(value) ? value.every(market) : !!market(value)
  if (path.startsWith('/markets/')) return !!market(value)
  if (path === '/uploads')
    return Array.isArray(value) ? value.every(upload) : !!upload(value)
  if (path.startsWith('/uploads/')) return !!upload(value)
  if (path === '/status') return typeof value?.connected === 'boolean'
  if (path === '/connect' || path === '/reconnect' || path === '/info')
    return (
      value?.connected === true &&
      date(value.expires_at) &&
      value.info?.protocol === 'powerx.plugin-publisher-local/v1' &&
      typeof value.info.plugin?.name === 'string' &&
      typeof value.info.plugin?.plugin_id === 'string' &&
      typeof value.info.plugin?.source_version === 'string' &&
      Array.isArray(value.info.features) &&
      value.info.features.every((feature: unknown) => typeof feature === 'string') &&
      ['native_available', 'container_available', 'container_ready', 'can_start'].every(key => typeof value.info.build_environment?.[key] === 'boolean') &&
      Array.isArray(value.info.build_targets)
    )
  if (path.includes('?')) {
    const item = path.startsWith('/plans') ? plan : build
    return (
      value &&
      Array.isArray(value.items) &&
      value.items.every(item) &&
      Number.isSafeInteger(value.total) &&
      value.total >= value.items.length &&
      Number.isSafeInteger(value.page) &&
      value.page > 0 &&
      Number.isSafeInteger(value.size) &&
      value.size > 0 &&
      value.size <= 100 &&
      value.items.length <= value.size
    )
  }
  if (path.startsWith('/plans')) return !!plan(value)
  if (path.endsWith('/logs')) return typeof value?.text === 'string'
  if (path.endsWith('/verify')) return !!artifact(value)
  return !!build(value)
}

export class PublisherClientError extends Error {
  constructor(
    public code: string,
    public requestID = ''
  ) {
    super(code)
  }
}
export function createPublisherClient(
  origin: string,
  transport: typeof fetch = fetch
) {
  async function request<T>(
    path: string,
    method = 'GET',
    body?: unknown
  ): Promise<T> {
    try {
      const response = await transport(publisherAPIPath + path, {
        method,
        credentials: 'omit',
        cache: 'no-store',
        redirect: 'error',
        headers: {
          'X-PowerX-Publisher-Intent': publisherIntent,
          'X-PowerX-Publisher-Origin': origin,
          ...(body === undefined ? {} : { 'Content-Type': 'application/json' })
        },
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: AbortSignal.timeout(path === '/connect' || path === '/reconnect' ? 210000 : 20000)
      })
      const value = await response.json()
      if (!response.ok) {
        if (!/^PUBLISHER_[A-Z_]+$/.test(value?.error?.code || ''))
          throw new PublisherClientError('PUBLISHER_PROXY_RESPONSE_INVALID')
        const requestID = value.error.params?.request_id
        if (
          requestID !== undefined &&
          (typeof requestID !== 'string' || !uuid.test(requestID))
        )
          throw new PublisherClientError('PUBLISHER_PROXY_RESPONSE_INVALID')
        throw new PublisherClientError(value.error.code, requestID || '')
      }
      if (
        !value ||
        Object.keys(value).length !== 1 ||
        !Object.hasOwn(value, 'data')
      )
        throw new PublisherClientError('PUBLISHER_PROXY_RESPONSE_INVALID')
      if (!validPayload(path, value.data))
        throw new PublisherClientError('PUBLISHER_PROXY_RESPONSE_INVALID')
      return value.data as T
    } catch (error) {
      if (error instanceof PublisherClientError) throw error
      throw new PublisherClientError('PUBLISHER_PROXY_UNAVAILABLE')
    }
  }
  return {
    requestTraces: (kind: 'uploads' | 'submissions', id: string, page = 1) =>
      request<PublisherPage<PublisherRequestTrace>>(
        `/${kind}/${encodeURIComponent(id)}/requests?page=${page}&size=20`
      ),
    submissions: () => request<PublisherSubmission[]>('/submissions'),
    submission: (id: string) =>
      request<PublisherSubmission>(`/submissions/${encodeURIComponent(id)}`),
    prepareSubmission: (connection: string, version: string) =>
      request<PublisherSubmissionPreparation>('/submissions/prepare', 'POST', {
        connection_id: connection,
        version_id: version
      }),
    previewSubmission: (input: PublisherSubmissionInput) =>
      request<PublisherSubmissionPreview>(
        '/submissions/preview',
        'POST',
        input
      ),
    startSubmission: (input: PublisherSubmissionInput) =>
      request<PublisherSubmission>('/submissions', 'POST', input),
    resumeSubmission: (id: string) =>
      request<PublisherSubmission>(
        `/submissions/${encodeURIComponent(id)}/resume`,
        'POST'
      ),
    cancelSubmission: (id: string) =>
      request<PublisherSubmission>(
        `/submissions/${encodeURIComponent(id)}/cancel`,
        'POST'
      ),
    publications: (connection: string, page = 1) =>
      request<PublisherPage<PublisherPublication>>(
        `/markets/${encodeURIComponent(connection)}/publications?page=${page}&size=20`
      ),
    markets: () => request<PublisherMarket[]>('/markets'),
    addMarket: (
      input: Pick<PublisherMarket, 'name' | 'api_root' | 'console_base'>
    ) => request<PublisherMarket>('/markets', 'POST', input),
    bindMarket: (id: string, token: string) =>
      request<PublisherMarket>(`/markets/${encodeURIComponent(id)}/bind`, 'POST', { token }),
    verifyMarket: (id: string) =>
      request<PublisherMarket>(
        `/markets/${encodeURIComponent(id)}/verify`,
        'POST'
      ),
    disconnectMarket: (id: string) =>
      request<PublisherMarket>(
        `/markets/${encodeURIComponent(id)}/disconnect`,
        'POST'
      ),
    marketVersions: (id: string, page = 1) =>
      request<PublisherPage<PublisherMarketVersion>>(
        `/markets/${encodeURIComponent(id)}/versions?page=${page}&size=20`
      ),
    uploads: () => request<PublisherUpload[]>('/uploads'),
    upload: (id: string) =>
      request<PublisherUpload>(`/uploads/${encodeURIComponent(id)}`),
    startUpload: (connection: string, build: string) =>
      request<PublisherUpload>('/uploads', 'POST', {
        connection_id: connection,
        build_id: build
      }),
    resumeUpload: (id: string) =>
      request<PublisherUpload>(
        `/uploads/${encodeURIComponent(id)}/resume`,
        'POST'
      ),
    cancelUpload: (id: string) =>
      request<PublisherUpload>(
        `/uploads/${encodeURIComponent(id)}/cancel`,
        'POST'
      ),
    status: () => request<{ connected: boolean }>('/status'),
    connect: () => request<PublisherConnection>('/connect', 'POST'),
    reconnect: () => request<PublisherConnection>('/reconnect', 'POST'),
    info: () => request<PublisherConnection>('/info'),
    startEnvironment: () => request<PublisherBuildEnvironment>('/build-environment/start', 'POST'),
    plans: (page = 1) =>
      request<PublisherPage<PublisherPlan>>(`/plans?page=${page}&size=20`),
    plan: (id: string) =>
      request<PublisherPlan>(`/plans/${encodeURIComponent(id)}`),
    create: (input: PublisherPlanInput) =>
      request<PublisherPlan>('/plans', 'POST', input),
    builds: (page = 1) =>
      request<PublisherPage<PublisherBuild>>(`/builds?page=${page}&size=20`),
    build: (id: string) =>
      request<PublisherBuild>(`/builds/${encodeURIComponent(id)}`),
    run: (id: string, executor: 'native' | 'container' = 'native') =>
      request<PublisherBuild>('/builds', 'POST', { release_plan_id: id, executor }),
    cancel: (id: string) =>
      request<PublisherBuild>(
        `/builds/${encodeURIComponent(id)}/cancel`,
        'POST'
      ),
    verify: (id: string) =>
      request<PublisherArtifact>(
        `/builds/${encodeURIComponent(id)}/verify`,
        'POST'
      ),
    logs: (id: string) =>
      request<{ text: string }>(`/builds/${encodeURIComponent(id)}/logs`)
  }
}
export type PublisherClient = ReturnType<typeof createPublisherClient>
