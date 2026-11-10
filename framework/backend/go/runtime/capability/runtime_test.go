package capability

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestRuntimeStreamUsesSelectedRegistryAndNeverFallsBack(t *testing.T) {
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tenant/invocations/stream" || r.Header.Get("Authorization") != "ApiKey selected-key" {
			t.Error("SELECTED_STREAM_CHANNEL_CHANGED")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: end\ndata: {}\n\n")
	}))
	defer core.Close()
	client, err := powerxcapability.NewClient(powerxcapability.Config{BaseURL: core.URL, AuthScheme: "apikey", APIKey: "selected-key"}, core.Client())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := NewRuntime(provider.ModeDelegated, registryStub{}, client)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := runtime.Streamer()
	if err != nil {
		t.Fatal(err)
	}
	events := 0
	if err := stream.InvokeStream(context.Background(), powerxcapability.StreamInput{CapabilityID: "com.corex.ai.llm.stream", Body: map[string]any{}}, func(powerxcapability.StreamEvent) error { events++; return nil }); err != nil || events != 1 {
		t.Fatal("SELECTED_STREAM_UNAVAILABLE", err, events)
	}
	local, err := NewRuntime(provider.ModeLocal, registryStub{}, client)
	if err != nil {
		t.Fatal(err)
	}
	if stream, err := local.Streamer(); stream != nil || err == nil {
		t.Fatal("LOCAL_STREAM_FELL_BACK_TO_CORE")
	}
}

type countingRegistry struct {
	registryStub
	calls *int
}

func (s countingRegistry) Invoke(context.Context, powerxcapability.InvokeInput) (*powerxcapability.InvokeResult, error) {
	*s.calls++
	return nil, nil
}

func TestDelegatedHTTPDenialDoesNotInvokeLocal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tenant/invocations" || r.Header.Get("Authorization") != "Bearer test-sts" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"reason_code":"TEST_GRANT_DENIED"}`))
	}))
	defer server.Close()
	client, err := powerxcapability.NewClientWithTokenProvider(powerxcapability.Config{BaseURL: server.URL}, powerxcapability.TokenProviderFunc(func(context.Context) (string, error) { return "test-sts", nil }), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	runtime, err := NewRuntime(provider.ModeDelegated, countingRegistry{calls: &calls}, client)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := runtime.Registry()
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.Invoke(context.Background(), powerxcapability.InvokeInput{CapabilityID: "com.test.operation"})
	var upstream *powerxcapability.HTTPError
	if !errors.As(err, &upstream) || upstream.StatusCode != 403 || upstream.ReasonCode != "TEST_GRANT_DENIED" || calls != 0 {
		t.Fatalf("err=%v local_calls=%d", err, calls)
	}
}
