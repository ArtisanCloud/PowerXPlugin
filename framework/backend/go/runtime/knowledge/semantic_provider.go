package knowledge

import "context"

const (
	OperationSemanticCapabilities = "semantic_capabilities"
	OperationSemanticQuery        = "semantic_query"
	OperationSemanticConfigure    = "semantic_configure"
	OperationDocumentVisibility   = "document_visibility"
)

type SemanticKnowledgeProvider interface {
	GetSemanticCapabilities(context.Context, string) (*SemanticCapabilities, error)
	ConfigureSemanticIndex(context.Context, string, SemanticConfigureInput) (*SemanticGeneration, error)
	QuerySemanticKnowledge(context.Context, SemanticQuery) (*SemanticResult, error)
	SetDocumentVisibility(context.Context, string, string, SemanticVisibilityInput) (*SemanticVisibility, error)
}
type SemanticDelegatedClient = SemanticKnowledgeProvider

func SemanticProvider(provider KnowledgeProvider) (SemanticKnowledgeProvider, error) {
	service, ok := provider.(SemanticKnowledgeProvider)
	if !ok || service == nil {
		return nil, Unsupported(OperationSemanticQuery)
	}
	return service, nil
}
func (p *DelegatedProvider) semanticClient(ctx context.Context, operation string) (SemanticDelegatedClient, error) {
	if err := RequireCapability(p.Capabilities(ctx), operation); err != nil {
		return nil, err
	}
	service, ok := p.client.(SemanticDelegatedClient)
	if !ok || service == nil {
		return nil, Unsupported(operation)
	}
	return service, nil
}
func (p *DelegatedProvider) GetSemanticCapabilities(ctx context.Context, space string) (*SemanticCapabilities, error) {
	client, err := p.semanticClient(ctx, OperationSemanticCapabilities)
	if err != nil {
		return nil, err
	}
	ctx, cancel := p.withTimeout(ctx)
	defer cancel()
	result, err := client.GetSemanticCapabilities(ctx, space)
	if err != nil {
		return nil, p.mapError(OperationSemanticCapabilities, "", err)
	}
	return result, nil
}
func (p *DelegatedProvider) ConfigureSemanticIndex(ctx context.Context, space string, input SemanticConfigureInput) (*SemanticGeneration, error) {
	client, err := p.semanticClient(ctx, OperationSemanticConfigure)
	if err != nil {
		return nil, err
	}
	ctx, cancel := p.withTimeout(ctx)
	defer cancel()
	result, err := client.ConfigureSemanticIndex(ctx, space, input)
	if err != nil {
		return nil, p.mapError(OperationSemanticConfigure, "", err)
	}
	return result, nil
}
func (p *DelegatedProvider) QuerySemanticKnowledge(ctx context.Context, input SemanticQuery) (*SemanticResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	client, err := p.semanticClient(ctx, OperationSemanticQuery)
	if err != nil {
		return nil, err
	}
	ctx, cancel := p.withTimeout(ctx)
	defer cancel()
	result, err := client.QuerySemanticKnowledge(ctx, input)
	if err != nil {
		return nil, p.mapError(OperationSemanticQuery, "", err)
	}
	return result, nil
}
func (p *DelegatedProvider) SetDocumentVisibility(ctx context.Context, space, document string, input SemanticVisibilityInput) (*SemanticVisibility, error) {
	client, err := p.semanticClient(ctx, OperationDocumentVisibility)
	if err != nil {
		return nil, err
	}
	ctx, cancel := p.withTimeout(ctx)
	defer cancel()
	result, err := client.SetDocumentVisibility(ctx, space, document, input)
	if err != nil {
		return nil, p.mapError(OperationDocumentVisibility, "", err)
	}
	return result, nil
}
func (p *LocalProvider) GetSemanticCapabilities(context.Context, string) (*SemanticCapabilities, error) {
	return nil, Unsupported(OperationSemanticCapabilities)
}
func (p *LocalProvider) ConfigureSemanticIndex(context.Context, string, SemanticConfigureInput) (*SemanticGeneration, error) {
	return nil, Unsupported(OperationSemanticConfigure)
}
func (p *LocalProvider) QuerySemanticKnowledge(context.Context, SemanticQuery) (*SemanticResult, error) {
	return nil, Unsupported(OperationSemanticQuery)
}
func (p *LocalProvider) SetDocumentVisibility(context.Context, string, string, SemanticVisibilityInput) (*SemanticVisibility, error) {
	return nil, Unsupported(OperationDocumentVisibility)
}
