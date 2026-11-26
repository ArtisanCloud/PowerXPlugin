import { constants } from 'node:fs'
import { open, lstat } from 'node:fs/promises'
import { dirname, isAbsolute } from 'node:path'
import { isIP } from 'node:net'
import {
  publisherIntent,
  type PublisherErrorBody,
  type PublisherInfo
} from './types'

const protocol = 'powerx.plugin-publisher-local/v1'
const uuid =
  '[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}'
const secret = /^[A-Za-z0-9_-]{43}$/
export const maxPublisherRequestBytes = 65536
const maxResponseBytes = 32 * 1024 * 1024 // Includes worst-case JSON escaping of the 4 MiB F03 log.
export interface PublisherProxyConfig {
  origin: string
  sessionFile: string
}
export interface ProxyRequest {
  path: string
  method: string
  headers: Headers
  remoteAddress: string
  localPort?: number
  secure?: boolean
  body?: Uint8Array
}
export class PublisherProxyError extends Error {
  constructor(
    public code: string,
    public status: number,
    public requestID = ''
  ) {
    super(code)
  }
  body(): { error: PublisherErrorBody } {
    return {
      error: {
        code: this.code,
        message_key: this.code,
        ...(this.requestID ? { params: { request_id: this.requestID } } : {})
      }
    }
  }
}
function fail(code: string, status = 400): never {
  throw new PublisherProxyError(code, status)
}
function loopback(address: string): boolean {
  const host = address.replace(/^\[|\]$/g, '')
  return (
    (isIP(host) === 4 && host.startsWith('127.')) ||
    host === '::1' ||
    /^::ffff:127\.\d+\.\d+\.\d+$/.test(host)
  )
}
export function validatePublisherBoundary(
  config: PublisherProxyConfig,
  request: ProxyRequest
): void {
  if (!config.origin || !config.sessionFile)
    fail('PUBLISHER_PROXY_NOT_CONFIGURED', 503)
  let origin: URL
  try {
    origin = new URL(config.origin)
  } catch {
    return fail('PUBLISHER_PROXY_NOT_CONFIGURED', 503)
  }
  if (
    !['http:', 'https:'].includes(origin.protocol) ||
    origin.origin !== config.origin ||
    origin.username ||
    origin.password ||
    (origin.hostname !== 'localhost' && !loopback(origin.hostname))
  )
    fail('PUBLISHER_PROXY_NOT_CONFIGURED', 503)
  if (
    !loopback(request.remoteAddress) ||
    request.headers.get('host') !== origin.host ||
    request.headers.get('x-powerx-publisher-intent') !== publisherIntent ||
    request.headers.get('x-powerx-publisher-origin') !== config.origin ||
    (request.headers.has('origin') &&
      request.headers.get('origin') !== config.origin) ||
    (request.headers.has('sec-fetch-site') &&
      request.headers.get('sec-fetch-site') !== 'same-origin') ||
    request.headers.has('authorization')
  )
    fail('PUBLISHER_FORBIDDEN', 403)
}

async function readBootstrap(file: string) {
  if (!isAbsolute(file)) fail('PUBLISHER_PROXY_NOT_CONFIGURED', 503)
  const parent = await lstat(dirname(file))
  if (
    !parent.isDirectory() ||
    parent.isSymbolicLink() ||
    parent.mode & 0o077 ||
    parent.uid !== process.getuid?.()
  )
    fail('PUBLISHER_INVALID_RECORD', 500)
  const handle = await open(file, constants.O_RDONLY | constants.O_NOFOLLOW)
  try {
    const stat = await handle.stat()
    if (
      !stat.isFile() ||
      stat.mode & 0o077 ||
      stat.uid !== process.getuid?.() ||
      stat.size > maxPublisherRequestBytes
    )
      fail('PUBLISHER_INVALID_RECORD', 500)
    const buffer = Buffer.alloc(maxPublisherRequestBytes + 1)
    let length = 0
    for (;;) {
      const { bytesRead } = await handle.read(
        buffer,
        length,
        buffer.length - length,
        length
      )
      length += bytesRead
      if (length > maxPublisherRequestBytes)
        fail('PUBLISHER_INVALID_RECORD', 500)
      if (!bytesRead) break
    }
    let data: any
    try {
      // Preserve a BOM so JSON.parse rejects it, matching Go's JSON decoder.
      const raw = new TextDecoder('utf-8', {
        fatal: true,
        ignoreBOM: true
      }).decode(buffer.subarray(0, length))
      data = JSON.parse(raw)
    } catch {
      fail('PUBLISHER_INVALID_RECORD', 500)
    }
    const keys = [
      'protocol',
      'instance_id',
      'endpoint',
      'session_file',
      'bootstrap_expires_at',
      'pid',
      'bootstrap_token'
    ]
    if (
      !data ||
      typeof data !== 'object' ||
      Object.keys(data).length !== keys.length ||
      keys.some((key) => !Object.hasOwn(data, key)) ||
      data.protocol !== protocol ||
      !new RegExp(`^${uuid}$`).test(data.instance_id) ||
      !secret.test(data.bootstrap_token) ||
      data.session_file !== '.px-plugin/publisher/tool-session.json' ||
      !Number.isInteger(data.pid) ||
      data.pid < 1 ||
      !Number.isFinite(Date.parse(data.bootstrap_expires_at))
    )
      fail('PUBLISHER_INVALID_RECORD', 500)
    const endpoint = new URL(data.endpoint)
    if (
      endpoint.protocol !== 'http:' ||
      endpoint.origin !== data.endpoint ||
      !loopback(endpoint.hostname) ||
      endpoint.username ||
      endpoint.password ||
      !endpoint.port
    )
      fail('PUBLISHER_INVALID_RECORD', 500)
    return {
      endpoint: endpoint.origin,
      bootstrap: data.bootstrap_token as string,
      expires: Date.parse(data.bootstrap_expires_at)
    }
  } finally {
    await handle.close()
  }
}

