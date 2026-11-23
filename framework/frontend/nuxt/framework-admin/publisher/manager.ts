import { spawn, type ChildProcess } from 'node:child_process'
import { lstat, mkdtemp, readFile, realpath, rm, writeFile } from 'node:fs/promises'
import { dirname, join, resolve } from 'node:path'
import { tmpdir } from 'node:os'
import { createPublisherProxy, PublisherProxyError, validatePublisherBoundary, type ProxyRequest } from './proxy'
import { activeBuildStages, activeSubmissionStages, activeUploadStages, directBuildFeature, type PublisherConnection } from './types'

export interface PublisherManagerOptions {
  rootDir: string
  workspace?: string
  executable?: string
}

// The workspace and executable come exclusively from trusted Nuxt configuration.
// Browser requests never supply a path, command, environment or CLI arguments.
export async function publisherWorkspace(rootDir: string): Promise<string> {
  let directory = await realpath(rootDir)
  for (let depth = 0; depth < 8; depth++) {
    const manifest = await lstat(join(directory, 'plugin.yaml')).catch(() => null)
    if (manifest?.isFile() && !manifest.isSymbolicLink()) return directory
    const parent = dirname(directory)
    if (parent === directory) break
    directory = parent
  }
  throw new PublisherProxyError('PUBLISHER_INVALID_WORKSPACE', 503)
}

export function managedPublisherOrigin(request: ProxyRequest): string {
  const host = request.headers.get('host') || ''
  let url: URL
  try { url = new URL(`${request.secure ? 'https' : 'http'}://${host}`) } catch { throw new PublisherProxyError('PUBLISHER_FORBIDDEN', 403) }
  if (!['localhost', '127.0.0.1', '[::1]'].includes(url.hostname) ||
      url.host !== host || !request.localPort || Number(url.port || (request.secure ? 443 : 80)) !== request.localPort)
    throw new PublisherProxyError('PUBLISHER_FORBIDDEN', 403)
  const origin = url.origin
  validatePublisherBoundary({ origin, sessionFile: '/managed/tool-session.json' }, request)
  return origin
}

const developmentEnvironment = () => Object.fromEntries(
  Object.entries(process.env).filter(([key]) => ['PATH', 'HOME', 'TMPDIR', 'LANG', 'LC_ALL', 'GOCACHE', 'GOMODCACHE', 'GOPATH', 'DOCKER_HOST', 'DOCKER_CONTEXT', 'DOCKER_CONFIG', 'HTTP_PROXY', 'HTTPS_PROXY', 'NO_PROXY', 'http_proxy', 'https_proxy', 'no_proxy'].includes(key))
)

type ProxyResponse = Awaited<ReturnType<ReturnType<typeof createPublisherProxy>>>
const hasDirectBuild = (response: ProxyResponse) =>
  (response.body as { data: PublisherConnection }).data.info.features.includes(directBuildFeature)

function containsActiveTask(items: unknown, stages: string[]) {
  if (!Array.isArray(items) || items.some(item => typeof item?.stage !== 'string'))
    throw new PublisherProxyError('PUBLISHER_PROXY_RESPONSE_INVALID', 502)
  return items.some(item => stages.includes(item.stage))
}

