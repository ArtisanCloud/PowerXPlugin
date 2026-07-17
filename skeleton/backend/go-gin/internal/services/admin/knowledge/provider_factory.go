package knowledge

import (
	"context"
	"strings"
	"time"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
)

type Factory struct {
	cfg    *config.Config
	client fwknowledge.DelegatedClient
	logger *pxlog.Entry
}

func NewProviderFactory(cfg *config.Config, client fwknowledge.DelegatedClient, logger *pxlog.Entry) *Factory {
	if logger == nil {
		logger = pxlog.WithComponent("knowledge.provider_factory")
	}
	return &Factory{cfg: cfg, client: client, logger: logger}
}

func (f *Factory) Build() (fwknowledge.KnowledgeProvider, error) {
	cfg := &config.Config{}
	if f != nil && f.cfg != nil {
		cfg = f.cfg
	}
	if cfg.Logging == nil {
		cfg.Logging = &config.LoggingConfig{DebugMode: true}
	}
	if err := cfg.NormalizeKnowledgeConfig(); err != nil {
		return nil, err
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.Knowledge.Mode))
	switch mode {
	case fwknowledge.ProviderModeDelegated, fwknowledge.ProviderModeThirdParty:
		timeout, _ := time.ParseDuration(cfg.Knowledge.DelegateTimeout)
		return fwknowledge.NewDelegatedProvider(fwknowledge.DelegatedProviderConfig{
			Name:    "powerx_delegated",
			Client:  f.delegatedClient(),
			Timeout: timeout,
		}), nil
	case fwknowledge.ProviderModeMock:
		return fwknowledge.NewMockProvider(), nil
	default:
		return fwknowledge.NewLocalProvider(fwknowledge.LocalProviderConfig{RequireTenant: cfg.Knowledge.RequireTenant}), nil
	}
}

func (f *Factory) delegatedClient() fwknowledge.DelegatedClient {
	if f != nil && f.client != nil {
		return f.client
	}
	return unavailableDelegatedClient{}
}

type unavailableDelegatedClient struct{}

func (unavailableDelegatedClient) ListKnowledgeSpaces(context.Context, fwknowledge.ListSpacesInput) ([]fwknowledge.KnowledgeSpace, error) {
	return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "knowledge delegated client is not configured")
}

func (unavailableDelegatedClient) GetKnowledgeCatalog(context.Context) (*fwknowledge.KnowledgeCatalog, error) {
	return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "knowledge delegated client is not configured")
}

func (unavailableDelegatedClient) SearchKnowledge(context.Context, fwknowledge.KnowledgeQuery) (*fwknowledge.KnowledgeSearchResult, error) {
	return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "knowledge delegated client is not configured")
}

func (unavailableDelegatedClient) UpsertKnowledgeDocument(context.Context, fwknowledge.KnowledgeDocument) (*fwknowledge.KnowledgeIndexJob, error) {
	return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "knowledge delegated client is not configured")
}

func (unavailableDelegatedClient) DeleteKnowledgeDocument(context.Context, fwknowledge.DeleteDocumentInput) (*fwknowledge.KnowledgeIndexJob, error) {
	return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "knowledge delegated client is not configured")
}

func (unavailableDelegatedClient) ReindexKnowledgeDocument(context.Context, fwknowledge.ReindexInput) (*fwknowledge.KnowledgeIndexJob, error) {
	return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "knowledge delegated client is not configured")
}