async function responseJSON(response: Response): Promise<any> {
  if (
    !response.headers
      .get('content-type')
      ?.toLowerCase()
      .startsWith('application/json')
  )
    fail('PUBLISHER_PROXY_RESPONSE_INVALID', 502)
  const reader = response.body?.getReader()
  if (!reader) fail('PUBLISHER_PROXY_RESPONSE_INVALID', 502)
  const chunks: Uint8Array[] = []
  let length = 0
  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      length += value.byteLength
      if (length > maxResponseBytes)
        fail('PUBLISHER_PROXY_RESPONSE_INVALID', 502)
      chunks.push(value)
    }
    const raw = Buffer.concat(chunks)
    try {
      return JSON.parse(
        new TextDecoder('utf-8', { fatal: true, ignoreBOM: true }).decode(raw)
      )
    } catch {
      fail('PUBLISHER_PROXY_RESPONSE_INVALID', 502)
    }
  } finally {
    await reader.cancel().catch(() => {})
  }
}

export function createPublisherProxy(config: PublisherProxyConfig) {
  let session: { endpoint: string; token: string; expires: string } | undefined
  let connecting: Promise<unknown> | undefined
  async function upstream(
    endpoint: string,
    token: string,
    path: string,
    method: string,
    body?: Uint8Array
  ) {
    const headers: Record<string, string> = {
      Origin: config.origin,
      Authorization: `Bearer ${token}`
    }
    if (body?.length) headers['Content-Type'] = 'application/json'
    const response = await fetch(endpoint + path, {
      method,
      headers,
      body: body?.length ? (body as BodyInit) : undefined,
      redirect: 'error',
      signal: AbortSignal.timeout(15000)
    })
    const data = await responseJSON(response)
    if (!response.ok) {
      const code = data?.error?.code
      if (typeof code !== 'string' || !/^PUBLISHER_[A-Z_]+$/.test(code))
        fail('PUBLISHER_PROXY_RESPONSE_INVALID', 502)
      if (response.status === 401 && code === 'PUBLISHER_UNAUTHORIZED') session = undefined
      const requestID = data?.error?.params?.request_id
      if (
        requestID !== undefined &&
        (typeof requestID !== 'string' ||
          !new RegExp(`^${uuid}$`).test(requestID))
      )
        fail('PUBLISHER_PROXY_RESPONSE_INVALID', 502)
      throw new PublisherProxyError(code, response.status, requestID || '')
    }
    if (!data || !Object.hasOwn(data, 'data') || Object.keys(data).length !== 1)
      fail('PUBLISHER_PROXY_RESPONSE_INVALID', 502)
    return { status: response.status, body: data }
  }
  async function info() {
    if (!session || Date.parse(session.expires) <= Date.now())
      fail('PUBLISHER_UNAUTHORIZED', 401)
    const result = await upstream(
      session.endpoint,
      session.token,
      '/v1/info',
      'GET'
    )
    const value = result.body.data as PublisherInfo
    if (
      value?.protocol !== protocol ||
      Object.keys(value).some(
        (k) => !['protocol', 'plugin', 'features', 'build_targets', 'build_environment'].includes(k)
      ) ||
      !value.plugin ||
      Object.keys(value.plugin).some(
        (k) => !['plugin_id', 'name', 'source_version'].includes(k)
      ) ||
      [
        value.plugin.plugin_id,
        value.plugin.name,
        value.plugin.source_version
      ].some((v) => typeof v !== 'string') ||
      !Array.isArray(value.features) ||
      !value.features.every((v) => typeof v === 'string') ||
      !Array.isArray(value.build_targets)
      || !['native_available', 'container_available', 'container_ready', 'can_start'].every(key => typeof value.build_environment?.[key as keyof typeof value.build_environment] === 'boolean')
    )
      fail('PUBLISHER_PROXY_RESPONSE_INVALID', 502)
    return {
      status: 200,
      body: {
        data: { connected: true, expires_at: session.expires, info: value }
      }
    }
  }
  async function connect() {
    if (session && Date.parse(session.expires) > Date.now()) return info()
    const bootstrap = await readBootstrap(config.sessionFile)
    if (bootstrap.expires <= Date.now()) fail('PUBLISHER_UNAUTHORIZED', 401)
    const result = await upstream(
      bootstrap.endpoint,
      bootstrap.bootstrap,
      '/v1/session',
      'POST'
    )
    const value = result.body.data
    if (
      !value ||
      !secret.test(value.session_token) ||
      !Number.isFinite(Date.parse(value.expires_at)) ||
      Date.parse(value.expires_at) <= Date.now()
    )
      fail('PUBLISHER_PROXY_RESPONSE_INVALID', 502)
    session = {
      endpoint: bootstrap.endpoint,
      token: value.session_token,
      expires: value.expires_at
    }
    return info()
  }
  return async (request: ProxyRequest) => {
    try {
      validatePublisherBoundary(config, request)
      if (!['GET', 'POST'].includes(request.method))
        fail('PUBLISHER_METHOD_NOT_ALLOWED', 405)
      if ((request.body?.length || 0) > maxPublisherRequestBytes)
        fail('PUBLISHER_REQUEST_TOO_LARGE', 413)
      const [path, query = ''] = request.path.split('?')
      if (!path) fail('PUBLISHER_NOT_FOUND', 404)
      if (
        path === '/connect' &&
        request.method === 'POST' &&
        !query &&
        !request.body?.length
      ) {
        if (!connecting)
          connecting = connect().finally(() => {
            connecting = undefined
          })
        return (await connecting) as { status: number; body: unknown }
      }
      if (path === '/info' && request.method === 'GET' && !query)
        return await info()
      if (path === '/status' && request.method === 'GET' && !query)
        return {
          status: 200,
          body: {
            data: {
              connected: !!session && Date.parse(session.expires) > Date.now()
            }
          }
        }
      const list = path === '/plans' || path === '/builds'
      const marketList =
        path === '/markets' || path === '/uploads' || path === '/submissions'
      const versionsList = new RegExp(
        `^/markets/${uuid}/(versions|publications)$`
      ).test(path)
      const requestList = new RegExp(
        `^/(uploads|submissions)/${uuid}/requests$`
      ).test(path)
      const read =
        requestList ||
        list ||
        marketList ||
        new RegExp(`^/(markets|uploads|submissions)/${uuid}$`).test(path) ||
        new RegExp(`^/markets/${uuid}/(versions|publications)$`).test(path) ||
        new RegExp(`^/(plans|builds)/${uuid}$`).test(path) ||
        new RegExp(`^/builds/${uuid}/logs$`).test(path)
      const write =
        /^\/submissions\/(prepare|preview)$/.test(path) ||
        new RegExp(`^/submissions/${uuid}/(resume|cancel)$`).test(path) ||
        list ||
        marketList ||
        new RegExp(`^/markets/${uuid}/(verify|disconnect|bind)$`).test(path) ||
        path === '/build-environment/start' ||
        new RegExp(`^/uploads/${uuid}/(resume|cancel)$`).test(path) ||
        new RegExp(`^/builds/${uuid}/(cancel|verify)$`).test(path)
      if (request.method === 'GET' ? !read : !write)
        fail('PUBLISHER_NOT_FOUND', 404)
      if (query) {
        const parsed = new URLSearchParams(query)
        if (
          (!list && !versionsList && !requestList) ||
          request.method !== 'GET'
        )
          fail('PUBLISHER_INVALID_INPUT')
        for (const [key, value] of parsed)
          if (
            !['page', 'size'].includes(key) ||
            parsed.getAll(key).length !== 1 ||
            !/^[1-9][0-9]{0,6}$/.test(value) ||
            Number(value) > (key === 'size' ? 100 : 1000000)
          )
            fail('PUBLISHER_INVALID_INPUT')
      }
      if (!session || Date.parse(session.expires) <= Date.now())
        fail('PUBLISHER_UNAUTHORIZED', 401)
      return await upstream(
        session.endpoint,
        session.token,
        '/v1' + request.path,
        request.method,
        request.body
      )
    } catch (error) {
      if (error instanceof PublisherProxyError) throw error
      // Never return an upstream exception, filesystem path, URL or credential.
      fail('PUBLISHER_PROXY_UNAVAILABLE', 503)
    }
  }
}
