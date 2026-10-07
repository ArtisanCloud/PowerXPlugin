import { describe, expect, it, vi, afterEach } from 'vitest'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { parse, compileScript } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'
import { createSSRApp, defineComponent, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
const require = createRequire(import.meta.url)
async function render(props: Record<string, unknown>) {
  vi.stubGlobal('useI18n', () => ({ t: (key: string) => key }))
  const { descriptor } = parse(readFileSync(new URL('../../app/components/knowledge/KnowledgeSpaceOverviewPanel.vue', import.meta.url), 'utf8'))
  const script = compileScript(descriptor, { id: 'knowledge-overview', inlineTemplate: true })
  const code = transformSync(script.content, { loader: 'ts', format: 'cjs' }).code
  const module = { exports: {} as any }
  new Function('require', 'module', 'exports', code)(require, module, module.exports)
  const app = createSSRApp(module.exports.default, props)
  app.component('UAlert', defineComponent({ props: ['title', 'description'], setup: props => () => h('div', [props.title, props.description]) }))
  app.component('UIcon', defineComponent({ render: () => null }))
  app.component('UBadge', defineComponent({ setup: (_, { slots }) => () => h('span', slots.default?.()) }))
  return renderToString(app)
}
afterEach(() => vi.unstubAllGlobals())
describe('shared knowledge overview', () => {
  it('shows a failed list as an error rather than an empty space list', async () => {
    const html = await render({ spaces: [], loading: false, error: 'knowledgeLab.proxyUnavailable' })
    expect(html).toContain('knowledgeLab.proxyUnavailable')
    expect(html).not.toContain('knowledgeSpaces.ui.noSpaces')
  })
  it('shows business names and a proxy description without exposing UUIDs', async () => {
    const uuid = '11111111-1111-4111-8111-111111111111'
    const html = await render({ spaces: [{ id: uuid, name: 'fixture', department: '-', statusLabel: 'knowledgeSpaces.status.active', statusColor: 'success' }], loading: false, descriptionKey: 'knowledgeLab.description' })
    expect(html).toContain('fixture')
    expect(html).toContain('knowledgeLab.description')
    expect(html).not.toContain('knowledgeSpaces.overview.description')
    expect(html).not.toContain(uuid)
  })
})
