package skills

import (
	"context"
	"testing"

	powerxskills "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/skills"
	frameworkskills "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
)

type recordingSkillRegistry struct {
	invocation frameworkskills.PluginSkillInvocation
}

func (r *recordingSkillRegistry) Invoke(_ context.Context, inv frameworkskills.PluginSkillInvocation) (frameworkskills.PluginSkillResult, error) {
	r.invocation = inv
	return frameworkskills.SuccessResult(inv, frameworkskills.ResultCompleted, "completed", map[string]any{"ok": true}), nil
}

func TestFrameworkLocalInvokerDerivesTenantFromTrustedContext(t *testing.T) {
	registry := &recordingSkillRegistry{}
	invoker := NewFrameworkLocalInvoker(registry, func(context.Context) (string, bool) { return "tenant-trusted", true }, "com.powerx.test")
	output, err := invoker.Invoke(context.Background(), powerxskills.InvokeInput{
		SkillID: "demo.skill",
		Payload: map[string]any{"action": "list"},
		Context: map[string]any{
			"tenant_uuid": "tenant-untrusted",
			"user_uuid":   "user-1",
			"agent_id":    "agent-1",
			"session_id":  "session-1",
			"trace_id":    "trace-1",
		},
	})
	if err != nil {
		t.Fatalf("Invoke(): %v", err)
	}
	if output.ProtocolUsed != "local" || output.TraceID != "trace-1" || output.Result["ok"] != true {
		t.Fatalf("unexpected output: %#v", output)
	}
	if registry.invocation.Context.TenantUUID != "tenant-trusted" {
		t.Fatalf("tenant = %q, want trusted context tenant", registry.invocation.Context.TenantUUID)
	}
}

func TestFrameworkLocalInvokerRejectsMissingTrustedTenant(t *testing.T) {
	invoker := NewFrameworkLocalInvoker(&recordingSkillRegistry{}, func(context.Context) (string, bool) { return "", false }, "com.powerx.test")
	_, err := invoker.Invoke(context.Background(), powerxskills.InvokeInput{SkillID: "demo.skill"})
	if err == nil {
		t.Fatal("Invoke() error = nil, want missing trusted tenant error")
	}
}
