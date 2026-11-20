import { apiGet, apiPost, apiDel } from "./_client";
import { createFetchSSE, type SSEStreamEvent } from "./useStream";

export interface AgentSession { session_uuid: string; agent_uuid: string; title: string; status: string; created_at: string; updated_at: string; revision: number }
export interface AgentMessage { message_uuid: string; session_uuid: string; role: "user" | "assistant"; content: string; sequence: number; created_at: string }
export interface AgentInvocation { invocation_uuid: string; session_uuid: string; message_uuid: string; trace_uuid: string; status: "running" | "cancelling" | "succeeded" | "failed" | "cancelled"; output: string; reason_code: string }
export interface AgentAttempt { session: string; content: string; appendKey: string; invokeKey: string; messageUUID?: string; invocationUUID?: string }
type Envelope<T> = { success: boolean; data: T };
type Page<T> = { items: T[]; total: number; page: number; page_size: number };
const root = "/plugin/agent/sessions";
function path(id: string) { if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(id) || /^0{8}-0{4}-0{4}-0{4}-0{12}$/.test(id)) throw new Error("AGENT_SESSION_INVALID_ARGUMENT"); return `${root}/${id}`; }
function data<T>(response: Envelope<T>): T { if (!response || response.success !== true || response.data == null) throw new Error("AGENT_SESSION_UPSTREAM_DEPENDENCY"); return response.data; }

export function useAgentSessionsApi() {
  async function listAll<T>(url: string): Promise<T[]> {
    const items: T[] = [];
    for (let page = 1; page <= 1000000; page++) {
      const result = data(await apiGet<Envelope<Page<T>>>(url, { page, page_size: 100 }));
      if (!Array.isArray(result.items) || result.page !== page || result.page_size !== 100 || !Number.isInteger(result.total) || result.total < 0) throw new Error("AGENT_SESSION_UPSTREAM_DEPENDENCY");
      items.push(...result.items);
      if (items.length >= result.total) return items;
      if (!result.items.length) throw new Error("AGENT_SESSION_UPSTREAM_DEPENDENCY");
    }
    throw new Error("AGENT_SESSION_UPSTREAM_DEPENDENCY");
  }
  return {
    async create(agent_uuid: string, title: string) { path(agent_uuid); return data(await apiPost<Envelope<AgentSession>>(root, { agent_uuid, title })); },
    list: () => listAll<AgentSession>(root),
    messages: (session: string) => listAll<AgentMessage>(`${path(session)}/messages`),
    async remove(session: string) { return data(await apiDel<Envelope<AgentSession>>(path(session))); },
    // Caller retains attempt across explicit retries. Never generate replacement
    // keys or append another user message because Invoke/stream disconnected.
    async submit(attempt: AgentAttempt): Promise<AgentInvocation> {
      const url = path(attempt.session);
      if (!attempt.messageUUID) {
        const message = data(await apiPost<Envelope<AgentMessage>>(`${url}/messages`, { role: "user", content: attempt.content }, { headers: { "Idempotency-Key": attempt.appendKey } }));
        path(message.message_uuid); attempt.messageUUID = message.message_uuid;
      }
      if (attempt.invocationUUID) return data(await apiGet<Envelope<AgentInvocation>>(`${url}/invocations/${attempt.invocationUUID}`));
      const invocation = data(await apiPost<Envelope<AgentInvocation>>(`${url}/invocations`, { message_uuid: attempt.messageUUID }, { headers: { "Idempotency-Key": attempt.invokeKey } }));
      path(invocation.invocation_uuid); attempt.invocationUUID = invocation.invocation_uuid;
      return invocation;
    },
    async cancel(session: string, invocation: string) { path(invocation); return data(await apiPost<Envelope<AgentInvocation>>(`${path(session)}/invocations/${invocation}/cancel`)); },
    async subscribe(session: string, invocation: string, signal: AbortSignal, onEvent: (event: SSEStreamEvent) => void, headers: Record<string,string> = {}) {
      path(invocation); let ended = false; let final = false; let failure = "";
      await createFetchSSE({ path: `${path(session)}/invocations/${invocation}/events`, signal, headers, onEvent(event) {
        const payload = event.payload as Record<string, unknown>;
        if (!payload || typeof payload !== "object") throw new Error("AGENT_SESSION_UPSTREAM_DEPENDENCY");
        switch (event.event) {
          case "state": case "final":
            if (payload.session_uuid !== session || payload.invocation_uuid !== invocation) throw new Error("AGENT_SESSION_UPSTREAM_DEPENDENCY");
            if (event.event === "final") { if (payload.status !== "succeeded" || typeof payload.output !== "string") throw new Error("AGENT_SESSION_UPSTREAM_DEPENDENCY"); final = true; }
            break;
          case "error": failure = String(payload.reason_code || "AGENT_SESSION_UPSTREAM_DEPENDENCY"); break;
          case "end": if (!['succeeded','failed','cancelled'].includes(String(payload.status)) || (payload.status === "succeeded" && !final) || (payload.status !== "succeeded" && !failure)) throw new Error("AGENT_SESSION_STREAM_INTERRUPTED"); ended = true; break;
          default: throw new Error("AGENT_SESSION_UPSTREAM_DEPENDENCY");
        }
        onEvent(event);
      }});
      if (failure) throw new Error(failure);
      if (!ended) throw new Error("AGENT_SESSION_STREAM_INTERRUPTED");
    },
  };
}
