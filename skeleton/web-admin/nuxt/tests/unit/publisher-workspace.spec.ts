import { afterEach, describe, expect, it, vi } from 'vitest'
import { createPublisherClient, PublisherClientError } from '../../../../../framework/frontend/nuxt/framework-admin/publisher/client'
import { createPublisherWorkspace, publisherPolling } from '../../../../../framework/frontend/nuxt/framework-admin/publisher/workspace'
import { directBuildFeature } from '../../../../../framework/frontend/nuxt/framework-admin/publisher/types'
import en from '../../../../../framework/frontend/nuxt/framework-admin/publisher/locales/en.json'
import zh from '../../../../../framework/frontend/nuxt/framework-admin/publisher/locales/zh.json'

const id = '00000000-0000-4000-8000-000000000001'
const plan = { schema: 'powerx.plugin-publisher-plan/v1', release_plan_id: id, plugin_id: 'com.example.publisher', version: '1.2.0', channel: 'stable', target: { os: 'linux', arch: 'amd64' }, release_notes: '', created_at: '2026-11-10T00:00:00Z' }
const build = (stage: string) => ({ schema: 'powerx.plugin-publisher-build/v1', build_id: id, release_plan_id: id, plugin_id: plan.plugin_id, version: plan.version, target: plan.target, stage, created_at: plan.created_at, updated_at: plan.created_at })
function fixture() {
  const client = {
    status: vi.fn(async () => ({ connected: true })),
    connect: vi.fn(async () => ({ connected: true, expires_at: '2026-12-01T00:00:00Z', info: { plugin: { plugin_id: plan.plugin_id }, features: [directBuildFeature], build_targets: [plan.target], build_environment: { native_available: true, container_available: true, container_ready: true, can_start: false } } })),
    reconnect: vi.fn(),
    info: vi.fn(), startEnvironment: vi.fn(async () => {}),
    plans: vi.fn(async () => ({ items: [plan], page: 1, size: 20, total: 1 })),
    plan: vi.fn(async () => plan), builds: vi.fn(async () => ({ items: [], page: 1, size: 20, total: 0 })),
    build: vi.fn(async () => build('ready')), run: vi.fn(async () => build('queued')),
    cancel: vi.fn(async () => build('cancelled')), verify: vi.fn(), create: vi.fn(async () => plan)
  }
  return { client, workspace: createPublisherWorkspace(client as any) }
}
afterEach(() => vi.useRealTimers())
describe('publisher page state and primary polling', () => {
  it('uses a separate same-origin client with no business credentials', async () => {
    const transport = vi.fn(async () => new Response(JSON.stringify({ data: plan }), { headers: { 'Content-Type': 'application/json' } }))
    await createPublisherClient('http://localhost:3131', transport as any).create(plan as any)
    expect(transport).toHaveBeenCalledTimes(1)
    expect(transport.mock.calls[0]).toMatchObject(['/__powerx/publisher-api/plans', { credentials: 'omit', redirect: 'error', method: 'POST', headers: { 'X-PowerX-Publisher-Origin': 'http://localhost:3131' } }])
    expect((transport.mock.calls[0] as any)[1].headers.Authorization).toBeUndefined()
  })
  it('rejects malformed or unknown-state responses and never retries an ambiguous write', async () => {
    for (const data of [{}, { ...build('ready'), artifact: undefined }, build('unknown_state')]) {
      const transport = vi.fn(async () => new Response(JSON.stringify({ data }), { headers: { 'Content-Type': 'application/json' } }))
      await expect(createPublisherClient('http://localhost:3131', transport as any).run(id)).rejects.toMatchObject({ code: 'PUBLISHER_PROXY_RESPONSE_INVALID' })
      expect(transport).toHaveBeenCalledTimes(1)
    }
    const transport = vi.fn(async () => { throw new TypeError() })
    await expect(createPublisherClient('http://localhost:3131', transport as any).run(id)).rejects.toMatchObject({ code: 'PUBLISHER_PROXY_UNAVAILABLE' })
    expect(transport).toHaveBeenCalledTimes(1)
  })
  it('restores the selected task and plan from server records on refresh', async () => {
    const { client, workspace: w } = fixture(); client.info.mockResolvedValue(await client.connect())
    client.build.mockResolvedValue(build('interrupted'))
    await w.restore(id, id)
    expect(w.selectedPlan.value?.release_plan_id).toBe(id)
    expect(w.selectedBuild.value?.stage).toBe('interrupted')
    expect(client.run).not.toHaveBeenCalled(); w.dispose()
  })
  it('polls only the same active build and stops at a terminal state', async () => {
    vi.useFakeTimers(); const { client, workspace: w } = fixture()
    await w.connect(); await w.run()
    await vi.advanceTimersByTimeAsync(publisherPolling.interval)
    expect(w.selectedBuild.value?.stage).toBe('ready')
    await vi.advanceTimersByTimeAsync(publisherPolling.interval * 5)
    expect(client.build).toHaveBeenCalledTimes(1); w.dispose()
  })
  it('retains a failed task and stops on transport errors without retrying writes', async () => {
    vi.useFakeTimers(); const { client, workspace: w } = fixture()
    client.build.mockRejectedValue(new PublisherClientError('PUBLISHER_PROXY_UNAVAILABLE'))
    await w.connect(); await w.run(); await vi.advanceTimersByTimeAsync(2000)
    expect(w.error.value).toBe('PUBLISHER_PROXY_UNAVAILABLE'); expect(w.pollPaused.value).toBe(true)
    await vi.advanceTimersByTimeAsync(20000)
    expect(client.run).toHaveBeenCalledTimes(1); expect(client.build).toHaveBeenCalledTimes(1); w.dispose()
  })
  it('does not let a late poll replace a cancellation result', async () => {
    vi.useFakeTimers(); const { client, workspace: w } = fixture(); let resolve: (value: any) => void = () => {}
    client.build.mockImplementation(() => new Promise(r => { resolve = r }))
    await w.connect(); await w.run(); await vi.advanceTimersByTimeAsync(2000)
    await w.cancel(); resolve(build('building')); await Promise.resolve(); await Promise.resolve()
    expect(w.selectedBuild.value?.stage).toBe('cancelled'); w.dispose()
  })
  it('bounds the polling window and permits explicit resumption of the same task', async () => {
    vi.useFakeTimers(); const { client, workspace: w } = fixture(); client.build.mockResolvedValue(build('building'))
    await w.connect(); await w.run(); vi.setSystemTime(Date.now() + publisherPolling.window)
    await vi.advanceTimersByTimeAsync(2000)
    expect(w.pollPaused.value).toBe(true); expect(client.build).not.toHaveBeenCalled()
    w.resume(); await vi.advanceTimersByTimeAsync(2000); expect(client.build).toHaveBeenCalledWith(id); w.dispose()
  })
  it('stops timers when leaving the page and blocks unsupported build platforms', async () => {
    vi.useFakeTimers(); const { client, workspace: w } = fixture(); await w.connect(); await w.run(); w.dispose()
    await vi.advanceTimersByTimeAsync(20000); expect(client.build).not.toHaveBeenCalled()
    const other = fixture(); const conn = await other.client.connect(); conn.info.build_targets = []
    other.client.connect.mockResolvedValue(conn); await other.workspace.connect(); await other.workspace.run()
    expect(other.workspace.error.value).toBe('PUBLISHER_BUILD_PLATFORM_REQUIRED'); expect(other.client.run).not.toHaveBeenCalled(); other.workspace.dispose()
  })
  it('keeps direct builds selected when a container is available but stopped', async () => {
    const { client, workspace } = fixture()
    const connection = await client.connect()
    const environment = connection.info.build_environment
    client.connect.mockResolvedValue({
      ...connection,
      info: {
        ...connection.info,
        build_environment: { ...environment, container_ready: false, can_start: true }
      }
    })
    await workspace.connect()
    client.info.mockResolvedValue(connection)
    await workspace.checkEnvironment()
    try {
      expect(await workspace.run()).toBe(true)
      expect(client.run).toHaveBeenCalledWith(id, 'native')
      expect(client.startEnvironment).not.toHaveBeenCalled()
    } finally {
      workspace.dispose()
    }
  })
  it('requires an explicit container selection when direct builds are unavailable', async () => {
    const { client, workspace } = fixture()
    const connection = await client.connect()
    connection.info.build_environment.native_available = false
    connection.info.build_environment.container_ready = false
    connection.info.build_environment.can_start = true
    client.connect.mockResolvedValue(connection)
    await workspace.connect()
    try {
      expect(await workspace.run()).toBe(false)
      expect(client.startEnvironment).not.toHaveBeenCalled()
      expect(client.run).not.toHaveBeenCalled()
    } finally {
      workspace.dispose()
    }
  })
  it('reconnects an outdated executor and rebuilds the selected failed plan directly', async () => {
    const { client, workspace } = fixture()
    const current = await client.connect()
    client.connect.mockResolvedValue({ ...current, info: { ...current.info, features: [] } })
    client.reconnect.mockResolvedValue(current)
    client.build.mockResolvedValue(build('failed'))
    await workspace.connect(id, id)
    try {
      expect(workspace.executorUpdateRequired.value).toBe(true)
      expect(await workspace.run()).toBe(false)
      expect(client.run).not.toHaveBeenCalled()
      expect(await workspace.reconnect()).toBe(true)
      expect(workspace.executorUpdateRequired.value).toBe(false)
      expect(workspace.selectedBuild.value?.stage).toBe('failed')
      expect(workspace.selectedPlan.value?.release_plan_id).toBe(id)
      expect(client.run).not.toHaveBeenCalled()
      expect(await workspace.run()).toBe(true)
      expect(client.run).toHaveBeenCalledExactlyOnceWith(id, 'native')
    } finally {
      workspace.dispose()
    }
  })
  it('reports when the running development server has yet to load reconnect', async () => {
    const { client, workspace } = fixture()
    client.reconnect.mockRejectedValue(new PublisherClientError('PUBLISHER_NOT_FOUND'))
    await workspace.connect()
    try {
      expect(await workspace.reconnect()).toBe(false)
      expect(workspace.error.value).toBe('PUBLISHER_EXECUTOR_HOST_UPDATE_REQUIRED')
      expect(client.run).not.toHaveBeenCalled()
    } finally {
      workspace.dispose()
    }
  })
  it('prepares the container after the build click, then starts the same plan once it is ready', async () => {
    vi.useFakeTimers()
    const { client, workspace } = fixture()
    const ready = await client.connect()
    const stopped = {
      ...ready,
      info: {
        ...ready.info,
        build_environment: { ...ready.info.build_environment, container_ready: false, can_start: true }
      }
    }
    client.connect.mockResolvedValue(stopped)
    client.info.mockResolvedValueOnce(stopped).mockResolvedValue(ready)
    await workspace.connect()
    workspace.executor.value = 'container'
    try {
      expect(client.startEnvironment).not.toHaveBeenCalled()
      const building = workspace.run()
      await vi.advanceTimersByTimeAsync(0)
      expect(client.startEnvironment).toHaveBeenCalledTimes(1)
      expect(await workspace.run()).toBe(false)
      expect(client.run).not.toHaveBeenCalled()
      await vi.advanceTimersByTimeAsync(1500)
      expect(await building).toBe(true)
      expect(client.run).toHaveBeenCalledExactlyOnceWith(id, 'container')
    } finally {
      workspace.dispose()
    }
  })
  it('keeps preparation failures separate from unknown build results', async () => {
    const { client, workspace } = fixture()
    const connection = await client.connect()
    connection.info.build_environment.container_ready = false
    connection.info.build_environment.can_start = true
    client.connect.mockResolvedValue(connection)
    client.startEnvironment.mockRejectedValue(new PublisherClientError('PUBLISHER_PROXY_UNAVAILABLE'))
    await workspace.connect()
    workspace.executor.value = 'container'
    try {
      expect(await workspace.run()).toBe(false)
      expect(client.run).not.toHaveBeenCalled()
      expect(workspace.error.value).toBe('PUBLISHER_PROXY_UNAVAILABLE')
      expect(workspace.needsReconcile.value).toBe(false)
    } finally {
      workspace.dispose()
    }
  })
  it('retains and monitors the active task when selecting a different release', async () => {
    vi.useFakeTimers(); const { client, workspace: w } = fixture(); await w.connect(); await w.run()
    const other = { ...plan, release_plan_id: '00000000-0000-4000-8000-000000000002', version: '1.3.0' }
    await w.selectPlan(other)
    expect(w.selectedPlan.value?.version).toBe(other.version); expect(w.ongoingBuild.value?.build_id).toBe(id); expect(w.running.value).toBe(true)
    await w.cancel(); expect(client.cancel).toHaveBeenCalledWith(id); expect(w.ongoingBuild.value).toBeNull(); w.dispose()
  })
  it('blocks another write after an unknown result until an explicit successful query', async () => {
    const { client, workspace: w } = fixture(); await w.connect()
    client.run.mockRejectedValue(new PublisherClientError('PUBLISHER_PROXY_UNAVAILABLE'))
    await w.run(); expect(w.needsReconcile.value).toBe(true)
    await w.run(); expect(client.run).toHaveBeenCalledTimes(1)
    await w.refresh(); expect(w.needsReconcile.value).toBe(false)
    w.dispose()
  })
  it('keeps locale keys identical and includes every stable proxy error', () => {
    const keys = (value: any, prefix = ''): string[] => Object.entries(value).flatMap(([key, child]) => typeof child === 'object' ? keys(child, prefix + key + '.') : [prefix + key]).sort()
    expect(keys(en)).toEqual(keys(zh))
    for (const code of ['PUBLISHER_PROXY_NOT_CONFIGURED', 'PUBLISHER_PROXY_UNAVAILABLE', 'PUBLISHER_PROXY_RESPONSE_INVALID', 'PUBLISHER_ARTIFACT_INVALID']) expect(Object.hasOwn(en.errors, code)).toBe(true)
  })
})
