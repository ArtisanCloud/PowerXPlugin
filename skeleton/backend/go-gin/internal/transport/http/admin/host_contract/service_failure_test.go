package host_contract

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	fwcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/capability"
	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	hostcontract "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

func TestCapabilityProbePreservesTypedServiceDiagnostics(t *testing.T) {
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		_, _ = fmt.Fprint(w, `{"error_code":"CAPABILITY_PLUGIN_RUNTIME_VERSION_MISMATCH","reason_code":"CAPABILITY_PLUGIN_RUNTIME_VERSION_MISMATCH","request_id":"core-request","details":{"trace_id":"core-trace","expected_provider_version":"1.2.3","actual_provider_version":"1.2.2","private":"not-for-ui"}}`)
	}))
	defer core.Close()
	client, err := powerxcapability.NewClient(powerxcapability.Config{BaseURL: core.URL, AuthScheme: "apikey", APIKey: "selected-key"}, core.Client())
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := fwcapability.NewRuntime(provider.ModeDelegated, nil, client)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(&app.Deps{CapabilityAccess: runtime, ProviderMode: provider.ModeDelegated})
	r := executeProbe(t, h, map[string]any{"module": "capability_registry", "operation": "capabilities.list", "input": map[string]any{}})
	var envelope struct {
		Error hostcontract.ErrorEnvelope `json:"error"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if r.Code != 409 || envelope.Error.ErrorCode != "CAPABILITY_PLUGIN_RUNTIME_VERSION_MISMATCH" || envelope.Error.ReasonCode != "CAPABILITY_PLUGIN_RUNTIME_VERSION_MISMATCH" || envelope.Error.TraceID != "core-trace" || envelope.Error.RequestID != "core-request" || envelope.Error.Details["expected_provider_version"] != "1.2.3" || envelope.Error.Details["actual_provider_version"] != "1.2.2" || envelope.Error.Details["private"] != nil {
		t.Fatal("SERVICE_DIAGNOSTICS_LOST", r.Code, envelope.Error)
	}
}
