package metadata

// This file contains the service-actor adapter for the PowerX tenant Metadata
// Host Contract. It intentionally does not use the admin metadata API: Core
// derives tenant identity from the STS credential carried by this client.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	"github.com/google/uuid"
)

type HostTokenProvider interface {
	Token(context.Context) (string, error)
}
type HostTokenProviderFunc func(context.Context) (string, error)

func (f HostTokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type HostClientConfig struct {
	BaseURL  string
	Timeout  time.Duration
	Locale   string
	PageSize int
}

type HostClient struct {
	baseURL  string
	tokens   HostTokenProvider
	http     *http.Client
	locale   string
	pageSize int
}

func NewHostClientWithTokenProvider(cfg HostClientConfig, tokens HostTokenProvider, httpClient *http.Client) (*HostClient, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("metadata host base_url is required")
	}
	if tokens == nil {
		return nil, errors.New("metadata host token provider is required")
	}
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	return &HostClient{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"), tokens: tokens, http: httpClient, locale: strings.TrimSpace(cfg.Locale), pageSize: pageSize}, nil
}

func (c *HostClient) ListDictionaryNamespaces(ctx context.Context, in ListDictionaryNamespacesRequest) (*Page[DictionaryNamespace], error) {
	var out Page[DictionaryNamespace]
	err := c.do(ctx, http.MethodGet, "/api/v1/tenant/metadata/dictionaries", c.listQuery(in.Page, in.PageSize, in.Locale, map[string]string{"module": in.Module, "status": in.Status, "q": in.Query}), nil, &out)
	return normalizePage(&out), err
}
func (c *HostClient) CreateDictionaryNamespace(ctx context.Context, in CreateDictionaryNamespaceRequest) (*DictionaryNamespace, error) {
	if strings.TrimSpace(in.Namespace) == "" || strings.TrimSpace(in.Module) == "" {
		return nil, hostInvalid("metadata.dictionary.create")
	}
	var out DictionaryNamespace
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/metadata/dictionaries", nil, map[string]any{"namespace": strings.TrimSpace(in.Namespace), "module": strings.TrimSpace(in.Module), "name_i18n": in.NameI18n, "description_i18n": in.DescriptionI18n}, &out)
	return &out, err
}
func (c *HostClient) UpdateDictionaryNamespace(ctx context.Context, in UpdateDictionaryNamespaceRequest) (*DictionaryNamespace, error) {
	if !validHostUUID(in.NamespaceUUID) {
		return nil, hostInvalid("metadata.dictionary.update")
	}
	var out DictionaryNamespace
	err := c.do(ctx, http.MethodPatch, "/api/v1/tenant/metadata/dictionaries/"+url.PathEscape(in.NamespaceUUID), nil, map[string]any{"name_i18n": in.NameI18n, "description_i18n": in.DescriptionI18n, "status": in.Status}, &out)
	return &out, err
}
func (c *HostClient) ResolveDictionaryNamespace(ctx context.Context, namespace string) (*DictionaryNamespace, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, hostInvalid("metadata.resolve")
	}

	return resolvePages(ctx, func(page int) (*Page[DictionaryNamespace], error) {
		return c.ListDictionaryNamespaces(ctx, ListDictionaryNamespacesRequest{Query: namespace, Page: page, PageSize: c.pageSize})
	}, func(item DictionaryNamespace) bool { return item.Namespace == namespace })
}
func (c *HostClient) ListDictionaryItems(ctx context.Context, in ListDictionaryItemsRequest) (*Page[DictionaryItem], error) {
	if !validHostUUID(in.NamespaceUUID) {
		return nil, hostInvalid("metadata.dictionary.items.list")
	}
	var out Page[DictionaryItem]
	err := c.do(ctx, http.MethodGet, "/api/v1/tenant/metadata/dictionaries/"+url.PathEscape(in.NamespaceUUID)+"/items", c.listQuery(in.Page, in.PageSize, in.Locale, map[string]string{"status": in.Status, "q": in.Query}), nil, &out)
	return normalizePage(&out), err
}
func (c *HostClient) CreateDictionaryItem(ctx context.Context, in CreateDictionaryItemRequest) (*DictionaryItem, error) {
	if !validHostUUID(in.NamespaceUUID) || strings.TrimSpace(in.Code) == "" {
		return nil, hostInvalid("metadata.dictionary.items.create")
	}
	var out DictionaryItem
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/metadata/dictionaries/"+url.PathEscape(in.NamespaceUUID)+"/items", nil, map[string]any{"code": strings.TrimSpace(in.Code), "label_i18n": in.LabelI18n, "description_i18n": in.DescriptionI18n, "sort_order": in.SortOrder, "metadata": in.Metadata}, &out)
	return &out, err
}
func (c *HostClient) UpdateDictionaryItem(ctx context.Context, in UpdateDictionaryItemRequest) (*DictionaryItem, error) {
	if !validHostUUID(in.ItemUUID) {
		return nil, hostInvalid("metadata.dictionary.items.update")
	}
	var out DictionaryItem
	err := c.do(ctx, http.MethodPatch, "/api/v1/tenant/metadata/dictionary-items/"+url.PathEscape(in.ItemUUID), nil, map[string]any{"label_i18n": in.LabelI18n, "description_i18n": in.DescriptionI18n, "sort_order": in.SortOrder, "status": in.Status, "metadata": in.Metadata}, &out)
	return &out, err
}
func (c *HostClient) ResolveDictionaryItem(ctx context.Context, namespace, code string) (*DictionaryItem, error) {
	namespace = strings.TrimSpace(namespace)
	code = strings.TrimSpace(code)
	if namespace == "" || code == "" {
		return nil, hostInvalid("metadata.resolve")
	}
	ns, err := c.ResolveDictionaryNamespace(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return resolvePages(ctx, func(page int) (*Page[DictionaryItem], error) {
		return c.ListDictionaryItems(ctx, ListDictionaryItemsRequest{NamespaceUUID: ns.UUID, Query: code, Page: page, PageSize: c.pageSize})
	}, func(item DictionaryItem) bool { return item.Code == code })
}

func (c *HostClient) ListTaxonomies(ctx context.Context, in ListTaxonomiesRequest) (*Page[Taxonomy], error) {
	var out Page[Taxonomy]
	err := c.do(ctx, http.MethodGet, "/api/v1/tenant/metadata/taxonomies", c.listQuery(in.Page, in.PageSize, in.Locale, map[string]string{"module": in.Module, "status": in.Status, "q": in.Query}), nil, &out)
	return normalizePage(&out), err
}
func (c *HostClient) CreateTaxonomy(ctx context.Context, in CreateTaxonomyRequest) (*Taxonomy, error) {
	if strings.TrimSpace(in.Namespace) == "" || strings.TrimSpace(in.Module) == "" {
		return nil, hostInvalid("metadata.taxonomy.create")
	}
	var out Taxonomy
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/metadata/taxonomies", nil, map[string]any{"namespace": strings.TrimSpace(in.Namespace), "module": strings.TrimSpace(in.Module), "name_i18n": in.NameI18n, "description_i18n": in.DescriptionI18n, "max_depth": in.MaxDepth}, &out)
	return &out, err
}
func (c *HostClient) ResolveTaxonomy(ctx context.Context, namespace string) (*Taxonomy, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, hostInvalid("metadata.resolve")
	}

	return resolvePages(ctx, func(page int) (*Page[Taxonomy], error) {
		return c.ListTaxonomies(ctx, ListTaxonomiesRequest{Query: namespace, Page: page, PageSize: c.pageSize})
	}, func(item Taxonomy) bool { return item.Namespace == namespace })
}
func (c *HostClient) ListTaxonomyNodes(ctx context.Context, in ListTaxonomyNodesRequest) (*Page[TaxonomyNode], error) {
	if !validHostUUID(in.TaxonomyUUID) {
		return nil, hostInvalid("metadata.taxonomy.nodes.list")
	}
	var out Page[TaxonomyNode]
	err := c.do(ctx, http.MethodGet, "/api/v1/tenant/metadata/taxonomies/"+url.PathEscape(in.TaxonomyUUID)+"/nodes", c.listQuery(in.Page, in.PageSize, in.Locale, map[string]string{"status": in.Status, "q": in.Query}), nil, &out)
	return normalizePage(&out), err
}
func (c *HostClient) CreateTaxonomyNode(ctx context.Context, in CreateTaxonomyNodeRequest) (*TaxonomyNode, error) {
	if !validHostUUID(in.TaxonomyUUID) || strings.TrimSpace(in.Code) == "" || (in.ParentUUID != nil && !validHostUUID(*in.ParentUUID)) {
		return nil, hostInvalid("metadata.taxonomy.nodes.create")
	}
	var out TaxonomyNode
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/metadata/taxonomies/"+url.PathEscape(in.TaxonomyUUID)+"/nodes", nil, map[string]any{"parent_uuid": in.ParentUUID, "code": strings.TrimSpace(in.Code), "label_i18n": in.LabelI18n, "description_i18n": in.DescriptionI18n, "sort_order": in.SortOrder}, &out)
	return &out, err
}
func (c *HostClient) UpdateTaxonomyNode(ctx context.Context, in UpdateTaxonomyNodeRequest) (*TaxonomyNode, error) {
	if !validHostUUID(in.NodeUUID) || in.Version <= 0 {
		return nil, hostInvalid("metadata.taxonomy.nodes.update")
	}
	var out TaxonomyNode
	err := c.do(ctx, http.MethodPatch, "/api/v1/tenant/metadata/taxonomy-nodes/"+url.PathEscape(in.NodeUUID), nil, map[string]any{"label_i18n": in.LabelI18n, "description_i18n": in.DescriptionI18n, "sort_order": in.SortOrder, "status": in.Status, "version": in.Version}, &out)
	return &out, err
}
func (c *HostClient) ResolveTaxonomyNode(ctx context.Context, namespace, code string) (*TaxonomyNode, error) {
	namespace = strings.TrimSpace(namespace)
	code = strings.TrimSpace(code)
	if namespace == "" || code == "" {
		return nil, hostInvalid("metadata.resolve")
	}
	taxonomy, err := c.ResolveTaxonomy(ctx, namespace)
	if err != nil {
		return nil, err
	}
	return resolvePages(ctx, func(page int) (*Page[TaxonomyNode], error) {
		return c.ListTaxonomyNodes(ctx, ListTaxonomyNodesRequest{TaxonomyUUID: taxonomy.UUID, Query: code, Page: page, PageSize: c.pageSize})
	}, func(item TaxonomyNode) bool { return item.Code == code })
}

