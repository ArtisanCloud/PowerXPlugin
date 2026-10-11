package metadata

import (
	"context"
	"strings"

	fwmetadata "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/metadata"
)

// FrameworkLocalAdapter is the only local metadata implementation injected
// into metadata.Runtime. Tenant scope comes from trusted request context, so
// its public DTOs stay identical to the delegated Host contract.
type FrameworkLocalAdapter struct{ service *Service }

func NewFrameworkLocalAdapter(service *Service) *FrameworkLocalAdapter {
	if service == nil {
		return nil
	}
	return &FrameworkLocalAdapter{service: service}
}

func (a *FrameworkLocalAdapter) tenant(ctx context.Context) (string, error) {
	if a == nil || a.service == nil {
		return "", &fwmetadata.Error{Code: fwmetadata.CodeClientUnavailable, Message: "metadata local adapter is not configured"}
	}
	tenant, ok := fwmetadata.TenantUUIDFromContext(ctx)
	if !ok || strings.TrimSpace(tenant) == "" {
		return "", &fwmetadata.Error{Code: fwmetadata.CodeInvalidRequest, Message: "metadata tenant scope is required"}
	}
	return tenant, nil
}
func unavailable(operation string) error {
	return &fwmetadata.Error{Code: fwmetadata.CodeOperationUnavailable, Message: "metadata local operation is not implemented", Operation: operation}
}

