package knowledge

import (
	"context"
	"time"
)

type DelegatedClient interface {
	ListKnowledgeSpaces(ctx context.Context, input ListSpacesInput) ([]KnowledgeSpace, error)
	SearchKnowledge(ctx context.Context, query KnowledgeQuery) (*KnowledgeSearchResult, error)
	UpsertKnowledgeDocument(ctx context.Context, document KnowledgeDocument) (*KnowledgeIndexJob, error)
	DeleteKnowledgeDocument(ctx context.Context, input DeleteDocumentInput) (*KnowledgeIndexJob, error)
	ReindexKnowledgeDocument(ctx context.Context, input ReindexInput) (*KnowledgeIndexJob, error)
}

type CatalogDelegatedClient interface {
	GetKnowledgeCatalog(ctx context.Context) (*KnowledgeCatalog, error)
}

// CapabilityDelegatedClient declares the operations actually exposed by the
// Host Contract. Without this declaration, delegated providers expose no
// business operation rather than guessing from a client implementation.
type CapabilityDelegatedClient interface {
	DelegatedCapabilities(ctx context.Context) ProviderCapabilities
}

type DelegatedProviderConfig struct {
	Name    string
	Client  DelegatedClient
	Timeout time.Duration
}

type DelegatedProvider struct {
	name    string
	client  DelegatedClient
	timeout time.Duration
}

func NewDelegatedProvider(cfg DelegatedProviderConfig) *DelegatedProvider {
	name := cfg.Name
	if name == "" {
		name = "powerx_delegated"
	}
	return &DelegatedProvider{name: name, client: cfg.Client, timeout: cfg.Timeout}
}

func (p *DelegatedProvider) Name() string { return p.name }

func (p *DelegatedProvider) Mode() string { return ProviderModeDelegated }

func (p *DelegatedProvider) Capabilities(ctx context.Context) ProviderCapabilities {
	if client, ok := p.client.(CapabilityDelegatedClient); ok && client != nil {
		caps := client.DelegatedCapabilities(ctx)
		caps.Provider = p.name
		caps.Mode = ProviderModeDelegated
		return caps
	}
	return BasicCapabilities(p.name, ProviderModeDelegated, OperationHealth)
}

func (p *DelegatedProvider) Catalog(ctx context.Context) (*KnowledgeCatalog, error) {
	if err := RequireCapability(p.Capabilities(ctx), OperationCatalog); err != nil {
		return nil, err
	}
	client, ok := p.client.(CatalogDelegatedClient)
	if !ok || client == nil {
		return nil, &Error{Code: CodeUnsupportedCapability, Message: "knowledge delegated catalog client is unavailable", Provider: p.name, Operation: OperationCatalog}
	}
	callCtx, cancel := p.withTimeout(ctx)
	defer cancel()
	catalog, err := client.GetKnowledgeCatalog(callCtx)
	if err != nil {
		return nil, p.mapError(OperationCatalog, "", err)
	}
	if catalog == nil {
		return nil, &Error{Code: CodeProviderUnavailable, Message: "knowledge delegated catalog response is empty", Provider: p.name, Operation: OperationCatalog, Retryable: true}
	}
	return catalog, nil
}

func (p *DelegatedProvider) ListSpaces(ctx context.Context, input ListSpacesInput) ([]KnowledgeSpace, error) {
	if err := RequireCapability(p.Capabilities(ctx), OperationRetrieve); err != nil {
		return nil, err
	}
	if p.client == nil {
		return nil, &Error{Code: CodeProviderUnavailable, Message: "knowledge delegated provider unavailable", Provider: p.name, Operation: OperationRetrieve, Retryable: true, TraceID: input.TraceID}
	}
	input = input.Normalized()
	callCtx, cancel := p.withTimeout(ctx)
	defer cancel()
	spaces, err := p.client.ListKnowledgeSpaces(callCtx, input)
	if err != nil {
		return nil, p.mapError(OperationRetrieve, input.TraceID, err)
	}
	if spaces == nil {
		spaces = []KnowledgeSpace{}
	}
	return spaces, nil
}

