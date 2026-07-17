package agent

import "time"

const (
	EventIntent                 = "intent"
	EventPlan                   = "plan"
	EventNodeStart              = "node_start"
	EventNodeEnd                = "node_end"
	EventToken                  = "token"
	EventFinal                  = "final"
	EventEnd                    = "end"
	EventError                  = "error"
	EventAgentRunStarted        = "agent_run.started"
	EventAgentRunResponsePlan   = "agent_run.response_plan"
	EventAgentRunIntentDetected = "agent_run.intent_detected"
	EventAgentRunPlanCreated    = "agent_run.plan_created"
	EventAgentRunTaskStatus     = "agent_run.task_status"
	EventAgentRunTaskStarted    = "agent_run.task_started"
	EventAgentRunAwaitingParams = "agent_run.awaiting_params"
	EventAgentRunTaskCompleted  = "agent_run.task_completed"
	EventAgentRunTaskFailed     = "agent_run.task_failed"
	EventAgentRunFinal          = "agent_run.final"
	EventAgentRunEnded          = "agent_run.ended"
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
	case EventIntent, EventPlan, EventNodeStart, EventNodeEnd, EventToken, EventFinal, EventEnd, EventError,
		EventAgentRunStarted, EventAgentRunResponsePlan, EventAgentRunIntentDetected, EventAgentRunPlanCreated,
		EventAgentRunTaskStatus, EventAgentRunTaskStarted, EventAgentRunAwaitingParams, EventAgentRunTaskCompleted,
		EventAgentRunTaskFailed, EventAgentRunFinal, EventAgentRunEnded:
		return true
	default:
		return false
	}
}
