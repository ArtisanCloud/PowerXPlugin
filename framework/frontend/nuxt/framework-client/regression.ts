export interface SuiteDefinition {
 uuid: string;
 title_key: string;
 description_key: string;
 key: string;
}
export interface StepDefinition {
 uuid: string;
 key: string;
 title_key: string;
 phase: "preflight" | "prepare" | "execute" | "verify" | "cleanup";
 boundary: string;
}
export interface CaseDefinition {
 uuid: string;
 suite_uuid: string;
 key: string;
 version: string;
 title_key: string;
 description_key: string;
 tags: Array<string>;
 supported_profile_kinds: Array<"controlled" | "live">;
 input_schema: Record<string, unknown>;
 execution_timeout_ms: number;
 cleanup_timeout_ms: number;
 steps: Array<StepDefinition>;
 required_assertion_keys: Array<string>;
}
export interface ProfileDefinition {
 uuid: string;
 key: string;
 title_key: string;
 kind: "controlled" | "live";
 dependency_declarations: Array<DependencyDeclaration>;
 fixture_namespace: string;
 binding_revision: string;
}
export interface DependencyEvidence {
 dependency_key: string;
 declared_mode: "real" | "test_double";
 actual_mode: "real" | "test_double" | "unavailable";
 boundary: string;
 reason_code: string;
}
export interface Evidence {
 kind: string;
 boundary: string;
 source_ref: string;
 observed_at: string;
 sanitized_data: unknown;
 resource_uuid?: string;
 trace_id?: string;
 request_id?: string;
}
export interface AssertionResult {
 uuid: string;
 key: string;
 title_key: string;
 status: "passed" | "failed" | "blocked" | "skipped";
 required: boolean;
 expected: unknown;
 actual: unknown;
 evidence: Array<Evidence>;
 reason_code?: string;
}
export interface StepResult {
 uuid: string;
 step_uuid: string;
 phase: "preflight" | "prepare" | "execute" | "verify" | "cleanup";
 title_key: string;
 status: "pending" | "running" | "passed" | "failed" | "blocked" | "skipped";
 started_at: string | null;
 finished_at: string | null;
 assertions: Array<AssertionResult>;
}
export interface FixtureResource {
 uuid: string;
 resource_type: string;
 resource_uuid: string;
 display_name: string;
 role_key: string;
 parent_resource_uuid: string;
 cleanup_evidence: Array<Evidence>;
 cleaned_at: string | null;
 tenant_uuid: string;
 run_uuid: string;
 case_execution_uuid: string;
 cleanup_key: string;
 cleanup_parameters: Record<string, unknown>;
 cleanup_state: "pending" | "running" | "succeeded" | "failed" | "not_required";
}
export interface Run {
 uuid: string;
 tenant_uuid: string;
 actor_uuid: string;
 profile_uuid: string;
 client_request_uuid: string;
 rerun_of_run_uuid: string | null;
 profile_snapshot: ProfileDefinition;
 selection_snapshot: Array<CaseDefinition>;
 input_snapshot: Record<string, unknown>;
 source_snapshot: Record<string, unknown>;
 phase: "queued" | "running" | "cancelling" | "completed" | "cancelled" | "interrupted";
 verdict: "pending" | "passed" | "failed" | "blocked";
 cleanup_state: "pending" | "running" | "succeeded" | "failed" | "not_required";
 reason_code: string;
 message_key: string;
 revision: number;
 created_at: string;
 updated_at: string;
 started_at: string | null;
 finished_at: string | null;
}
export interface CaseExecution {
 uuid: string;
 tenant_uuid: string;
 run_uuid: string;
 case_uuid: string;
 ordinal: number;
 definition_snapshot: CaseDefinition;
 input_snapshot: Record<string, unknown>;
 dependency_evidence: Array<DependencyEvidence>;
 phase: "pending" | "preflight" | "preparing" | "executing" | "verifying" | "cleaning" | "finished";
 verdict: "pending" | "passed" | "failed" | "blocked" | "skipped";
 steps: Array<StepResult>;
 fixture_resources: Array<FixtureResource>;
 cleanup_state: "pending" | "running" | "succeeded" | "failed" | "not_required";
 reason_code: string;
 message_key: string;
 revision: number;
 created_at: string;
 updated_at: string;
 started_at: string | null;
 finished_at: string | null;
 missing_assertion_keys: Array<string>;
}
export interface RegressionEvent {
 uuid: string;
 tenant_uuid: string;
 run_uuid: string;
 case_execution_uuid: string | null;
 sequence: number;
 revision: number;
 kind: "run.created" | "run.updated" | "case.updated" | "assertion.recorded" | "cleanup.updated" | "run.finished";
 payload: RunEventPayload | CaseEventPayload | AssertionEventPayload;
 created_at: string;
 updated_at: string;
 meta: EventMeta;
}
export interface RunDetail {
 run: Run;
 case_executions: Array<CaseExecution>;
 last_event_uuid: string | null;
}
export interface Report {
 schema_version: "v1";
 run: Run;
 case_executions: Array<CaseExecution>;
 events: Array<RegressionEvent>;
 exported_at: string;
}
export interface CreateRun {
 client_request_uuid: string;
 profile_uuid: string;
 case_uuids: Array<string>;
 inputs?: Record<string, unknown>;
 rerun_of_run_uuid?: string;
}
export interface SSEEnvelope {
 protocol: "sse";
 channel: string;
 event_type: "powerx.regression.run.updated.v1";
 payload: RegressionEvent;
 plugin_id: string;
 tenant_uuid: string;
 member_uuid: string;
 trace_id: string;
 request_id: string;
 timestamp: string;
}
export interface DependencyDeclaration {
 dependency_key: string;
 mode: "real" | "test_double";
 boundary: string;
 required_capabilities: Array<string>;
}
export interface RunEventPayload {
 run: Run;
}
export interface CaseEventPayload {
 run: Run;
 case_execution: CaseExecution;
}
export interface AssertionEventPayload {
 run: Run;
 case_execution: CaseExecution;
 step_result_uuid: string;
 assertion: AssertionResult;
}
export interface EventMeta {
 tenant_uuid: string;
 request_id: string;
 trace_id: string;
 source_plugin: string;
 occurred_at: string;
 payload_version: "v1";
}


