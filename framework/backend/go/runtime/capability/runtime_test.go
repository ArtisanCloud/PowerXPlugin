package capability

import (
	"context"
	"errors"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type registryStub struct{ status string }

func (s registryStub) List(context.Context, powerxcapability.ListInput) ([]powerxcapability.Capability, error) {
	return nil, nil
}
func (s registryStub) GrantStatus(context.Context, powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error) {
	return []powerxcapability.GrantStatusItem{{Status: s.status}}, nil
}
func (s registryStub) Resolve(context.Context, powerxcapability.ResolveInput) (*powerxcapability.ResolveResult, error) {
	return nil, nil
}
func (s registryStub) Invoke(context.Context, powerxcapability.InvokeInput) (*powerxcapability.InvokeResult, error) {
	return nil, nil
}
func (s registryStub) GetInvocation(context.Context, string) (*powerxcapability.Invocation, error) {
	return nil, nil
}

func TestRuntimeSelectsOnlyTrustedMode(t *testing.T) {
	for _, tc := range []struct {
		mode provider.Mode
		want string
	}{{provider.ModeLocal, "local"}, {provider.ModeDelegated, "delegated"}} {
		runtime, err := NewRuntime(tc.mode, registryStub{status: "local"}, registryStub{status: "delegated"})
		if err != nil {
			t.Fatalf("NewRuntime(): %v", err)
		}
		registry, err := runtime.Registry()
		if err != nil {
			t.Fatalf("Registry(): %v", err)
		}
		items, err := registry.GrantStatus(context.Background(), powerxcapability.GrantStatusInput{})
		if err != nil || items[0].Status != tc.want {
			t.Fatalf("GrantStatus()=%#v,%v", items, err)
		}
	}
}
func TestRuntimeFailsClosedWhenSelectedAdapterMissing(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeDelegated, registryStub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = runtime.Registry()
	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("Registry() error=%v", err)
	}
}

func TestLocalRuntimeDoesNotUseCoreDebugRegistry(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeLocal, nil, registryStub{status: "granted"})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := runtime.Registry()
	var moduleErr *module.Error
	if registry != nil || !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("registry=%T err=%v", registry, err)
	}
}
