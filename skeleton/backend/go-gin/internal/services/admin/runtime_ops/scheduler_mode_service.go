package runtime_ops

import "strings"

// ModeValidationRequest describes runtime mode validation input.
type ModeValidationRequest struct {
	PowerXProxy       string `json:"powerx_proxy"`
	TaskBusProvider   string `json:"taskbus_provider"`
	GatewayAuthScheme string `json:"gateway_auth_scheme,omitempty"`
}

// ModeValidationResult describes runtime mode validation output.
type ModeValidationResult struct {
	Valid           bool   `json:"valid"`
	ExecutionMode   string `json:"execution_mode"`
	TaskBusProvider string `json:"taskbus_provider"`
	Message         string `json:"message,omitempty"`
}

// SchedulerModeService validates runtime mode/provider consistency.
type SchedulerModeService struct{}

// NewSchedulerModeService constructs scheduler mode service.
func NewSchedulerModeService() *SchedulerModeService {
	return &SchedulerModeService{}
}

// Validate checks POWERX_PROXY and taskbus provider pairing.
func (s *SchedulerModeService) Validate(req ModeValidationRequest) ModeValidationResult {
	proxy := strings.TrimSpace(req.PowerXProxy)
	provider := strings.ToLower(strings.TrimSpace(req.TaskBusProvider))

	result := ModeValidationResult{
		Valid:           false,
		TaskBusProvider: provider,
	}

	switch proxy {
	case "1":
		result.ExecutionMode = "delegated_proxy"
		if provider == "host" {
			result.Valid = true
			result.Message = "mode/provider matched"
			return result
		}
		result.Message = "mode conflict: delegated_proxy requires taskbus_provider=host"
		return result
	default:
		result.ExecutionMode = "standalone_local"
		if provider == "redis" {
			result.Valid = true
			result.Message = "mode/provider matched"
			return result
		}
		result.Message = "mode conflict: standalone_local requires taskbus_provider=redis"
		return result
	}
}
