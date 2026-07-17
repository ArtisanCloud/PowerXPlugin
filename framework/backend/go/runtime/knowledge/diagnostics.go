package knowledge

import "time"

const (
	DiagProvider      = "provider"
	DiagProviderMode  = "provider_mode"
	DiagOperation     = "operation"
	DiagTenantUUID    = "tenant_uuid"
	DiagPluginID      = "plugin_id"
	DiagAgentUUID     = "agent_uuid"
	DiagSkillID       = "skill_id"
	DiagTraceID       = "trace_id"
	DiagLatencyMS     = "latency_ms"
	DiagAuditedBypass = "audited_bypass"
)

type Diagnostics struct {
	Provider      string
	ProviderMode  string
	Operation     string
	TenantUUID    string
	PluginID      string
	AgentUUID     string
	SkillID       string
	TraceID       string
	StartedAt     time.Time
	FinishedAt    time.Time
	AuditedBypass bool
	Fields        map[string]any
}

func (d Diagnostics) Map() map[string]any {
	out := map[string]any{}
	if d.Provider != "" {
		out[DiagProvider] = d.Provider
	}
	if d.ProviderMode != "" {
		out[DiagProviderMode] = d.ProviderMode
	}
	if d.Operation != "" {
		out[DiagOperation] = d.Operation
	}
	if d.TenantUUID != "" {
		out[DiagTenantUUID] = d.TenantUUID
	}
	if d.PluginID != "" {
		out[DiagPluginID] = d.PluginID
	}
	if d.AgentUUID != "" {
		out[DiagAgentUUID] = d.AgentUUID
	}
	if d.SkillID != "" {
		out[DiagSkillID] = d.SkillID
	}
	if d.TraceID != "" {
		out[DiagTraceID] = d.TraceID
	}
	if !d.StartedAt.IsZero() && !d.FinishedAt.IsZero() {
		out[DiagLatencyMS] = d.FinishedAt.Sub(d.StartedAt).Milliseconds()
	}
	if d.AuditedBypass {
		out[DiagAuditedBypass] = true
	}
	for key, value := range RedactMap(d.Fields) {
		out[key] = value
	}
	return out
}

func QueryDiagnostics(provider KnowledgeProvider, operation string, q KnowledgeQuery, started time.Time) map[string]any {
	finished := time.Now()
	return Diagnostics{
		Provider:     provider.Name(),
		ProviderMode: provider.Mode(),
		Operation:    operation,
		TenantUUID:   q.TenantUUID,
		PluginID:     q.PluginID,
		AgentUUID:    q.AgentUUID,
		SkillID:      q.SkillID,
		TraceID:      q.TraceID,
		StartedAt:    started,
		FinishedAt:   finished,
	}.Map()
}
