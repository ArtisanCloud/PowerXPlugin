export const publisherPagePath = '/powerx/publisher'
export const publisherAPIPath = '/__powerx/publisher-api'
export const publisherIntent = 'local-publisher'
export const directBuildFeature = 'direct-build-apple-silicon'
export const activeBuildStages = [
  'queued',
  'prechecking',
  'building',
  'packing',
  'verifying'
]
export const activeUploadStages = ['queued', 'uploading', 'reconciling', 'processing']
export const activeSubmissionStages = [...activeUploadStages, 'sending', 'submitted']
export interface PublisherErrorBody {
  code: string
  message_key: string
  params?: Record<string, string>
}
export interface PublisherPlan {
  schema: string
  release_plan_id: string
  plugin_id: string
  version: string
  channel: 'stable'
  target: { os: 'linux'; arch: 'amd64' }
  release_notes: string
  created_at: string
}
export type PublisherPlanInput = Pick<
  PublisherPlan,
  'version' | 'channel' | 'target' | 'release_notes'
>
export interface PublisherArtifact {
  path: string
  sha256: string
  size: number
  file_count: number
}
export interface PublisherBuild {
  schema: string
  build_id: string
  release_plan_id: string
  plugin_id: string
  version: string
  target: { os: string; arch: string }
  stage: string
  executor?: 'native' | 'container'
  created_at: string
  updated_at: string
  artifact?: PublisherArtifact
  error?: PublisherErrorBody
  source?: {
    vcs: string
    commit?: string
    dirty: boolean | null
    sha256: string
  }
}
export interface PublisherInfo {
  protocol: string
  plugin: { plugin_id: string; name: string; source_version: string }
  features: string[]
  build_targets: { os: string; arch: string }[]
  build_environment: PublisherBuildEnvironment
}
export interface PublisherBuildEnvironment {
  native_available: boolean
  container_available: boolean
  container_ready: boolean
  can_start: boolean
}
export interface PublisherPage<T> {
  items: T[]
  total: number
  page: number
  size: number
}
export interface PublisherConnection {
  connected: true
  expires_at: string
  info: PublisherInfo
}
export interface PublisherMarket {
  connection_id: string
  name: string
  api_root: string
  console_base: string
  features: string[]
  verified_at?: string
  identity?: {
    user: { user_id: string; name: string }
    developer: { developer_id: string; name: string }
    credential: {
      token_id: string
      plugin_ids: string[]
      scopes: string[]
      expires_at: string
    }
  }
}
export interface PublisherMarketVersion {
  version_id: string
  plugin_id: string
  version: string
  channel: string
  availability_status: string
  market_version_url: string
  artifacts: {
    artifact_id: string
    os: string
    arch: string
    sha256: string
    size: number
  }[]
}
export interface PublisherUpload {
  schema: string
  upload_id: string
  operation_id: string
  connection_id: string
  build_id: string
  release_plan_id: string
  plugin_id: string
  version: string
  stage: string
  attempts: number
  bytes_sent: number
  artifact: PublisherArtifact
  created_at: string
  updated_at: string
  error?: PublisherErrorBody
  receipt?: { version_id: string; market_version_url: string }
  current_version?: PublisherMarketVersion
}
export interface PublisherMetadata {
  description: string
  license: string
  min_core: string
  support_url: string
  icon_url: string
  security_statement: string
  privacy_statement: string
}
export interface PublisherSubmissionInput {
  connection_id: string
  version_id: string
  version: string
  metadata: PublisherMetadata
  expected_artifacts: PublisherMarketVersion['artifacts']
}
export interface PublisherSubmissionPreview {
  input: PublisherSubmissionInput
  request_sha256: string
}
export interface PublisherSubmissionPreparation {
  version: PublisherMarketVersion
  options: {
    versions: string[]
    licenses: string[]
    source: string
    version_source: string
  }
}
export interface PublisherPublication {
  publication_id: string
  plugin_id: string
  version_id: string
  version: string
  status: 'submitted' | 'rejected' | 'approved' | 'published'
  metadata: PublisherMetadata
  artifacts: PublisherMarketVersion['artifacts']
  submitted_at: string
  reviewed_at: string | null
  review_reason: string
  published_at: string | null
  publish_reason: string
  market_publication_url: string
}
export interface PublisherSubmission extends PublisherSubmissionInput {
  schema: string
  submission_id: string
  operation_id: string
  plugin_id: string
  request_sha256: string
  stage: string
  attempts: number
  created_at: string
  updated_at: string
  receipt?: {
    publication_id: string
    version_id: string
    market_publication_url: string
  }
  current_publication?: PublisherPublication
  error?: PublisherErrorBody
}

export interface PublisherRequestTrace {
  schema: 'powerx.plugin-publisher-request/v1'
  task_kind: 'uploads' | 'submissions'
  task_id: string
  operation_id: string
  client_request_id: string
  server_request_id?: string
  method: 'GET' | 'POST'
  response_type: string
  http_status: number
  started_at: string
  finished_at?: string
}
