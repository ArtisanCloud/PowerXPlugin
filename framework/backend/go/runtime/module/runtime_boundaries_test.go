package module_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/agent"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/ai"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/capability"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/integration"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/media"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/metadata"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/notifications"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/pluginrelease"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/pluginruntime"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/skills"
)

// Exercise real exported accessors, not a mock Factory, so new module wrappers
// cannot accidentally reintroduce nil dereferences or return a usable adapter.
func TestUninitializedRuntimesFailClosed(t *testing.T) {
	for _, tc := range []struct {
		runtime   any
		accessors []string
	}{
		{(*agent.Runtime)(nil), []string{"Agent", "Lifecycle", "Sessions"}},
		{(*ai.Runtime)(nil), []string{"Generative"}},
		{(*capability.Runtime)(nil), []string{"Registry"}},
		{(*customerfw.Runtime)(nil), []string{"Auth", "ExternalIdentity", "Membership"}},
		{(*integration.Runtime)(nil), []string{"Gateway"}},
		{(*knowledge.Runtime)(nil), []string{"Provider"}},
		{(*media.Runtime)(nil), []string{"Assets", "Media"}},
		{(*metadata.Runtime)(nil), []string{"Service"}},
		{(*notifications.Runtime)(nil), []string{"Publisher"}},
		{(*pluginrelease.Runtime)(nil), []string{"Service"}},
		{(*pluginruntime.Runtime)(nil), []string{"Service"}},
		{(*skills.Runtime)(nil), []string{"Invoker"}},
	} {
		for _, value := range []reflect.Value{reflect.ValueOf(tc.runtime), reflect.New(reflect.TypeOf(tc.runtime).Elem())} {
			state := "zero"
			if value.IsNil() {
				state = "nil"
			}
			if mode := value.MethodByName("Mode"); mode.IsValid() {
				t.Run(value.Type().String()+"/"+state+"/Mode", func(t *testing.T) {
					if mode.Call(nil)[0].String() != "" {
						t.Fatal("uninitialized runtime reported configured mode")
					}
				})
			}
			for _, accessor := range tc.accessors {
				t.Run(value.Type().String()+"/"+state+"/"+accessor, func(t *testing.T) {
					results := value.MethodByName(accessor).Call(nil)
					if len(results) != 2 || !results[0].IsNil() || results[1].IsNil() {
						t.Fatalf("accessor returned usable result: %v", results)
					}
					err := results[1].Interface().(error)
					var moduleErr *module.Error
					if !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
						t.Fatalf("err=%v", err)
					}
				})
			}
		}
	}
}

func TestNilFactoryMode(t *testing.T) {
	var factory *module.Factory[any]
	if factory.Mode() != "" {
		t.Fatal("nil factory reported configured mode")
	}
	if _, err := factory.Resolve(); err == nil {
		t.Fatal("nil factory resolved")
	}
}
