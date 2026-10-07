import { describe, expect, it, vi, afterEach } from 'vitest'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { parse, compileScript } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'
import { createRenderer, defineComponent, nextTick, ref } from 'vue'
const nativeRequire = createRequire(import.meta.url)
const blank = defineComponent({ render: () => null })
function workspace() {
  const filename = new URL('../../app/components/knowledge/KnowledgeLabWorkspace.vue', import.meta.url)
  const { descriptor } = parse(readFileSync(filename, 'utf8'))
  const script = compileScript(descriptor, { id: 'knowledge-routing' })
  const code = transformSync(script.content, { loader: 'ts', format: 'cjs' }).code
  const module = { exports: {} as any }
  const require = (id: string) => id === 'vue' ? nativeRequire(id) : id.endsWith('.vue') ? { default: blank } : { useApiClient: () => client, apiPost: vi.fn() }
  new Function('require', 'module', 'exports', code)(require, module, module.exports)
  return { ...module.exports.default, render: () => null }
}
const client = { get: vi.fn(), post: vi.fn(), delete: vi.fn() }
const renderer = createRenderer<any, any>({ patchProp() {}, insert() {}, remove() {}, createElement: () => ({}), createText: () => ({}), createComment: () => ({}), setText() {}, setElementText() {}, parentNode: () => null, nextSibling: () => null })
const uuid = '11111111-1111-4111-8111-111111111111'
async function mount(section: string, requestedUUID = uuid, translate = (key: string) => key) {
  vi.stubGlobal('useRoute', () => ({ path: `/powerx/knowledge-lab/${requestedUUID}/${section}`, params: { uuid: requestedUUID }, query: {} }))
  vi.stubGlobal('navigateTo', vi.fn())
  vi.stubGlobal('useI18n', () => ({ t: translate, te: () => true }))
  client.get.mockImplementation(async (path: string) => {
    if (path.endsWith('/provider')) return { mode: 'delegated', capabilities: { operations: [] } }
    if (path.endsWith('/departments')) return { items: [{ department_uuid: uuid, name: 'host-department' }] }
    if (path.endsWith('/catalog')) throw { data: { code: 'KNOWLEDGE_UNSUPPORTED_CAPABILITY', error: 'knowledgeLab.catalogNotExposed' } }
    if (path.endsWith('/spaces')) return { spaces: [{ id: uuid, name: 'fixture', status: 'active' }] }
    if (path.includes('/ingestions')) return { records: [] }
    if (path.endsWith('/policy')) return { strategies: [] }
    throw new Error(`Unexpected request: ${path}`)
  })
  const app = renderer.createApp(workspace(), { section }); app.mount({})
  for (let n = 0; n < 12; n++) await nextTick()
  return app
}
afterEach(() => { vi.clearAllMocks(); vi.unstubAllGlobals() })
describe('knowledge Lab route authority', () => {
  it.each(['ingestion', 'records', 'strategy', 'sources', 'playground'])('leaves %s data access to the shared workspace adapter', async section => {
    const app = await mount(section)
    expect(client.get).not.toHaveBeenCalled()
    expect(client.post).not.toHaveBeenCalled()
    app.unmount()
  })
  it('keeps creation blocked without host directories and never reads Local catalogs', async () => {
    const app = await mount('create')
    app._instance!.setupState.createForm.spaceName = 'fixture'
    await app._instance!.setupState.createKnowledgeSpace()
    expect(client.post).not.toHaveBeenCalled()
    expect(app._instance!.setupState.creationDepartments).toEqual([{ value: uuid, label: 'host-department' }])
    expect(app._instance!.setupState.creationDepartmentsError).toBe('')
    expect(client.get).toHaveBeenCalledWith('/admin/runtime/knowledge-lab/catalog')
    expect(client.get.mock.calls.every(([path]) => path.startsWith('/admin/runtime/knowledge-lab/'))).toBe(true)
    expect(client.get.mock.calls.some(([path]) => path.endsWith('/spaces'))).toBe(false)
    app.unmount()
  })
  it('preserves a chosen department only while it remains in the refreshed Core directory', async () => {
    const app = await mount('create')
    const state = app._instance!.setupState
    state.createForm.departmentUUID = uuid
    await state.loadCreationDepartments()
    expect(state.createForm.departmentUUID).toBe(uuid)
    client.get.mockResolvedValueOnce({ items: [] })
    await state.loadCreationDepartments()
    expect(state.createForm.departmentUUID).toBe('')
    expect(state.creationDepartments).toEqual([])
    app.unmount()
  })
  it('clears an unavailable department after a failed refresh and retains the error', async () => {
    const app = await mount('create')
    const state = app._instance!.setupState
    state.createForm.departmentUUID = uuid
    client.get.mockRejectedValueOnce({ data: { error: 'knowledgeLab.departmentsUnavailable' } })
    await state.loadCreationDepartments()
    expect(state.createForm.departmentUUID).toBe('')
    expect(state.creationDepartments).toEqual([])
    expect(state.creationDepartmentsError).toBe('knowledgeLab.departmentsUnavailable')
    app.unmount()
  })
  it('does not synthesize strategy or Profile defaults from an empty catalog', async () => {
    const app = await mount('create')
    const state = app._instance!.setupState
    client.get.mockResolvedValueOnce({ scenes: [], strategy_packages: [] })
    await state.loadKnowledgeCatalog()
    expect(state.createForm).toMatchObject({ sceneKey: '', strategyPackageKey: '', ingestionProfileKey: '', indexProfileKey: '', ragProfileKey: '', featureFlags: [] })
    expect(state.canCreateSpace).toBe(false)
    expect(client.post).not.toHaveBeenCalled()
    app.unmount()
  })
  it('updates directory errors when the language changes without another request', async () => {
    const language = ref('zh')
    const app = await mount('create', uuid, key => `${language.value}:${key}`)
    const state = app._instance!.setupState
    client.get.mockRejectedValueOnce({ data: { error: 'knowledgeLab.departmentsUnavailable' } })
    await state.loadCreationDepartments()
    const calls = client.get.mock.calls.length
    expect(state.catalogError).toBe('zh:knowledgeLab.catalogNotExposed')
    expect(state.creationDepartmentsError).toBe('zh:knowledgeLab.departmentsUnavailable')
    language.value = 'en'
    await nextTick()
    expect(state.catalogError).toBe('en:knowledgeLab.catalogNotExposed')
    expect(state.creationDepartmentsError).toBe('en:knowledgeLab.departmentsUnavailable')
    expect(client.get.mock.calls.length).toBe(calls)
    app.unmount()
  })
  it('submits typed service fields for a ready catalog and returns to the list', async () => {
    const app = await mount('create'); const state = app._instance!.setupState
    const profile = { uuid, key: 'p1_general', version: 1 }
    client.get.mockResolvedValueOnce({ strategy_packages: [{ key: 'A_simple', label: 'fixture', available: true, recommended_profile_key: 'p1_general', recommended_scenes: ['support_faq'], profiles: { ingestion: profile, index: profile, rag: profile } }], policy_templates: [{ uuid, name: 'fixture', version: 'v1' }], default_policy_template_uuid: uuid })
    await state.loadKnowledgeCatalog()
    state.createForm.spaceName = 'fixture'; state.createForm.departmentUUID = uuid
    expect(state.canCreateSpace).toBe(true)
    client.post.mockRejectedValueOnce({ data: { code: 'KNOWLEDGE_SPACE_CONFLICT', error: 'knowledgeLab.gatewayFailed', trace_id: 'core-conflict-trace' } })
    await state.createKnowledgeSpace()
    expect(client.post).toHaveBeenCalledTimes(1)
    expect(navigateTo).not.toHaveBeenCalled()
    expect(state.createError).toContain('core-conflict-trace')
    client.post.mockResolvedValueOnce({ item: { space_uuid: uuid, status: 'pending_iam' } })
    await state.createKnowledgeSpace()
    expect(client.post).toHaveBeenCalledWith('/admin/runtime/knowledge-lab/spaces', { name: 'fixture', department_uuid: uuid, strategy_key: 'A_simple', scene_key: 'support_faq', policy_template_uuid: uuid, ingestion_profile_uuid: uuid, index_profile_uuid: uuid, rag_profile_uuid: uuid })
    expect(navigateTo).toHaveBeenCalledWith('/powerx/knowledge-lab'); app.unmount()
  })
  it('does not submit a strategy that Core marks unavailable', async () => {
    const app = await mount('create'); const state = app._instance!.setupState
    client.get.mockResolvedValueOnce({ strategy_packages: [{ key: 'A_simple', label: 'fixture', available: false, unavailable_reasons: ['rag_profile_not_published'] }], policy_templates: [{ uuid, name: 'fixture', version: 'v1' }], default_policy_template_uuid: uuid })
    await state.loadKnowledgeCatalog()
    state.createForm.spaceName = 'fixture'; state.createForm.departmentUUID = uuid
    await state.createKnowledgeSpace()
    expect(state.canCreateSpace).toBe(false); expect(client.post).not.toHaveBeenCalled()
    expect(state.strategyUnavailableDescription).toContain('knowledgeLab.strategyUnavailable'); app.unmount()
  })
  it('opening ingestion only initializes the editable form and submits nothing', async () => {
    const app = await mount('ingestion')
    expect(client.get.mock.calls.some(([path]) => path.includes('/ingestions') || path.endsWith('/policy'))).toBe(false)
    expect(client.post).not.toHaveBeenCalled(); app.unmount()
  })
})
