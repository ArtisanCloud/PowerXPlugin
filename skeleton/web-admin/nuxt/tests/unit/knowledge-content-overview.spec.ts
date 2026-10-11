import { afterEach, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { compileScript, parse } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'
import { createRenderer, nextTick } from 'vue'
const native = createRequire(import.meta.url)
function loadModule(path: string) {
  const m = { exports: {} as any }
  const code = transformSync(readFileSync(new URL(path, import.meta.url), 'utf8'), {
    loader: 'ts',
    format: 'cjs'
  }).code
  new Function('require', 'module', 'exports', code)(native, m, m.exports)
  return m.exports
}
const helpers = loadModule('../../app/composables/knowledge/contentOverview.ts')
const renderer = createRenderer<any, any>({
  patchProp() {},
  insert() {},
  remove() {},
  createElement: () => ({}),
  createText: () => ({}),
  createComment: () => ({}),
  setText() {},
  setElementText() {},
  parentNode: () => null,
  nextSibling: () => null
})
const space = '11111111-1111-4111-8111-111111111111'
const doc = '22222222-2222-4222-8222-222222222222'
const otherSpace = '33333333-3333-4333-8333-333333333333'
const otherDoc = '44444444-4444-4444-8444-444444444444'
const generation = '55555555-5555-4555-8555-555555555555'
const nextGeneration = '66666666-6666-4666-8666-666666666666'
const document = { uuid: doc, title: 'fixture', preview: 'fixture-source', suggested_questions: [] }
function overview(values: any = {}) {
  return {
    space_uuid: space,
    scope: 'space_documents',
    topics: [],
    documents: [document],
    total: 1,
    ...values
  }
}
afterEach(() => vi.unstubAllGlobals())
async function flush() {
  for (let n = 0; n < 10; n++) await nextTick()
}
async function mount(workspace: any, props: any = {}) {
  const t = vi.fn((key: string) => key)
  vi.stubGlobal('useI18n', () => ({ t }))
  const { descriptor } = parse(
    readFileSync(
      new URL('../../app/components/knowledge/KnowledgeContentOverview.vue', import.meta.url),
      'utf8'
    )
  )
  const code = transformSync(compileScript(descriptor, { id: 'content-overview' }).content, {
    loader: 'ts',
    format: 'cjs'
  }).code
  const m = { exports: {} as any }
  new Function('require', 'module', 'exports', code)(
    (id: string) =>
      id === 'vue'
        ? native(id)
        : id.endsWith('contentOverview')
          ? helpers
          : id.endsWith('requestFailure')
            ? loadModule('../../app/composables/knowledge/requestFailure.ts')
            : { useKnowledgeWorkspace: () => workspace },
    m,
    m.exports
  )
  const question = vi.fn()
  const app = renderer.createApp(
    { ...m.exports.default, render: () => null },
    { spaceUUID: space, spaceName: 'fixture', onQuestion: question, ...props }
  )
  app.mount({})
  await flush()
  return {
    app,
    state: app._instance!.setupState,
    question,
    t,
    update: (values: any) => Object.assign(app._instance!.props, values)
  }
}
it('uses exact Local source previews without inventing summaries or suggested questions', () => {
  const content = '😀'.repeat(321)
  const value = helpers.localContentOverview(space, [
    { uuid: doc, space_uuid: space, title: 'fixture', content }
  ])
  expect(Array.from(value.documents[0].preview)).toHaveLength(320)
  expect(value.documents[0].preview).toBe('😀'.repeat(320))
  expect(value.documents[0].suggested_questions).toEqual([])
  expect(value.summary).toBeUndefined()
  expect(value.scope).toBe('space_documents')
  expect(value.total).toBe(1)
  expect(() =>
    helpers.localContentOverview(space, [
      { uuid: doc, space_uuid: otherSpace, title: 'fixture', content }
    ])
  ).toThrow()
})
it('does not call Local documents or retrieval to disguise a missing PowerX directory', async () => {
  const workspace = { mode: 'powerx', api: { documents: vi.fn() }, semanticQuery: vi.fn() }
  const { app, state } = await mount(workspace)
  expect(state.state).toBe('unavailable')
  expect(state.overview).toBeUndefined()
  expect(workspace.api.documents).not.toHaveBeenCalled()
  expect(workspace.semanticQuery).not.toHaveBeenCalled()
  app.unmount()
})
it('loads the directory and fills a source-bound question without submitting retrieval', async () => {
  const service = vi
    .fn()
    .mockResolvedValue(
      overview({ documents: [{ ...document, suggested_questions: ['fixture-question'] }] })
    )
  const search = vi.fn()
  const { app, state, question, t } = await mount({ contentOverview: service, search })
  expect(state.state).toBe('ready')
  expect(state.suggestions).toEqual([
    { documentUUID: doc, title: 'fixture', text: 'fixture-question' }
  ])
  state.pickQuestion('fixture-question')
  expect(question).toHaveBeenCalledExactlyOnceWith('fixture-question')
  expect(search).not.toHaveBeenCalled()
  state.directoryOpen = true
  state.askDocument(document)
  expect(state.directoryOpen).toBe(false)
  expect(t).toHaveBeenCalledWith('knowledgeContent.articleQuestion', { title: 'fixture' })
  expect(question).toHaveBeenCalledTimes(1)
  state.emitPendingQuestion()
  expect(question).toHaveBeenLastCalledWith('knowledgeContent.articleQuestion')
  app.unmount()
})
it('does not replace the question during an active retrieval request', async () => {
  const service = vi.fn().mockResolvedValue(overview())
  const { app, state, question } = await mount({ contentOverview: service }, { disabled: true })
  state.pickQuestion('fixture-question')
  expect(question).not.toHaveBeenCalled()
  app.unmount()
})
it('keeps a permission error distinct from an empty directory and retries explicitly', async () => {
  const service = vi
    .fn()
    .mockRejectedValue({ statusCode: 403, data: { code: 'KNOWLEDGE_FORBIDDEN' } })
  const { app, state } = await mount({ contentOverview: service })
  expect(state.state).toBe('error')
  expect(state.overview).toBeUndefined()
  expect(state.error).toBeTruthy()
  service.mockResolvedValue(overview({ documents: [], total: 0 }))
  await state.load()
  expect(state.state).toBe('ready')
  expect(state.overview.total).toBe(0)
  expect(state.error).toBe('')
  app.unmount()
})
it('ignores an old space response after navigation', async () => {
  let resolveOld: (value: any) => void = () => {}
  const service = vi
    .fn()
    .mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveOld = resolve
        })
    )
    .mockResolvedValueOnce(
      overview({ space_uuid: otherSpace, documents: [{ ...document, uuid: otherDoc }] })
    )
  const { app, state, update } = await mount({ contentOverview: service })
  update({ spaceUUID: otherSpace })
  await flush()
  resolveOld(overview())
  await flush()
  expect(state.overview.space_uuid).toBe(otherSpace)
  expect(state.overview.documents[0].uuid).toBe(otherDoc)
  app.unmount()
})
it('rejects mismatched corpus versions rather than displaying stale suggestions', async () => {
  const service = vi
    .fn()
    .mockResolvedValue(
      overview({ scope: 'queryable_documents', corpus_generation: nextGeneration })
    )
  const { app, state } = await mount(
    { contentOverview: service },
    { retrievalMode: 'semantic', corpusGeneration: generation }
  )
  expect(service).toHaveBeenCalledWith(space, { mode: 'semantic', corpus_generation: generation })
  expect(state.state).toBe('error')
  expect(state.overview).toBeUndefined()
  expect(state.suggestions).toEqual([])
  app.unmount()
})
it('loads remaining pages and rejects duplicate or changed directory pages', async () => {
  const service = vi
    .fn()
    .mockResolvedValueOnce(overview({ total: 2, next_cursor: 'cursor-1' }))
    .mockResolvedValueOnce(overview({ documents: [{ ...document, uuid: otherDoc }], total: 2 }))
  const { app, state } = await mount({ contentOverview: service })
  await state.loadMore()
  expect(service).toHaveBeenLastCalledWith(space, { cursor: 'cursor-1' })
  expect(state.overview.documents.map((item: any) => item.uuid)).toEqual([doc, otherDoc])
  expect(state.overview.next_cursor).toBeUndefined()
  service.mockResolvedValueOnce(overview({ total: 2, next_cursor: 'cursor-1' }))
  await state.load()
  service.mockResolvedValueOnce(overview({ total: 2 }))
  await state.loadMore()
  expect(state.pageError).toBeTruthy()
  expect(state.overview.documents).toHaveLength(1)
  app.unmount()
})
it('rejects cross-space or malformed directory identities and incomplete first pages', () => {
  for (const value of [
    overview({ space_uuid: otherSpace }),
    overview({ topics: [doc] }),
    overview({ documents: [{ ...document, title: doc }] }),
    overview({
      documents: [{ ...document, suggested_questions: ['fixture-question', 'fixture-question'] }]
    }),
    overview({ documents: [{ ...document, uuid: '1' }] }),
    overview({ documents: [document, document], total: 2 }),
    overview({ total: 2 }),
    overview({ documents: [{ ...document, suggested_questions: null }] })
  ]) {
    expect(() => helpers.validateContentOverview(value, space, {})).toThrow()
  }
})
