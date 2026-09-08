package knowledge

import (
	"context"
	"testing"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
)

func TestProviderFactoryBuildsLocal(t *testing.T) {
	provider, err := NewProviderFactory(&config.Config{Knowledge: &config.KnowledgeConfig{Mode: "local", RequireTenant: true}}, fwprovider.ModeLocal, nil, nil).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if provider.Mode() != fwknowledge.ProviderModeLocal {
		t.Fatalf("expected local provider, got %s", provider.Mode())
	}
}

func TestProviderFactoryDoesNotLetKnowledgeConfigOverrideProviderMode(t *testing.T) {
	provider, err := NewProviderFactory(&config.Config{Knowledge: &config.KnowledgeConfig{Mode: "delegated"}}, fwprovider.ModeLocal, nil, nil).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if provider.Mode() != fwknowledge.ProviderModeLocal {
		t.Fatalf("expected local provider, got %s", provider.Mode())
	}
}

func TestProviderFactoryBuildsDelegated(t *testing.T) {
	provider, err := NewProviderFactory(&config.Config{Knowledge: &config.KnowledgeConfig{Mode: "delegated", DelegateTimeout: "1s"}}, fwprovider.ModeDelegated, knowledgeDelegatedClientStub{}, nil).Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if provider.Mode() != fwknowledge.ProviderModeDelegated {
		t.Fatalf("expected delegated provider, got %s", provider.Mode())
	}
}

func TestProviderFactoryRejectsDelegatedModeWithoutClient(t *testing.T) {
	_, err := NewProviderFactory(&config.Config{Knowledge: &config.KnowledgeConfig{Mode: "delegated"}}, fwprovider.ModeDelegated, nil, nil).Build()
	if err == nil {
		t.Fatal("expected delegated mode without client to fail")
	}
	if fwknowledge.CodeOf(err) != fwknowledge.CodeProviderUnavailable {
		t.Fatalf("error=%v, want provider unavailable", err)
	}
}

type knowledgeDelegatedClientStub struct{}

func (knowledgeDelegatedClientStub) ListKnowledgeSpaces(context.Context, fwknowledge.ListSpacesInput) ([]fwknowledge.KnowledgeSpace, error) {
	return nil, nil
}
func (knowledgeDelegatedClientStub) GetKnowledgeCatalog(context.Context) (*fwknowledge.KnowledgeCatalog, error) {
	return nil, nil
}
func (knowledgeDelegatedClientStub) SearchKnowledge(context.Context, fwknowledge.KnowledgeQuery) (*fwknowledge.KnowledgeSearchResult, error) {
	return nil, nil
}
func (knowledgeDelegatedClientStub) UpsertKnowledgeDocument(context.Context, fwknowledge.KnowledgeDocument) (*fwknowledge.KnowledgeIndexJob, error) {
	return nil, nil
}
func (knowledgeDelegatedClientStub) DeleteKnowledgeDocument(context.Context, fwknowledge.DeleteDocumentInput) (*fwknowledge.KnowledgeIndexJob, error) {
	return nil, nil
}
func (knowledgeDelegatedClientStub) ReindexKnowledgeDocument(context.Context, fwknowledge.ReindexInput) (*fwknowledge.KnowledgeIndexJob, error) {
	return nil, nil
}
func (knowledgeDelegatedClientStub) GetKnowledgeIndexJob(context.Context, fwknowledge.IndexJobQuery) (*fwknowledge.KnowledgeIndexJob, error) {
	return nil, nil
}