func (c *HostClient) ListTags(ctx context.Context, in ListTagsRequest) (*Page[Tag], error) {
	var out Page[Tag]
	err := c.do(ctx, http.MethodGet, "/api/v1/tenant/metadata/tags", c.listQuery(in.Page, in.PageSize, in.Locale, map[string]string{"namespace": in.Namespace, "resource_type": in.ResourceType, "status": in.Status, "q": in.Query}), nil, &out)
	return normalizePage(&out), err
}
func (c *HostClient) CreateTag(ctx context.Context, in CreateTagRequest) (*Tag, error) {
	if strings.TrimSpace(in.Namespace) == "" || strings.TrimSpace(in.ResourceType) == "" || strings.TrimSpace(in.Code) == "" {
		return nil, hostInvalid("metadata.tag.create")
	}
	var out Tag
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/metadata/tags", nil, map[string]any{"namespace": strings.TrimSpace(in.Namespace), "resource_type": strings.TrimSpace(in.ResourceType), "code": strings.TrimSpace(in.Code), "color": strings.TrimSpace(in.Color), "label_i18n": in.LabelI18n, "description_i18n": in.DescriptionI18n}, &out)
	return &out, err
}
func (c *HostClient) ResolveTag(ctx context.Context, resourceType, namespace, code string) (*Tag, error) {
	resourceType = strings.TrimSpace(resourceType)
	namespace = strings.TrimSpace(namespace)
	code = strings.TrimSpace(code)
	if resourceType == "" || namespace == "" || code == "" {
		return nil, hostInvalid("metadata.resolve")
	}

	return resolvePages(ctx, func(page int) (*Page[Tag], error) {
		return c.ListTags(ctx, ListTagsRequest{ResourceType: resourceType, Namespace: namespace, Query: code, Page: page, PageSize: c.pageSize})
	}, func(item Tag) bool {
		return item.ResourceType == resourceType && item.Namespace == namespace && item.Code == code
	})
}
func (c *HostClient) CreateTagBinding(ctx context.Context, in CreateTagBindingRequest) (*TagBinding, error) {
	if !validHostUUID(in.TagUUID) || strings.TrimSpace(in.ResourceType) == "" || !validHostUUID(in.ResourceUUID) {
		return nil, hostInvalid("metadata.tag_binding.create")
	}
	var out TagBinding
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/metadata/tag-bindings", nil, map[string]any{"tag_uuid": strings.TrimSpace(in.TagUUID), "resource_type": strings.TrimSpace(in.ResourceType), "resource_uuid": strings.TrimSpace(in.ResourceUUID)}, &out)
	return &out, err
}
func (c *HostClient) DeleteTagBinding(ctx context.Context, in DeleteTagBindingRequest) error {
	if !validHostUUID(in.BindingUUID) {
		return hostInvalid("metadata.tag_binding.delete")
	}
	return c.do(ctx, http.MethodDelete, "/api/v1/tenant/metadata/tag-bindings/"+url.PathEscape(in.BindingUUID), nil, nil, nil)
}
func (c *HostClient) ListResourceTypes(ctx context.Context, in ListResourceTypesRequest) (*Page[ResourceType], error) {
	var out Page[ResourceType]
	err := c.do(ctx, http.MethodGet, "/api/v1/tenant/metadata/resource-types", c.listQuery(in.Page, in.PageSize, in.Locale, map[string]string{"module": in.Module, "status": in.Status, "q": in.Query}), nil, &out)
	return normalizePage(&out), err
}
func (c *HostClient) CreateResourceType(ctx context.Context, in CreateResourceTypeRequest) (*ResourceType, error) {
	if strings.TrimSpace(in.ResourceType) == "" || strings.TrimSpace(in.Module) == "" {
		return nil, hostInvalid("metadata.resource_type.create")
	}
	var out ResourceType
	err := c.do(ctx, http.MethodPost, "/api/v1/tenant/metadata/resource-types", nil, map[string]any{"resource_type": strings.TrimSpace(in.ResourceType), "module": strings.TrimSpace(in.Module), "name_i18n": in.NameI18n, "description_i18n": in.DescriptionI18n, "validator_key": strings.TrimSpace(in.ValidatorKey), "binding_enabled": in.BindingEnabled}, &out)
	return &out, err
}
func (c *HostClient) UpdateResourceType(ctx context.Context, in UpdateResourceTypeRequest) (*ResourceType, error) {
	if !validHostUUID(in.ResourceTypeUUID) {
		return nil, hostInvalid("metadata.resource_type.update")
	}
	var out ResourceType
	err := c.do(ctx, http.MethodPatch, "/api/v1/tenant/metadata/resource-types/"+url.PathEscape(in.ResourceTypeUUID), nil, map[string]any{"name_i18n": in.NameI18n, "description_i18n": in.DescriptionI18n, "validator_key": in.ValidatorKey, "binding_enabled": in.BindingEnabled, "status": in.Status}, &out)
	return &out, err
}
func (c *HostClient) ResolveResourceType(ctx context.Context, resourceType string) (*ResourceType, error) {
	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		return nil, hostInvalid("metadata.resolve")
	}

	return resolvePages(ctx, func(page int) (*Page[ResourceType], error) {
		return c.ListResourceTypes(ctx, ListResourceTypesRequest{Query: resourceType, Page: page, PageSize: c.pageSize})
	}, func(item ResourceType) bool { return item.ResourceType == resourceType })
}

