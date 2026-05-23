package scheduler

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidMode             = errors.New("scheduler: invalid mode")
	ErrProviderUnavailable     = errors.New("scheduler: provider unavailable")
	ErrProviderNotConfigured   = errors.New("scheduler provider is not configured")
	ErrHostProviderUnavailable = errors.New("scheduler host provider is not available")
	ErrJobNotFound             = errors.New("scheduler: job not found")
	ErrHandlerNotFound         = errors.New("scheduler: handler not found")
	ErrInvalidJobSpec          = errors.New("scheduler: invalid job spec")
	ErrSchedulerUnavailable    = errors.New("scheduler: unavailable")
)

type HostRequestError struct {
	StatusCode int
	Endpoint   string
	Body       string
}

func (e *HostRequestError) Error() string {
	if e == nil {
		return "scheduler host request failed"
	}
	parts := []string{fmt.Sprintf("scheduler host request failed: status=%d", e.StatusCode)}
	if strings.TrimSpace(e.Endpoint) != "" {
		parts = append(parts, "endpoint="+strings.TrimSpace(e.Endpoint))
	}
	if strings.TrimSpace(e.Body) != "" {
		parts = append(parts, "body="+strings.TrimSpace(e.Body))
	}
	return strings.Join(parts, " ")
}
