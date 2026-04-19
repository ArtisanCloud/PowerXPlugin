package federated

import "testing"

func TestSessionServiceInvalidatesMember(t *testing.T) {
	svc := NewSessionService()
	if svc.IsInvalidated(10) {
		t.Fatalf("IsInvalidated before unbind = true, want false")
	}
	svc.InvalidateMember(10)
	if !svc.IsInvalidated(10) {
		t.Fatalf("IsInvalidated after unbind = false, want true")
	}
}
