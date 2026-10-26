package host_contract

import (
	"encoding/json"
	"fmt"
	fwmedia "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/media"
	core "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/media"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostDerivativeOperationsDispatchAndDiagnostics(t *testing.T) {
	asset := "11111111-1111-4111-8111-111111111111"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var in struct {
			Payload struct {
				Body map[string]any `json:"body"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		if r.URL.Path != "/api/v2/tenant/invocations" || r.Header.Get("Authorization") != "ApiKey controlled" || in.Payload.Body["asset_uuid"] != asset {
			t.Fatal("HOST_MEDIA_BINDING_INVALID")
		}
		if in.Payload.Body["operation"] == "ensure" {
			w.WriteHeader(403)
			fmt.Fprint(w, `{"error_code":"MEDIA_FORBIDDEN","reason_code":"MEDIA_FORBIDDEN","trace_id":"media-trace","request_id":"media-request","details":{"grant":"denied"}}`)
			return
		}
		fmt.Fprintf(w, `{"data":{"protocol_used":"core_internal","fallback_used":false,"payload":{"result":{"asset_uuid":%q,"upload_state":"ready","status":"not_requested","recipe":"standard-image-v2","attempts":0,"retryable":false,"items":[{"variant":"preview","status":"not_requested"},{"variant":"thumbnail","status":"not_requested"},{"variant":"thumbnail_64","status":"not_requested"}]}}}}`, asset)
	}))
	defer server.Close()
	client, err := core.NewHostClientWithAPIKey(core.HostClientConfig{BaseURL: server.URL, APIPrefix: "/api/v2"}, "controlled", nil)
	if err != nil {
		t.Fatal(err)
	}
	runtime, _ := fwmedia.NewRuntime(provider.ModeDelegated, nil, client)
	handler := NewHandler(&app.Deps{MediaCatalog: runtime})
	response := executeProbe(t, handler, map[string]any{"module": "media", "operation": "derivatives.status", "input": map[string]any{"asset_uuid": asset}})
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	request := map[string]any{"module": "media", "operation": "derivatives.ensure", "input": map[string]any{"asset_uuid": asset, "retry_failed": true}}
	if r := executeProbe(t, handler, request); r.Code != 409 || calls != 1 {
		t.Fatal("ENSURE_CONFIRMATION_BYPASSED")
	}
	request["confirm"] = true
	response = executeProbe(t, handler, request)
	var out struct {
		Error struct {
			Code    string `json:"error_code"`
			Trace   string `json:"trace_id"`
			Request string `json:"request_id"`
			Status  int    `json:"http_status"`
		} `json:"error"`
	}
	_ = json.Unmarshal(response.Body.Bytes(), &out)
	if response.Code != 403 || out.Error.Code != "MEDIA_FORBIDDEN" || out.Error.Trace != "media-trace" || out.Error.Request != "media-request" || out.Error.Status != 403 || calls != 2 {
		t.Fatal("HOST_MEDIA_DIAGNOSTICS_LOST")
	}
	if runtime.Mode() != provider.ModeDelegated {
		t.Fatal("MEDIA_CHANNEL_CHANGED_AFTER_FAILURE")
	}
}
