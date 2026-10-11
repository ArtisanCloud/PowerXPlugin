package host_contract

import (
	"encoding/json"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/runtimeexample"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"testing"
)

func TestStorageProbeWorkflow(t *testing.T) {
	cache, tasks, err := runtimeexample.Build(provider.ModeLocal, hostapi.Config{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(&app.Deps{CacheRuntime: cache, TaskCenterRuntime: tasks, ProviderMode: provider.ModeLocal})
	invoke := func(module, op string, input map[string]any, confirm bool, status int) map[string]any {
		t.Helper()
		w := executeProbe(t, h, map[string]any{"module": module, "operation": op, "input": input, "confirm": confirm})
		if w.Code != status {
			t.Fatalf("%s %s: %d %s", module, op, w.Code, w.Body.String())
		}
		var out map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	in := map[string]any{"namespace": "test", "key": "k", "value_base64": "", "ttl_ms": 10000}
	invoke("cache", "set", in, false, 409)
	invoke("cache", "set", in, true, 200)
	out := invoke("cache", "get", map[string]any{"namespace": "test", "key": "k"}, false, 200)
	if !out["data"].(map[string]any)["result"].(map[string]any)["found"].(bool) {
		t.Fatal(out)
	}
	invoke("cache", "delete", map[string]any{"namespace": "test", "key": "k"}, true, 200)
	invoke("cache", "get", map[string]any{"namespace": "test", "key": "k", "tenant_uuid": probeTenantUUID}, false, 400)
	out = invoke("taskcenter", "create", map[string]any{"type": "test", "idempotency_key": "once", "payload": nil}, true, 200)
	task := out["data"].(map[string]any)["result"].(map[string]any)["task"].(map[string]any)
	id := task["task_uuid"].(string)
	invoke("taskcenter", "get", map[string]any{"task_uuid": id}, false, 200)
	update := map[string]any{"task_uuid": id, "expected_revision": 1, "state": "running", "progress": 10}
	invoke("taskcenter", "update", update, true, 200)
	invoke("taskcenter", "update", update, true, 409)
}
