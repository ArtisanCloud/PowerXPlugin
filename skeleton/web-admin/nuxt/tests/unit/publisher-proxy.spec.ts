import { afterEach, describe, expect, it } from 'vitest'
import { createServer, type Server } from 'node:http'
import { mkdtemp, writeFile, chmod, rm, symlink } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { createPublisherProxy, PublisherProxyError, type ProxyRequest } from '../../../../../framework/frontend/nuxt/framework-admin/publisher/proxy'
import { publisherIntent } from '../../../../../framework/frontend/nuxt/framework-admin/publisher/types'
import { createApp, toNodeListener } from 'h3'
import { createPublisherDevHandler } from '../../../../../framework/frontend/nuxt/framework-admin/publisher/runtime/handler'
import { createPublisherManager } from '../../../../../framework/frontend/nuxt/framework-admin/publisher/manager'

const cleanup: (() => Promise<unknown>)[] = []
afterEach(async () => { for (const close of cleanup.splice(0).reverse()) await close() })
const origin = 'http://127.0.0.1:3131'
const token = 's'.repeat(43), bootstrap = 'b'.repeat(43)
const info = { protocol: 'powerx.plugin-publisher-local/v1', plugin: { plugin_id: 'com.example.publisher', name: 'fixture', source_version: '0.1.0' }, features: ['release-plans', 'local-builds'], build_targets: [], build_environment: {native_available:false,container_available:false,container_ready:false,can_start:false} }
function request(path = '/connect', method = 'POST', extra: Record<string, string> = {}): ProxyRequest {
  return { path, method, remoteAddress: '127.0.0.1', headers: new Headers({ host: '127.0.0.1:3131', origin, 'x-powerx-publisher-origin': origin, 'x-powerx-publisher-intent': publisherIntent, ...extra }) }
}
async function fixture(handler?: (path: string) => { status?: number; body?: unknown; raw?: string; headers?: Record<string, string> }) {
  const seen: { path: string; method: string; authorization?: string; cookie?: string; origin?: string }[] = []
  const server = createServer((req, res) => {
    seen.push({ path: req.url!, method: req.method!, authorization: req.headers.authorization, cookie: req.headers.cookie, origin: req.headers.origin })
    const data = handler?.(req.url!)
    res.writeHead(data?.status || 200, { 'Content-Type': 'application/json', ...data?.headers })
    res.end(data?.raw ?? JSON.stringify(data?.body || { data: req.url === '/v1/session' ? { session_token: token, expires_at: new Date(Date.now() + 3600000).toISOString() } : info }))
  })
  await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve))
  cleanup.push(async () => { server.closeAllConnections(); await new Promise<void>(resolve => server.close(() => resolve())) })
  const endpoint = `http://127.0.0.1:${(server.address() as { port: number }).port}`
  const dir = await mkdtemp(join(tmpdir(), 'publisher-proxy-unit-')); await chmod(dir, 0o700)
  cleanup.push(() => rm(dir, { recursive: true, force: true }))
  const file = join(dir, 'tool-session.json')
  const record = { protocol: info.protocol, instance_id: '00000000-0000-4000-8000-000000000001', endpoint, session_file: '.px-plugin/publisher/tool-session.json', bootstrap_expires_at: new Date(Date.now() + 300000).toISOString(), pid: process.pid, bootstrap_token: bootstrap }
  await writeFile(file, JSON.stringify(record), { mode: 0o600 })
  return { seen, endpoint, dir, file, record, proxy: createPublisherProxy({ origin, sessionFile: file }) }
}

