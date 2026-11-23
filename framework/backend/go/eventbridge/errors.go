package eventbridge

import "errors"

var (
	ErrNotConfigured   = errors.New("eventbridge not configured")
	ErrTaskBusRequired = errors.New("eventbridge taskbus provider not configured")
)
