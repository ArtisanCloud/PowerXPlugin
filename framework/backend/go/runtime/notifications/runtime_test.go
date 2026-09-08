package notifications

import (
	"context"
	"errors"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	powerxnotifications "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

type publisherStub struct{ uuid string }

func (s publisherStub) Create(context.Context, powerxnotifications.CreateInput) (*powerxnotifications.Notification, error) {
	return &powerxnotifications.Notification{UUID: s.uuid}, nil
}

func TestRuntimeSelectsOnlyTrustedMode(t *testing.T) {
	for _, tc := range []struct {
		mode provider.Mode
		want string
	}{{provider.ModeLocal, "local"}, {provider.ModeDelegated, "delegated"}} {
		t.Run(string(tc.mode), func(t *testing.T) {
			runtime, err := NewRuntime(tc.mode, publisherStub{uuid: "local"}, publisherStub{uuid: "delegated"})
			if err != nil {
				t.Fatalf("NewRuntime(): %v", err)
			}
			publisher, err := runtime.Publisher()
			if err != nil {
				t.Fatalf("Publisher(): %v", err)
			}
			notification, err := publisher.Create(context.Background(), powerxnotifications.CreateInput{Title: "title", Content: "content"})
			if err != nil || notification.UUID != tc.want {
				t.Fatalf("Create() = %#v, %v", notification, err)
			}
		})
	}
}

func TestRuntimeFailsClosedWhenSelectedAdapterMissing(t *testing.T) {
	runtime, err := NewRuntime(provider.ModeDelegated, publisherStub{uuid: "local"}, nil)
	if err != nil {
		t.Fatalf("NewRuntime(): %v", err)
	}
	_, err = runtime.Publisher()
	var moduleErr *module.Error
	if !errors.As(err, &moduleErr) || moduleErr.Code != "FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE" {
		t.Fatalf("Publisher() error=%v", err)
	}
}
