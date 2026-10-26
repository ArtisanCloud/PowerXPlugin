package notifications

import (
	core "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
)

func TestCardExtensionUsesOnlySelectedAdapter(t *testing.T) {
	client, err := core.NewClient(core.Config{BaseURL: "https://core.test", AuthScheme: "apikey", APIKey: "controlled"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	local, _ := NewRuntime(provider.ModeLocal, publisherStub{}, client)
	if _, err := local.CardInvalidator(); err == nil {
		t.Fatal("LOCAL_CARD_ADAPTER_MUST_FAIL")
	}
	if _, err := local.ReadAcknowledger(); err == nil {
		t.Fatal("LOCAL_READ_ADAPTER_MUST_FAIL")
	}
	delegated, _ := NewRuntime(provider.ModeDelegated, publisherStub{}, client)
	invalidator, err := delegated.CardInvalidator()
	if err != nil || invalidator != client {
		t.Fatal("DELEGATED_BINDING_CHANGED")
	}
	host, err := NewCoreHostRuntime(client)
	if err != nil || host.Channel() != "core_host" || host.Mode() != provider.ModeLocal {
		t.Fatal("CORE_HOST_BINDING_INVALID")
	}
	invalidator, err = host.CardInvalidator()
	if err != nil || invalidator != client {
		t.Fatal("CORE_HOST_BINDING_CHANGED")
	}
	if _, err := NewCoreHostRuntime(nil); err == nil {
		t.Fatal("NIL_CORE_HOST_ACCEPTED")
	}
}
