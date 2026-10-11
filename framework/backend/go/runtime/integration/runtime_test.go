package integration

import (
	"context"
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxintegration "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/integration"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"net/http"
	"net/http/httptest"
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

type rejectingGateway struct {
	gatewayStub
	calls *int
	err   error
}

func (s rejectingGateway) InvokeRoute(context.Context, string, powerxintegration.InvokeRouteInput) (*powerxintegration.InvokeRouteOutput, error) {
	*s.calls++
	return nil, s.err
}

func TestLocalRuntimeCannotUseDelegatedGateway(t *testing.T) {
	calls := 0
	runtime, err := NewRuntime(provider.ModeLocal, nil, rejectingGateway{calls: &calls})
	if err != nil {
		t.Fatal(err)
	}
	gateway, err := runtime.Gateway()
	if gateway != nil || err == nil || calls != 0 {
		t.Fatalf("gateway=%T err=%v calls=%d", gateway, err, calls)
	}
}

func TestDelegatedHTTPDenialDoesNotInvokeLocal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tenant/integration/routes/test-route/invoke" || r.Header.Get("Authorization") != "Bearer test-sts" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"reason_code":"TEST_GRANT_DENIED"}`))
	}))
	defer server.Close()
	client, err := powerxintegration.NewClientWithTokenProvider(powerxintegration.Config{BaseURL: server.URL}, powerxintegration.TokenProviderFunc(func(context.Context) (string, error) { return "test-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	runtime, err := NewRuntime(provider.ModeDelegated, rejectingGateway{calls: &calls}, client)
	if err != nil {
		t.Fatal(err)
	}
	gateway, err := runtime.Gateway()
	if err != nil {
		t.Fatal(err)
	}
	_, err = gateway.InvokeRoute(context.Background(), "test-route", powerxintegration.InvokeRouteInput{Payload: map[string]any{"operation": "test"}})
	var upstream *powerxintegration.HTTPError
	if !errors.As(err, &upstream) || upstream.StatusCode != 403 || upstream.ReasonCode != "TEST_GRANT_DENIED" || calls != 0 {
		t.Fatalf("err=%v local_calls=%d", err, calls)
	}
}
