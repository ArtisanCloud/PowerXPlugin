package skills

import (
	"context"
	"log/slog"
)

const (
	LogFieldPluginID   = "plugin_id"
	LogFieldTenantUUID = "tenant_uuid"
	LogFieldSkillID    = "skill_id"
	LogFieldSessionID  = "session_id"
	LogFieldTraceID    = "trace_id"
	LogFieldComponent  = "component"
)

func LogAttrs(inv PluginSkillInvocation, pluginID string) []slog.Attr {
	if pluginID == "" {
		pluginID = inv.Context.PluginID
	}
	return []slog.Attr{
		slog.String(LogFieldPluginID, pluginID),
		slog.String(LogFieldTenantUUID, inv.Context.TenantUUID),
		slog.String(LogFieldSkillID, inv.SkillID),
		slog.String(LogFieldSessionID, inv.Context.SessionID),
		slog.String(LogFieldTraceID, inv.Context.TraceID),
		slog.String(LogFieldComponent, "agent_skill_bridge"),
	}
}

func LogInvoke(ctx context.Context, logger *slog.Logger, inv PluginSkillInvocation, pluginID, status string) {
	if logger == nil {
		return
	}
	attrs := LogAttrs(inv, pluginID)
	attrs = append(attrs, slog.String("status", status))
	logger.LogAttrs(ctx, slog.LevelInfo, "plugin skill invocation", attrs...)
}
