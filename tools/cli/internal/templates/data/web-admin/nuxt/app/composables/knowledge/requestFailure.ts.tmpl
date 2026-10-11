type FailureInfo = { key: string; params: Record<string,string|number>; traceId?: string }
function record(value:unknown):Record<string,unknown>|undefined {return value !== null && typeof value==='object' ? value as Record<string,unknown> : undefined}
export function knowledgeRequestFailure(error:unknown):FailureInfo {
 const value=record(error), response=record(value?.response)
 const payload=record(value?.data)||record(response?._data)
 const nested=record(payload?.error)
 const rawCode=payload?.reason_code || payload?.error_code || nested?.code || payload?.code
 const code=typeof rawCode==='string' ? rawCode.trim() : undefined
 const status=Number(value?.statusCode || response?.status || value?.status)
 const rawTrace=payload?.trace_id || payload?.request_id
 const traceId=typeof rawTrace==='string' ? rawTrace : undefined
 if(code)return {key:'knowledgeLab.workspace.requestFailed',params:{code},traceId}
 if(status===404)return {key:'knowledgeSemantic.errors.routeUnavailable',params:{status},traceId}
 if(Number.isInteger(status)&&status>=400&&status<=599)return {key:'knowledgeSemantic.errors.httpFailed',params:{status},traceId}
 return {key:'knowledgeSemantic.errors.requestFailed',params:{},traceId}
}
export function formatKnowledgeRequestFailure(error:unknown,t:(key:string,params?:Record<string,string|number>)=>string):string {
 const failure=knowledgeRequestFailure(error)
 return t(failure.key,failure.params)+(failure.traceId?` (${failure.traceId})`:'')
}
