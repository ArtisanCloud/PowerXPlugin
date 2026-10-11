package skills

import (
	"context"
	"strings"

	powerxskills "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/skills"
	frameworkskills "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
)

// TrustedTenantResolver obtains scope from authenticated request context. It
// deliberately does not accept a tenant UUID from an InvokeInput payload.
type TrustedTenantResolver func(context.Context) (string, bool)

type pluginSkillRegistry interface {
	Invoke(context.Context, frameworkskills.PluginSkillInvocation) (frameworkskills.PluginSkillResult, error)
}

// FrameworkLocalInvoker adapts a plugin-owned local Skill registry to the
// same typed contract used by delegated PowerX Skill calls.
type FrameworkLocalInvoker struct {
	registry      pluginSkillRegistry
	resolveTenant TrustedTenantResolver
	pluginID      string
}

func NewFrameworkLocalInvoker(registry pluginSkillRegistry, resolveTenant TrustedTenantResolver, pluginID string) *FrameworkLocalInvoker {
	return &FrameworkLocalInvoker{registry: registry, resolveTenant: resolveTenant, pluginID: strings.TrimSpace(pluginID)}
}

func (i *FrameworkLocalInvoker) Invoke(ctx context.Context, input powerxskills.InvokeInput) (*powerxskills.InvokeOutput, error) {
	if i == nil || i.registry == nil || i.resolveTenant == nil {
		return nil, frameworkskills.NewError(frameworkskills.ErrCodeExecutorUnavailable, "local skill invoker is not configured")
	}
	tenantUUID, ok := i.resolveTenant(ctx)
	tenantUUID = strings.TrimSpace(tenantUUID)
	if !ok || tenantUUID == "" {
		return nil, frameworkskills.NewError(frameworkskills.ErrCodeContextMissing, "trusted tenant context is required")
	}
	skillID := strings.TrimSpace(input.SkillID)
	invocation := frameworkskills.PluginSkillInvocation{
		SkillID: skillID,
		Version: strings.TrimSpace(input.Version),
		Input:   input.Payload,
		Context: frameworkskills.PluginSkillInvocationContext{
			TenantUUID: tenantUUID,
			UserUUID:   inputContextString(input.Context, "user_uuid"),
			AgentID:    inputContextString(input.Context, "agent_id"),
			SessionID:  inputContextString(input.Context, "session_id"),
			MessageID:  inputContextString(input.Context, "message_id"),
			SkillID:    skillID,
			TraceID:    inputContextString(input.Context, "trace_id"),
			Channel:    inputContextString(input.Context, "channel"),
			Locale:     inputContextString(input.Context, "locale"),
			Capability: inputContextString(input.Context, "capability"),
			RequestID:  inputContextString(input.Context, "request_id"),
			PluginID:   i.pluginID,
		},
		IdempotencyKey: inputContextString(input.Context, "idempotency_key"),
	}
	result, err := i.registry.Invoke(ctx, invocation)
	if err != nil {
		return nil, err
	}
	return &powerxskills.InvokeOutput{
		TraceID:      result.TraceID,
		Status:       result.Status,
		ProtocolUsed: "local",
		FallbackUsed: false,
		Result:       result.Data,
	}, nil
}

func inputContextString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}
