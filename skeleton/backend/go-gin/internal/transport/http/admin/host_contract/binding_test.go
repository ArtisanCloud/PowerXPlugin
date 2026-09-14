package host_contract

import (
	"encoding/json"
	ai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ai"
	knowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	media "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/media"
	provider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"testing"
)

func TestStatusChecksSelectedBindingWithoutBusinessCalls(t *testing.T) {
	for _, mode := range []provider.Mode{provider.ModeLocal, provider.ModeDelegated} {
		mr, _ := media.NewRuntime(mode, nil, nil)
		ar, _ := ai.NewRuntime(mode, nil, nil)
		h := NewHandler(&app.Deps{ProviderMode: mode, MediaCatalog: mr, AIInvocation: ar})
		for _, name := range []string{"cache", "taskcenter", "iam", "knowledge", "media", "ai", "agent", "capability_registry", "integration_gateway", "skills", "notifications", "plugin_runtime"} {
			w := executeProbe(t, h, map[string]any{"module": name, "operation": "status"})
			if w.Code != 200 {
				t.Fatalf("%s %s %d %s", mode, name, w.Code, w.Body.String())
			}
			var out struct {
				Data struct {
					Reason string `json:"reason_code"`
					Result struct {
						Available bool `json:"adapter_available"`
						Verified  bool `json:"connectivity_verified"`
					} `json:"result"`
				} `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
				t.Fatal(err)
			}
			if out.Data.Reason != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" || out.Data.Result.Available || out.Data.Result.Verified {
				t.Fatal(w.Body.String())
			}
		}
		w := executeProbe(t, h, map[string]any{"module": "media", "operation": "assets.list"})
		if w.Code != 503 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
func TestKnowledgeLocalProbeUsesSelectedProvider(t *testing.T) {
	h := NewHandler(&app.Deps{ProviderMode: provider.ModeLocal, KnowledgeProvider: knowledge.NewLocalProvider(knowledge.LocalProviderConfig{RequireTenant: true})})
	for _, op := range []string{"status", "spaces.list"} {
		w := executeProbe(t, h, map[string]any{"module": "knowledge", "operation": op})
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
