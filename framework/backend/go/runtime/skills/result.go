package skills

const (
	ResultQueued    = "queued"
	ResultRunning   = "running"
	ResultCompleted = "completed"
	ResultFailed    = "failed"
	ResultDenied    = "denied"
)

type PluginSkillResult struct {
	Success bool           `json:"success"`
	SkillID string         `json:"skill_id,omitempty"`
	Status  string         `json:"status"`
	Message string         `json:"message,omitempty"`
	TaskID  string         `json:"task_id,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
	TraceID string         `json:"trace_id,omitempty"`
	Error   *SkillError    `json:"error,omitempty"`
}

func SuccessResult(inv PluginSkillInvocation, status, message string, data map[string]any) PluginSkillResult {
	return PluginSkillResult{
		Success: true,
		SkillID: inv.SkillID,
		Status:  status,
		Message: message,
		Data:    data,
		TraceID: inv.Context.TraceID,
	}
}

func ErrorResult(inv PluginSkillInvocation, err *SkillError) PluginSkillResult {
	if err == nil {
		err = NewError(ErrCodeExecutionFailed, "skill execution failed")
	}
	if err.TraceID == "" {
		err.TraceID = inv.Context.TraceID
	}
	if err.RequestID == "" {
		err.RequestID = inv.Context.RequestID
	}
	return PluginSkillResult{
		Success: false,
		SkillID: inv.SkillID,
		Status:  ResultFailed,
		TraceID: inv.Context.TraceID,
		Error:   err,
	}
}
