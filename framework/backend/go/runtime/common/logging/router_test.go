package logging

import (
	"context"
	"testing"
)

type okSink struct{ name SinkType }

func (s *okSink) Name() SinkType { return s.name }
func (s *okSink) Emit(_ context.Context, _ Event) error { return nil }

func TestRouterFanOutSuccess(t *testing.T) {
	reg := NewSinkRegistry()
	_ = reg.Register(&okSink{name: SinkStdout})
	_ = reg.Register(&okSink{name: SinkFile})

	r, err := NewRouter(Policy{
		Mode:  ModeStandalone,
		Sinks: []SinkType{SinkStdout, SinkFile},
		Format: "json",
		Level:  "info",
		Retry: RetryPolicy{Enabled: true, MaxAttempts: 2, BackoffMS: 10},
	}, reg)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	outcomes := r.Route(context.Background(), Event{Message: "fanout", Level: "info"})
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 outcomes, got %+v", outcomes)
	}
	for _, item := range outcomes {
		if item.Status != OutcomeSuccess {
			t.Fatalf("expected success, got %+v", item)
		}
	}
}
