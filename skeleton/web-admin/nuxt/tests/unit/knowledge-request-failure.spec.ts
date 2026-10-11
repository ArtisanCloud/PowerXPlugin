import {expect,it} from 'vitest'
import {knowledgeRequestFailure,formatKnowledgeRequestFailure} from '../../app/composables/knowledge/requestFailure'
it('reports a missing route as HTTP 404 rather than inventing an upstream dependency failure',()=>{
 expect(knowledgeRequestFailure({statusCode:404,data:'404 page not found'})).toEqual({key:'knowledgeSemantic.errors.routeUnavailable',params:{status:404},traceId:undefined})
})
it('preserves Core machine codes, actual HTTP status and trace while handling structured errors',()=>{
 const value=knowledgeRequestFailure({response:{status:503,_data:{code:503,reason_code:'KNOWLEDGE_VECTOR_CHANNEL_UNAVAILABLE',request_id:'fixture'}}})
 expect(value).toEqual({key:'knowledgeLab.workspace.requestFailed',params:{code:'KNOWLEDGE_VECTOR_CHANNEL_UNAVAILABLE'},traceId:'fixture'})
 expect(formatKnowledgeRequestFailure({data:{code:'KNOWLEDGE_FORBIDDEN',trace_id:'fixture'}},key=>key)).toBe('knowledgeLab.workspace.requestFailed (fixture)')
})
it('keeps unstructured failures explicit without parsing a free-text body or assigning a machine code',()=>{
 expect(knowledgeRequestFailure({response:{status:502,_data:'unstructured-body'}})).toMatchObject({key:'knowledgeSemantic.errors.httpFailed',params:{status:502}})
 expect(knowledgeRequestFailure({data:'unstructured-body'})).toMatchObject({key:'knowledgeSemantic.errors.requestFailed',params:{}})
})
