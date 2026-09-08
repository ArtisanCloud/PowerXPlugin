// Package agent provides mode-bound Agent lifecycle and service-session
// contracts. It keeps plugin business code independent from PowerX transport.
package agent

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/agent"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// LifecycleService includes lifecycle control and existing standalone invoke
// operations. The Core delegated adapter rejects these old Invoke/StreamSSE
// shapes: service-actor execution must use SessionService. Local plugins inject
// their implementation; no request can select the runtime mode.
type LifecycleService interface {
	Invoke(context.Context, powerxagent.AgentInvokeRequest) (powerxagent.AgentInvokeResponse, error)
	StreamSSE(context.Context, url.Values, func(powerxagent.AgentStreamEvent) error) error
	GetHealthSummary(context.Context, string) (*powerxagent.HealthSummary, error)
	ListHealthHistory(context.Context, string, int, int) (*powerxagent.HealthHistory, error)
	GetBridgeState(context.Context, string, int) (*json.RawMessage, error)
	Freeze(context.Context, string, powerxagent.BridgeControlInput) (*powerxagent.BridgeLifecycleResult, error)
	Recover(context.Context, string, powerxagent.BridgeControlInput) (*powerxagent.BridgeLifecycleResult, error)
	Rebalance(context.Context, string, powerxagent.BridgeRebalanceInput) (*powerxagent.BridgeLifecycleResult, error)
}

// AgentService is the preferred name for new plugin code.
type AgentService = LifecycleService

type SessionService = powerxagent.SessionService
type RuntimeOption func(*Runtime) error

// WithSessions binds the independent Session contract without requiring local
// lifecycle implementations to implement a different business sub-contract.
func WithSessions(local, delegated SessionService) RuntimeOption {
	return func(r *Runtime) error {
		var err error
		r.sessions, err = module.NewFactory("agent.sessions", r.mode,
			module.Binding[SessionService]{Value: local, Available: local != nil},
			module.Binding[SessionService]{Value: delegated, Available: delegated != nil})
		return err
	}
}

type Runtime struct {
	mode      provider.Mode
	lifecycle *module.Factory[LifecycleService]
	sessions  *module.Factory[SessionService]
}

// NewRuntime binds supplied local/delegated implementations to one trusted
// startup mode. A missing implementation fails at consumption time.
func NewRuntime(mode provider.Mode, local, delegated LifecycleService, options ...RuntimeOption) (*Runtime, error) {
	lifecycle, err := module.NewFactory("agent.lifecycle", mode,
		module.Binding[LifecycleService]{Value: local, Available: local != nil},
		module.Binding[LifecycleService]{Value: delegated, Available: delegated != nil},
	)
	if err != nil {
		return nil, err
	}
	r := &Runtime{mode: mode, lifecycle: lifecycle}
	for _, option := range options {
		if option != nil {
			if err := option(r); err != nil {
				return nil, err
			}
		}
	}
	return r, nil
}

func (r *Runtime) Sessions() (SessionService, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "agent.sessions")
	}
	return r.sessions.Resolve()
}

func (r *Runtime) Mode() provider.Mode {
	if r == nil {
		return ""
	}
	return r.mode
}

func (r *Runtime) Lifecycle() (LifecycleService, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "agent runtime is unavailable")
	}
	return r.lifecycle.Resolve()
}

// Agent returns lifecycle and legacy standalone invocation operations.
// Service-actor execution uses Sessions(), not the old human-session inputs.
func (r *Runtime) Agent() (AgentService, error) { return r.Lifecycle() }
