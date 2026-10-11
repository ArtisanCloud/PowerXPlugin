import { afterEach,expect,it,vi } from 'vitest'
import {readFileSync} from 'node:fs'
import {createRequire} from 'node:module'
import {webcrypto} from 'node:crypto'
import {compileScript,parse} from '@vue/compiler-sfc'
import {transformSync} from 'esbuild'
import {createRenderer,nextTick} from 'vue'
const native=createRequire(import.meta.url)
function loadModule(path:string){const m={exports:{} as any};const code=transformSync(readFileSync(new URL(path,import.meta.url),'utf8'),{loader:'ts',format:'cjs'}).code;new Function('require','module','exports',code)(native,m,m.exports);return m.exports}
const renderer=createRenderer<any,any>({patchProp(){},insert(){},remove(){},createElement:()=>({}),createText:()=>({}),createComment:()=>({}),setText(){},setElementText(){},parentNode:()=>null,nextSibling:()=>null})
const space='11111111-1111-4111-8111-111111111111';const generation='22222222-2222-4222-8222-222222222222'
afterEach(()=>vi.unstubAllGlobals())
it('bundle checksums and offsets use the exact UTF-8 text and Unicode codepoints',async()=>{
 vi.stubGlobal('crypto',webcrypto)
 const helpers=loadModule('../../app/composables/knowledge/semanticBundle.ts')
 const draft=helpers.newSemanticDraft();const indexing={mode:'hybrid',embedding_profile:{uuid:space,version:7},artifact_roles:['source_chunk']}
 const text='中😀é\n';const profile='画像😀';const ingestion={chunk_overlap:0,page_priority:false,separators:[]}
 const first=await helpers.semanticBundle(draft,text,indexing,profile,ingestion,'raw_document','fixture',['category'],[])
 expect(first.artifacts[0].source_ref.char_end).toBe(Array.from(text).length)
 expect(first.artifacts[1].source_ref.char_end).toBe(Array.from(profile).length)
 expect(first.artifacts[1].source_ref.source_uuid).toBe(first.artifacts[1].knowledge_profile_uuid)
 expect(first.artifacts[0].checksum).toMatch(/^[a-f0-9]{64}$/)
 expect(first.artifacts[0].text).toBe(text)
 expect(first.indexing.artifact_roles).toEqual(['source_chunk','knowledge_profile'])
 const same=await helpers.semanticBundle(draft,text,indexing,profile,ingestion,'raw_document','fixture',['category'],[])
 expect(same.idempotency_key).toBe(first.idempotency_key)
 const changed=await helpers.semanticBundle(draft,text,indexing,profile,{...ingestion,chunk_size:640},'raw_document','fixture',['category'],[])
 expect(changed.idempotency_key).not.toBe(first.idempotency_key)
 expect(changed.external_ref).toEqual(first.external_ref)
})
async function mount(workspace:any){
 vi.stubGlobal('useI18n',()=>({t:(key:string)=>key}));vi.stubGlobal('useRoute',()=>({params:{uuid:space}}))
 const {descriptor}=parse(readFileSync(new URL('../../app/components/knowledge/KnowledgeSearchWorkspace.vue',import.meta.url),'utf8'))
 const code=transformSync(compileScript(descriptor,{id:'semantic-search'}).content,{loader:'ts',format:'cjs'}).code;const m={exports:{} as any};new Function('require','module','exports',code)((id:string)=>id==='vue'?native(id):id.endsWith('requestFailure')?loadModule('../../app/composables/knowledge/requestFailure.ts'):{useKnowledgeWorkspace:()=>workspace},m,m.exports)
 const app=renderer.createApp({...m.exports.default,render:()=>null});app.mount({});for(let n=0;n<8;n++)await nextTick();return {app,state:app._instance!.setupState}
}
it('semantic search preserves the full query, typed filters and generations without a lexical fallback',async()=>{
 const workspace={basePath:'/powerx/knowledge-lab',api:{spaces:vi.fn().mockResolvedValue({items:[{uuid:space,name:'fixture'}]})},semanticQuery:vi.fn().mockResolvedValue({items:[{chunk_uuid:'a'},{chunk_uuid:'b'}]}),search:vi.fn()}
 const {app,state}=await mount(workspace)
 state.setCapabilities({configured:true,ready:true,indexed:true,indexed_query_modes:['semantic','hybrid'],generation:{corpus_generation:generation}})
 const text='  用自然语言描述需要应对的声誉问题😀。  ';state.query=text;state.filters.category_codes=['public_relations'];state.filters.artifact_roles=['knowledge_profile','source_chunk']
 await state.search();expect(workspace.semanticQuery).toHaveBeenCalledWith(expect.objectContaining({query:text,mode:'semantic',filters:{category_codes:['public_relations'],artifact_roles:['knowledge_profile','source_chunk'],document_uuids:[],tag_uuids:[]},required_generations:{[space]:generation}}))
 expect(state.semanticResult.items).toHaveLength(2)
 workspace.semanticQuery.mockRejectedValue({data:{code:'KNOWLEDGE_FORBIDDEN'}});await state.search();expect(state.semanticResult).toBeUndefined();expect(workspace.search).not.toHaveBeenCalled();expect(state.error).toBeTruthy();app.unmount()
})