func (a *FrameworkLocalAdapter) ListDictionaryNamespaces(ctx context.Context, r fwmetadata.ListDictionaryNamespacesRequest) (*fwmetadata.Page[fwmetadata.DictionaryNamespace], error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.ListDictionaryNamespaces(ctx, ListOptions{TenantUUID: t, Module: r.Module, Status: r.Status, Query: r.Query, Locale: r.Locale, Page: r.Page, PageSize: r.PageSize})
}
func (a *FrameworkLocalAdapter) CreateDictionaryNamespace(ctx context.Context, r fwmetadata.CreateDictionaryNamespaceRequest) (*fwmetadata.DictionaryNamespace, error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.CreateDictionaryNamespace(ctx, CreateDictionaryNamespaceInput{TenantUUID: t, Namespace: r.Namespace, Module: r.Module, NameI18n: r.NameI18n, DescriptionI18n: r.DescriptionI18n})
}
func (a *FrameworkLocalAdapter) UpdateDictionaryNamespace(context.Context, fwmetadata.UpdateDictionaryNamespaceRequest) (*fwmetadata.DictionaryNamespace, error) {
	return nil, unavailable("metadata.dictionary.update")
}
func (a *FrameworkLocalAdapter) ResolveDictionaryNamespace(context.Context, string) (*fwmetadata.DictionaryNamespace, error) {
	return nil, unavailable("metadata.dictionary.resolve")
}
func (a *FrameworkLocalAdapter) ListDictionaryItems(ctx context.Context, r fwmetadata.ListDictionaryItemsRequest) (*fwmetadata.Page[fwmetadata.DictionaryItem], error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.ListDictionaryItems(ctx, r.NamespaceUUID, ListOptions{TenantUUID: t, Status: r.Status, Query: r.Query, Locale: r.Locale, Page: r.Page, PageSize: r.PageSize})
}
func (a *FrameworkLocalAdapter) CreateDictionaryItem(ctx context.Context, r fwmetadata.CreateDictionaryItemRequest) (*fwmetadata.DictionaryItem, error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.CreateDictionaryItem(ctx, CreateDictionaryItemInput{TenantUUID: t, NamespaceUUID: r.NamespaceUUID, Code: r.Code, LabelI18n: r.LabelI18n, DescriptionI18n: r.DescriptionI18n, SortOrder: r.SortOrder})
}
func (a *FrameworkLocalAdapter) UpdateDictionaryItem(context.Context, fwmetadata.UpdateDictionaryItemRequest) (*fwmetadata.DictionaryItem, error) {
	return nil, unavailable("metadata.dictionary_item.update")
}
func (a *FrameworkLocalAdapter) ResolveDictionaryItem(context.Context, string, string) (*fwmetadata.DictionaryItem, error) {
	return nil, unavailable("metadata.dictionary_item.resolve")
}
func (a *FrameworkLocalAdapter) ListTaxonomies(ctx context.Context, r fwmetadata.ListTaxonomiesRequest) (*fwmetadata.Page[fwmetadata.Taxonomy], error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.ListTaxonomies(ctx, ListOptions{TenantUUID: t, Module: r.Module, Status: r.Status, Query: r.Query, Locale: r.Locale, Page: r.Page, PageSize: r.PageSize})
}
func (a *FrameworkLocalAdapter) CreateTaxonomy(ctx context.Context, r fwmetadata.CreateTaxonomyRequest) (*fwmetadata.Taxonomy, error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.CreateTaxonomy(ctx, CreateTaxonomyInput{TenantUUID: t, Namespace: r.Namespace, Module: r.Module, NameI18n: r.NameI18n, DescriptionI18n: r.DescriptionI18n, MaxDepth: r.MaxDepth})
}
func (a *FrameworkLocalAdapter) ResolveTaxonomy(context.Context, string) (*fwmetadata.Taxonomy, error) {
	return nil, unavailable("metadata.taxonomy.resolve")
}
func (a *FrameworkLocalAdapter) ListTaxonomyNodes(ctx context.Context, r fwmetadata.ListTaxonomyNodesRequest) (*fwmetadata.Page[fwmetadata.TaxonomyNode], error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.ListTaxonomyNodes(ctx, r.TaxonomyUUID, ListOptions{TenantUUID: t, Status: r.Status, Query: r.Query, Locale: r.Locale, Page: r.Page, PageSize: r.PageSize})
}
func (a *FrameworkLocalAdapter) CreateTaxonomyNode(ctx context.Context, r fwmetadata.CreateTaxonomyNodeRequest) (*fwmetadata.TaxonomyNode, error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.CreateTaxonomyNode(ctx, CreateTaxonomyNodeInput{TenantUUID: t, TaxonomyUUID: r.TaxonomyUUID, ParentUUID: r.ParentUUID, Code: r.Code, LabelI18n: r.LabelI18n, DescriptionI18n: r.DescriptionI18n, SortOrder: r.SortOrder})
}
func (a *FrameworkLocalAdapter) UpdateTaxonomyNode(ctx context.Context, r fwmetadata.UpdateTaxonomyNodeRequest) (*fwmetadata.TaxonomyNode, error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.UpdateTaxonomyNode(ctx, UpdateTaxonomyNodeInput{TenantUUID: t, NodeUUID: r.NodeUUID, LabelI18n: i18nPtr(r.LabelI18n), DescriptionI18n: i18nPtr(r.DescriptionI18n), SortOrder: r.SortOrder, Status: r.Status, Version: r.Version})
}
func (a *FrameworkLocalAdapter) ResolveTaxonomyNode(context.Context, string, string) (*fwmetadata.TaxonomyNode, error) {
	return nil, unavailable("metadata.taxonomy_node.resolve")
}
func (a *FrameworkLocalAdapter) ListTags(ctx context.Context, r fwmetadata.ListTagsRequest) (*fwmetadata.Page[fwmetadata.Tag], error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.ListTags(ctx, ListOptions{TenantUUID: t, Namespace: r.Namespace, ResourceType: r.ResourceType, Status: r.Status, Query: r.Query, Locale: r.Locale, Page: r.Page, PageSize: r.PageSize})
}
func (a *FrameworkLocalAdapter) CreateTag(ctx context.Context, r fwmetadata.CreateTagRequest) (*fwmetadata.Tag, error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.CreateTag(ctx, CreateTagInput{TenantUUID: t, Namespace: r.Namespace, ResourceType: r.ResourceType, Code: r.Code, Color: r.Color, LabelI18n: r.LabelI18n, DescriptionI18n: r.DescriptionI18n})
}
func (a *FrameworkLocalAdapter) UpdateTag(ctx context.Context, r fwmetadata.UpdateTagRequest) (*fwmetadata.Tag, error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.UpdateTag(ctx, UpdateTagInput{TenantUUID: t, TagUUID: r.TagUUID, LabelI18n: i18nPtr(r.LabelI18n), DescriptionI18n: i18nPtr(r.DescriptionI18n), Color: r.Color, Status: r.Status})
}
func (a *FrameworkLocalAdapter) ResolveTag(context.Context, string, string, string) (*fwmetadata.Tag, error) {
	return nil, unavailable("metadata.tag.resolve")
}
func (a *FrameworkLocalAdapter) ListTagBindings(ctx context.Context, r fwmetadata.ListTagBindingsRequest) ([]fwmetadata.TagBinding, error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.ListTagBindings(ctx, ListTagBindingsInput{TenantUUID: t, ResourceType: r.ResourceType, ResourceUUID: r.ResourceUUID, Locale: r.Locale})
}
func (a *FrameworkLocalAdapter) ReplaceTagBindings(ctx context.Context, r fwmetadata.ReplaceTagBindingsRequest) ([]fwmetadata.TagBinding, error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.ReplaceTagBindings(ctx, ReplaceTagBindingsInput{TenantUUID: t, ResourceType: r.ResourceType, ResourceUUID: r.ResourceUUID, TagUUIDs: r.TagUUIDs})
}
func (a *FrameworkLocalAdapter) ListResourceTypes(ctx context.Context, r fwmetadata.ListResourceTypesRequest) (*fwmetadata.Page[fwmetadata.ResourceType], error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.ListResourceTypes(ctx, ListOptions{TenantUUID: t, Module: r.Module, Status: r.Status, Query: r.Query, Locale: r.Locale, Page: r.Page, PageSize: r.PageSize})
}
func (a *FrameworkLocalAdapter) CreateResourceType(ctx context.Context, r fwmetadata.CreateResourceTypeRequest) (*fwmetadata.ResourceType, error) {
	t, e := a.tenant(ctx)
	if e != nil {
		return nil, e
	}
	return a.service.CreateResourceType(ctx, CreateResourceTypeInput{TenantUUID: t, ResourceType: r.ResourceType, Module: r.Module, NameI18n: r.NameI18n, DescriptionI18n: r.DescriptionI18n, ValidatorKey: r.ValidatorKey, BindingEnabled: r.BindingEnabled})
}
func (a *FrameworkLocalAdapter) UpdateResourceType(context.Context, fwmetadata.UpdateResourceTypeRequest) (*fwmetadata.ResourceType, error) {
	return nil, unavailable("metadata.resource_type.update")
}
func (a *FrameworkLocalAdapter) ResolveResourceType(context.Context, string) (*fwmetadata.ResourceType, error) {
	return nil, unavailable("metadata.resource_type.resolve")
}

var _ fwmetadata.Service = (*FrameworkLocalAdapter)(nil)

func i18nPtr(value *fwmetadata.I18nMap) *map[string]string {
	if value == nil {
		return nil
	}
	out := map[string]string(*value)
	return &out
}
