package scheduler

import "errors"

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
