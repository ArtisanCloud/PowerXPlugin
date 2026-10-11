package pluginruntime

import (
	"context"
	"errors"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxruntime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/pluginruntime"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
)

type runtimeStub struct{ uuid string }

func (s runtimeStub) ListKnowledgeSpaces(context.Context, powerxruntime.ListKnowledgeSpacesInput) (*powerxruntime.ListKnowledgeSpacesOutput, error) {
	return &powerxruntime.ListKnowledgeSpacesOutput{Items: []powerxruntime.KnowledgeSpace{{UUID: s.uuid}}}, nil
}
func (runtimeStub) InstantiateAgent(context.Context, powerxruntime.InstantiateAgentInput) (*powerxruntime.Agent, error) {
	return nil, nil
}
func (runtimeStub) ListAgents(context.Context, string, string) ([]powerxruntime.Agent, error) {
	return nil, nil
}
func TestRuntimeSelectsOnlyTrustedMode(t *testing.T) {
	for _, tc := range []struct {
		mode provider.Mode
		want string
	}{{provider.ModeLocal, "local"}, {provider.ModeDelegated, "delegated"}} {
		runtime, err := NewRuntime(tc.mode, runtimeStub{"local"}, runtimeStub{"delegated"})
		if err != nil {
			t.Fatal(err)
		}
		service, err := runtime.Service()
		if err != nil {
			t.Fatal(err)
		}
		out, err := service.ListKnowledgeSpaces(context.Background(), powerxruntime.ListKnowledgeSpacesInput{})
		if err != nil || out.Items[0].UUID != tc.want {
			t.Fatalf("ListKnowledgeSpaces=%#v,%v", out, err)
		}
	}
}
func TestRuntimeFailsClosedWhenSelectedAdapterMissing(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeDelegated, runtimeStub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = runtime.Service()
	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("Service error=%v", err)
	}
}
