package runtimeexample

import (
	"context"
	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"net/http"
	"testing"
)

func TestLocalAISettingsShareExecutionConfiguration(t *testing.T) {
	s := localAITest(t, func(http.ResponseWriter, *http.Request) { t.Error("settings read must not invoke model") })
	ctx := authx.ContextWithTenantUUID(context.Background(), tenant)
	profiles, e := s.ModelProfiles(ctx, "")
	if e != nil || len(profiles) != 1 {
		t.Fatalf("profiles: %v %v", profiles, e)
	}
	models, e := s.ListLLMModels(ctx, "")
	if e != nil || profiles[0]["model_key"] != models.Items[0].ModelKey || profiles[0]["model"] != models.Items[0].Model {
		t.Fatalf("configuration drift: %v", e)
	}
	profiles[0]["modalities"].([]string)[0] = "mutated"
	again, e := s.ModelProfiles(ctx, "")
	if e != nil || again[0]["modalities"].([]string)[0] != "llm" {
		t.Fatal("mutable snapshot")
	}
	h, e := s.Health(ctx, "")
	if e != nil || h["status"] != "not_verified" {
		t.Fatalf("health: %v %v", h, e)
	}
	empty, e := NewLocalAI(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	h, e = empty.Health(ctx, "")
	if e != nil || h["status"] != "not_configured" {
		t.Fatalf("empty: %v %v", h, e)
	}
	if _, e = s.ModelProfiles(context.Background(), ""); e == nil {
		t.Fatal("missing tenant accepted")
	}
}
