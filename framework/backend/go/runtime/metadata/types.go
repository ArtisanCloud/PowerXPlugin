package metadata

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

const (
	CapabilityDictionaryRead     = "com.corex.metadata.dictionary.read"
	CapabilityDictionaryManage   = "com.corex.metadata.dictionary.manage"
	CapabilityTaxonomyRead       = "com.corex.metadata.taxonomy.read"
	CapabilityTaxonomyManage     = "com.corex.metadata.taxonomy.manage"
	CapabilityTagRead            = "com.corex.metadata.tag.read"
	CapabilityTagManage          = "com.corex.metadata.tag.manage"
	CapabilityResourceTypeRead   = "com.corex.metadata.resource_type.read"
	CapabilityResourceTypeManage = "com.corex.metadata.resource_type.manage"

	DefaultPageSize = 100
)

type GatewayInvoker interface {
	Invoke(ctx context.Context, req gateway.InvokeRequest) (*gateway.Response, error)
}

type Config struct {
	Invoker    GatewayInvoker
	TenantUUID string
	Locale     string
	PageSize   int
}

type Client struct {
	invoker    GatewayInvoker
	tenantUUID string
	locale     string
	pageSize   int
}

type I18nMap map[string]string

type Display struct {
	DisplayName        string `json:"display_name,omitempty"`
	DisplayDescription string `json:"display_description,omitempty"`
}

type Pagination struct {
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

type Page[T any] struct {
	Items      []T        `json:"items"`
	Pagination Pagination `json:"pagination"`
}

type DictionaryNamespace struct {
	UUID            string  `json:"uuid"`
	Namespace       string  `json:"namespace"`
	Module          string  `json:"module"`
	NameI18n        I18nMap `json:"name_i18n"`
	DescriptionI18n I18nMap `json:"description_i18n,omitempty"`
	Status          string  `json:"status"`
	ItemCount       int64   `json:"item_count"`
	Display
}

type DictionaryItem struct {
	UUID            string  `json:"uuid"`
	NamespaceUUID   string  `json:"namespace_uuid"`
	Code            string  `json:"code"`
	LabelI18n       I18nMap `json:"label_i18n"`
	DescriptionI18n I18nMap `json:"description_i18n,omitempty"`
	Status          string  `json:"status"`
	SortOrder       int     `json:"sort_order"`
	ReferenceCount  int64   `json:"reference_count"`
	Display
}

type Taxonomy struct {
	UUID            string  `json:"uuid"`
	Namespace       string  `json:"namespace"`
	Module          string  `json:"module"`
	NameI18n        I18nMap `json:"name_i18n"`
	DescriptionI18n I18nMap `json:"description_i18n,omitempty"`
	MaxDepth        int     `json:"max_depth"`
	Status          string  `json:"status"`
	Display
}

type TaxonomyNode struct {
	UUID            string  `json:"uuid"`
	TaxonomyUUID    string  `json:"taxonomy_uuid"`
	ParentUUID      *string `json:"parent_uuid,omitempty"`
	Code            string  `json:"code"`
	LabelI18n       I18nMap `json:"label_i18n"`
	DescriptionI18n I18nMap `json:"description_i18n,omitempty"`
	Path            string  `json:"path"`
	Depth           int     `json:"depth"`
	SortOrder       int     `json:"sort_order"`
	Status          string  `json:"status"`
	ReferenceCount  int64   `json:"reference_count"`
	Version         int64   `json:"version"`
	Display
}

type Tag struct {
	UUID            string  `json:"uuid"`
	Namespace       string  `json:"namespace"`
	ResourceType    string  `json:"resource_type"`
	Code            string  `json:"code"`
	LabelI18n       I18nMap `json:"label_i18n"`
	DescriptionI18n I18nMap `json:"description_i18n,omitempty"`
	Color           string  `json:"color,omitempty"`
	Status          string  `json:"status"`
	UsageCount      int64   `json:"usage_count"`
	Display
}

type TagBinding struct {
	TagUUID      string `json:"tag_uuid"`
	ResourceType string `json:"resource_type"`
	ResourceUUID string `json:"resource_uuid"`
	Tag          *Tag   `json:"tag,omitempty"`
}

type ResourceType struct {
	UUID            string  `json:"uuid"`
	ResourceType    string  `json:"resource_type"`
	Module          string  `json:"module"`
	NameI18n        I18nMap `json:"name_i18n"`
	DescriptionI18n I18nMap `json:"description_i18n,omitempty"`
	ValidatorKey    string  `json:"validator_key,omitempty"`
	BindingEnabled  bool    `json:"binding_enabled"`
	ValidatorStatus string  `json:"validator_status"`
	Status          string  `json:"status"`
	Display
}

type ListDictionaryNamespacesRequest struct {
	Module    string
	Status    string
	Query     string
	Locale    string
	Page      int
	PageSize  int
	RequestID string
}

type ListDictionaryItemsRequest struct {
	NamespaceUUID string
	Status        string
	Query         string
	Locale        string
	Page          int
	PageSize      int
	RequestID     string
}

type ListTaxonomiesRequest struct {
	Module    string
	Status    string
	Query     string
	Locale    string
	Page      int
	PageSize  int
	RequestID string
}

type ListTaxonomyNodesRequest struct {
	TaxonomyUUID string
	Status       string
	Query        string
	Locale       string
	Page         int
	PageSize     int
	RequestID    string
}

type ListTagsRequest struct {
	Namespace    string
	ResourceType string
	Status       string
	Query        string
	Locale       string
	Page         int
	PageSize     int
	RequestID    string
}

type ListTagBindingsRequest struct {
	ResourceType string
	ResourceUUID string
	Locale       string
	RequestID    string
}

type ReplaceTagBindingsRequest struct {
	ResourceType string
	ResourceUUID string
	TagUUIDs     []string
	RequestID    string
}

type ReplaceTagBindingsByCodeRequest struct {
	ResourceType string
	ResourceUUID string
	Namespace    string
	TagCodes     []string
	RequestID    string
}

type ListResourceTypesRequest struct {
	Module    string
	Status    string
	Query     string
	Locale    string
	Page      int
	PageSize  int
	RequestID string
}

type CreateDictionaryNamespaceRequest struct {
	Namespace       string
	Module          string
	NameI18n        I18nMap
	DescriptionI18n I18nMap
	RequestID       string
}

type CreateDictionaryItemRequest struct {
	NamespaceUUID   string
	Code            string
	LabelI18n       I18nMap
	DescriptionI18n I18nMap
	SortOrder       int
	RequestID       string
}

type CreateTaxonomyRequest struct {
	Namespace       string
	Module          string
	NameI18n        I18nMap
	DescriptionI18n I18nMap
	MaxDepth        int
	RequestID       string
}

type CreateTaxonomyNodeRequest struct {
	TaxonomyUUID    string
	ParentUUID      *string
	Code            string
	LabelI18n       I18nMap
	DescriptionI18n I18nMap
	SortOrder       int
	RequestID       string
}

type CreateTagRequest struct {
	Namespace       string
	ResourceType    string
	Code            string
	Color           string
	LabelI18n       I18nMap
	DescriptionI18n I18nMap
	RequestID       string
}

type CreateResourceTypeRequest struct {
	ResourceType    string
	Module          string
	NameI18n        I18nMap
	DescriptionI18n I18nMap
	ValidatorKey    string
	BindingEnabled  bool
	RequestID       string
}
