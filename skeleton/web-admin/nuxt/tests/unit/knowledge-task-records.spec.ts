import {afterEach,expect,it,vi} from 'vitest'
import {readFileSync} from 'node:fs'
import {createRequire} from 'node:module'
import {compileScript,parse} from '@vue/compiler-sfc'
import {transformSync} from 'esbuild'
import {createRenderer,nextTick} from 'vue'
import * as processModel from '../../app/composables/knowledge/taskProcess'
const native=createRequire(import.meta.url)
const renderer=createRenderer<any,any>({patchProp(){},insert(){},remove(){},createElement:()=>({}),createText:()=>({}),createComment:()=>({}),setText(){},setElementText(){},parentNode:()=>null,nextSibling:()=>null})
const space='11111111-1111-4111-8111-111111111111',job='22222222-2222-4222-8222-222222222222'
async function mount(indexJob:any){
 const workspace={mode:'powerx',basePath:'/powerx/knowledge-lab',capabilities:{documents:false,realtime:false},api:{spaces:vi.fn(async()=>({items:[{uuid:space,name:'fixture'}]}))},indexJob}
 vi.stubGlobal('useRoute',()=>({path:'/powerx/knowledge-lab/'+space+'/records',params:{uuid:space},query:{job_uuid:job,job_title:'fixture'}}));vi.stubGlobal('useRuntimeConfig',()=>({public:{}}));vi.stubGlobal('useI18n',()=>({locale:{value:'zh'},t:(k:string)=>k}));vi.stubGlobal('useToast',()=>({add:vi.fn()}))
 const {descriptor}=parse(readFileSync(new URL('../../app/components/knowledge/KnowledgeSpaceRecordsWorkspace.vue',import.meta.url),'utf8'))
 const code=transformSync(compileScript(descriptor,{id:'record-test'}).content,{loader:'ts',format:'cjs'}).code
 const m={exports:{} as any};new Function('require','module','exports',code)((id:string)=>id==='vue'?native(id):id.endsWith('taskProcess')?processModel:id.endsWith('useKnowledgeWorkspace')?{useKnowledgeWorkspace:()=>workspace}:{createPluginWsClient:vi.fn(),resolveApiBase:()=>'',getAuthToken:()=>''},m,m.exports)
 const app=renderer.createApp({...m.exports.default,render:()=>null});app.mount({});for(let i=0;i<15;i++)await nextTick();return {app,state:app._instance!.setupState}
}
afterEach(()=>vi.unstubAllGlobals())
it('refreshes a queued task into its actual terminal state without polling',async()=>{
 const get=vi.fn().mockResolvedValueOnce({job_id:job,space_id:space,status:'queued'}).mockResolvedValueOnce({job_id:job,space_id:space,status:'succeeded',chunk_count:2,started_at:'2026-10-08T10:00:00.010Z',finished_at:'2026-10-08T10:00:00.023Z'})
 const {app,state}=await mount(get);expect(state.submittedTasks[0].phase).toBe('queued');expect(state.submittedTasks[0].durationMs).toBeUndefined();expect(get).toHaveBeenCalledTimes(1)
 await state.openHostJob(job);expect(get).toHaveBeenCalledTimes(2);expect(state.submittedTasks[0]).toMatchObject({phase:'succeeded',chunks:2,durationMs:13});expect(state.selectedHostJob.status).toBe('succeeded');app.unmount()
})
it('does not render cached success when the detail request is rejected',async()=>{
 const get=vi.fn().mockResolvedValueOnce({job_id:job,space_id:space,status:'succeeded'}).mockRejectedValueOnce({data:{code:'KNOWLEDGE_FORBIDDEN',trace_id:'fixture-trace'}})
 const {app,state}=await mount(get);await state.openHostJob(job);expect(state.selectedHostJob).toBeUndefined();expect(state.hostDetailError).toContain('fixture-trace');expect(state.hostDetailError).toContain('knowledgeLab.workspace.requestFailed');app.unmount()
})
it('keeps a failed initial read explicit rather than displaying fabricated empty history',async()=>{
 const get=vi.fn().mockRejectedValue({data:{code:'KNOWLEDGE_FORBIDDEN',trace_id:'fixture-trace'}})
 const {app,state}=await mount(get);expect(state.loadError).toContain('fixture-trace');expect(state.submittedJobs).toEqual([]);expect(get).toHaveBeenCalledTimes(1);app.unmount()
})
