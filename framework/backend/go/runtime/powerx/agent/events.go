package agent

import "time"

const (
	EventIntent    = "intent"
	EventPlan      = "plan"
	EventNodeStart = "node_start"
	EventNodeEnd   = "node_end"
	EventToken     = "token"
	EventFinal     = "final"
	EventEnd       = "end"
	EventError     = "error"
)

type AgentStreamEvent struct {
	Type      string         `json:"type"`
	TraceID   string         `json:"trace_id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	PlanID    string         `json:"plan_id,omitempty"`
	NodeID    string         `json:"node_id,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"created_at,omitempty"`
}

func IsKnownEventType(t string) bool {
	switch t {
	case EventIntent, EventPlan, EventNodeStart, EventNodeEnd, EventToken, EventFinal, EventEnd, EventError:
		return true
	default:
		return false
	}
}
