// Package integration defines the dual-mode Framework boundary for tenant
// Integration Gateway route discovery and invocation.
package integration

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxintegration "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/integration"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type Gateway interface {
	ListRoutes(context.Context, powerxintegration.ListRoutesInput) ([]powerxintegration.RouteSummary, error)
	GetRoute(context.Context, string) (*powerxintegration.RouteDetail, error)
	InvokeRoute(context.Context, string, powerxintegration.InvokeRouteInput) (*powerxintegration.InvokeRouteOutput, error)
}

type Runtime struct {
	mode    provider.Mode
	gateway *module.Factory[Gateway]
}

func NewRuntime(mode provider.Mode, local, delegated Gateway) (*Runtime, error) {
	gateway, err := module.NewFactory("integration.gateway", mode, module.Binding[Gateway]{Value: local, Available: local != nil}, module.Binding[Gateway]{Value: delegated, Available: delegated != nil})
	if err != nil {
		return nil, err
	}
	return &Runtime{mode: mode, gateway: gateway}, nil
}
func (r *Runtime) Mode() provider.Mode {
	if r == nil {
		return ""
	}
	return r.mode
}
func (r *Runtime) Gateway() (Gateway, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "integration runtime is unavailable")
	}
	return r.gateway.Resolve()
}
