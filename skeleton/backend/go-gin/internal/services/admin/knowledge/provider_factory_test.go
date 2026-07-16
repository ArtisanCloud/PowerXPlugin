package knowledge

import (
	"testing"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
)

func TestProviderFactoryBuildsLocal(t *testing.T) {
	provider, err := NewProviderFactory(&config.Config{Knowledge: &config.KnowledgeConfig{Mode: "local", RequireTenant: true}}, nil, nil).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if provider.Mode() != fwknowledge.ProviderModeLocal {
		t.Fatalf("expected local provider, got %s", provider.Mode())
	}
}

func TestProviderFactoryBuildsMock(t *testing.T) {
	provider, err := NewProviderFactory(&config.Config{Knowledge: &config.KnowledgeConfig{Mode: "mock"}}, nil, nil).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if provider.Mode() != fwknowledge.ProviderModeMock {
		t.Fatalf("expected mock provider, got %s", provider.Mode())
	}
}

func TestProviderFactoryBuildsDelegated(t *testing.T) {
	provider, err := NewProviderFactory(&config.Config{Knowledge: &config.KnowledgeConfig{Mode: "delegated", DelegateTimeout: "1s"}}, nil, nil).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if provider.Mode() != fwknowledge.ProviderModeDelegated {
		t.Fatalf("expected delegated provider, got %s", provider.Mode())
	}
}
