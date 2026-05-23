package logging

import (
	"context"
	"errors"
	"testing"
)

type stubSink struct {
	name SinkType
	err  error
}

func (s *stubSink) Name() SinkType { return s.name }
func (s *stubSink) Emit(_ context.Context, _ Event) error {
	return s.err
}

func TestValidatePolicyHostRequiresAuthorizedExtraSink(t *testing.T) {
	p := ResolvePolicy(Policy{
		Mode:  ModeHost,
		Sinks: []SinkType{SinkStdout, SinkLoki},
		Retry: RetryPolicy{Enabled: true, MaxAttempts: 1, BackoffMS: 1},
	})
	if err := ValidatePolicy(p); err == nil {
		t.Fatalf("expected validation error for unauthorized sink")
	}

	p.AuthorizedExtraSinks = []SinkType{SinkLoki}
	if err := ValidatePolicy(p); err != nil {
		t.Fatalf("expected host policy to be valid, got: %v", err)
	}
}

func TestResolvePolicyDefaultsStandaloneEvenWhenPowerXProxyEnabled(t *testing.T) {
	t.Setenv("POWERX_PROXY", "1")
	p := ResolvePolicy(Policy{
		Sinks: []SinkType{SinkFile},
		Retry: RetryPolicy{Enabled: true, MaxAttempts: 2, BackoffMS: 100},
	})
	if p.Mode != ModeStandalone {
		t.Fatalf("expected standalone mode, got %s", p.Mode)
	}
	if len(p.Sinks) != 1 || p.Sinks[0] != SinkFile {
		t.Fatalf("expected sinks=[file], got=%v", p.Sinks)
	}
}

func TestResolveWithHostModeFalsePreservesFileOutput(t *testing.T) {
	t.Setenv("POWERX_PROXY", "1")
	p := ResolveWithHostMode(Policy{
		Sinks:  []SinkType{SinkFile},
		Format: "text",
		Retry:  RetryPolicy{Enabled: true, MaxAttempts: 2, BackoffMS: 100},
	}, false)
	if p.Mode != ModeStandalone {
		t.Fatalf("expected standalone mode, got %s", p.Mode)
	}
	if p.Format != "text" {
		t.Fatalf("expected text format, got %s", p.Format)
	}
	if len(p.Sinks) != 1 || p.Sinks[0] != SinkFile {
		t.Fatalf("expected sinks=[file], got=%v", p.Sinks)
	}
}

func TestResolveWithHostModeTrueForcesStdoutJSON(t *testing.T) {
	p := ResolveWithHostMode(Policy{
		Sinks: []SinkType{SinkFile},
		Retry: RetryPolicy{Enabled: true, MaxAttempts: 2, BackoffMS: 100},
	}, true)
	if p.Mode != ModeHost {
		t.Fatalf("expected host mode, got %s", p.Mode)
	}
	if p.Format != "json" {
		t.Fatalf("expected json format, got %s", p.Format)
	}
	if len(p.Sinks) != 1 || p.Sinks[0] != SinkStdout {
		t.Fatalf("expected sinks=[stdout], got=%v", p.Sinks)
	}
	if len(p.AuthorizedExtraSinks) != 0 {
		t.Fatalf("expected authorized_extra_sinks to be cleared in host mode, got=%v", p.AuthorizedExtraSinks)
	}
}

func TestSinkRegistryAndRouter(t *testing.T) {
	reg := NewSinkRegistry()
	if err := reg.Register(&stubSink{name: SinkStdout}); err != nil {
		t.Fatalf("register sink: %v", err)
	}
	if _, ok := reg.Resolve(SinkStdout); !ok {
		t.Fatalf("expected stdout sink to be resolved")
	}

	router, err := NewRouter(Policy{
		Mode:  ModeStandalone,
		Sinks: []SinkType{SinkStdout},
		Retry: RetryPolicy{Enabled: true, MaxAttempts: 1, BackoffMS: 100},
	}, reg)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	outcomes := router.Route(context.Background(), Event{Message: "test", Level: "info"})
	if len(outcomes) != 1 || outcomes[0].Status != OutcomeSuccess {
		t.Fatalf("unexpected outcomes: %+v", outcomes)
	}
}

func TestRouterRetryOutcome(t *testing.T) {
	reg := NewSinkRegistry()
	if err := reg.Register(&stubSink{name: SinkStdout, err: errors.New("boom")}); err != nil {
		t.Fatalf("register sink: %v", err)
	}
	router, err := NewRouter(Policy{
		Mode:  ModeStandalone,
		Sinks: []SinkType{SinkStdout},
		Retry: RetryPolicy{Enabled: true, MaxAttempts: 2, BackoffMS: 100},
	}, reg)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	outcomes := router.Route(context.Background(), Event{Message: "test", Level: "info"})
	if len(outcomes) != 2 {
		t.Fatalf("expected 2 outcomes (retry + failed), got %+v", outcomes)
	}
	if outcomes[0].Status != OutcomeRetrying || outcomes[1].Status != OutcomeFailed {
		t.Fatalf("unexpected retry outcomes: %+v", outcomes)
	}
}
