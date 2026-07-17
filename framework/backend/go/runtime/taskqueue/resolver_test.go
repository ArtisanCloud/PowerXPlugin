package taskqueue

import (
	"testing"

	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

func TestResolveProviderModeProxySelectsHostLink(t *testing.T) {
	result := ResolveProviderMode(ResolveInput{
		PowerXProxy: "1",
		EnvMode:     string(fwprovider.ModeLocal),
	})
	if result.Mode != LinkModeHost {
		t.Fatalf("mode=%s want=%s", result.Mode, LinkModeHost)
	}
	if result.Source != "env:POWERX_PROXY" {
		t.Fatalf("source=%s", result.Source)
	}
}

func TestResolveProviderModeDelegatedWithoutProxyKeepsLocalLink(t *testing.T) {
	result := ResolveProviderMode(ResolveInput{
		PowerXProxy: "0",
		EnvMode:     string(fwprovider.ModeDelegated),
	})
	if result.Mode != LinkModeLocal {
		t.Fatalf("mode=%s want=%s", result.Mode, LinkModeLocal)
	}
}

func TestResolveProviderModeConfigDelegatedWithoutProxyKeepsLocalLink(t *testing.T) {
	result := ResolveProviderMode(ResolveInput{
		PowerXProxy:        "0",
		ConfigProviderMode: fwprovider.ModeDelegated,
	})
	if result.Mode != LinkModeLocal {
		t.Fatalf("mode=%s want=%s", result.Mode, LinkModeLocal)
	}
}
