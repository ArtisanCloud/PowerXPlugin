package skills

import (
	"errors"
	"net/http"
)

func MapError(err error, inv PluginSkillInvocation) (int, PluginSkillResult) {
	var skillErr *SkillError
	if !errors.As(err, &skillErr) {
		skillErr = WrapError(ErrCodeExecutionFailed, "skill execution failed", err)
	}
	status := http.StatusInternalServerError
	switch skillErr.Code {
	case ErrCodeContextMissing, ErrCodeCapabilityMismatch, ErrCodeInvalidInvocation, ErrCodeSchemaInvalid:
		status = http.StatusBadRequest
	case ErrCodeNotFound:
		status = http.StatusNotFound
	case ErrCodeAuthDenied:
		status = http.StatusUnauthorized
	case ErrCodeExecutorUnavailable:
		status = http.StatusServiceUnavailable
	}
	return status, ErrorResult(inv, skillErr)
}
