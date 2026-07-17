package skills

import (
	"errors"
	"fmt"
)

const (
	ErrCodeNotFound            = "skill.not_found"
	ErrCodeContextMissing      = "skill.plugin_context_missing"
	ErrCodeCapabilityMismatch  = "skill.plugin_capability_mismatch"
	ErrCodeExecutorUnavailable = "skill.plugin_executor_unavailable"
	ErrCodeExecutionFailed     = "skill.execution_failed"
	ErrCodeSchemaInvalid       = "skill.schema_invalid"
	ErrCodeAuthDenied          = "skill.auth_denied"
	ErrCodeManifestInvalid     = "skill.manifest_invalid"
	ErrCodeDuplicateManifest   = "skill.manifest_duplicate"
	ErrCodeInvalidInvocation   = "skill.invocation_invalid"
)

type SkillError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	TraceID   string         `json:"trace_id,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

func (e *SkillError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

func NewError(code, message string) *SkillError {
	return &SkillError{Code: code, Message: message}
}

func WrapError(code, message string, err error) *SkillError {
	if err == nil {
		return NewError(code, message)
	}
	var skillErr *SkillError
	if errors.As(err, &skillErr) {
		return skillErr
	}
	out := NewError(code, message)
	out.Details = map[string]any{"cause": err.Error()}
	return out
}

func MissingFieldError(field string) *SkillError {
	return NewError(ErrCodeContextMissing, fmt.Sprintf("%s is required", field))
}
