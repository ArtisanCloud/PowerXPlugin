package runtimeexample

import (
	"context"
	"testing"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	powerxruntime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/pluginruntime"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
)

func TestLocalPluginRuntimeCreatesAndScopesAgents(t *testing.T) {
	ai := localAITest(t, nil)
	knowledge := fwknowledge.NewLocalProvider(fwknowledge.LocalProviderConfig{RequireTenant: true})
	runtime, err := NewLocalPluginRuntime(ai, knowledge)
	if err != nil {
		t.Fatal(err)
	}
	ctx := authx.ContextWithTenantUUID(context.Background(), tenant)
	created, err := runtime.InstantiateAgent(ctx, powerxruntime.InstantiateAgentInput{Name: "Local agent", Parameters: map[string]any{"model_key": "chat"}})
	if err != nil {
		t.Fatal(err)
	}
	if created.UUID == "" || created.Source != "local" || created.Parameters["model_key"] != "chat" {
		t.Fatalf("created=%+v", created)
	}
	items, err := runtime.ListAgents(ctx, "local", "active")
	if err != nil || len(items) != 1 || items[0].UUID != created.UUID {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	otherCtx := authx.ContextWithTenantUUID(context.Background(), other)
	items, err = runtime.ListAgents(otherCtx, "", "")
	if err != nil || len(items) != 0 {
		t.Fatalf("cross-tenant items=%+v err=%v", items, err)
	}
}

func TestLocalPluginRuntimeRequiresConfiguredModel(t *testing.T) {
	ai := localAITest(t, nil)
	runtime, err := NewLocalPluginRuntime(ai, fwknowledge.NewLocalProvider(fwknowledge.LocalProviderConfig{RequireTenant: true}))
	if err != nil {
		t.Fatal(err)
	}
	ctx := authx.ContextWithTenantUUID(context.Background(), tenant)
	if _, err = runtime.InstantiateAgent(ctx, powerxruntime.InstantiateAgentInput{Name: "Local agent"}); err == nil {
		t.Fatal("expected model_key requirement")
	}
	if _, err = runtime.InstantiateAgent(ctx, powerxruntime.InstantiateAgentInput{Name: "Local agent", Parameters: map[string]any{"model_key": "missing"}}); err == nil {
		t.Fatal("expected unknown model rejection")
	}
}
