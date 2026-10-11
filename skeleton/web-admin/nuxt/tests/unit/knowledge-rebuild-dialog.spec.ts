import { afterEach, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { compileScript, parse } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'
import { createRenderer, nextTick } from 'vue'
const native = createRequire(import.meta.url)
const renderer = createRenderer<any,any>({patchProp(){},insert(){},remove(){},createElement:()=>({}),createText:()=>({}),createComment:()=>({}),setText(){},setElementText(){},parentNode:()=>null,nextSibling:()=>null})
const schema='powerx.knowledge.document-ingestion/v1'
const space='11111111-1111-4111-8111-111111111111';const document='22222222-2222-4222-8222-222222222222'
const contract={schema,chunk_size_max:65536,priorities:['normal','high'],segment_modes:['unit'],segment_size_policies:['cap','target'],page_priority_supported:false,anchor_mode_requirements:{},defaults:{schema,priority:'normal',segment_mode:'unit',segment_size_policy:'cap',chunk_size:1024,chunk_overlap:0,segment_order:['page','segment','separator','size'],separators:[],page_priority:false}}
async function settle(){for(let n=0;n<10;n++){await Promise.resolve();await nextTick()}}
async function mount(workspace:any){
 vi.stubGlobal('useI18n',()=>({t:(key:string)=>key}));let key=0;vi.stubGlobal('crypto',{randomUUID:()=>`fixture-${++key}`})
 const {descriptor}=parse(readFileSync(new URL('../../app/components/knowledge/KnowledgeRebuildDialog.vue',import.meta.url),'utf8'))
 const code=transformSync(compileScript(descriptor,{id:'rebuild'}).content,{loader:'ts',format:'cjs'}).code
 const m={exports:{} as any};new Function('require','module','exports',code)((id:string)=>id==='vue'?native(id):id.endsWith('useKnowledgeWorkspace')?{useKnowledgeWorkspace:()=>workspace}:{documentIngestionSchema:schema},m,m.exports)
 const app=renderer.createApp({...m.exports.default,render:()=>null},{spaceUUID:space,documentUUID:document})
 app.mount({});return {app,state:app._instance!.setupState}
}
afterEach(()=>vi.unstubAllGlobals())
it('reuse does not read current profiles and retries preserve the same request key',async()=>{
 const workspace={api:{strategyPackages:vi.fn()},capabilities:{ingestionSettings:true},ingestionContract:contract,rebuildIndex:vi.fn().mockRejectedValue({data:{code:'KNOWLEDGE_INDEX_CONFLICT'}})}
 const {app,state}=await mount(workspace);state.open=true;await settle();expect(state.canSubmit).toBe(true)
 await state.submit();await state.submit()
 expect(workspace.api.strategyPackages).not.toHaveBeenCalled()
 expect(workspace.rebuildIndex).toHaveBeenCalledTimes(2)
 for(const args of workspace.rebuildIndex.mock.calls){expect(args.slice(0,2)).toEqual([space,document]);expect(args[2]).toEqual({mode:'reuse_snapshot',idempotency_key:'fixture-1'})}
 expect(state.open).toBe(true);app.unmount()
})
it('new settings load the published contract and use a new key when the request changes',async()=>{
 const workspace={api:{strategyPackages:vi.fn().mockResolvedValue({items:[]})},capabilities:{ingestionSettings:true},ingestionContract:contract,rebuildIndex:vi.fn().mockRejectedValue({data:{code:'KNOWLEDGE_INDEX_CONFLICT'}})}
 const {app,state}=await mount(workspace);state.open=true;await settle();await state.submit()
 state.mode='apply_config';await settle();expect(workspace.api.strategyPackages).toHaveBeenCalledTimes(1)
 state.settings.chunk_size=640;state.settings.chunk_overlap=80;expect(state.canSubmit).toBe(true)
 await state.submit();await state.submit()
 const request=workspace.rebuildIndex.mock.calls[1][2]
 expect(request).toMatchObject({mode:'apply_config',idempotency_key:'fixture-2',ingestion:{schema,chunk_size:640,chunk_overlap:80,page_priority:false,separators:[]}})
 expect(workspace.rebuildIndex.mock.calls[2][2]).toEqual(request)
 expect(request.ingestion).not.toHaveProperty('ingestion_profile')
 state.settings.chunk_overlap=640;expect(state.canSubmit).toBe(false);app.unmount()
})
