package integration

import (
	"context"
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxintegration "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/integration"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
)

type gatewayStub struct{ slug string }

func (s gatewayStub) ListRoutes(context.Context, powerxintegration.ListRoutesInput) ([]powerxintegration.RouteSummary, error) {
	return []powerxintegration.RouteSummary{{RouteSlug: s.slug}}, nil
}
func (gatewayStub) GetRoute(context.Context, string) (*powerxintegration.RouteDetail, error) {
	return nil, nil
}
func (gatewayStub) InvokeRoute(context.Context, string, powerxintegration.InvokeRouteInput) (*powerxintegration.InvokeRouteOutput, error) {
	return nil, nil
}
func TestRuntimeSelectsOnlyTrustedMode(t *testing.T) {
	for _, tc := range []struct {
		mode provider.Mode
		want string
	}{{provider.ModeLocal, "local"}, {provider.ModeDelegated, "delegated"}} {
		runtime, err := NewRuntime(tc.mode, gatewayStub{"local"}, gatewayStub{"delegated"})
		if err != nil {
			t.Fatal(err)
		}
		gateway, err := runtime.Gateway()
		if err != nil {
			t.Fatal(err)
		}
		routes, err := gateway.ListRoutes(context.Background(), powerxintegration.ListRoutesInput{})
		if err != nil || routes[0].RouteSlug != tc.want {
			t.Fatalf("ListRoutes=%#v,%v", routes, err)
		}
	}
}
func TestRuntimeFailsClosedWhenSelectedAdapterMissing(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeDelegated, gatewayStub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = runtime.Gateway()
	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("Gateway error=%v", err)
	}
}