func (p *DelegatedProvider) Search(ctx context.Context, query KnowledgeQuery) (*KnowledgeSearchResult, error) {
	if err := RequireCapability(p.Capabilities(ctx), OperationSearch); err != nil {
		return nil, err
	}
	if p.client == nil {
		return nil, &Error{Code: CodeProviderUnavailable, Message: "knowledge delegated provider unavailable", Provider: p.name, Operation: OperationSearch, Retryable: true, TraceID: query.TraceID}
	}
	query = query.Normalized()
	if err := query.Validate(TenantRequiredForVisibility(query.Visibility)); err != nil {
		return nil, err
	}
	callCtx, cancel := p.withTimeout(ctx)
	defer cancel()
	result, err := p.client.SearchKnowledge(callCtx, query)
	if err != nil {
		return nil, p.mapError(OperationSearch, query.TraceID, err)
	}
	return NormalizeSearchResult(p, query, result)
}

func (p *DelegatedProvider) UpsertDocument(ctx context.Context, document KnowledgeDocument) (*KnowledgeIndexJob, error) {
	if err := RequireCapability(p.Capabilities(ctx), OperationUpsert); err != nil {
		return nil, err
	}
	if p.client == nil {
		return nil, &Error{Code: CodeProviderUnavailable, Message: "knowledge delegated provider unavailable", Provider: p.name, Operation: OperationUpsert, Retryable: true}
	}
	document, err := ValidateDocument(document)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := p.withTimeout(ctx)
	defer cancel()
	job, err := p.client.UpsertKnowledgeDocument(callCtx, document)
	if err != nil {
		return nil, p.mapError(OperationUpsert, "", err)
	}
	return job, nil
}

func (p *DelegatedProvider) DeleteDocument(ctx context.Context, input DeleteDocumentInput) (*KnowledgeIndexJob, error) {
	if err := RequireCapability(p.Capabilities(ctx), OperationDelete); err != nil {
		return nil, err
	}
	if p.client == nil {
		return nil, &Error{Code: CodeProviderUnavailable, Message: "knowledge delegated provider unavailable", Provider: p.name, Operation: OperationDelete, Retryable: true}
	}
	callCtx, cancel := p.withTimeout(ctx)
	defer cancel()
	job, err := p.client.DeleteKnowledgeDocument(callCtx, input)
	if err != nil {
		return nil, p.mapError(OperationDelete, input.TraceID, err)
	}
	return job, nil
}

func (p *DelegatedProvider) Reindex(ctx context.Context, input ReindexInput) (*KnowledgeIndexJob, error) {
	if err := RequireCapability(p.Capabilities(ctx), OperationReindex); err != nil {
		return nil, err
	}
	if p.client == nil {
		return nil, &Error{Code: CodeProviderUnavailable, Message: "knowledge delegated provider unavailable", Provider: p.name, Operation: OperationReindex, Retryable: true}
	}
	callCtx, cancel := p.withTimeout(ctx)
	defer cancel()
	job, err := p.client.ReindexKnowledgeDocument(callCtx, input)
	if err != nil {
		return nil, p.mapError(OperationReindex, input.TraceID, err)
	}
	return job, nil
}

func (p *DelegatedProvider) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if p.timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, p.timeout)
}

func (p *DelegatedProvider) mapError(operation, traceID string, err error) error {
	if fwerr, ok := err.(*Error); ok {
		fwerr.Provider = p.name
		fwerr.Operation = operation
		if fwerr.TraceID == "" {
			fwerr.TraceID = traceID
		}
		return fwerr
	}
	if err == context.DeadlineExceeded {
		return &Error{Code: CodeProviderUnavailable, Message: "knowledge delegated provider timeout", Provider: p.name, Operation: operation, Retryable: true, TraceID: traceID, Cause: err}
	}
	return &Error{Code: CodeProviderUnavailable, Message: RedactString(err.Error()), Provider: p.name, Operation: operation, Retryable: true, TraceID: traceID, Cause: err}
}
