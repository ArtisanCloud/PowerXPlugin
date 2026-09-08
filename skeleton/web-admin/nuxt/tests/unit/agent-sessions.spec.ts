import { beforeEach, describe, expect, it, vi } from "vitest";
const mocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn(), del: vi.fn(), stream: vi.fn() }));
vi.mock("../../app/composables/api/_client", () => ({ apiGet: mocks.get, apiPost: mocks.post, apiDel: mocks.del }));
vi.mock("../../app/composables/api/useStream", () => ({ createFetchSSE: mocks.stream }));
import { useAgentSessionsApi, type AgentAttempt } from "../../app/composables/api/useAgentSessions";
const session="12345678-1234-4234-8234-123456789abc", message="12345678-1234-4234-8234-123456789abd", invocation="12345678-1234-4234-8234-123456789abe";
const attempt = (): AgentAttempt => ({ session, content:"test.input", appendKey:"append-key", invokeKey:"invoke-key" });
describe("service sessions",()=>{
 beforeEach(()=>vi.resetAllMocks());
 it("keeps the same caller keys and does not append again when invoke failed",async()=>{
  const api=useAgentSessionsApi(), a=attempt();mocks.post.mockResolvedValueOnce({success:true,data:{message_uuid:message}}).mockRejectedValueOnce(new Error("AGENT_SESSION_UPSTREAM_DEPENDENCY"));
  await expect(api.submit(a)).rejects.toThrow("AGENT_SESSION_UPSTREAM_DEPENDENCY");
  expect(a.messageUUID).toBe(message);
  mocks.post.mockResolvedValueOnce({success:true,data:{invocation_uuid:invocation}});await api.submit(a);
  expect(mocks.post).toHaveBeenNthCalledWith(1,`/plugin/agent/sessions/${session}/messages`,{role:"user",content:"test.input"},{headers:{"Idempotency-Key":"append-key"}});
  expect(mocks.post.mock.calls[1]).toEqual(mocks.post.mock.calls[2]);
  expect(a.invocationUUID).toBe(invocation);
  mocks.get.mockResolvedValueOnce({success:true,data:{invocation_uuid:invocation}});await api.submit(a);expect(mocks.post).toHaveBeenCalledTimes(3);
 });
 it("uses only UUID contract and explicit cancel",async()=>{
  mocks.post.mockResolvedValue({success:true,data:{session_uuid:session}});const api=useAgentSessionsApi();await api.create(session,"test.title");expect(mocks.post.mock.calls[0][1]).toEqual({agent_uuid:session,title:"test.title"});
  await api.cancel(session,invocation);expect(mocks.post).toHaveBeenLastCalledWith(`/plugin/agent/sessions/${session}/invocations/${invocation}/cancel`);
  await expect(api.create("123","test.title")).rejects.toThrow("AGENT_SESSION_INVALID_ARGUMENT");
 });
 it("lists explicit pages without agent or tenant overrides",async()=>{
  mocks.get.mockResolvedValueOnce({success:true,data:{items:[{session_uuid:session}],total:2,page:1,page_size:100}}).mockResolvedValueOnce({success:true,data:{items:[{session_uuid:message}],total:2,page:2,page_size:100}});
  const result=await useAgentSessionsApi().list();expect(result).toHaveLength(2);expect(mocks.get.mock.calls[1][1]).toEqual({page:2,page_size:100});
 });
 it("subscribes without invoking and requires final/end",async()=>{
  const api=useAgentSessionsApi(), signal=new AbortController().signal;
  mocks.stream.mockImplementation(async({onEvent}:any)=>{onEvent({event:"final",payload:{session_uuid:session,invocation_uuid:invocation,status:"succeeded",output:"test.output"}});onEvent({event:"end",payload:{status:"succeeded"}})});
  await api.subscribe(session,invocation,signal,()=>{});expect(mocks.post).not.toHaveBeenCalled();expect(mocks.stream.mock.calls[0][0].params).toBeUndefined();
  mocks.stream.mockResolvedValue(undefined);await expect(api.subscribe(session,invocation,signal,()=>{})).rejects.toThrow("AGENT_SESSION_STREAM_INTERRUPTED");
 });
 it("does not turn an error/end into success or cancel on disconnect",async()=>{
  mocks.stream.mockImplementation(async({onEvent}:any)=>{onEvent({event:"error",payload:{reason_code:"AGENT_SESSION_FORBIDDEN"}});onEvent({event:"end",payload:{status:"failed"}})});
  await expect(useAgentSessionsApi().subscribe(session,invocation,new AbortController().signal,()=>{})).rejects.toThrow("AGENT_SESSION_FORBIDDEN");
  mocks.stream.mockRejectedValue(new DOMException("test.abort","AbortError"));await expect(useAgentSessionsApi().subscribe(session,invocation,new AbortController().signal,()=>{})).rejects.toMatchObject({name:"AbortError"});expect(mocks.post).not.toHaveBeenCalled();
 });
});
