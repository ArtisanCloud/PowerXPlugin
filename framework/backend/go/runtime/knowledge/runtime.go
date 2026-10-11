package knowledge

import (
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// Runtime selects the explicitly supplied provider at bootstrap. Plugin-owned
// local persistence is never constructed as a delegated failure fallback.
type Runtime struct {
	knowledge *module.Factory[KnowledgeProvider]
}

func NewRuntime(mode provider.Mode, local, delegated KnowledgeProvider) (*Runtime, error) {
	factory, err := module.NewFactory("knowledge.provider", mode,
		module.Binding[KnowledgeProvider]{Value: local, Available: local != nil},
		module.Binding[KnowledgeProvider]{Value: delegated, Available: delegated != nil})
	if err != nil {
		return nil, err
	}
	return &Runtime{knowledge: factory}, nil
}

func (r *Runtime) Provider() (KnowledgeProvider, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "knowledge.runtime_unavailable")
	}
	return r.knowledge.Resolve()
}

func (r *Runtime) Mode() provider.Mode {
	if r == nil || r.knowledge == nil {
		return ""
	}
	return r.knowledge.Mode()
}
