package agent

import (
	"fmt"
	"strings"
)

type AgentRunState struct {
	Run           map[string]any   `json:"run,omitempty"`
	ResponsePlan  map[string]any   `json:"response_plan,omitempty"`
	Intent        map[string]any   `json:"intent,omitempty"`
	Plan          map[string]any   `json:"plan,omitempty"`
	Tasks         []AgentTaskState `json:"tasks,omitempty"`
	PendingParams []AgentTaskState `json:"pending_params,omitempty"`
	Results       []AgentTaskState `json:"results,omitempty"`
	Errors        []AgentTaskState `json:"errors,omitempty"`
	TraceLinks    []map[string]any `json:"trace_links,omitempty"`
	Ended         bool             `json:"ended,omitempty"`
}

type AgentTaskState struct {
	RunID         string           `json:"run_id,omitempty"`
	SessionID     string           `json:"session_id,omitempty"`
	MessageID     string           `json:"message_id,omitempty"`
	TraceID       string           `json:"trace_id,omitempty"`
	TaskID        string           `json:"task_id,omitempty"`
	TeamID        string           `json:"team_id,omitempty"`
	AgentID       string           `json:"agent_id,omitempty"`
	AgentName     string           `json:"agent_name,omitempty"`
	NodeKind      string           `json:"node_kind,omitempty"`
	NodeRef       string           `json:"node_ref,omitempty"`
	SkillID       string           `json:"skill_id,omitempty"`
	CapabilityID  string           `json:"capability_id,omitempty"`
	Action        string           `json:"action,omitempty"`
	Status        string           `json:"status,omitempty"`
	MissingFields []string         `json:"missing_fields,omitempty"`
	Result        any              `json:"result,omitempty"`
	Links         []map[string]any `json:"links,omitempty"`
	Error         any              `json:"error,omitempty"`
}

func ReduceAgentRunState(state AgentRunState, event AgentStreamEvent) AgentRunState {
	if state.Run == nil {
		state.Run = map[string]any{}
	}
	payload := event.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	switch event.Type {
	case EventAgentRunStarted:
		mergeRunIdentity(state.Run, payload)
	case EventAgentRunResponsePlan:
		state.ResponsePlan = payload
	case EventAgentRunIntentDetected:
		state.Intent = payload
	case EventAgentRunPlanCreated:
		state.Plan = payload
	case EventAgentRunTaskStatus, EventAgentRunTaskStarted, EventAgentRunTaskCompleted, EventAgentRunTaskFailed, EventAgentRunAwaitingParams:
		task := taskFromPayload(payload)
		if event.Type == EventAgentRunAwaitingParams {
			task.Status = "awaiting_params"
		}
		state.Tasks = upsertTask(state.Tasks, task)
		state.PendingParams = filterTasks(state.Tasks, "awaiting_params")
		state.Results = filterTasks(state.Tasks, "completed")
		state.Errors = filterTasks(state.Tasks, "failed")
	case EventAgentRunFinal:
		mergeRunIdentity(state.Run, payload)
	case EventAgentRunEnded:
		state.Ended = true
	}
	return state
}

func mergeRunIdentity(run map[string]any, payload map[string]any) {
	for _, key := range []string{"run_id", "session_id", "message_id", "trace_id"} {
		if value, ok := payload[key]; ok && strings.TrimSpace(toString(value)) != "" {
			run[key] = value
		}
	}
}

func taskFromPayload(payload map[string]any) AgentTaskState {
	task := AgentTaskState{
		RunID:        toString(payload["run_id"]),
		SessionID:    toString(payload["session_id"]),
		MessageID:    toString(payload["message_id"]),
		TraceID:      toString(payload["trace_id"]),
		TaskID:       firstString(payload["task_id"], payload["node_id"]),
		TeamID:       toString(payload["team_id"]),
		AgentID:      toString(payload["agent_id"]),
		AgentName:    toString(payload["agent_name"]),
		NodeKind:     toString(payload["node_kind"]),
		NodeRef:      toString(payload["node_ref"]),
		SkillID:      toString(payload["skill_id"]),
		CapabilityID: toString(payload["capability_id"]),
		Action:       toString(payload["action"]),
		Status:       firstString(payload["status"], "pending"),
		Result:       payload["result"],
		Error:        payload["error"],
	}
	if values, ok := payload["missing_fields"].([]string); ok {
		task.MissingFields = values
	}
	return task
}

func upsertTask(tasks []AgentTaskState, task AgentTaskState) []AgentTaskState {
	if strings.TrimSpace(task.TaskID) == "" {
		task.TaskID = task.NodeRef
	}
	for i := range tasks {
		if tasks[i].TaskID == task.TaskID {
			tasks[i] = task
			return tasks
		}
	}
	return append(tasks, task)
}

func filterTasks(tasks []AgentTaskState, status string) []AgentTaskState {
	out := make([]AgentTaskState, 0)
	for _, task := range tasks {
		if strings.EqualFold(task.Status, status) {
			out = append(out, task)
		}
	}
	return out
}

func firstString(values ...any) string {
	for _, value := range values {
		if s := strings.TrimSpace(toString(value)); s != "" {
			return s
		}
	}
	return ""
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
