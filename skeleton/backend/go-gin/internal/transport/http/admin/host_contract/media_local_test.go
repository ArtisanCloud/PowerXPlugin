package host_contract

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	media "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/media"
	provider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/runtimeexample"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"testing"
)

func TestLocalMediaProbeCreatesAndListsRealRecord(t *testing.T) {
	runtime, err := media.NewRuntime(provider.ModeLocal, runtimeexample.NewLocalMedia("http://127.0.0.1:8078"), nil)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(&app.Deps{MediaCatalog: runtime, ProviderMode: provider.ModeLocal})
	input := map[string]any{"name": "local-test", "mime_type": "application/octet-stream", "size_bytes": 0, "checksum": fmt.Sprintf("%x", sha256.Sum256(nil))}
	body := map[string]any{"module": "media", "operation": "asset.create", "input": input, "confirm": false}
	if w := executeProbe(t, h, body); w.Code != 409 {
		t.Fatal(w.Code, w.Body.String())
	}
	body["confirm"] = true
	w := executeProbe(t, h, body)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct {
		Data struct {
			Result struct {
				Output struct {
					UUID string `json:"asset_uuid"`
				} `json:"output"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	id := result.Data.Result.Output.UUID
	if id == "" {
		t.Fatal(w.Body.String())
	}
	w = executeProbe(t, h, map[string]any{"module": "media", "operation": "asset.get", "input": map[string]any{"asset_uuid": id}})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = executeProbe(t, h, map[string]any{"module": "media", "operation": "assets.list"})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
}
