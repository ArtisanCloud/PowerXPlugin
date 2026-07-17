package skills

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validInvocation() PluginSkillInvocation {
	return PluginSkillInvocation{
		SkillID: "mediax.video_rebuilder.cn",
		Version: "1.0.0",
		Input:   map[string]any{"urls": []any{"https://example.com/video.mp4"}},
		Context: PluginSkillInvocationContext{
			TenantUUID: "tenant_001",
			UserUUID:   "user_001",
			AgentID:    "agent_001",
			SessionID:  "session_001",
			MessageID:  "message_001",
			SkillID:    "mediax.video_rebuilder.cn",
			TraceID:    "trace_001",
			Channel:    "web",
			Locale:     "zh-CN",
			Capability: "creation.video_automation.ingest",
		},
	}
}

func TestValidateInvocationContextRequiresTenant(t *testing.T) {
	ctx := validInvocation().Context
	ctx.TenantUUID = ""
	err := ValidateInvocationContext(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), ErrCodeContextMissing)
	require.Contains(t, err.Error(), "tenant_uuid is required")
}
