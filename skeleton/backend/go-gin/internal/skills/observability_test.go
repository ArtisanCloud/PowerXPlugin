package skills

import (
	"testing"

	runtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
	"github.com/stretchr/testify/require"
)

func TestSkillLogFieldsIncludeTraceTenantSessionAndSkill(t *testing.T) {
	inv := runtime.PluginSkillInvocation{
		SkillID: TemplateSkillID,
		Context: runtime.PluginSkillInvocationContext{
			TenantUUID: "tenant_001",
			SessionID:  "session_001",
			TraceID:    "trace_001",
			PluginID:   "com.powerx.plugins.base",
		},
	}
	attrs := runtime.LogAttrs(inv, "")
	fields := map[string]string{}
	for _, attr := range attrs {
		fields[attr.Key] = attr.Value.String()
	}
	require.Equal(t, "tenant_001", fields[runtime.LogFieldTenantUUID])
	require.Equal(t, "session_001", fields[runtime.LogFieldSessionID])
	require.Equal(t, "trace_001", fields[runtime.LogFieldTraceID])
	require.Equal(t, TemplateSkillID, fields[runtime.LogFieldSkillID])
	require.Equal(t, "com.powerx.plugins.base", fields[runtime.LogFieldPluginID])
}
