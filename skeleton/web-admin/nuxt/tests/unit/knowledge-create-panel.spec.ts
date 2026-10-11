import { describe, expect, it, vi, afterEach } from 'vitest'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { parse, compileScript } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'
import { createRenderer, createSSRApp, defineComponent, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
const require = createRequire(import.meta.url)
function panel(inlineTemplate = true) {
  vi.stubGlobal('useI18n', () => ({ t: (key: string) => key }))
  const { descriptor } = parse(readFileSync(new URL('../../app/components/knowledge/KnowledgeSpaceCreatePanel.vue', import.meta.url), 'utf8'))
  const code = transformSync(compileScript(descriptor, { id: 'knowledge-create', inlineTemplate }).content, { loader: 'ts', format: 'cjs' }).code
  const module = { exports: {} as any }
  new Function('require', 'module', 'exports', code)(require, module, module.exports)
  return module.exports.default
}
const base = { name: 'fixture', department: '', strategy: '', overviewPath: '/powerx/knowledge-lab', departmentOptions: [], strategyOptions: [], canSave: false, executionStatus: 'knowledgeLab.readinessUnavailable' }
afterEach(() => vi.unstubAllGlobals())
describe('shared knowledge creation form', () => {
  it('keeps basic information, strategy and profile fields visible when host catalogs are unavailable', async () => {
    const app = createSSRApp(panel(), { ...base, departmentsError: 'knowledgeLab.departmentsUnavailable', strategyError: 'knowledgeLab.catalogNotExposed' })
    app.component('UFormField', defineComponent({ props: ['label'], setup: (props, { slots }) => () => h('label', [props.label, slots.default?.(), slots.help?.()]) }))
    app.component('UInput', defineComponent({ props: ['disabled', 'modelValue'], setup: props => () => h('input', { disabled: props.disabled, value: props.modelValue }) }))
    app.component('USelectMenu', defineComponent({ props: ['disabled'], setup: props => () => h('select', { disabled: props.disabled }) }))
    app.component('UButton', defineComponent({ props: ['disabled', 'to'], setup: (props, { slots }) => () => h(props.to ? 'a' : 'button', { href: props.to, disabled: props.disabled }, slots.default?.()) }))
    const html = await renderToString(app)
    for (const key of ['knowledgeSpaces.create.basicTitle', 'knowledgeSpaces.ui.spaceName', 'knowledgeSpaces.ui.department', 'knowledgeSpaces.ui.strategyPackage', 'knowledgeSpaces.ui.strategyAutoMapping', 'knowledgeLab.departmentsUnavailable', 'knowledgeLab.catalogNotExposed']) expect(html).toContain(key)
    expect(html).toContain('value="fixture"')
    expect(html).toContain('<select disabled')
    expect(html).not.toContain('<input disabled')
    expect(html).toContain('<button disabled')
  })
  it.each([
    { canSave: false },
    { canSave: true, departmentsError: 'knowledgeLab.departmentsUnavailable' },
    { canSave: true, strategyError: 'knowledgeLab.catalogNotExposed' },
    { canSave: true, saving: true },
  ])('blocks keyboard/programmatic submission when unavailable: %j', props => {
    const saved = vi.fn()
    const renderer = createRenderer<any, any>({ patchProp() {}, insert() {}, remove() {}, createElement: () => ({}), createText: () => ({}), createComment: () => ({}), setText() {}, setElementText() {}, parentNode: () => null, nextSibling: () => null })
    const app = renderer.createApp({ ...panel(false), render: () => null }, { ...base, ...props, onSave: saved })
    app.mount({}); app._instance!.setupState.submit()
    expect(saved).not.toHaveBeenCalled(); app.unmount()
  })
  it('emits one save for a valid ready form', () => {
    const saved = vi.fn()
    const renderer = createRenderer<any, any>({ patchProp() {}, insert() {}, remove() {}, createElement: () => ({}), createText: () => ({}), createComment: () => ({}), setText() {}, setElementText() {}, parentNode: () => null, nextSibling: () => null })
    const app = renderer.createApp({ ...panel(false), render: () => null }, { ...base, canSave: true, onSave: saved })
    app.mount({}); app._instance!.setupState.submit()
    expect(saved).toHaveBeenCalledTimes(1); app.unmount()
  })
})
