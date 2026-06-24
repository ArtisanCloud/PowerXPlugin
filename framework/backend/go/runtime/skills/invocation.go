package skills

type PluginSkillInvocation struct {
	SkillID        string                       `json:"skill_id"`
	Version        string                       `json:"version,omitempty"`
	Input          map[string]any               `json:"input,omitempty"`
	Context        PluginSkillInvocationContext `json:"context"`
	IdempotencyKey string                       `json:"idempotency_key,omitempty"`
}

type PluginSkillInvocationContext struct {
	TenantUUID string `json:"tenant_uuid"`
	UserUUID   string `json:"user_uuid"`
	AgentID    string `json:"agent_id"`
	SessionID  string `json:"session_id"`
	MessageID  string `json:"message_id,omitempty"`
	SkillID    string `json:"skill_id"`
	TraceID    string `json:"trace_id"`
	Channel    string `json:"channel,omitempty"`
	Locale     string `json:"locale,omitempty"`
	Capability string `json:"capability,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	PluginID   string `json:"plugin_id,omitempty"`
}
