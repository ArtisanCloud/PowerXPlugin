import { afterEach, describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { compileScript, parse } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'
import { createRenderer, nextTick } from 'vue'
const requireNative = createRequire(import.meta.url)
const renderer = createRenderer<any, any>({ patchProp() {}, insert() {}, remove() {}, createElement: () => ({}), createText: () => ({}), createComment: () => ({}), setText() {}, setElementText() {}, parentNode: () => null, nextSibling: () => null })
const uuid = '11111111-1111-4111-8111-111111111111'
function adapter(client: any) {
 const code = transformSync(readFileSync(new URL('../../app/composables/api/usePowerXKnowledgeWorkspace.ts', import.meta.url), 'utf8'), { loader:'ts',format:'cjs' }).code
 const m = { exports:{} as any }; new Function('require','module','exports',code)(() => ({useApiClient:()=>client}),m,m.exports)
 return m.exports.usePowerXKnowledgeWorkspace('/powerx/knowledge-lab')
}
async function mount(workspace: any) {
 vi.stubGlobal('useRoute',()=>({params:{uuid}}));vi.stubGlobal('useRuntimeConfig',()=>({public:{}}));vi.stubGlobal('useI18n',()=>({t:(k:string)=>k}));vi.stubGlobal('useToast',()=>({add:vi.fn()}));vi.stubGlobal('navigateTo',vi.fn())
 const {descriptor} = parse(readFileSync(new URL('../../app/components/knowledge/KnowledgeIngestionWorkspace.vue',import.meta.url),'utf8'))
 const code = transformSync(compileScript(descriptor,{id:'shared'}).content,{loader:'ts',format:'cjs'}).code
 const m={exports:{} as any};new Function('require','module','exports',code)((id:string)=>id==='vue'?requireNative(id):id.endsWith('useKnowledgeWorkspace')?{useKnowledgeWorkspace:()=>workspace}:{createPluginWsClient:vi.fn(),resolveApiBase:()=>'',getAuthToken:()=>''},m,m.exports)
 const app=renderer.createApp({...m.exports.default,render:()=>null});app.mount({});for(let i=0;i<12;i++)await nextTick();return {app,state:app._instance!.setupState}
}
const files=[{name:'one.txt',size:1,lastModified:1,text:async()=>'one'},{name:'two.md',size:1,lastModified:2,text:async()=>'two'}]
function client() {return {get:vi.fn(async(p:string)=>p.endsWith('/spaces')?{spaces:[{id:uuid,name:'fixture',status:'active'}]}:{strategy_packages:[{key:'A_simple',recommended_profile_key:'p0_basic',available:true,dependencies:{index:[],runtime:[],assets:[]},unavailable_reasons:[]}]}),post:vi.fn()}}
afterEach(()=>vi.unstubAllGlobals())
describe('shared ingestion with PowerX',()=>{
 it('keeps the Core configuration draft and blocks writes when snapshots are unsupported',async()=>{
  const c=client();const {app,state}=await mount(adapter(c));await state.readFiles({target:{files}})
  expect(state.uploadFiles).toHaveLength(2)
  expect(state.ingestion.ingestion_profile).toBe('p0_basic')
  state.step=2;expect(state.canContinue).toBe(true)
  state.ingestion.chunk_size=640;state.step=3;expect(state.canContinue).toBe(true)
  state.step=4;expect(state.canSubmit).toBe(false);await state.submit()
  expect(c.post).not.toHaveBeenCalled();expect(navigateTo).not.toHaveBeenCalled()
  expect(state.ingestion.chunk_size).toBe(640)
  expect(state.uploadFiles.every((x:any)=>x.status==='ready')).toBe(true)
  expect(c.get.mock.calls.every(([p])=>p.startsWith('/admin/runtime/knowledge-lab/'))).toBe(true);app.unmount()
 })
 it('retains multi-file submission and per-file failures for an adapter supporting snapshots',async()=>{
  const c=client();const workspace=adapter(c)
  workspace.mode='local';workspace.capabilities.ingestionSettings=true
  workspace.submitDocument=vi.fn().mockResolvedValueOnce({documentUUID:uuid,jobUUID:uuid,status:'succeeded'}).mockRejectedValueOnce({data:{code:'INDEX_WRITE_FAILED'}})
  const {app,state}=await mount(workspace);await state.readFiles({target:{files}});expect(state.canSubmit).toBe(true);await state.submit()
  expect(workspace.submitDocument).toHaveBeenCalledTimes(2)
  expect(workspace.submitDocument.mock.calls[0][1].ingestion).toMatchObject({ingestion_profile:'p0_basic',chunk_size:800})
  expect(state.uploadFiles.map((x:any)=>x.status)).toEqual(['completed','failed']);expect(navigateTo).not.toHaveBeenCalled();app.unmount()
 })
 it('rejects unsupported settings before any host write',async()=>{
  const c=client();await expect(adapter(c).submitDocument(uuid,{title:'fixture',content:'fixture',ingestion:{}})).rejects.toMatchObject({data:{code:'KNOWLEDGE_UNSUPPORTED_CAPABILITY'}});expect(c.post).not.toHaveBeenCalled()
 })
})
