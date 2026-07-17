package skills

import (
	"context"
	"strings"
)

type ExecutorHandler func(context.Context, PluginSkillInvocation) (PluginSkillResult, error)

func (r *Registry) RegisterExecutor(skillID string, handler ExecutorHandler) error {
	if strings.TrimSpace(skillID) == "" {
		return NewError(ErrCodeExecutorUnavailable, "skill_id is required")
	}
	if handler == nil {
		return NewError(ErrCodeExecutorUnavailable, "executor handler is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executors[skillID] = handler
	return nil
}

func (r *Registry) Invoke(ctx context.Context, inv PluginSkillInvocation) (PluginSkillResult, error) {
	if strings.TrimSpace(inv.SkillID) == "" {
		return ErrorResult(inv, NewError(ErrCodeInvalidInvocation, "skill_id is required")), NewError(ErrCodeInvalidInvocation, "skill_id is required")
	}
	if err := ValidateInvocationContext(inv.Context); err != nil {
		return ErrorResult(inv, WrapError(ErrCodeContextMissing, "plugin context is missing", err)), err
	}
	if inv.Context.SkillID != "" && inv.Context.SkillID != inv.SkillID {
		err := NewError(ErrCodeInvalidInvocation, "context.skill_id must match invocation.skill_id")
		return ErrorResult(inv, err), err
	}
	manifest, ok := r.Get(inv.SkillID, inv.Version)
	if !ok {
		err := NewError(ErrCodeNotFound, "skill is not registered")
		return ErrorResult(inv, err), err
	}
	if inv.Context.Capability != "" && manifest.Executor.Capability != "" && inv.Context.Capability != manifest.Executor.Capability {
		err := NewError(ErrCodeCapabilityMismatch, "capability does not match manifest executor")
		return ErrorResult(inv, err), err
	}
	r.mu.RLock()
	handler := r.executors[inv.SkillID]
	r.mu.RUnlock()
	if handler == nil {
		err := NewError(ErrCodeExecutorUnavailable, "executor handler is not registered")
		return ErrorResult(inv, err), err
	}
	result, err := handler(ctx, inv)
	if err != nil {
		return result, WrapError(ErrCodeExecutionFailed, "skill execution failed", err)
	}
	if result.SkillID == "" {
		result.SkillID = inv.SkillID
	}
	if result.TraceID == "" {
		result.TraceID = inv.Context.TraceID
	}
	return result, nil
}
