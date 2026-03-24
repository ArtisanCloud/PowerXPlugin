package runtime_ops

import "testing"

func TestSchedulerModeServiceValidate(t *testing.T) {
	svc := NewSchedulerModeService()

	tests := []struct {
		name          string
		request       ModeValidationRequest
		expectValid   bool
		expectMode    string
		expectMessage string
	}{
		{
			name: "standalone_local with redis",
			request: ModeValidationRequest{
				PowerXProxy:     "0",
				TaskBusProvider: "redis",
			},
			expectValid:   true,
			expectMode:    ExecutionModeStandaloneLocal,
			expectMessage: "mode/provider matched",
		},
		{
			name: "delegated_proxy with host",
			request: ModeValidationRequest{
				PowerXProxy:     "1",
				TaskBusProvider: "host",
			},
			expectValid:   true,
			expectMode:    ExecutionModeDelegatedProxy,
			expectMessage: "mode/provider matched",
		},
		{
			name: "conflict delegated with redis",
			request: ModeValidationRequest{
				PowerXProxy:     "1",
				TaskBusProvider: "redis",
			},
			expectValid:   false,
			expectMode:    ExecutionModeDelegatedProxy,
			expectMessage: "mode conflict: delegated_proxy requires taskbus_provider=host",
		},
		{
			name: "conflict standalone with host",
			request: ModeValidationRequest{
				PowerXProxy:     "0",
				TaskBusProvider: "host",
			},
			expectValid:   false,
			expectMode:    ExecutionModeStandaloneLocal,
			expectMessage: "mode conflict: standalone_local requires taskbus_provider=redis",
		},
		{
			name: "invalid provider",
			request: ModeValidationRequest{
				PowerXProxy:     "1",
				TaskBusProvider: "memory",
			},
			expectValid:   false,
			expectMode:    ExecutionModeDelegatedProxy,
			expectMessage: "invalid taskbus_provider: must be host or redis",
		},
		{
			name: "proxy malformed defaults to standalone",
			request: ModeValidationRequest{
				PowerXProxy:     "unexpected",
				TaskBusProvider: "redis",
			},
			expectValid:   true,
			expectMode:    ExecutionModeStandaloneLocal,
			expectMessage: "mode/provider matched",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.Validate(tt.request)
			if result.Valid != tt.expectValid {
				t.Fatalf("valid mismatch, got=%v want=%v", result.Valid, tt.expectValid)
			}
			if result.ExecutionMode != tt.expectMode {
				t.Fatalf("execution_mode mismatch, got=%s want=%s", result.ExecutionMode, tt.expectMode)
			}
			if result.Message != tt.expectMessage {
				t.Fatalf("message mismatch, got=%s want=%s", result.Message, tt.expectMessage)
			}
		})
	}
}