it('keeps semantic search blocked until the model and published index are ready',async()=>{
 const workspace={basePath:'/powerx/knowledge-lab',api:{spaces:vi.fn().mockResolvedValue({items:[{uuid:space,name:'fixture'}]})},semanticQuery:vi.fn(),search:vi.fn()}
 const {app,state}=await mount(workspace);state.query='fixture'
 for(const ready of [{configured:false,ready:false,indexed:false},{configured:true,ready:true,indexed:false}]){state.setCapabilities({...ready,indexed_query_modes:[]});expect(state.canSearch).toBe(false);await state.search()}
 expect(workspace.semanticQuery).not.toHaveBeenCalled();expect(workspace.search).not.toHaveBeenCalled();app.unmount()
})
it('loads readiness while index settings are closed and uses the default semantic query',async()=>{
 const capabilities={configured:true,ready:true,indexed:true,indexed_query_modes:['semantic','hybrid'],generation:{corpus_generation:generation}}
 const workspace={api:{spaces:vi.fn().mockResolvedValue({items:[{uuid:space,name:'fixture'}]})},semanticCapabilities:vi.fn().mockResolvedValue(capabilities),semanticQuery:vi.fn().mockResolvedValue({items:[]}),search:vi.fn()}
 const {app,state}=await mount(workspace)
 expect(state.indexSettingsOpen).toBe(false);expect(workspace.semanticCapabilities).toHaveBeenCalledExactlyOnceWith(space)
 state.selectQuestion('fixture-question');expect(state.query).toBe('fixture-question');expect(workspace.semanticQuery).not.toHaveBeenCalled();state.loading=true;state.selectQuestion('other-question');expect(state.query).toBe('fixture-question');state.loading=false
 expect(state.canSearch).toBe(true);await state.search()
 expect(workspace.semanticQuery).toHaveBeenCalledWith(expect.objectContaining({mode:'semantic',top_k:5,required_generations:{[space]:generation}}));expect(workspace.search).not.toHaveBeenCalled();app.unmount()
})
it('blocks a failed readiness request and preserves the question when refreshing',async()=>{
 const service=vi.fn().mockRejectedValue({statusCode:503,data:{code:'KNOWLEDGE_UPSTREAM_DEPENDENCY'}})
 const workspace={api:{spaces:vi.fn().mockResolvedValue({items:[{uuid:space,name:'fixture'}]})},semanticCapabilities:service,semanticQuery:vi.fn(),search:vi.fn()}
 const {app,state}=await mount(workspace);state.query='fixture'
 expect(state.indexError).toBeTruthy();expect(state.canSearch).toBe(false);await state.search();expect(workspace.search).not.toHaveBeenCalled()
 service.mockResolvedValue({configured:true,ready:true,indexed:true,indexed_query_modes:['semantic']});await state.loadCapabilities()
 expect(state.query).toBe('fixture');expect(state.indexError).toBe('');expect(state.canSearch).toBe(true);app.unmount()
})
async function mountPanel(service:any,props:any={}){
 vi.stubGlobal('useI18n',()=>({t:(key:string)=>key}))
 const {descriptor}=parse(readFileSync(new URL('../../app/components/knowledge/KnowledgeSemanticIndexPanel.vue',import.meta.url),'utf8'))
 const code=transformSync(compileScript(descriptor,{id:'semantic-panel'}).content,{loader:'ts',format:'cjs'}).code;const m={exports:{} as any}
 new Function('require','module','exports',code)((id:string)=>id==='vue'?native(id):id.endsWith('requestFailure')?loadModule('../../app/composables/knowledge/requestFailure.ts'):{useKnowledgeWorkspace:()=>({semanticCapabilities:service})},m,m.exports)
 const app=renderer.createApp({...m.exports.default,render:()=>null},{spaceUUID:space,...props});app.mount({});for(let n=0;n<8;n++)await nextTick();return {app,state:app._instance!.setupState}
}
it('opens settings with the existing capabilities without invalidating search results',async()=>{
 const initialState={configured:true,ready:true,indexed:true,generation:{model_key:'fixture'}};const changed=vi.fn();const failed=vi.fn();const service=vi.fn().mockRejectedValue({statusCode:503,data:{code:'KNOWLEDGE_UPSTREAM_DEPENDENCY'}})
 const {app,state}=await mountPanel(service,{initialState,onChanged:changed,onFailed:failed})
 expect(state.state).toEqual(initialState);expect(state.modelKey).toBe('fixture');expect(service).not.toHaveBeenCalled();expect(changed).not.toHaveBeenCalled()
 await state.load();expect(changed).toHaveBeenCalledWith(undefined);expect(failed).toHaveBeenCalledWith(state.error);expect(state.error).toBeTruthy();app.unmount()
})
it('treats a successful unconfigured capability response as readiness data rather than a request error',async()=>{
 const stateValue={configured:false,ready:false,indexed:false,reason_code:'KNOWLEDGE_SEMANTIC_INDEX_NOT_READY'}
 const {app,state}=await mountPanel(vi.fn().mockResolvedValue(stateValue));expect(state.error).toBe('');expect(state.state).toEqual(stateValue);app.unmount()
})
it('shows an unavailable route explicitly without blaming Core dependencies',async()=>{
 const {app,state}=await mountPanel(vi.fn().mockRejectedValue({statusCode:404,data:'404 page not found'}));expect(state.error).toBe('knowledgeSemantic.errors.routeUnavailable');expect(state.state).toBeUndefined();app.unmount()
})
