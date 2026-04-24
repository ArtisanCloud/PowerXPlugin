package logger

import "testing"

func TestGovernancePolicyStatus(t *testing.T) {
	tests := []struct {
		name           string
		policy         GovernancePolicy
		violationCount int
		want           GovernanceStatus
	}{
		{
			name:           "resolved when no violation",
			policy:         GovernancePolicy{Mode: "warn"},
			violationCount: 0,
			want:           GovernanceResolved,
		},
		{
			name:           "detect mode returns detected",
			policy:         GovernancePolicy{Mode: "detect"},
			violationCount: 2,
			want:           GovernanceDetected,
		},
		{
			name:           "warn mode returns warned",
			policy:         GovernancePolicy{Mode: "warn"},
			violationCount: 3,
			want:           GovernanceWarned,
		},
		{
			name:           "block mode returns blocked",
			policy:         GovernancePolicy{Mode: "block"},
			violationCount: 1,
			want:           GovernanceBlocked,
		},
		{
			name:           "unknown mode falls back warned",
			policy:         GovernancePolicy{Mode: "custom"},
			violationCount: 1,
			want:           GovernanceWarned,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.Status(tt.violationCount)
			if got != tt.want {
				t.Fatalf("status mismatch, got=%s want=%s", got, tt.want)
			}
		})
	}
}

func TestGovernancePolicyShouldBlockByDeadline(t *testing.T) {
	tests := []struct {
		name   string
		policy GovernancePolicy
		want   bool
	}{
		{
			name:   "current lower than deadline",
			policy: GovernancePolicy{CurrentVersion: "1.1.0", DeadlineVersion: "1.2.0"},
			want:   false,
		},
		{
			name:   "current equals deadline",
			policy: GovernancePolicy{CurrentVersion: "1.2.0", DeadlineVersion: "1.2.0"},
			want:   true,
		},
		{
			name:   "current greater than deadline",
			policy: GovernancePolicy{CurrentVersion: "v2.0.1", DeadlineVersion: "1.9.9"},
			want:   true,
		},
		{
			name:   "missing versions",
			policy: GovernancePolicy{CurrentVersion: "", DeadlineVersion: "1.0.0"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.policy.ShouldBlockByDeadline()
			if got != tt.want {
				t.Fatalf("deadline decision mismatch, got=%v want=%v", got, tt.want)
			}
		})
	}
}
