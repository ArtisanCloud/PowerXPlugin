package eventbridge

import "errors"

var (
	ErrInvalidMode     = errors.New("invalid eventbridge mode (expected: local, taskbus, dual)")
	ErrNotConfigured   = errors.New("eventbridge not configured")
	ErrTaskBusRequired = errors.New("eventbridge taskbus provider not configured")
)