func (c *HostClient) listQuery(page, size int, locale string, values map[string]string) url.Values {
	q := url.Values{}
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = c.pageSize
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(size))
	if locale = firstNonEmpty(locale, c.locale); locale != "" {
		q.Set("locale", locale)
	}
	for k, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			q.Set(k, v)
		}
	}
	return q
}
func normalizePage[T any](page *Page[T]) *Page[T] {
	if page != nil && page.Items == nil {
		page.Items = []T{}
	}
	return page
}
func hostInvalid(operation string) *Error {
	return &Error{Code: CodeInvalidArgument, Message: "metadata host request has invalid arguments", Operation: operation}
}

func (c *HostClient) do(ctx context.Context, method, path string, query url.Values, input, out any) error {
	if c == nil || c.http == nil || c.tokens == nil {
		return &Error{Code: CodeClientUnavailable, Message: "metadata host client is not configured"}
	}
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		if err != nil {
			return &Error{Code: CodeInvalidArgument, Message: "metadata request cannot be encoded", Cause: err}
		}
		body = bytes.NewReader(raw)
	}
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return &Error{Code: CodeGatewayFailed, Message: "metadata host request cannot be created", Cause: err}
	}
	token, err := c.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return &Error{Code: CodeClientUnavailable, Message: "metadata host token is unavailable", Cause: err}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return &Error{Code: CodeGatewayFailed, Message: "metadata host request failed", Cause: err}
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if readErr != nil || len(raw) > 1<<20 {
		return &Error{Code: CodeDecodeFailed, Message: "metadata.response_read_failed", Cause: readErr}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return metadataHostError(resp.StatusCode, raw)
	}
	if out == nil {
		return nil
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return &Error{Code: CodeDecodeFailed, Message: "metadata host response is invalid", Cause: err}
	}
	if len(envelope.Data) == 0 {
		return &Error{Code: CodeDecodeFailed, Message: "metadata host response data is missing"}
	}
	var wrapped struct {
		Payload json.RawMessage `json:"payload"`
	}
	if json.Unmarshal(envelope.Data, &wrapped) != nil || len(wrapped.Payload) == 0 || string(wrapped.Payload) == "null" {
		return &Error{Code: CodeDecodeFailed, Message: "metadata.response_payload_missing"}
	}
	envelope.Data = wrapped.Payload
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return &Error{Code: CodeDecodeFailed, Message: "metadata host response cannot be decoded", Cause: err}
	}
	return validateHostObject(out)
}

