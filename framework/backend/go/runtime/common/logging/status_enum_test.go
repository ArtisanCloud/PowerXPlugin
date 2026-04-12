package logging

import "testing"

func TestStatusEnumConsistency(t *testing.T) {
	expected := []string{
		StatusQueued,
		StatusProcessing,
		StatusSucceeded,
		StatusFailed,
		StatusSkipped,
	}

	if len(expected) != len(StatusEnum) {
		t.Fatalf("status enum length = %d, want %d", len(StatusEnum), len(expected))
	}

	for i, want := range expected {
		if got := StatusEnum[i]; got != want {
			t.Fatalf("status enum[%d] = %q, want %q", i, got, want)
		}
		if !IsAllowedStatus(want) {
			t.Fatalf("expected %q to be allowed status", want)
		}
	}

	for _, invalid := range []string{"done", "ok", "pending"} {
		if IsAllowedStatus(invalid) {
			t.Fatalf("expected %q to be rejected", invalid)
		}
	}
}
