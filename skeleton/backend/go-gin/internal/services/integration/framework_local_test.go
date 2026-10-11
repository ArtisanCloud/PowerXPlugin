package integration

import (
	"context"
	"sync"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	capdto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	intdto "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/integration"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLocalFrameworkCapabilityAndIntegration(t *testing.T) {
	db, _ := setupTemplateService(t)
	plugin := "com.powerx.plugins.base"
	entries := []capabilities.CatalogEntry{}
	for _, action := range []string{"create", "list", "read", "update", "delete"} {
		entries = append(entries, capabilities.CatalogEntry{ID: plugin + ".template." + action, ProviderPluginID: plugin})
	}
	entries = append(entries, capabilities.CatalogEntry{ID: "com.corex.iam.directory.read"})
	r, e := NewLocalCapabilityRuntime(plugin, entries, db)
	require.NoError(t, e)
	ctx := authx.ContextWithTenantUUID(context.Background(), uuid.NewString())
	other := authx.ContextWithTenantUUID(context.Background(), uuid.NewString())
	catalog, e := r.List(ctx, capdto.ListInput{})
	require.NoError(t, e)
	require.Len(t, catalog, 5)
	_, e = r.List(context.Background(), capdto.ListInput{})
	require.Error(t, e)
	resolved, e := r.Resolve(ctx, capdto.ResolveInput{Method: "INVOKE", Endpoint: catalog[0].Protocols[0].Endpoint})
	require.NoError(t, e)
	require.Equal(t, catalog[0].CapabilityID, resolved.CapabilityID)
	input := capdto.InvokeInput{CapabilityID: plugin + ".template.create", IdempotencyKey: "create-one", Payload: map[string]any{"name": "local-contract", "description": "fixture", "content": "fixture"}}
	created, e := r.Invoke(ctx, input)
	require.NoError(t, e)
	object := created.Payload["template"].(map[string]any)
	id := object["uuid"].(string)
	require.NotEmpty(t, id)
	require.NotContains(t, object, "id")
	replay, e := r.Invoke(ctx, input)
	require.NoError(t, e)
	require.Equal(t, created.TraceID, replay.TraceID)
	var wg sync.WaitGroup
	for n := 0; n < 16; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := r.Invoke(ctx, input)
			if err != nil || got == nil || got.TraceID != replay.TraceID {
				t.Errorf("concurrent replay: %v", err)
			}
		}()
	}
	wg.Wait()
	created.Payload["template"].(map[string]any)["uuid"] = "mutated"
	replay, e = r.Invoke(ctx, input)
	require.NoError(t, e)
	require.Equal(t, id, replay.Payload["template"].(map[string]any)["uuid"])
	input.Payload["name"] = "changed"
	_, e = r.Invoke(ctx, input)
	require.Error(t, e)
	record, e := r.GetInvocation(ctx, replay.TraceID)
	require.NoError(t, e)
	require.Equal(t, "succeeded", record.Status)
	_, e = r.GetInvocation(other, replay.TraceID)
	require.Error(t, e)
	routes, e := r.ListRoutes(ctx, intdto.ListRoutesInput{CapabilityID: plugin + ".template.read"})
	require.NoError(t, e)
	require.Len(t, routes, 1)
	result, e := r.InvokeRoute(ctx, routes[0].RouteUUID, intdto.InvokeRouteInput{Payload: map[string]any{"template_uuid": id}})
	require.NoError(t, e)
	require.Equal(t, id, result.Result["template"].(map[string]any)["uuid"])
	_, e = r.GetRoute(other, routes[0].RouteUUID)
	require.Error(t, e)
	_, e = r.Invoke(ctx, capdto.InvokeInput{CapabilityID: "com.corex.iam.directory.read"})
	require.Error(t, e)
	_, e = r.Invoke(ctx, capdto.InvokeInput{CapabilityID: plugin + ".template.read", Payload: map[string]any{"template_id": 123}})
	require.Error(t, e)
	_, e = r.Invoke(ctx, capdto.InvokeInput{CapabilityID: plugin + ".template.read", Payload: map[string]any{"template_uuid": id, "tenant_uuid": uuid.NewString()}})
	require.Error(t, e)
	_, e = r.GrantStatus(ctx, capdto.GrantStatusInput{CapabilityIDs: []string{plugin + ".template.read"}})
	var contractErr *hostapi.HTTPError
	require.ErrorAs(t, e, &contractErr)
	require.Equal(t, "CAPABILITY_LOCAL_GRANT_STATUS_UNAVAILABLE", contractErr.ReasonCode)
}
