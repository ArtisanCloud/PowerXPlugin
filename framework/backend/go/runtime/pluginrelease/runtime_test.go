package pluginrelease

import (
	"context"
	"errors"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerx "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/pluginrelease"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type serviceStub struct{ id string }

func (s serviceStub) StartInstallSession(context.Context, powerx.StartInstallSessionInput) (*powerx.InstallSession, error) {
	return &powerx.InstallSession{SessionUUID: s.id}, nil
}
func (s serviceStub) GetInstallSession(context.Context, string) (*powerx.InstallSession, error) {
	return &powerx.InstallSession{SessionUUID: s.id}, nil
}
func (s serviceStub) StopInstallSession(context.Context, string, powerx.StopInstallSessionInput) (*powerx.InstallSession, error) {
	return &powerx.InstallSession{SessionUUID: s.id}, nil
}
func (s serviceStub) StartImportJob(context.Context, powerx.StartImportJobInput) (*powerx.ImportJob, error) {
	return &powerx.ImportJob{JobUUID: s.id}, nil
}
func (s serviceStub) GetImportJob(context.Context, string) (*powerx.ImportJob, error) {
	return &powerx.ImportJob{JobUUID: s.id}, nil
}

func TestRuntimeUsesBootstrapModeAndFailsClosed(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeDelegated, serviceStub{id: "local"}, serviceStub{id: "delegated"})
	if err != nil {
		t.Fatal(err)
	}
	service, err := runtime.Service()
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.StartInstallSession(context.Background(), powerx.StartInstallSessionInput{})
	if err != nil || result.SessionUUID != "delegated" {
		t.Fatalf("StartInstallSession() = %#v, %v", result, err)
	}

	missing, err := NewRuntime(provider.ModeDelegated, serviceStub{id: "local"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = missing.Service()
	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("missing Service() error = %v", err)
	}
}
