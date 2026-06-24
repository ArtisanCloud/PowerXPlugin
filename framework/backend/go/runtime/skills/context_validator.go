package skills

import "strings"

func ValidateInvocationContext(ctx PluginSkillInvocationContext) error {
	required := map[string]string{
		"tenant_uuid": ctx.TenantUUID,
		"user_uuid":   ctx.UserUUID,
		"agent_id":    ctx.AgentID,
		"session_id":  ctx.SessionID,
		"skill_id":    ctx.SkillID,
		"trace_id":    ctx.TraceID,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return MissingFieldError(field)
		}
	}
	return nil
}
