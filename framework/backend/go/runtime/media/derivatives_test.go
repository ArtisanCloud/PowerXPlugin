package media

import (
	powerxmedia "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/media"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"testing"
)

func TestImageDerivativesExtensionKeepsSelectedAdapter(t *testing.T) {
	core, err := powerxmedia.NewHostClientWithAPIKey(powerxmedia.HostClientConfig{BaseURL: "https://core.invalid"}, "controlled", nil)
	if err != nil {
		t.Fatal(err)
	}
	local, _ := NewRuntime(provider.ModeLocal, catalogStub{}, core)
	if _, err := local.ImageDerivatives(); err == nil {
		t.Fatal("LOCAL_DERIVATIVES_FALLBACK_OCCURRED")
	}
	delegated, _ := NewRuntime(provider.ModeDelegated, catalogStub{}, core)
	selected, err := delegated.ImageDerivatives()
	if err != nil || selected != core {
		t.Fatal("DELEGATED_DERIVATIVES_BINDING_CHANGED")
	}
	host, err := NewCoreHostRuntime(core)
	if err != nil || host.Channel() != "core_host" || host.Mode() != provider.ModeLocal {
		t.Fatal("CORE_HOST_MEDIA_BINDING_INVALID")
	}
	selected, err = host.ImageDerivatives()
	if err != nil || selected != core {
		t.Fatal("CORE_HOST_MEDIA_EXTENSION_NOT_BOUND")
	}
	if _, err := NewCoreHostRuntime(nil); err == nil {
		t.Fatal("NIL_CORE_HOST_ACCEPTED")
	}
}
