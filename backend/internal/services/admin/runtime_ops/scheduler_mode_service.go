package runtime_ops

import "strings"

const (
	ExecutionModeStandaloneLocal = "standalone_local"
	ExecutionModeDelegatedProxy  = "delegated_proxy"
)

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
	proxy := normalizePowerXProxy(req.PowerXProxy)
	provider := strings.ToLower(strings.TrimSpace(req.TaskBusProvider))

	result := ModeValidationResult{
		Valid:           false,
		ExecutionMode:   resolveExecutionMode(proxy),
		TaskBusProvider: provider,
	}
	if provider != "host" && provider != "redis" {
		result.Message = "invalid taskbus_provider: must be host or redis"
		return result
	}

	switch proxy {
	case "1":
		if provider == "host" {
			result.Valid = true
			result.Message = "mode/provider matched"
			return result
		}
		result.Message = "mode conflict: delegated_proxy requires taskbus_provider=host"
		return result
	default:
		if provider == "redis" {
			result.Valid = true
			result.Message = "mode/provider matched"
			return result
		}
		result.Message = "mode conflict: standalone_local requires taskbus_provider=redis"
		return result
	}
}

func normalizePowerXProxy(raw string) string {
	if strings.TrimSpace(raw) == "1" {
		return "1"
	}
	return "0"
}

func resolveExecutionMode(proxy string) string {
	if normalizePowerXProxy(proxy) == "1" {
		return ExecutionModeDelegatedProxy
	}
	return ExecutionModeStandaloneLocal
}