export function createPublisherManager(options: PublisherManagerOptions) {
  const registryKey = Symbol.for('powerx.publisher.managed-instances.v1')
  const state = globalThis as unknown as Record<symbol, Map<string, { close(): Promise<void> }>>
  const registry = state[registryKey] ??= new Map()
  const key = resolve(options.rootDir, options.workspace || '.')
  const previousClosed = registry.get(key)?.close()
  let child: ChildProcess | undefined
  let directory = ''
  let origin = ''
  let closed = false
  let preparing: Promise<void> | undefined
  let proxy: ReturnType<typeof createPublisherProxy> | undefined
  let compiler: ChildProcess | undefined
  let closing: Promise<void> | undefined
  let sessionFile = ''
  let renewing: Promise<void> | undefined
  let reconnecting: Promise<ProxyResponse> | undefined
  let writes = 0
  function failure(phase: string, detail: string) {
    const diagnostic = detail.slice(0, 4096).split('\n').map(line =>
      /token|password|authorization|cookie|secret|api[_-]?key/i.test(line)
        ? '[PUBLISHER_DIAGNOSTIC_REDACTED]'
        : line.replace(/https?:\/\/[^\s"']+/g, '[url]').replaceAll(options.rootDir, '[workspace]').replaceAll(directory || '\0', '[runtime]')
    ).join('\n')
    if (!closed) console.warn('PUBLISHER_EXECUTOR_PREPARATION_FAILED', { phase, diagnostic })
    return new PublisherProxyError('PUBLISHER_EXECUTOR_PREPARATION_FAILED', 503)
  }

  async function stop(process: ChildProcess | undefined) {
    if (!process || process.exitCode !== null || process.signalCode !== null) return
    const exited = new Promise<void>(done => process!.once('exit', () => done()))
    try { globalThis.process.kill(-process.pid!, 'SIGTERM') } catch { return }
    let timer: ReturnType<typeof setTimeout> | undefined
    await Promise.race([exited, new Promise<void>(done => { timer = setTimeout(done, 20000) })])
    clearTimeout(timer)
    if (process.exitCode === null && process.signalCode === null) {
      try { globalThis.process.kill(-process.pid!, 'SIGKILL') } catch {}
      await exited
    }
  }

  async function executable(workspaceRoot: string) {
    if (options.executable) return options.executable
    // Resolve the trusted source checkout from its plugin workspace, not from
    // import.meta.url (which dev loaders may rewrite or preserve as a symlink).
    let repository = workspaceRoot
    let source = false
    for (let depth = 0; depth < 8; depth++) {
      const cli = await readFile(join(repository, 'tools/cli/go.mod'), 'utf8').catch(() => '')
      const sdk = await readFile(join(repository, 'framework/backend/go/go.mod'), 'utf8').catch(() => '')
      if (cli.startsWith('module github.com/powerx-plugin/cli\n') && sdk.startsWith('module github.com/ArtisanCloud/PowerXPlugin/framework/backend/go\n')) { source = true; break }
      const parent = dirname(repository)
      if (parent === repository) break
      repository = parent
    }
    if (!source) return 'px-plugin'
    if (directory) await rm(directory, { recursive: true, force: true })
    directory = await mkdtemp(join(tmpdir(), 'powerx-publisher-runtime-'))
    const binary = join(directory, 'px-plugin')
    const workspace = join(directory, 'go.work')
    await writeFile(workspace, `go 1.24.0\n\nuse (\n ${JSON.stringify(join(repository, 'framework/backend/go'))}\n ${JSON.stringify(join(repository, 'tools/cli'))}\n)\n`, { mode: 0o600 })
    await new Promise<void>((done, reject) => {
      let diagnostic = ''
      const process = compiler = spawn('go', ['build', '-o', binary, './cmd/px-plugin'], {
        cwd: join(repository, 'tools/cli'), detached: true,
        env: { ...developmentEnvironment(), GOWORK: workspace }, stdio: ['ignore', 'ignore', 'pipe']
      })
      process.stderr?.on('data', chunk => { diagnostic = (diagnostic + chunk.toString()).slice(0, 4096) })
      const timeout = setTimeout(() => { void stop(process); reject(new PublisherProxyError('PUBLISHER_EXECUTOR_PREPARATION_FAILED', 503)) }, 150000)
      process.once('error', () => { clearTimeout(timeout); reject(new PublisherProxyError('PUBLISHER_EXECUTOR_PREPARATION_FAILED', 503)) })
      process.once('exit', code => { clearTimeout(timeout); compiler = undefined; code === 0 ? done() : reject(failure('compiler', diagnostic || String(code))) })
    })
    return binary
  }

  async function start(workspace: string, nextOrigin: string) {
    if (closed) throw new PublisherProxyError('PUBLISHER_PROXY_UNAVAILABLE', 503)
    const binary = await executable(workspace)
    if (closed) throw new PublisherProxyError('PUBLISHER_PROXY_UNAVAILABLE', 503)
    const process = child = spawn(binary, ['publisher', 'serve', '--entry', workspace, '--origin', nextOrigin, '--parent-pid', String(globalThis.process.pid)], {
      cwd: workspace, detached: true, env: developmentEnvironment(), stdio: ['ignore', 'pipe', 'pipe']
    })
    await new Promise<void>((done, reject) => {
      let output = ''
      let diagnostic = ''
      const timeout = setTimeout(() => reject(new PublisherProxyError('PUBLISHER_EXECUTOR_PREPARATION_FAILED', 503)), 15000)
      const fail = () => { clearTimeout(timeout); reject(failure('executor', diagnostic || 'STARTUP_FAILED')) }
      process.once('error', fail)
      process.once('exit', fail)
      process.stderr?.on('data', chunk => { diagnostic = (diagnostic + chunk.toString()).slice(0, 4096) })
      process.stdout?.on('data', chunk => {
        output += chunk.toString()
        if (output.length > 65536) { fail(); return }
        if (!output.includes('\n')) return
        try {
          const data = JSON.parse(output.split('\n')[0]!).data
          if (data.protocol !== 'powerx.plugin-publisher-local/v1' || data.pid !== process.pid) { fail(); return }
          clearTimeout(timeout); process.removeListener('exit', fail); done()
        } catch { fail() }
      })
    })
    origin = nextOrigin
    sessionFile = join(workspace, '.px-plugin/publisher/tool-session.json')
    proxy = createPublisherProxy({ origin, sessionFile })
    process.once('exit', () => { if (child === process) { child = undefined; proxy = undefined } })
  }

  async function connect(request: ProxyRequest, nextOrigin: string) {
    if (!preparing && !proxy) preparing = (async () => {
      const workspace = options.workspace ? await realpath(resolve(options.rootDir, options.workspace)) : await publisherWorkspace(options.rootDir)
      try { await start(workspace, nextOrigin) } catch (error) {
        await stop(child); child = undefined
        if (directory) { await rm(directory, { recursive: true, force: true }); directory = '' }
        throw error
      }
    })().finally(() => { preparing = undefined })
    if (preparing) await preparing
    if (!proxy) throw new PublisherProxyError('PUBLISHER_UNAUTHORIZED', 401)
    try { return await proxy(request) } catch (error) {
      if (!(error instanceof PublisherProxyError) || error.code !== 'PUBLISHER_UNAUTHORIZED') throw error
      if (!renewing) renewing = (async () => {
        const previous = await lstat(sessionFile)
        if (!child?.kill('SIGUSR1')) throw new PublisherProxyError('PUBLISHER_PROXY_UNAVAILABLE', 503)
        const deadline = Date.now() + 5000
        while (Date.now() < deadline && !closed) {
          await new Promise(resolve => setTimeout(resolve, 25))
          const current = await lstat(sessionFile)
          if (current.ino !== previous.ino || current.mtimeMs !== previous.mtimeMs) return
        }
        throw new PublisherProxyError('PUBLISHER_PROXY_UNAVAILABLE', 503)
      })().finally(() => { renewing = undefined })
      await renewing
      return proxy(request)
    }
  }

  async function reconnect(request: ProxyRequest, nextOrigin: string) {
    const connectRequest = { ...request, path: '/connect' }
    const wasConnected = !!proxy
    const current = await connect(connectRequest, nextOrigin)
    if (hasDirectBuild(current)) return current
    const previousProxy = proxy
    if (!wasConnected || !previousProxy)
      throw new PublisherProxyError('PUBLISHER_EXECUTOR_UPDATE_REQUIRED', 409)

    // The manager rejects new writes while reading the owned executor's durable queue.
    const read = async (path: string) => {
      const result = await previousProxy({ ...request, path, method: 'GET', body: undefined })
      return (result.body as { data: unknown }).data
    }
    const [uploads, submissions] = await Promise.all([read('/uploads'), read('/submissions')])
    if (containsActiveTask(uploads, activeUploadStages) || containsActiveTask(submissions, activeSubmissionStages))
      throw new PublisherProxyError('PUBLISHER_EXECUTOR_BUSY', 409)
    for (let page = 1; ; page++) {
      const builds = await read(`/builds?page=${page}&size=100`) as { items: unknown; total: number; page: number; size: number }
      if (!builds || builds.page !== page || builds.size !== 100 || !Number.isSafeInteger(builds.total) || builds.total < 0)
        throw new PublisherProxyError('PUBLISHER_PROXY_RESPONSE_INVALID', 502)
      if (containsActiveTask(builds.items, activeBuildStages))
        throw new PublisherProxyError('PUBLISHER_EXECUTOR_BUSY', 409)
      if (page * builds.size >= builds.total) break
    }

    await stop(child)
    child = undefined
    proxy = undefined
    const updated = await connect(connectRequest, nextOrigin)
    if (!hasDirectBuild(updated)) throw new PublisherProxyError('PUBLISHER_EXECUTOR_UPDATE_REQUIRED', 409)
    return updated
  }

  const manager = {
    async request(request: ProxyRequest) {
      const nextOrigin = managedPublisherOrigin(request)
      await previousClosed
      if (origin && origin !== nextOrigin) throw new PublisherProxyError('PUBLISHER_FORBIDDEN', 403)
      if (closed) throw new PublisherProxyError('PUBLISHER_PROXY_UNAVAILABLE', 503)
      if (request.path === '/reconnect') {
        if (request.method !== 'POST') throw new PublisherProxyError('PUBLISHER_METHOD_NOT_ALLOWED', 405)
        if (request.body?.length) throw new PublisherProxyError('PUBLISHER_INVALID_INPUT', 400)
        if (writes) throw new PublisherProxyError('PUBLISHER_EXECUTOR_BUSY', 409)
        return reconnecting ??= reconnect(request, nextOrigin).finally(() => { reconnecting = undefined })
      }
      if (reconnecting) throw new PublisherProxyError('PUBLISHER_EXECUTOR_BUSY', 409)
      if (!proxy && request.method === 'GET' && request.path === '/status')
        return { status: 200, body: { data: { connected: false } } }
      const write = request.method === 'POST'
      if (write) writes++
      try {
        if (write && request.path === '/connect' && !request.body?.length)
          return await connect(request, nextOrigin)
        if (!proxy) throw new PublisherProxyError('PUBLISHER_UNAUTHORIZED', 401)
        return await proxy(request)
      } finally {
        if (write) writes--
      }
    },
    close() {
      return closing ??= (async () => {
        closed = true
        await stop(compiler)
        await preparing?.catch(() => {})
        await reconnecting?.catch(() => {})
        await stop(child)
        if (directory) await rm(directory, { recursive: true, force: true })
        child = undefined; proxy = undefined
        if (registry.get(key) === manager) registry.delete(key)
      })()
    }
  }
  registry.set(key, manager)
  return manager
}
