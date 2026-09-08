package skills

import (
	"context"
	"errors"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxskills "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/skills"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type invokerStub struct{ status string }

func (s invokerStub) Invoke(context.Context, powerxskills.InvokeInput) (*powerxskills.InvokeOutput, error) {
	return &powerxskills.InvokeOutput{Status: s.status}, nil
}

func TestRuntimeSelectsOnlyTrustedMode(t *testing.T) {
	for _, tc := range []struct {
		mode provider.Mode
		want string
	}{{provider.ModeLocal, "local"}, {provider.ModeDelegated, "delegated"}} {
		t.Run(string(tc.mode), func(t *testing.T) {
			runtime, err := NewRuntime(tc.mode, invokerStub{status: "local"}, invokerStub{status: "delegated"})
			if err != nil {
				t.Fatalf("NewRuntime(): %v", err)
			}
			invoker, err := runtime.Invoker()
			if err != nil {
				t.Fatalf("Invoker(): %v", err)
			}
			result, err := invoker.Invoke(context.Background(), powerxskills.InvokeInput{SkillID: "skill-uuid"})
			if err != nil || result.Status != tc.want {
				t.Fatalf("Invoke() = %#v, %v", result, err)
			}
		})
	}
}

func TestRuntimeFailsClosedWhenSelectedAdapterMissing(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeDelegated, invokerStub{status: "local"}, nil)
	if err != nil {
		t.Fatalf("NewRuntime(): %v", err)
	}
	_, err = runtime.Invoker()
	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("Invoker() error=%v", err)
	}
}
