package agent

type AgentInvokeRequest struct {
	AgentID   string         `json:"agent_id"`
	SessionID string         `json:"session_id,omitempty"`
	Message   string         `json:"message"`
	TraceID   string         `json:"trace_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type AgentInvokeResponse struct {
	SessionID string         `json:"session_id,omitempty"`
	TraceID   string         `json:"trace_id,omitempty"`
	Message   string         `json:"message,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}
