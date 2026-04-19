package auth

import (
	"testing"
)

type captureSink struct {
	events []FederatedAuditEvent
}

func (c *captureSink) Emit(event FederatedAuditEvent) {
	c.events = append(c.events, event)
}

func TestFederatedAuditEventFieldsCompleteness(t *testing.T) {
	sink := &captureSink{}
	SetFederatedAuditSinkForTests(sink)
	t.Cleanup(func() { SetFederatedAuditSinkForTests(nil) })

	svc := NewFederatedAuditService("com.powerx.plugin.test")
	svc.Record(FederatedAuditEvent{
		Provider:         "wecom",
		TenantUUID:       "tenant-a",
		ExternalIdentity: "wx-uid-1",
		BindingOutcome:   "login_success",
		RiskDecision:     "allow",
		ReasonCode:       "ok",
		TraceID:          "trace-1",
		Evidence:         map[string]string{"state": "s1"},
	})

	if len(sink.events) != 1 {
		t.Fatalf("len(events)=%d, want 1", len(sink.events))
	}
	e := sink.events[0]
	if e.PluginID == "" || e.Provider == "" || e.TenantUUID == "" || e.TraceID == "" {
		t.Fatalf("required fields missing: %+v", e)
	}
	if e.BindingOutcome == "" || e.RiskDecision == "" || e.ReasonCode == "" {
		t.Fatalf("decision fields missing: %+v", e)
	}
	if e.OccurredAt.IsZero() {
		t.Fatalf("OccurredAt not set")
	}
}
