// Package skills defines the mode-bound Framework contract for tenant Skill
// invocation.
package skills

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxskills "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/skills"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// Invoker is the shared business boundary for a tenant-scoped Skill call.
// Implementations must derive tenancy and caller authority from their trusted
// runtime context, never from InvokeInput.
type Invoker interface {
	Invoke(context.Context, powerxskills.InvokeInput) (*powerxskills.InvokeOutput, error)
}

type Runtime struct {
	mode    provider.Mode
	invoker *module.Factory[Invoker]
}

func NewRuntime(mode provider.Mode, local, delegated Invoker) (*Runtime, error) {
	invoker, err := module.NewFactory("skills.invoker", mode,
		module.Binding[Invoker]{Value: local, Available: local != nil},
		module.Binding[Invoker]{Value: delegated, Available: delegated != nil},
	)
	if err != nil {
		return nil, err
	}
	return &Runtime{mode: mode, invoker: invoker}, nil
}

func (r *Runtime) Mode() provider.Mode {
	if r == nil {
		return ""
	}
	return r.mode
}

func (r *Runtime) Invoker() (Invoker, error) {
	if r == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "skills runtime is unavailable")
	}
	return r.invoker.Resolve()
}
