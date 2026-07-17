package knowledge

import "context"

type ProviderCapabilities struct {
	Provider   string         `json:"provider"`
	Mode       string         `json:"mode"`
	Operations []string       `json:"operations"`
	Limits     map[string]any `json:"limits,omitempty"`
	Health     string         `json:"health,omitempty"`
}

type ProviderInfo struct {
	Name         string               `json:"name"`
	Mode         string               `json:"mode"`
	Capabilities ProviderCapabilities `json:"capabilities"`
}

type KnowledgeProvider interface {
	Name() string
	Mode() string
	Capabilities(ctx context.Context) ProviderCapabilities
	ListSpaces(ctx context.Context, input ListSpacesInput) ([]KnowledgeSpace, error)
	Catalog(ctx context.Context) (*KnowledgeCatalog, error)
	Search(ctx context.Context, query KnowledgeQuery) (*KnowledgeSearchResult, error)
	UpsertDocument(ctx context.Context, document KnowledgeDocument) (*KnowledgeIndexJob, error)
	DeleteDocument(ctx context.Context, input DeleteDocumentInput) (*KnowledgeIndexJob, error)
	Reindex(ctx context.Context, input ReindexInput) (*KnowledgeIndexJob, error)
}

func Supports(caps ProviderCapabilities, operation string) bool {
	for _, op := range caps.Operations {
		if op == operation {
			return true
		}
	}
	return false
}

func RequireCapability(caps ProviderCapabilities, operation string) error {
	if Supports(caps, operation) {
		return nil
	}
	err := Unsupported(operation)
	err.Provider = caps.Provider
	return err
}

func BasicCapabilities(provider, mode string, operations ...string) ProviderCapabilities {
	return ProviderCapabilities{
		Provider:   provider,
		Mode:       mode,
		Operations: append([]string(nil), operations...),
		Limits:     map[string]any{},
		Health:     "ready",
	}
}
