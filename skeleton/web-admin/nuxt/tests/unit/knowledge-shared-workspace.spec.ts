import { afterEach, describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { compileScript, parse } from '@vue/compiler-sfc'
import { transformSync } from 'esbuild'
import { createRenderer, nextTick } from 'vue'
const requireNative = createRequire(import.meta.url)
const renderer = createRenderer<any, any>({ patchProp() {}, insert() {}, remove() {}, createElement: () => ({}), createText: () => ({}), createComment: () => ({}), setText() {}, setElementText() {}, parentNode: () => null, nextSibling: () => null })
const processModule={exports:{} as any}
new Function('module','exports',transformSync(readFileSync(new URL('../../app/composables/knowledge/taskProcess.ts',import.meta.url),'utf8'),{loader:'ts',format:'cjs'}).code)(processModule,processModule.exports)
const uuid = '11111111-1111-4111-8111-111111111111'
function adapter(client: any) {
 const code = transformSync(readFileSync(new URL('../../app/composables/api/usePowerXKnowledgeWorkspace.ts', import.meta.url), 'utf8'), { loader:'ts',format:'cjs' }).code
 const m = { exports:{} as any }; new Function('require','module','exports',code)((id:string) => id==='vue'?requireNative(id):id.endsWith('taskProcess')?processModule.exports:id.endsWith('knowledgeHostTypes')?{documentIngestionSchema:'powerx.knowledge.document-ingestion/v1'}:{useApiClient:()=>client},m,m.exports)
 return m.exports.usePowerXKnowledgeWorkspace('/powerx/knowledge-lab')
}
async function mount(workspace: any) {
 vi.stubGlobal('useRoute',()=>({params:{uuid}}));vi.stubGlobal('useRuntimeConfig',()=>({public:{}}));vi.stubGlobal('useI18n',()=>({t:(k:string)=>k}));vi.stubGlobal('useToast',()=>({add:vi.fn()}));vi.stubGlobal('navigateTo',vi.fn())
 const {descriptor} = parse(readFileSync(new URL('../../app/components/knowledge/KnowledgeIngestionWorkspace.vue',import.meta.url),'utf8'))
 const code = transformSync(compileScript(descriptor,{id:'shared'}).content,{loader:'ts',format:'cjs'}).code
 const m={exports:{} as any};new Function('require','module','exports',code)((id:string)=>id==='vue'?requireNative(id):id.endsWith('taskProcess')?processModule.exports:id.endsWith('knowledgeHostTypes')?{documentIngestionSchema:'powerx.knowledge.document-ingestion/v1'}:id.endsWith('useKnowledgeWorkspace')?{useKnowledgeWorkspace:()=>workspace}:{createPluginWsClient:vi.fn(),resolveApiBase:()=>'',getAuthToken:()=>''},m,m.exports)
 const app=renderer.createApp({...m.exports.default,render:()=>null});app.mount({});for(let i=0;i<12;i++)await nextTick();return {app,state:app._instance!.setupState}
}
const files=[{name:'one.txt',size:1,lastModified:1,text:async()=>'one'},{name:'two.md',size:1,lastModified:2,text:async()=>'two'}]
function client() {return {get:vi.fn(async(p:string)=>p.endsWith('/spaces')?{spaces:[{id:uuid,name:'fixture',status:'active'}]}:{strategy_packages:[{key:'A_simple',recommended_profile_key:'p0_basic',available:true,dependencies:{index:[],runtime:[],assets:[]},unavailable_reasons:[]}]}),post:vi.fn()}}
afterEach(()=>vi.unstubAllGlobals())
describe('shared ingestion with PowerX',()=>{
 it('keeps the Core configuration draft and blocks writes when snapshots are unsupported',async()=>{
  const c=client();const {app,state}=await mount(adapter(c));await state.readFiles({target:{files}})
  expect(state.uploadFiles).toHaveLength(2)
  expect(state.ingestion.ingestion_profile).toBe('')
  state.step=2;expect(state.canContinue).toBe(false)
  state.ingestion.chunk_size=640;state.step=3;expect(state.canContinue).toBe(false)
  state.step=4;expect(state.canSubmit).toBe(false);await state.submit()
  expect(c.post).not.toHaveBeenCalled();expect(navigateTo).not.toHaveBeenCalled()
  expect(state.ingestion.chunk_size).toBe(640)
  expect(state.uploadFiles.every((x:any)=>x.status==='ready')).toBe(true)
  expect(c.get.mock.calls.every(([p])=>p.startsWith('/admin/runtime/knowledge-lab/'))).toBe(true);app.unmount()
 })
 it('retains multi-file submission and per-file failures for an adapter supporting snapshots',async()=>{
  const c=client();const workspace=adapter(c)
  workspace.mode='local';workspace.capabilities.ingestionSettings=true
  workspace.api.strategyPackages=vi.fn(async()=>({items:[{key:'A_simple',profile_key:'p0_basic',ready:true}]}))
  workspace.api.strategyPackages=vi.fn(async()=>({items:[{key:'A_simple',profile_key:'p0_basic',ready:true}]}))
  workspace.submitDocument=vi.fn().mockResolvedValueOnce({documentUUID:uuid,jobUUID:uuid,status:'succeeded'}).mockRejectedValueOnce({data:{code:'INDEX_WRITE_FAILED',trace_id:'fixture'}})
  const {app,state}=await mount(workspace);await state.readFiles({target:{files}});expect(state.canSubmit).toBe(true);await state.submit()
  expect(workspace.submitDocument).toHaveBeenCalledTimes(2)
  expect(workspace.submitDocument.mock.calls[0][1].ingestion).toMatchObject({ingestion_profile:'p0_basic',chunk_size:800})
  expect(state.uploadFiles.map((x:any)=>x.status)).toEqual(['completed','failed']);expect(navigateTo).not.toHaveBeenCalled();expect(state.uploadFiles[1].errorTrace).toBe('fixture');app.unmount()
 })
 it('rejects unsupported settings before any host write',async()=>{
  const c=client();await expect(adapter(c).submitDocument(uuid,{title:'fixture',content:'fixture',ingestion:{}})).rejects.toMatchObject({data:{code:'KNOWLEDGE_UNSUPPORTED_CAPABILITY'}});expect(c.post).not.toHaveBeenCalled()
 })
})

const contract={schema:'powerx.knowledge.document-ingestion/v1',index_mode:'lexical_chunks',length_unit:'unicode_rune',content_types:['text/plain','text/markdown'],chunk_size_min:0,chunk_size_max:65536,priorities:['normal','high'],segment_modes:['unit','heading','clause','table_row','code_block','conversation'],segment_size_policies:['cap','target'],segment_order_items:['page','segment','separator','size'],page_priority_supported:false,strategy_override_supported:false,anchor_mode_requirements:{anchor_heading_path:'heading',anchor_clause_id:'clause',anchor_sentence_index:'clause',anchor_row_number:'table_row',anchor_speaker:'conversation'},defaults:{priority:'normal',segment_mode:'unit',segment_size_policy:'cap',chunk_size:1024,chunk_overlap:0,segment_order:['page','segment','separator','size'],separators:[],page_priority:false}}
it('submits Core snapshots with explicit false, empty arrays and preserved order',async()=>{
 const c=client();c.get.mockImplementation(async(p:string)=>p.endsWith('/spaces')?{spaces:[{id:uuid,name:'fixture',status:'active'}]}:{strategy_packages:[],document_ingestion:contract})
 const job='22222222-2222-4222-8222-222222222222';c.post.mockResolvedValue({item:{job_id:job,document_id:uuid,status:'queued'}})
 const {app,state}=await mount(adapter(c));await state.readFiles({target:{files}})
 state.ingestion.chunk_size=640;state.ingestion.chunk_overlap=80
 expect(state.canSubmit).toBe(true)
 expect(state.ingestion.anchor_heading_path).toBe(false)
 await state.submit()
 expect(c.post).toHaveBeenCalledTimes(2)
 for(const [path,body] of c.post.mock.calls){expect(path).toBe('/admin/runtime/knowledge-lab/spaces/'+uuid+'/documents');expect(body.ingestion).toMatchObject({schema:contract.schema,chunk_size:640,chunk_overlap:80,page_priority:false,anchor_heading_path:false,separators:[],segment_order:contract.segment_order_items});expect(body).not.toHaveProperty('ingestionProfile');expect(body.ingestion).not.toHaveProperty('processor_profile');expect(body.ingestion).not.toHaveProperty('strategy')}
 expect(state.uploadFiles.map((x:any)=>x.status)).toEqual(['accepted','accepted'])
 expect(state.uploadTasks.map((x:any)=>x.phase)).toEqual(['queued','queued'])
 expect(state.uploadTasks.every((x:any)=>x.durationKind==='request' && x.durationMs>=0)).toBe(true)
 expect(navigateTo).toHaveBeenCalled();app.unmount()
})
it('rejects incompatible modes, anchors, overlap and formats before submit',async()=>{
 const c=client();c.get.mockImplementation(async(p:string)=>p.endsWith('/spaces')?{spaces:[{id:uuid,name:'fixture',status:'active'}]}:{strategy_packages:[],document_ingestion:contract})
 const {app,state}=await mount(adapter(c));await state.readFiles({target:{files}})
 state.ingestion.segment_mode='semantic';expect(state.canSubmit).toBe(false)
 state.ingestion.segment_mode='heading';expect(state.canSubmit).toBe(false)
 state.ingestion.segment_mode='unit';state.ingestion.anchor_heading_path=true;expect(state.canSubmit).toBe(false)
 state.ingestion.anchor_heading_path=false;state.ingestion.chunk_overlap=1024;expect(state.canSubmit).toBe(false)
 state.ingestion.chunk_size=0;state.ingestion.chunk_overlap=0;expect(state.canSubmit).toBe(true)
 state.ingestion.segment_size_policy='target';expect(state.canSubmit).toBe(false)
 await state.submit();expect(c.post).not.toHaveBeenCalled();app.unmount()
})

it('sends document and space rebuilds to distinct routes without widening a rejected document request',async()=>{
 const c=client();const workspace=adapter(c);const doc='22222222-2222-4222-8222-222222222222';const job='33333333-3333-4333-8333-333333333333'
 c.post.mockResolvedValueOnce({item:{job_id:job,document_id:doc,status:'queued'}}).mockResolvedValueOnce({item:{job_id:job,status:'queued'}}).mockRejectedValueOnce({data:{code:'KNOWLEDGE_INDEX_CONFLICT'}})
 const request={mode:'reuse_snapshot',idempotency_key:'fixture-request'}
 await workspace.rebuildIndex(uuid,doc,request);await workspace.rebuildIndex(uuid,undefined,request)
 await expect(workspace.rebuildIndex(uuid,doc,request)).rejects.toMatchObject({data:{code:'KNOWLEDGE_INDEX_CONFLICT'}})
 expect(c.post.mock.calls.map(([path])=>path)).toEqual(['/admin/runtime/knowledge-lab/spaces/'+uuid+'/documents/'+doc+'/indexes:rebuild','/admin/runtime/knowledge-lab/spaces/'+uuid+'/indexes:rebuild','/admin/runtime/knowledge-lab/spaces/'+uuid+'/documents/'+doc+'/indexes:rebuild'])
 expect(c.post).toHaveBeenCalledTimes(3)
 for(const [,body] of c.post.mock.calls){expect(body).toEqual(request);expect(body).not.toHaveProperty('tenant_uuid')}
 await expect(workspace.rebuildIndex(uuid,doc,{mode:'apply_config',idempotency_key:'fixture-request'})).rejects.toMatchObject({data:{code:'KNOWLEDGE_INVALID_ARGUMENT'}})
 expect(c.post).toHaveBeenCalledTimes(3)
})

it('retries only known pre-acceptance failures and preserves accepted jobs, content and settings',async()=>{
 const c=client();c.get.mockImplementation(async(p:string)=>p.endsWith('/spaces')?{spaces:[{id:uuid,name:'fixture',status:'active'}]}:{strategy_packages:[],document_ingestion:contract})
 const firstJob='22222222-2222-4222-8222-222222222222';const secondJob='33333333-3333-4333-8333-333333333333'
 c.post.mockResolvedValueOnce({item:{job_id:firstJob,document_id:uuid,status:'queued'}}).mockRejectedValueOnce({data:{code:'KNOWLEDGE_PROFILE_UNAVAILABLE',trace_id:'fixture-trace'}}).mockResolvedValueOnce({item:{job_id:secondJob,document_id:uuid,status:'queued'}})
 const {app,state}=await mount(adapter(c));await state.readFiles({target:{files}})
 state.ingestion.chunk_size=640;state.ingestion.chunk_overlap=80
 await state.submit()
 expect(state.uploadFiles.map((x:any)=>x.status)).toEqual(['accepted','failed'])
 expect(state.profileUnavailable).toBe(true);expect(state.canRetryFailed).toBe(true)
 expect(state.uploadErrorLabel('KNOWLEDGE_PROFILE_UNAVAILABLE')).toBe('knowledgeLab.workspace.ingestionErrors.KNOWLEDGE_PROFILE_UNAVAILABLE')
 expect(state.uploadFiles[1].errorTrace).toBe('fixture-trace');expect(navigateTo).not.toHaveBeenCalled()
 const failedBody=c.post.mock.calls[1][1]
 await state.submit(true)
 expect(c.post).toHaveBeenCalledTimes(3);expect(c.post.mock.calls[2][1]).toEqual(failedBody)
 expect(state.uploadFiles.map((x:any)=>x.status)).toEqual(['accepted','accepted'])
 expect(state.uploadFiles[0].jobUUID).toBe(firstJob);expect(state.profileUnavailable).toBe(false)
 expect(navigateTo).toHaveBeenCalledWith({path:'/powerx/knowledge-lab/'+uuid+'/records',query:{job_uuid:[firstJob,secondJob],job_title:['one.txt','two.md']}})
 await state.submit(true);expect(c.post).toHaveBeenCalledTimes(3);app.unmount()
})
it('does not offer retry for unknown delivery results or unreadable files',async()=>{
 const c=client();c.get.mockImplementation(async(p:string)=>p.endsWith('/spaces')?{spaces:[{id:uuid,name:'fixture',status:'active'}]}:{strategy_packages:[],document_ingestion:contract})
 c.post.mockRejectedValue({data:{code:'KNOWLEDGE_UPSTREAM_DEPENDENCY',trace_id:'fixture-trace'}})
 const {app,state}=await mount(adapter(c));await state.readFiles({target:{files}});await state.submit()
 expect(state.canRetryFailed).toBe(false);await state.submit(true);expect(c.post).toHaveBeenCalledTimes(2)
 state.uploadFiles[0].errorCode='KNOWLEDGE_PROFILE_UNAVAILABLE';state.uploadFiles[0].content=''
 expect(state.canRetryFailed).toBe(false);app.unmount()
})