func metadataHostError(status int, raw []byte) *Error {
	fallback := "METADATA_UPSTREAM_DEPENDENCY"
	switch status {
	case http.StatusBadRequest:
		fallback = "METADATA_INVALID_ARGUMENT"
	case http.StatusUnauthorized:
		fallback = "METADATA_UNAUTHORIZED"
	case http.StatusForbidden:
		fallback = "METADATA_FORBIDDEN"
	case http.StatusNotFound:
		fallback = "METADATA_NOT_FOUND"
	case http.StatusConflict:
		fallback = "METADATA_CONFLICT"
	}
	code := ErrorCode(hostcontract.ParseReasonCode(raw, fallback))
	return &Error{Code: code, StatusCode: status, Message: fmt.Sprintf("metadata host request failed: status=%d", status), Details: map[string]any{"status_code": status, "reason_code": string(code)}}
}

var _ Service = (*HostClient)(nil)

func validHostUUID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed != uuid.Nil && strings.EqualFold(parsed.String(), value)
}

func validateHostObject(out any) error {
	var id string
	switch value := out.(type) {
	case *DictionaryNamespace:
		id = value.UUID
	case *DictionaryItem:
		id = value.UUID
	case *Taxonomy:
		id = value.UUID
	case *TaxonomyNode:
		id = value.UUID
	case *Tag:
		id = value.UUID
	case *TagBinding:
		id = value.BindingUUID
	case *ResourceType:
		id = value.UUID
	case *Page[DictionaryNamespace]:
		return validateHostPage(value)
	case *Page[DictionaryItem]:
		return validateHostPage(value)
	case *Page[Taxonomy]:
		return validateHostPage(value)
	case *Page[TaxonomyNode]:
		return validateHostPage(value)
	case *Page[Tag]:
		return validateHostPage(value)
	case *Page[ResourceType]:
		return validateHostPage(value)
	default:
		return nil
	}
	if !validHostUUID(id) {
		return &Error{Code: CodeDecodeFailed, Message: "metadata.response_uuid_invalid"}
	}
	return nil
}

func validateHostPage[T any](page *Page[T]) error {
	if page.Pagination.Page < 1 || page.Pagination.PageSize < 1 || page.Pagination.Total < int64(len(page.Items)) {
		return &Error{Code: CodeDecodeFailed, Message: "metadata.response_pagination_invalid"}
	}
	for i := range page.Items {
		if err := validateHostObject(&page.Items[i]); err != nil {
			return err
		}
	}
	return nil
}
