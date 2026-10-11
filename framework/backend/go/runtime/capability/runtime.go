// Package capability defines the Framework dual-mode boundary for tenant
// capability discovery, invocation, and effective-grant inspection.
package capability

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// Registry intentionally matches the typed Core contract so local adapters can
// be plugin supplied without exposing transport or credential details.
type Registry interface {
	List(context.Context, powerxcapability.ListInput) ([]powerxcapability.Capability, error)
	GrantStatus(context.Context, powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error)
	Resolve(context.Context, powerxcapability.ResolveInput) (*powerxcapability.ResolveResult, error)
	Invoke(context.Context, powerxcapability.InvokeInput) (*powerxcapability.InvokeResult, error)
	GetInvocation(context.Context, string) (*powerxcapability.Invocation, error)
}

type Runtime struct {
	mode     provider.Mode
	registry *module.Factory[Registry]
}

func NewRuntime(mode provider.Mode, local, delegated Registry) (*Runtime, error) {
	registry, err := module.NewFactory("capability.registry", mode,
		module.Binding[Registry]{Value: local, Available: local != nil},
		module.Binding[Registry]{Value: delegated, Available: delegated != nil},
	)
	if err != nil {
		return nil, err
	}
	return &Runtime{mode: mode, registry: registry}, nil
}
func (r *Runtime) Mode() provider.Mode {
	if r == nil {
		return ""
	}
	return r.mode
}
func (r *Runtime) Registry() (Registry, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "capability runtime is unavailable")
	}
	return r.registry.Resolve()
}