describe('local publisher development proxy', () => {
  it('keeps bootstrap/session secrets server-side and exchanges once under concurrency', async () => {
    const f = await fixture()
    const results = await Promise.all([f.proxy(request()), f.proxy(request())])
    expect(f.seen.filter(r => r.path === '/v1/session')).toHaveLength(1)
    expect(f.seen[0].authorization).toBe(`Bearer ${bootstrap}`)
    expect(f.seen[1].authorization).toBe(`Bearer ${token}`)
    expect(JSON.stringify(results)).not.toContain(token)
    expect(JSON.stringify(results)).not.toContain(bootstrap)
    expect(JSON.stringify(results)).not.toContain(f.file)
    expect(results[0]).toMatchObject({ status: 200, body: { data: { connected: true, info } } })
  })
  it.each([
    ['foreign_host', { host: 'evil.example' }],
    ['foreign_origin', { origin: 'https://evil.example' }],
    ['missing_intent', { 'x-powerx-publisher-intent': '' }],
    ['forged_page_origin', { 'x-powerx-publisher-origin': 'https://evil.example' }],
    ['cross_site', { 'sec-fetch-site': 'cross-site' }],
    ['business_bearer', { authorization: 'Bearer business' }]
  ])('rejects %s before connecting', async (_name, headers) => {
    const f = await fixture()
    await expect(f.proxy(request('/connect', 'POST', headers))).rejects.toMatchObject({ code: 'PUBLISHER_FORBIDDEN', status: 403 })
    expect(f.seen).toHaveLength(0)
  })
  it('rejects a LAN peer even with forged loopback headers', async () => {
    const f = await fixture(); const r = request(); r.remoteAddress = '192.168.1.2'
    await expect(f.proxy(r)).rejects.toMatchObject({ code: 'PUBLISHER_FORBIDDEN' })
  })
  it('does not forward browser cookies or accept arbitrary upstream paths', async () => {
    const f = await fixture(); await f.proxy(request('/connect', 'POST', { cookie: 'business=fixture' }))
    for (const path of ['/session', '/../../internal', '/plans?target=http://evil.example', '/plans?size=20&size=21', '/builds/not-a-uuid']) {
      await expect(f.proxy(request(path, 'GET'))).rejects.toBeInstanceOf(PublisherProxyError)
    }
    expect(f.seen.every(r => !r.cookie && r.origin === origin)).toBe(true)
    expect(f.seen).toHaveLength(2)
  })
  it('rejects oversized bodies and unsupported methods without replaying writes', async () => {
    const f = await fixture(); await f.proxy(request())
    const r = request('/plans', 'POST'); r.body = new Uint8Array(65537)
    await expect(f.proxy(r)).rejects.toMatchObject({ code: 'PUBLISHER_REQUEST_TOO_LARGE', status: 413 })
    await expect(f.proxy(request('/plans', 'OPTIONS'))).rejects.toMatchObject({ code: 'PUBLISHER_METHOD_NOT_ALLOWED' })
    expect(f.seen).toHaveLength(2)
  })
  it('rejects public, linked and expired bootstrap files', async () => {
    const f = await fixture(); await chmod(f.file, 0o644)
    await expect(f.proxy(request())).rejects.toMatchObject({ code: 'PUBLISHER_INVALID_RECORD' })
    await chmod(f.file, 0o600)
    await writeFile(f.file, JSON.stringify({ ...f.record, bootstrap_expires_at: '2000-01-01T00:00:00Z' }))
    await expect(f.proxy(request())).rejects.toMatchObject({ code: 'PUBLISHER_UNAUTHORIZED' })
    const linked = join(f.dir, 'linked.json'); await symlink(f.file, linked)
    await expect(createPublisherProxy({ origin, sessionFile: linked })(request())).rejects.toBeInstanceOf(PublisherProxyError)
    expect(f.seen).toHaveLength(0)
  })
  it('rejects non-loopback endpoints and never follows a credential redirect', async () => {
    const f = await fixture(() => ({ status: 302, headers: { Location: 'https://evil.example' } }))
    await expect(f.proxy(request())).rejects.toMatchObject({ code: 'PUBLISHER_PROXY_UNAVAILABLE' })
    expect(f.seen).toHaveLength(1)
    await writeFile(f.file, JSON.stringify({ ...f.record, endpoint: 'http://example.com:8077' }))
    await expect(f.proxy(request())).rejects.toMatchObject({ code: 'PUBLISHER_INVALID_RECORD' })
    expect(f.seen).toHaveLength(1)
  })
  it('requires explicit reconnect after expiry/revocation and does not replay', async () => {
    let reject = false
    const f = await fixture(path => reject && path !== '/v1/session' ? { status: 401, body: { error: { code: 'PUBLISHER_UNAUTHORIZED' } } } : {})
    await f.proxy(request()); reject = true
    await expect(f.proxy(request('/plans', 'GET'))).rejects.toMatchObject({ code: 'PUBLISHER_UNAUTHORIZED' })
    await expect(f.proxy(request('/plans', 'GET'))).rejects.toMatchObject({ code: 'PUBLISHER_UNAUTHORIZED' })
    expect(f.seen).toHaveLength(3)
  })
  it('market_credential_rejection_preserves_local_executor_session', async () => {
    const f = await fixture(path => path.endsWith('/verify') ? { status: 401, body: { error: { code: 'PUBLISHER_MARKET_CREDENTIAL' } } } : {})
    await f.proxy(request())
    await expect(f.proxy(request('/markets/00000000-0000-4000-8000-000000000001/verify', 'POST'))).rejects.toMatchObject({ code: 'PUBLISHER_MARKET_CREDENTIAL' })
    expect(await f.proxy(request('/info', 'GET'))).toMatchObject({ status: 200 })
    expect(f.seen.filter(item => item.path === '/v1/session')).toHaveLength(1)
  })
  it('requires private server configuration and ignores forward headers', async () => {
    await expect(createPublisherProxy({ origin: '', sessionFile: '' })(request())).rejects.toMatchObject({ code: 'PUBLISHER_PROXY_NOT_CONFIGURED' })
    const f = await fixture(); const r = request('/connect', 'POST', { host: 'evil.example', 'x-forwarded-host': '127.0.0.1:3131' })
    await expect(f.proxy(r)).rejects.toMatchObject({ code: 'PUBLISHER_FORBIDDEN' })
  })
  it('rejects BOM-prefixed private records and upstream JSON instead of stripping bytes', async () => {
    const f = await fixture(path => path === '/v1/info' ? { raw: '\ufeff' + JSON.stringify({ data: info }) } : {})
    await expect(f.proxy(request())).rejects.toMatchObject({ code: 'PUBLISHER_PROXY_RESPONSE_INVALID', status: 502 })
    const other = await fixture(); await writeFile(other.file, '\ufeff' + JSON.stringify(other.record))
    await expect(other.proxy(request())).rejects.toMatchObject({ code: 'PUBLISHER_INVALID_RECORD', status: 500 })
    expect(other.seen).toHaveLength(0)
  })
  it('checks the real peer on the outer development handler and bounds the HTTP body', async () => {
    const f = await fixture()
    const app = createApp()
    const server = createServer(toNodeListener(app))
    await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve))
    cleanup.push(async () => { server.closeAllConnections(); await new Promise<void>(resolve => server.close(() => resolve())) })
    const localOrigin = `http://127.0.0.1:${(server.address() as { port: number }).port}`
    const manager = createPublisherManager({ rootDir: f.dir })
    cleanup.push(() => manager.close())
    app.use('/__powerx/publisher-api', createPublisherDevHandler(manager))
    const base = localOrigin + '/__powerx/publisher-api'
    const headers = { ...Object.fromEntries(request().headers), host: new URL(localOrigin).host, origin: localOrigin, 'x-powerx-publisher-origin': localOrigin }
    const status = await fetch(base + '/status', { headers })
    expect(status.status).toBe(200); expect(await status.json()).toEqual({ data: { connected: false } })
    const rejected = await fetch(base + '/plans', { method: 'POST', headers, body: 'a'.repeat(65537) })
    expect(rejected.status).toBe(413); expect(await rejected.json()).toMatchObject({ error: { code: 'PUBLISHER_REQUEST_TOO_LARGE' } })
    expect(f.seen).toHaveLength(0)
  })
})

