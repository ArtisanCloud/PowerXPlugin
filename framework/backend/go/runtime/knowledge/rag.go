package knowledge

import "context"

type RAGContext struct {
	TenantUUID string
	PluginID   string
	AgentUUID  string
	SkillID    string
	CallerType string
	Locale     string
	TraceID    string
	Visibility string
}

type RAGRetriever struct {
	Provider KnowledgeProvider
}

func NewRAGRetriever(provider KnowledgeProvider) *RAGRetriever {
	return &RAGRetriever{Provider: provider}
}

func (r *RAGRetriever) Retrieve(ctx context.Context, ragCtx RAGContext, question string, opts ...func(*KnowledgeQuery)) (*KnowledgeSearchResult, error) {
	if r == nil || r.Provider == nil {
		return nil, NewError(CodeProviderUnavailable, "knowledge provider unavailable")
	}
	query := KnowledgeQuery{
		Query:      question,
		TenantUUID: ragCtx.TenantUUID,
		PluginID:   ragCtx.PluginID,
		AgentUUID:  ragCtx.AgentUUID,
		SkillID:    ragCtx.SkillID,
		CallerType: ragCtx.CallerType,
		Locale:     ragCtx.Locale,
		TraceID:    ragCtx.TraceID,
		Visibility: ragCtx.Visibility,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&query)
		}
	}
	result, err := r.Provider.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return NormalizeSearchResult(r.Provider, query, result)
}
