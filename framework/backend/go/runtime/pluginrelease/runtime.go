package pluginrelease

import (
	"context"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerx "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/pluginrelease"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type Service interface {
	StartInstallSession(context.Context, powerx.StartInstallSessionInput) (*powerx.InstallSession, error)
	GetInstallSession(context.Context, string) (*powerx.InstallSession, error)
	StopInstallSession(context.Context, string, powerx.StopInstallSessionInput) (*powerx.InstallSession, error)
	StartImportJob(context.Context, powerx.StartImportJobInput) (*powerx.ImportJob, error)
	GetImportJob(context.Context, string) (*powerx.ImportJob, error)
}
type Runtime struct{ service *module.Factory[Service] }

func NewRuntime(mode provider.Mode, local, delegated Service) (*Runtime, error) {
	factory, err := module.NewFactory("plugin_release.service", mode, module.Binding[Service]{Value: local, Available: local != nil}, module.Binding[Service]{Value: delegated, Available: delegated != nil})
	if err != nil {
		return nil, err
	}
	return &Runtime{service: factory}, nil
}
func (r *Runtime) Service() (Service, error) {
	if r == nil || r.service == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "plugin release runtime is unavailable")
	}
	return r.service.Resolve()
}

var _ Service = (*powerx.Client)(nil)
