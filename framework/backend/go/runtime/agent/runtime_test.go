package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/agent"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type lifecycleStub struct{ status string }

func TestSessionFactorySelectionAndMissingAdapter(t *testing.T) {
	local, delegated := &powerxagent.Client{}, &powerxagent.Client{}
	for _, mode := range []provider.Mode{provider.ModeLocal, provider.ModeDelegated} {
		r, err := NewRuntime(mode, nil, nil, WithSessions(local, delegated))
		if err != nil {
			t.Fatal(err)
		}
		got, err := r.Sessions()
		if err != nil {
			t.Fatal(err)
		}
		want := local
		if mode == provider.ModeDelegated {
			want = delegated
		}
		if got != want {
			t.Fatal("adapter selection")
		}
	}
	var missing *powerxagent.Client
	r, err := NewRuntime(provider.ModeDelegated, nil, nil, WithSessions(local, missing))
	if err != nil {
		t.Fatal(err)
	}
	for _, runtime := range []*Runtime{nil, {}, r} {
		got, err := runtime.Sessions()
		var target *module.Error
		if got != nil || !errors.As(err, &target) || target.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
			t.Fatalf("%v %v", got, err)
		}
	}
}

func (s lifecycleStub) Invoke(context.Context, powerxagent.AgentInvokeRequest) (powerxagent.AgentInvokeResponse, error) {
	return powerxagent.AgentInvokeResponse{Message: s.status, SessionID: s.status}, nil
}
func (s lifecycleStub) StreamSSE(_ context.Context, _ url.Values, callback func(powerxagent.AgentStreamEvent) error) error {
	return callback(powerxagent.AgentStreamEvent{Type: "final"})
}

func (s lifecycleStub) GetHealthSummary(context.Context, string) (*powerxagent.HealthSummary, error) {
	return &powerxagent.HealthSummary{Status: s.status}, nil
}
func (lifecycleStub) ListHealthHistory(context.Context, string, int, int) (*powerxagent.HealthHistory, error) {
	return &powerxagent.HealthHistory{}, nil
}
func (lifecycleStub) GetBridgeState(context.Context, string, int) (*json.RawMessage, error) {
	return nil, nil
}
func (lifecycleStub) Freeze(context.Context, string, powerxagent.BridgeControlInput) (*powerxagent.BridgeLifecycleResult, error) {
	return nil, nil
}
func (lifecycleStub) Recover(context.Context, string, powerxagent.BridgeControlInput) (*powerxagent.BridgeLifecycleResult, error) {
	return nil, nil
}
func (lifecycleStub) Rebalance(context.Context, string, powerxagent.BridgeRebalanceInput) (*powerxagent.BridgeLifecycleResult, error) {
	return nil, nil
}

func TestRuntimeUsesOnlySelectedLifecycleAdapter(t *testing.T) {
	for _, tc := range []struct {
		mode provider.Mode
		want string
	}{{provider.ModeLocal, "local"}, {provider.ModeDelegated, "delegated"}} {
		runtime, err := NewRuntime(tc.mode, lifecycleStub{status: "local"}, lifecycleStub{status: "delegated"})
		if err != nil {
			t.Fatalf("NewRuntime(): %v", err)
		}
		service, err := runtime.Lifecycle()
		if err != nil {
			t.Fatalf("Lifecycle(): %v", err)
		}
		summary, err := service.GetHealthSummary(context.Background(), "agent-uuid")
		if err != nil || summary.Status != tc.want {
			t.Fatalf("GetHealthSummary() = %#v, %v", summary, err)
		}
	}
}

func TestRuntimeSelectsAgentInvocationAdapter(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeDelegated, lifecycleStub{status: "local"}, lifecycleStub{status: "delegated"})
	if err != nil {
		t.Fatal(err)
	}
	service, err := runtime.Agent()
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Invoke(context.Background(), powerxagent.AgentInvokeRequest{AgentID: "agent-uuid", Message: "hello"})
	if err != nil || result.Message != "delegated" {
		t.Fatalf("Invoke() = %#v, %v", result, err)
	}
}

func TestRuntimeFailsClosedWhenSelectedAdapterMissing(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeDelegated, lifecycleStub{status: "local"}, nil)
	if err != nil {
		t.Fatalf("NewRuntime(): %v", err)
	}
	_, err = runtime.Lifecycle()
	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("Lifecycle() error = %v", err)
	}
}
