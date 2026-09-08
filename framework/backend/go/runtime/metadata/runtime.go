package metadata

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/module"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
)

// Service is the complete metadata business boundary. A plugin supplies its
// own local implementation; Framework supplies the delegated *HostClient.
// There is deliberately no request-time mode switch or local fallback.
type Service interface {
	ListDictionaryNamespaces(context.Context, ListDictionaryNamespacesRequest) (*Page[DictionaryNamespace], error)
	CreateDictionaryNamespace(context.Context, CreateDictionaryNamespaceRequest) (*DictionaryNamespace, error)
	UpdateDictionaryNamespace(context.Context, UpdateDictionaryNamespaceRequest) (*DictionaryNamespace, error)
	ResolveDictionaryNamespace(context.Context, string) (*DictionaryNamespace, error)
	ListDictionaryItems(context.Context, ListDictionaryItemsRequest) (*Page[DictionaryItem], error)
	CreateDictionaryItem(context.Context, CreateDictionaryItemRequest) (*DictionaryItem, error)
	UpdateDictionaryItem(context.Context, UpdateDictionaryItemRequest) (*DictionaryItem, error)
	ResolveDictionaryItem(context.Context, string, string) (*DictionaryItem, error)
	ListTaxonomies(context.Context, ListTaxonomiesRequest) (*Page[Taxonomy], error)
	CreateTaxonomy(context.Context, CreateTaxonomyRequest) (*Taxonomy, error)
	ResolveTaxonomy(context.Context, string) (*Taxonomy, error)
	ListTaxonomyNodes(context.Context, ListTaxonomyNodesRequest) (*Page[TaxonomyNode], error)
	CreateTaxonomyNode(context.Context, CreateTaxonomyNodeRequest) (*TaxonomyNode, error)
	UpdateTaxonomyNode(context.Context, UpdateTaxonomyNodeRequest) (*TaxonomyNode, error)
	ResolveTaxonomyNode(context.Context, string, string) (*TaxonomyNode, error)
	ListTags(context.Context, ListTagsRequest) (*Page[Tag], error)
	CreateTag(context.Context, CreateTagRequest) (*Tag, error)
	ResolveTag(context.Context, string, string, string) (*Tag, error)
	CreateTagBinding(context.Context, CreateTagBindingRequest) (*TagBinding, error)
	DeleteTagBinding(context.Context, DeleteTagBindingRequest) error
	ListResourceTypes(context.Context, ListResourceTypesRequest) (*Page[ResourceType], error)
	CreateResourceType(context.Context, CreateResourceTypeRequest) (*ResourceType, error)
	UpdateResourceType(context.Context, UpdateResourceTypeRequest) (*ResourceType, error)
	ResolveResourceType(context.Context, string) (*ResourceType, error)
}

// Runtime binds exactly one metadata adapter during startup. In local mode the
// plugin must inject Service; in delegated mode Framework's HostClient is used.
type Runtime struct{ service *module.Factory[Service] }

func NewRuntime(mode provider.Mode, local, delegated Service) (*Runtime, error) {
	service, err := module.NewFactory("metadata.service", mode,
		module.Binding[Service]{Value: local, Available: local != nil},
		module.Binding[Service]{Value: delegated, Available: delegated != nil},
	)
	if err != nil {
		return nil, err
	}
	return &Runtime{service: service}, nil
}

func (r *Runtime) Mode() provider.Mode {
	if r == nil || r.service == nil {
		return ""
	}
	return r.service.Mode()
}

func (r *Runtime) Service() (Service, error) {
	if r == nil || r.service == nil {
		return nil, module.NewError("FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE", "metadata runtime is unavailable")
	}
	return r.service.Resolve()
}
