package host_contract

import (
	fwagent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/agent"
	fwai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ai"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/runtimeexample"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"testing"
)

func TestLocalAIAndAgentProbeUsesSelectedRuntime(t *testing.T) {
	ai, e := runtimeexample.NewLocalAI(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	agent, e := runtimeexample.NewLocalAgent("test.plugin", nil, ai)
	if e != nil {
		t.Fatal(e)
	}
	ar, e := fwai.NewRuntime(provider.ModeLocal, ai, nil)
	if e != nil {
		t.Fatal(e)
	}
	ag, e := fwagent.NewRuntime(provider.ModeLocal, agent, nil, fwagent.WithSessions(agent, nil))
	if e != nil {
		t.Fatal(e)
	}
	h := NewHandler(&app.Deps{AIInvocation: ar, AgentLifecycle: ag, ProviderMode: provider.ModeLocal})
	for _, tc := range []struct {
		module, op string
		input      map[string]any
		confirm    bool
		status     int
	}{
		{"ai", "models.list", nil, false, 200},
		{"agent", "sessions.list", nil, false, 200},
		{"agent", "sessions.list", map[string]any{"unknown": 1}, false, 400},
		{"ai", "llm.invoke", map[string]any{"model_key": "missing"}, false, 409},
		{"ai", "llm.invoke", map[string]any{"model_key": "missing"}, true, 503},
		{"agent", "session.create", map[string]any{"agent_uuid": "numeric"}, true, 400},
	} {
		t.Run(tc.module+tc.op, func(t *testing.T) {
			w := executeProbe(t, h, map[string]any{"module": tc.module, "operation": tc.op, "input": tc.input, "confirm": tc.confirm})
			if w.Code != tc.status {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
		})
	}
}
