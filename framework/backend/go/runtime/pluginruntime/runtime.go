// Package pluginruntime defines the Framework dual-mode contract for
// tenant-scoped PowerX runtime objects.
package pluginruntime

import (
	"context"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxruntime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/pluginruntime"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type Service interface {
	ListKnowledgeSpaces(context.Context, powerxruntime.ListKnowledgeSpacesInput) (*powerxruntime.ListKnowledgeSpacesOutput, error)
	InstantiateAgent(context.Context, powerxruntime.InstantiateAgentInput) (*powerxruntime.Agent, error)
	ListAgents(context.Context, string, string) ([]powerxruntime.Agent, error)
}
type Runtime struct {
	mode    provider.Mode
	service *module.Factory[Service]
}

func NewRuntime(mode provider.Mode, local, delegated Service) (*Runtime, error) {
	service, err := module.NewFactory("plugin_runtime.service", mode, module.Binding[Service]{Value: local, Available: local != nil}, module.Binding[Service]{Value: delegated, Available: delegated != nil})
	if err != nil {
		return nil, err
	}
	return &Runtime{mode: mode, service: service}, nil
}
func (r *Runtime) Mode() provider.Mode {
	if r == nil {
		return ""
	}
	return r.mode
}
func (r *Runtime) Service() (Service, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "plugin runtime is unavailable")
	}
	return r.service.Resolve()
}
