package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

const preferredProtocolREST = "rest"

type restPayload struct {
	Method   string         `json:"method"`
	Endpoint string         `json:"endpoint"`
	Query    map[string]any `json:"query,omitempty"`
	Body     any            `json:"body,omitempty"`
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.Invoker == nil {
		return nil, errors.New("metadata: gateway invoker is required")
	}
	pageSize := cfg.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	return &Client{
		invoker:    cfg.Invoker,
		tenantUUID: strings.TrimSpace(cfg.TenantUUID),
		locale:     strings.TrimSpace(cfg.Locale),
		pageSize:   pageSize,
	}, nil
}

func (c *Client) ListDictionaryNamespaces(ctx context.Context, req ListDictionaryNamespacesRequest) (*Page[DictionaryNamespace], error) {
	query := c.pageQuery(req.Page, req.PageSize, req.Locale)
	setQuery(query, "module", req.Module)
	setQuery(query, "status", req.Status)
	setQuery(query, "q", req.Query)
	return invokePage[DictionaryNamespace](c, ctx, "metadata.dictionary.list_namespaces", CapabilityDictionaryRead, http.MethodGet, "/api/v1/admin/metadata/dictionaries", query, nil, req.RequestID)
}

func (c *Client) CreateDictionaryNamespace(ctx context.Context, req CreateDictionaryNamespaceRequest) (*DictionaryNamespace, error) {
	if strings.TrimSpace(req.Namespace) == "" {
		return nil, invalid("metadata.dictionary.create_namespace", "metadata: namespace is required")
	}
	if strings.TrimSpace(req.Module) == "" {
		return nil, invalid("metadata.dictionary.create_namespace", "metadata: module is required")
	}
	body := map[string]any{
		"namespace":        strings.TrimSpace(req.Namespace),
		"module":           strings.TrimSpace(req.Module),
		"name_i18n":        req.NameI18n,
		"description_i18n": req.DescriptionI18n,
	}
	var out DictionaryNamespace
	if err := c.invokePayload(ctx, "metadata.dictionary.create_namespace", CapabilityDictionaryManage, http.MethodPost, "/api/v1/admin/metadata/dictionaries", nil, body, req.RequestID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ResolveDictionaryNamespace(ctx context.Context, namespace string) (*DictionaryNamespace, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, invalid("metadata.dictionary.resolve_namespace", "metadata: namespace is required")
	}
	page, err := c.ListDictionaryNamespaces(ctx, ListDictionaryNamespacesRequest{Query: namespace, PageSize: c.pageSize})
	if err != nil {
		return nil, err
	}
	for _, item := range page.Items {
		if strings.TrimSpace(item.Namespace) == namespace {
			return &item, nil
		}
	}
	return nil, &Error{Code: CodeNotFound, Message: "metadata: dictionary namespace not found", Operation: "metadata.dictionary.resolve_namespace", Details: map[string]any{"namespace": namespace}}
}

func (c *Client) ListDictionaryItems(ctx context.Context, req ListDictionaryItemsRequest) (*Page[DictionaryItem], error) {
	if strings.TrimSpace(req.NamespaceUUID) == "" {
		return nil, invalid("metadata.dictionary.list_items", "metadata: namespace_uuid is required")
	}
	query := c.pageQuery(req.Page, req.PageSize, req.Locale)
	setQuery(query, "status", req.Status)
	setQuery(query, "q", req.Query)
	endpoint := fmt.Sprintf("/api/v1/admin/metadata/dictionaries/%s/items", strings.TrimSpace(req.NamespaceUUID))
	return invokePage[DictionaryItem](c, ctx, "metadata.dictionary.list_items", CapabilityDictionaryRead, http.MethodGet, endpoint, query, nil, req.RequestID)
}

func (c *Client) CreateDictionaryItem(ctx context.Context, req CreateDictionaryItemRequest) (*DictionaryItem, error) {
	if strings.TrimSpace(req.NamespaceUUID) == "" {
		return nil, invalid("metadata.dictionary.create_item", "metadata: namespace_uuid is required")
	}
	if strings.TrimSpace(req.Code) == "" {
		return nil, invalid("metadata.dictionary.create_item", "metadata: code is required")
	}
	body := map[string]any{
		"code":             strings.TrimSpace(req.Code),
		"label_i18n":       req.LabelI18n,
		"description_i18n": req.DescriptionI18n,
		"sort_order":       req.SortOrder,
	}
	endpoint := fmt.Sprintf("/api/v1/admin/metadata/dictionaries/%s/items", strings.TrimSpace(req.NamespaceUUID))
	var out DictionaryItem
	if err := c.invokePayload(ctx, "metadata.dictionary.create_item", CapabilityDictionaryManage, http.MethodPost, endpoint, nil, body, req.RequestID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ResolveDictionaryItem(ctx context.Context, namespace, code string) (*DictionaryItem, error) {
	namespace = strings.TrimSpace(namespace)
	code = strings.TrimSpace(code)
	if namespace == "" {
		return nil, invalid("metadata.dictionary.resolve_item", "metadata: namespace is required")
	}
	if code == "" {
		return nil, invalid("metadata.dictionary.resolve_item", "metadata: code is required")
	}
	ns, err := c.ResolveDictionaryNamespace(ctx, namespace)
	if err != nil {
		return nil, err
	}
	page, err := c.ListDictionaryItems(ctx, ListDictionaryItemsRequest{NamespaceUUID: ns.UUID, Query: code, PageSize: c.pageSize})
	if err != nil {
		return nil, err
	}
	for _, item := range page.Items {
		if strings.TrimSpace(item.Code) == code {
			return &item, nil
		}
	}
	return nil, &Error{Code: CodeNotFound, Message: "metadata: dictionary item not found", Operation: "metadata.dictionary.resolve_item", Details: map[string]any{"namespace": namespace, "code": code}}
}

func (c *Client) ListTaxonomies(ctx context.Context, req ListTaxonomiesRequest) (*Page[Taxonomy], error) {
	query := c.pageQuery(req.Page, req.PageSize, req.Locale)
	setQuery(query, "module", req.Module)
	setQuery(query, "status", req.Status)
	setQuery(query, "q", req.Query)
	return invokePage[Taxonomy](c, ctx, "metadata.taxonomy.list", CapabilityTaxonomyRead, http.MethodGet, "/api/v1/admin/metadata/taxonomies", query, nil, req.RequestID)
}

func (c *Client) CreateTaxonomy(ctx context.Context, req CreateTaxonomyRequest) (*Taxonomy, error) {
	if strings.TrimSpace(req.Namespace) == "" {
		return nil, invalid("metadata.taxonomy.create", "metadata: namespace is required")
	}
	if strings.TrimSpace(req.Module) == "" {
		return nil, invalid("metadata.taxonomy.create", "metadata: module is required")
	}
	body := map[string]any{
		"namespace":        strings.TrimSpace(req.Namespace),
		"module":           strings.TrimSpace(req.Module),
		"name_i18n":        req.NameI18n,
		"description_i18n": req.DescriptionI18n,
		"max_depth":        req.MaxDepth,
	}
	var out Taxonomy
	if err := c.invokePayload(ctx, "metadata.taxonomy.create", CapabilityTaxonomyManage, http.MethodPost, "/api/v1/admin/metadata/taxonomies", nil, body, req.RequestID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ResolveTaxonomy(ctx context.Context, namespace string) (*Taxonomy, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, invalid("metadata.taxonomy.resolve", "metadata: taxonomy namespace is required")
	}
	page, err := c.ListTaxonomies(ctx, ListTaxonomiesRequest{Query: namespace, PageSize: c.pageSize})
	if err != nil {
		return nil, err
	}
	for _, item := range page.Items {
		if strings.TrimSpace(item.Namespace) == namespace {
			return &item, nil
		}
	}
	return nil, &Error{Code: CodeNotFound, Message: "metadata: taxonomy not found", Operation: "metadata.taxonomy.resolve", Details: map[string]any{"namespace": namespace}}
}

func (c *Client) ListTaxonomyNodes(ctx context.Context, req ListTaxonomyNodesRequest) (*Page[TaxonomyNode], error) {
	if strings.TrimSpace(req.TaxonomyUUID) == "" {
		return nil, invalid("metadata.taxonomy.list_nodes", "metadata: taxonomy_uuid is required")
	}
	query := c.pageQuery(req.Page, req.PageSize, req.Locale)
	setQuery(query, "status", req.Status)
	setQuery(query, "q", req.Query)
	endpoint := fmt.Sprintf("/api/v1/admin/metadata/taxonomies/%s/nodes", strings.TrimSpace(req.TaxonomyUUID))
	return invokePage[TaxonomyNode](c, ctx, "metadata.taxonomy.list_nodes", CapabilityTaxonomyRead, http.MethodGet, endpoint, query, nil, req.RequestID)
}

func (c *Client) CreateTaxonomyNode(ctx context.Context, req CreateTaxonomyNodeRequest) (*TaxonomyNode, error) {
	if strings.TrimSpace(req.TaxonomyUUID) == "" {
		return nil, invalid("metadata.taxonomy.create_node", "metadata: taxonomy_uuid is required")
	}
	if strings.TrimSpace(req.Code) == "" {
		return nil, invalid("metadata.taxonomy.create_node", "metadata: code is required")
	}
	body := map[string]any{
		"parent_uuid":      req.ParentUUID,
		"code":             strings.TrimSpace(req.Code),
		"label_i18n":       req.LabelI18n,
		"description_i18n": req.DescriptionI18n,
		"sort_order":       req.SortOrder,
	}
	endpoint := fmt.Sprintf("/api/v1/admin/metadata/taxonomies/%s/nodes", strings.TrimSpace(req.TaxonomyUUID))
	var out TaxonomyNode
	if err := c.invokePayload(ctx, "metadata.taxonomy.create_node", CapabilityTaxonomyManage, http.MethodPost, endpoint, nil, body, req.RequestID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ResolveTaxonomyNode(ctx context.Context, taxonomyNamespace, code string) (*TaxonomyNode, error) {
	taxonomyNamespace = strings.TrimSpace(taxonomyNamespace)
	code = strings.TrimSpace(code)
	if taxonomyNamespace == "" {
		return nil, invalid("metadata.taxonomy.resolve_node", "metadata: taxonomy namespace is required")
	}
	if code == "" {
		return nil, invalid("metadata.taxonomy.resolve_node", "metadata: code is required")
	}
	taxonomy, err := c.ResolveTaxonomy(ctx, taxonomyNamespace)
	if err != nil {
		return nil, err
	}
	page, err := c.ListTaxonomyNodes(ctx, ListTaxonomyNodesRequest{TaxonomyUUID: taxonomy.UUID, Query: code, PageSize: c.pageSize})
	if err != nil {
		return nil, err
	}
	for _, item := range page.Items {
		if strings.TrimSpace(item.Code) == code {
			return &item, nil
		}
	}
	return nil, &Error{Code: CodeNotFound, Message: "metadata: taxonomy node not found", Operation: "metadata.taxonomy.resolve_node", Details: map[string]any{"taxonomy": taxonomyNamespace, "code": code}}
}

func (c *Client) ListTags(ctx context.Context, req ListTagsRequest) (*Page[Tag], error) {
	query := c.pageQuery(req.Page, req.PageSize, req.Locale)
	setQuery(query, "namespace", req.Namespace)
	setQuery(query, "resource_type", req.ResourceType)
	setQuery(query, "status", req.Status)
	setQuery(query, "q", req.Query)
	return invokePage[Tag](c, ctx, "metadata.tag.list", CapabilityTagRead, http.MethodGet, "/api/v1/admin/metadata/tags", query, nil, req.RequestID)
}

func (c *Client) CreateTag(ctx context.Context, req CreateTagRequest) (*Tag, error) {
	if strings.TrimSpace(req.Namespace) == "" {
		return nil, invalid("metadata.tag.create", "metadata: namespace is required")
	}
	if strings.TrimSpace(req.ResourceType) == "" {
		return nil, invalid("metadata.tag.create", "metadata: resource_type is required")
	}
	if strings.TrimSpace(req.Code) == "" {
		return nil, invalid("metadata.tag.create", "metadata: code is required")
	}
	body := map[string]any{
		"namespace":        strings.TrimSpace(req.Namespace),
		"resource_type":    strings.TrimSpace(req.ResourceType),
		"code":             strings.TrimSpace(req.Code),
		"color":            strings.TrimSpace(req.Color),
		"label_i18n":       req.LabelI18n,
		"description_i18n": req.DescriptionI18n,
	}
	var out Tag
	if err := c.invokePayload(ctx, "metadata.tag.create", CapabilityTagManage, http.MethodPost, "/api/v1/admin/metadata/tags", nil, body, req.RequestID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ResolveTag(ctx context.Context, resourceType, namespace, code string) (*Tag, error) {
	resourceType = strings.TrimSpace(resourceType)
	namespace = strings.TrimSpace(namespace)
	code = strings.TrimSpace(code)
	if resourceType == "" {
		return nil, invalid("metadata.tag.resolve", "metadata: resource_type is required")
	}
	if namespace == "" {
		return nil, invalid("metadata.tag.resolve", "metadata: namespace is required")
	}
	if code == "" {
		return nil, invalid("metadata.tag.resolve", "metadata: code is required")
	}
	page, err := c.ListTags(ctx, ListTagsRequest{ResourceType: resourceType, Namespace: namespace, Query: code, PageSize: c.pageSize})
	if err != nil {
		return nil, err
	}
	for _, item := range page.Items {
		if strings.TrimSpace(item.ResourceType) == resourceType && strings.TrimSpace(item.Namespace) == namespace && strings.TrimSpace(item.Code) == code {
			return &item, nil
		}
	}
	return nil, &Error{Code: CodeNotFound, Message: "metadata: tag not found", Operation: "metadata.tag.resolve", Details: map[string]any{"resource_type": resourceType, "namespace": namespace, "code": code}}
}

func (c *Client) ListTagBindings(ctx context.Context, req ListTagBindingsRequest) ([]TagBinding, error) {
	if strings.TrimSpace(req.ResourceType) == "" {
		return nil, invalid("metadata.tag_binding.list", "metadata: resource_type is required")
	}
	if strings.TrimSpace(req.ResourceUUID) == "" {
		return nil, invalid("metadata.tag_binding.list", "metadata: resource_uuid is required")
	}
	query := map[string]any{
		"resource_type": strings.TrimSpace(req.ResourceType),
		"resource_uuid": strings.TrimSpace(req.ResourceUUID),
	}
	setQuery(query, "locale", firstNonEmpty(req.Locale, c.locale))
	var envelope struct {
		Items []TagBinding `json:"items"`
	}
	if err := c.invokePayload(ctx, "metadata.tag_binding.list", CapabilityTagRead, http.MethodGet, "/api/v1/admin/metadata/tag-bindings", query, nil, req.RequestID, &envelope); err != nil {
		return nil, err
	}
	if envelope.Items == nil {
		envelope.Items = []TagBinding{}
	}
	return envelope.Items, nil
}

func (c *Client) ReplaceTagBindings(ctx context.Context, req ReplaceTagBindingsRequest) ([]TagBinding, error) {
	if strings.TrimSpace(req.ResourceType) == "" {
		return nil, invalid("metadata.tag_binding.replace", "metadata: resource_type is required")
	}
	if strings.TrimSpace(req.ResourceUUID) == "" {
		return nil, invalid("metadata.tag_binding.replace", "metadata: resource_uuid is required")
	}
	body := map[string]any{
		"resource_type": strings.TrimSpace(req.ResourceType),
		"resource_uuid": strings.TrimSpace(req.ResourceUUID),
		"tag_uuids":     append([]string(nil), req.TagUUIDs...),
	}
	var envelope struct {
		Items []TagBinding `json:"items"`
	}
	if err := c.invokePayload(ctx, "metadata.tag_binding.replace", CapabilityTagManage, http.MethodPut, "/api/v1/admin/metadata/tag-bindings", nil, body, req.RequestID, &envelope); err != nil {
		return nil, err
	}
	if envelope.Items == nil {
		envelope.Items = []TagBinding{}
	}
	return envelope.Items, nil
}

func (c *Client) ReplaceTagBindingsByCode(ctx context.Context, req ReplaceTagBindingsByCodeRequest) ([]TagBinding, error) {
	if strings.TrimSpace(req.ResourceType) == "" {
		return nil, invalid("metadata.tag_binding.replace_by_code", "metadata: resource_type is required")
	}
	if strings.TrimSpace(req.ResourceUUID) == "" {
		return nil, invalid("metadata.tag_binding.replace_by_code", "metadata: resource_uuid is required")
	}
	if strings.TrimSpace(req.Namespace) == "" {
		return nil, invalid("metadata.tag_binding.replace_by_code", "metadata: namespace is required")
	}
	tagUUIDs := make([]string, 0, len(req.TagCodes))
	for _, code := range req.TagCodes {
		tag, err := c.ResolveTag(ctx, req.ResourceType, req.Namespace, code)
		if err != nil {
			return nil, err
		}
		tagUUIDs = append(tagUUIDs, tag.UUID)
	}
	return c.ReplaceTagBindings(ctx, ReplaceTagBindingsRequest{
		ResourceType: req.ResourceType,
		ResourceUUID: req.ResourceUUID,
		TagUUIDs:     tagUUIDs,
		RequestID:    req.RequestID,
	})
}

func (c *Client) ListResourceTypes(ctx context.Context, req ListResourceTypesRequest) (*Page[ResourceType], error) {
	query := c.pageQuery(req.Page, req.PageSize, req.Locale)
	setQuery(query, "module", req.Module)
	setQuery(query, "status", req.Status)
	setQuery(query, "q", req.Query)
	return invokePage[ResourceType](c, ctx, "metadata.resource_type.list", CapabilityResourceTypeRead, http.MethodGet, "/api/v1/admin/metadata/resource-types", query, nil, req.RequestID)
}

func (c *Client) CreateResourceType(ctx context.Context, req CreateResourceTypeRequest) (*ResourceType, error) {
	if strings.TrimSpace(req.ResourceType) == "" {
		return nil, invalid("metadata.resource_type.create", "metadata: resource_type is required")
	}
	if strings.TrimSpace(req.Module) == "" {
		return nil, invalid("metadata.resource_type.create", "metadata: module is required")
	}
	body := map[string]any{
		"resource_type":    strings.TrimSpace(req.ResourceType),
		"module":           strings.TrimSpace(req.Module),
		"name_i18n":        req.NameI18n,
		"description_i18n": req.DescriptionI18n,
		"validator_key":    strings.TrimSpace(req.ValidatorKey),
		"binding_enabled":  req.BindingEnabled,
	}
	var out ResourceType
	if err := c.invokePayload(ctx, "metadata.resource_type.create", CapabilityResourceTypeManage, http.MethodPost, "/api/v1/admin/metadata/resource-types", nil, body, req.RequestID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ResolveResourceType(ctx context.Context, resourceType string) (*ResourceType, error) {
	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		return nil, invalid("metadata.resource_type.resolve", "metadata: resource_type is required")
	}
	page, err := c.ListResourceTypes(ctx, ListResourceTypesRequest{Query: resourceType, PageSize: c.pageSize})
	if err != nil {
		return nil, err
	}
	for _, item := range page.Items {
		if strings.TrimSpace(item.ResourceType) == resourceType {
			return &item, nil
		}
	}
	return nil, &Error{Code: CodeNotFound, Message: "metadata: resource type not found", Operation: "metadata.resource_type.resolve", Details: map[string]any{"resource_type": resourceType}}
}

func invokePage[T any](c *Client, ctx context.Context, operation, capabilityID, method, endpoint string, query map[string]any, body any, requestID string) (*Page[T], error) {
	var page Page[T]
	if err := c.invokePayload(ctx, operation, capabilityID, method, endpoint, query, body, requestID, &page); err != nil {
		return nil, err
	}
	if page.Items == nil {
		page.Items = []T{}
	}
	return &page, nil
}

func (c *Client) invokePayload(ctx context.Context, operation, capabilityID, method, endpoint string, query map[string]any, body any, requestID string, out any) error {
	if c == nil || c.invoker == nil {
		return &Error{Code: CodeClientUnavailable, Message: "metadata: gateway invoker is required", Operation: operation}
	}
	resp, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{
		CapabilityID:      capabilityID,
		PreferredProtocol: preferredProtocolREST,
		RequestID:         strings.TrimSpace(requestID),
		TenantUUID:        c.tenantUUID,
		Payload: restPayload{
			Method:   method,
			Endpoint: endpoint,
			Query:    cleanQuery(query),
			Body:     body,
		},
	})
	if err != nil {
		return mapInvokeError(operation, resp, err)
	}
	if resp == nil {
		return &Error{Code: CodeGatewayFailed, Message: "metadata: empty gateway response", Operation: operation}
	}
	payload, err := extractPayload(resp.Data)
	if err != nil {
		return &Error{Code: CodeDecodeFailed, Message: err.Error(), Operation: operation, TraceID: resp.TraceID, Cause: err}
	}
	if out == nil {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return &Error{Code: CodeDecodeFailed, Message: "metadata: encode response payload failed", Operation: operation, TraceID: resp.TraceID, Cause: err}
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return &Error{Code: CodeDecodeFailed, Message: "metadata: decode response payload failed", Operation: operation, TraceID: resp.TraceID, Cause: err}
	}
	return nil
}

func (c *Client) pageQuery(page, pageSize int, locale string) map[string]any {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = c.pageSize
	}
	query := map[string]any{
		"page":      page,
		"page_size": pageSize,
	}
	setQuery(query, "locale", firstNonEmpty(locale, c.locale))
	return query
}

func extractPayload(data map[string]any) (any, error) {
	if data == nil {
		return nil, errors.New("metadata: response data is empty")
	}
	payload, ok := data["payload"]
	if !ok {
		return nil, errors.New("metadata: response payload is missing")
	}
	if payload == nil {
		return nil, errors.New("metadata: response payload is null")
	}
	return payload, nil
}

func mapInvokeError(operation string, resp *gateway.Response, err error) error {
	code := CodeGatewayFailed
	traceID := ""
	if resp != nil {
		traceID = resp.TraceID
	}
	status := 0
	var invokeErr *gateway.InvocationError
	if errors.As(err, &invokeErr) && invokeErr != nil {
		status = invokeErr.StatusCode
	}
	switch status {
	case http.StatusUnauthorized:
		code = CodeUnauthorized
	case http.StatusForbidden:
		code = CodeForbidden
	case http.StatusNotFound:
		code = CodeNotFound
	case http.StatusConflict:
		code = CodeConflict
	}
	return &Error{Code: code, Message: err.Error(), Operation: operation, TraceID: traceID, Cause: err}
}

func setQuery(query map[string]any, key, value string) {
	if query == nil {
		return
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	query[key] = value
}

func cleanQuery(query map[string]any) map[string]any {
	if len(query) == 0 {
		return nil
	}
	out := make(map[string]any, len(query))
	for key, value := range query {
		if strings.TrimSpace(key) == "" || value == nil {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