it('forwards only the validated request UUID in fixed error diagnostics', async () => {
 const id='00000000-0000-4000-8000-000000000007'
 const f=await fixture(path=>path==='/v1/submissions/prepare'?{status:403,body:{error:{code:'PUBLISHER_MARKET_REJECTED',params:{request_id:id,prose:'fixture_private'}}}}:{})
 await f.proxy(request())
 try {await f.proxy(request('/submissions/prepare','POST'));throw new Error('expected_failure')}catch(error){
  expect(error).toBeInstanceOf(PublisherProxyError)
  expect((error as PublisherProxyError).body()).toEqual({error:{code:'PUBLISHER_MARKET_REJECTED',message_key:'PUBLISHER_MARKET_REJECTED',params:{request_id:id}}})
 }
})
it('allows task request pagination and rejects write or arbitrary query access', async()=>{
 const f=await fixture();await f.proxy(request())
 const path='/submissions/00000000-0000-4000-8000-000000000007/requests'
 await f.proxy(request(path+'?page=2&size=20','GET'))
 expect(f.seen.at(-1)?.path).toBe('/v1'+path+'?page=2&size=20')
 await expect(f.proxy(request(path,'POST'))).rejects.toMatchObject({code:'PUBLISHER_NOT_FOUND'})
 await expect(f.proxy(request(path+'?token=fixture','GET'))).rejects.toMatchObject({code:'PUBLISHER_INVALID_INPUT'})
})