import type { RealtimeClient } from "./realtime";
export interface RegressionClientOptions { pluginId: string; request: <T>(path:string,options?:Record<string,unknown>)=>Promise<T>; realtime: RealtimeClient }
const base="/admin/regression";
export const regressionUUID=(value:unknown):value is string=>typeof value==="string"&&/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(value);
const invalid=()=>new Error("REGRESSION_EVENT_META_INVALID");
export function validateRegressionEvent(value:unknown,detail:RunDetail,pluginId:string):SSEEnvelope {
 const e=value as SSEEnvelope;
 if(!e||e.protocol!=="sse"||e.event_type!=="powerx.regression.run.updated.v1"||e.plugin_id!==pluginId||e.tenant_uuid!==detail.run.tenant_uuid)throw invalid();
 const p=e.payload,m=p?.meta;
 if(!p||!regressionUUID(p.uuid)||p.run_uuid!==detail.run.uuid||p.tenant_uuid!==detail.run.tenant_uuid||!Number.isInteger(p.sequence)||!Number.isInteger(p.revision)||p.sequence!==p.revision||p.revision<1||!m||m.tenant_uuid!==p.tenant_uuid||m.source_plugin!==pluginId||m.payload_version!=="v1"||!m.request_id?.trim()||!m.trace_id?.trim()||!m.occurred_at?.endsWith("Z")||Number.isNaN(Date.parse(m.occurred_at))||e.request_id!==m.request_id||e.trace_id!==m.trace_id)throw invalid();
 const payload=p.payload as AssertionEventPayload;
 if(!payload?.run||payload.run.uuid!==detail.run.uuid||payload.run.tenant_uuid!==detail.run.tenant_uuid||payload.run.revision!==p.revision)throw invalid();
 if(["case.updated","cleanup.updated","assertion.recorded"].includes(p.kind)){
  if(!payload.case_execution||payload.case_execution.run_uuid!==p.run_uuid||payload.case_execution.tenant_uuid!==p.tenant_uuid||payload.case_execution.uuid!==p.case_execution_uuid)throw invalid();
 }
 if(p.kind==="assertion.recorded"&&(!payload.assertion||!regressionUUID(payload.step_result_uuid)))throw invalid();
 if(!["run.created","run.updated","run.finished","case.updated","cleanup.updated","assertion.recorded"].includes(p.kind))throw invalid();
 return e;
}
export function applyRegressionEvent(detail:RunDetail,event:SSEEnvelope):RunDetail{
 const p=event.payload,body=p.payload as CaseEventPayload;
 if(p.revision<=detail.run.revision)return detail;
 const cases=[...detail.case_executions];
 if(body.case_execution){const index=cases.findIndex(c=>c.uuid===body.case_execution.uuid);if(index<0)throw invalid();cases[index]=body.case_execution}
 return {run:body.run,case_executions:cases,last_event_uuid:p.uuid};
}
export const regressionTerminal=(phase:string)=>["completed","cancelled","interrupted"].includes(phase);
export function createRegressionClient(options:RegressionClientOptions){
 const read=async<T>(path:string,init?:Record<string,unknown>):Promise<T>=>{const response=await options.request<{success:boolean;data:T;error?:{code:string}}>(path,init);if(response?.success!==true||response.data===undefined)throw new Error(response?.error?.code||"REGRESSION_INVALID_REQUEST");return response.data};
 return {
  suites:()=>read<{items:SuiteDefinition[]}>(base+"/suites"),
  cases:(query:Record<string,string>={})=>read<{items:CaseDefinition[]}>(base+"/cases",{query}),
  profiles:()=>read<{items:ProfileDefinition[]}>(base+"/profiles"),
  runs:(page=1)=>read<{items:Run[];total:number}>(base+"/runs",{query:{page,page_size:20}}),
  get:(id:string)=>{if(!regressionUUID(id))throw invalid();return read<RunDetail>(base+"/runs/"+id)},
  create:(body:CreateRun)=>read<RunDetail>(base+"/runs",{method:"POST",body}),
  cancel:(id:string)=>read<RunDetail>(base+"/runs/"+id+"/cancel",{method:"POST"}),
  cleanup:(id:string)=>read<RunDetail>(base+"/runs/"+id+"/cleanup",{method:"POST"}),
  report:(id:string)=>options.request<Report>(base+"/runs/"+id+"/report"),
  async watch(initial:RunDetail,signal:AbortSignal,onSnapshot:(snapshot:RunDetail)=>void){
   let current=initial;
   await options.realtime.streamSSE({path:base+"/runs/"+initial.run.uuid+"/events",params:initial.last_event_uuid?{after_event_uuid:initial.last_event_uuid}:{},signal,onEvent(event){const typed=validateRegressionEvent(event.payload,current,options.pluginId);current=applyRegressionEvent(current,typed);onSnapshot(current)}});
   if(!signal.aborted&&(!regressionTerminal(current.run.phase)||current.run.cleanup_state==='running'))throw new Error("REGRESSION_STREAM_INTERRUPTED");
  }
 };
}
export type RegressionClient=ReturnType<typeof createRegressionClient>;
