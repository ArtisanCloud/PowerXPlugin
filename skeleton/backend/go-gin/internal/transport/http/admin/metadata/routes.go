package metadata

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	fwmetadata "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/metadata"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	metadatasvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/metadata"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	admincommon "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/common"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(admin *gin.RouterGroup, deps *app.Deps) {
	if admin == nil || deps == nil {
		return
	}
	h := &Handler{
		mode:      deps.IAMMode,
		delegated: deps.Metadata,
		local:     metadatasvc.NewService(deps.DB),
	}
	group := admin.Group("/metadata")
	group.GET("/mode", h.Mode)
	group.GET("/dictionaries", h.ListDictionaryNamespaces)
	group.POST("/dictionaries", h.CreateDictionaryNamespace)
	group.GET("/dictionaries/:namespace_uuid/items", h.ListDictionaryItems)
	group.POST("/dictionaries/:namespace_uuid/items", h.CreateDictionaryItem)
	group.GET("/taxonomies", h.ListTaxonomies)
	group.POST("/taxonomies", h.CreateTaxonomy)
	group.GET("/taxonomies/:taxonomy_uuid/nodes", h.ListTaxonomyNodes)
	group.POST("/taxonomies/:taxonomy_uuid/nodes", h.CreateTaxonomyNode)
	group.GET("/tags", h.ListTags)
	group.POST("/tags", h.CreateTag)
	group.GET("/resource-types", h.ListResourceTypes)
	group.POST("/resource-types", h.CreateResourceType)
}

type Handler struct {
	mode      iamservice.IAMMode
	delegated *fwmetadata.Client
	local     *metadatasvc.Service
}

type createDictionaryNamespaceRequest struct {
	Namespace       string            `json:"namespace"`
	Module          string            `json:"module"`
	NameI18n        map[string]string `json:"name_i18n"`
	DescriptionI18n map[string]string `json:"description_i18n"`
}

type createDictionaryItemRequest struct {
	Code            string            `json:"code"`
	LabelI18n       map[string]string `json:"label_i18n"`
	DescriptionI18n map[string]string `json:"description_i18n"`
	SortOrder       int               `json:"sort_order"`
}

type createTaxonomyRequest struct {
	Namespace       string            `json:"namespace"`
	Module          string            `json:"module"`
	NameI18n        map[string]string `json:"name_i18n"`
	DescriptionI18n map[string]string `json:"description_i18n"`
	MaxDepth        int               `json:"max_depth"`
}

type createTaxonomyNodeRequest struct {
	ParentUUID      *string           `json:"parent_uuid"`
	Code            string            `json:"code"`
	LabelI18n       map[string]string `json:"label_i18n"`
	DescriptionI18n map[string]string `json:"description_i18n"`
	SortOrder       int               `json:"sort_order"`
}

type createTagRequest struct {
	Namespace       string            `json:"namespace"`
	ResourceType    string            `json:"resource_type"`
	Code            string            `json:"code"`
	Color           string            `json:"color"`
	LabelI18n       map[string]string `json:"label_i18n"`
	DescriptionI18n map[string]string `json:"description_i18n"`
}

type createResourceTypeRequest struct {
	ResourceType    string            `json:"resource_type"`
	Module          string            `json:"module"`
	NameI18n        map[string]string `json:"name_i18n"`
	DescriptionI18n map[string]string `json:"description_i18n"`
	ValidatorKey    string            `json:"validator_key"`
	BindingEnabled  bool              `json:"binding_enabled"`
}

func (h *Handler) Mode(c *gin.Context) {
	mode := strings.TrimSpace(h.mode.String())
	if mode == "" {
		mode = string(iamservice.IAMModeLocal)
	}
	contracts.ResponseSuccess(c, gin.H{
		"mode":                mode,
		"delegated_available": h.delegated != nil,
		"local_available":     h.local != nil,
	})
}

func (h *Handler) ListDictionaryNamespaces(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	if h.isDelegated() {
		page, err := h.delegated.ListDictionaryNamespaces(c.Request.Context(), fwmetadata.ListDictionaryNamespacesRequest{
			Module: opts.Module, Status: opts.Status, Query: opts.Query, Locale: opts.Locale, Page: opts.Page, PageSize: opts.PageSize, RequestID: requestID(c),
		})
		respondPage(c, page, err)
		return
	}
	page, err := h.local.ListDictionaryNamespaces(c.Request.Context(), opts)
	respondPage(c, page, err)
}

func (h *Handler) CreateDictionaryNamespace(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	var req createDictionaryNamespaceRequest
	if !bindJSON(c, &req) {
		return
	}
	if h.isDelegated() {
		item, err := h.delegated.CreateDictionaryNamespace(c.Request.Context(), fwmetadata.CreateDictionaryNamespaceRequest{
			Namespace: req.Namespace, Module: req.Module, NameI18n: req.NameI18n, DescriptionI18n: req.DescriptionI18n, RequestID: requestID(c),
		})
		respondItem(c, item, err)
		return
	}
	item, err := h.local.CreateDictionaryNamespace(c.Request.Context(), metadatasvc.CreateDictionaryNamespaceInput{
		TenantUUID: opts.TenantUUID, Namespace: req.Namespace, Module: req.Module, NameI18n: req.NameI18n, DescriptionI18n: req.DescriptionI18n,
	})
	respondItem(c, item, err)
}

func (h *Handler) ListDictionaryItems(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	namespaceUUID := strings.TrimSpace(c.Param("namespace_uuid"))
	if namespaceUUID == "" {
		contracts.ResponseBadRequest(c, "namespace_uuid is required")
		return
	}
	if h.isDelegated() {
		page, err := h.delegated.ListDictionaryItems(c.Request.Context(), fwmetadata.ListDictionaryItemsRequest{
			NamespaceUUID: namespaceUUID, Status: opts.Status, Query: opts.Query, Locale: opts.Locale, Page: opts.Page, PageSize: opts.PageSize, RequestID: requestID(c),
		})
		respondPage(c, page, err)
		return
	}
	page, err := h.local.ListDictionaryItems(c.Request.Context(), namespaceUUID, opts)
	respondPage(c, page, err)
}

func (h *Handler) CreateDictionaryItem(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	namespaceUUID := strings.TrimSpace(c.Param("namespace_uuid"))
	if namespaceUUID == "" {
		contracts.ResponseBadRequest(c, "namespace_uuid is required")
		return
	}
	var req createDictionaryItemRequest
	if !bindJSON(c, &req) {
		return
	}
	if h.isDelegated() {
		item, err := h.delegated.CreateDictionaryItem(c.Request.Context(), fwmetadata.CreateDictionaryItemRequest{
			NamespaceUUID: namespaceUUID, Code: req.Code, LabelI18n: req.LabelI18n, DescriptionI18n: req.DescriptionI18n, SortOrder: req.SortOrder, RequestID: requestID(c),
		})
		respondItem(c, item, err)
		return
	}
	item, err := h.local.CreateDictionaryItem(c.Request.Context(), metadatasvc.CreateDictionaryItemInput{
		TenantUUID: opts.TenantUUID, NamespaceUUID: namespaceUUID, Code: req.Code, LabelI18n: req.LabelI18n, DescriptionI18n: req.DescriptionI18n, SortOrder: req.SortOrder,
	})
	respondItem(c, item, err)
}

func (h *Handler) ListTaxonomies(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	if h.isDelegated() {
		page, err := h.delegated.ListTaxonomies(c.Request.Context(), fwmetadata.ListTaxonomiesRequest{
			Module: opts.Module, Status: opts.Status, Query: opts.Query, Locale: opts.Locale, Page: opts.Page, PageSize: opts.PageSize, RequestID: requestID(c),
		})
		respondPage(c, page, err)
		return
	}
	page, err := h.local.ListTaxonomies(c.Request.Context(), opts)
	respondPage(c, page, err)
}

func (h *Handler) CreateTaxonomy(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	var req createTaxonomyRequest
	if !bindJSON(c, &req) {
		return
	}
	if h.isDelegated() {
		item, err := h.delegated.CreateTaxonomy(c.Request.Context(), fwmetadata.CreateTaxonomyRequest{
			Namespace: req.Namespace, Module: req.Module, NameI18n: req.NameI18n, DescriptionI18n: req.DescriptionI18n, MaxDepth: req.MaxDepth, RequestID: requestID(c),
		})
		respondItem(c, item, err)
		return
	}
	item, err := h.local.CreateTaxonomy(c.Request.Context(), metadatasvc.CreateTaxonomyInput{
		TenantUUID: opts.TenantUUID, Namespace: req.Namespace, Module: req.Module, NameI18n: req.NameI18n, DescriptionI18n: req.DescriptionI18n, MaxDepth: req.MaxDepth,
	})
	respondItem(c, item, err)
}

func (h *Handler) ListTaxonomyNodes(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	taxonomyUUID := strings.TrimSpace(c.Param("taxonomy_uuid"))
	if taxonomyUUID == "" {
		contracts.ResponseBadRequest(c, "taxonomy_uuid is required")
		return
	}
	if h.isDelegated() {
		page, err := h.delegated.ListTaxonomyNodes(c.Request.Context(), fwmetadata.ListTaxonomyNodesRequest{
			TaxonomyUUID: taxonomyUUID, Status: opts.Status, Query: opts.Query, Locale: opts.Locale, Page: opts.Page, PageSize: opts.PageSize, RequestID: requestID(c),
		})
		respondPage(c, page, err)
		return
	}
	page, err := h.local.ListTaxonomyNodes(c.Request.Context(), taxonomyUUID, opts)
	respondPage(c, page, err)
}

func (h *Handler) CreateTaxonomyNode(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	taxonomyUUID := strings.TrimSpace(c.Param("taxonomy_uuid"))
	if taxonomyUUID == "" {
		contracts.ResponseBadRequest(c, "taxonomy_uuid is required")
		return
	}
	var req createTaxonomyNodeRequest
	if !bindJSON(c, &req) {
		return
	}
	if h.isDelegated() {
		item, err := h.delegated.CreateTaxonomyNode(c.Request.Context(), fwmetadata.CreateTaxonomyNodeRequest{
			TaxonomyUUID: taxonomyUUID, ParentUUID: req.ParentUUID, Code: req.Code, LabelI18n: req.LabelI18n, DescriptionI18n: req.DescriptionI18n, SortOrder: req.SortOrder, RequestID: requestID(c),
		})
		respondItem(c, item, err)
		return
	}
	item, err := h.local.CreateTaxonomyNode(c.Request.Context(), metadatasvc.CreateTaxonomyNodeInput{
		TenantUUID: opts.TenantUUID, TaxonomyUUID: taxonomyUUID, ParentUUID: req.ParentUUID, Code: req.Code, LabelI18n: req.LabelI18n, DescriptionI18n: req.DescriptionI18n, SortOrder: req.SortOrder,
	})
	respondItem(c, item, err)
}

func (h *Handler) ListTags(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	if h.isDelegated() {
		page, err := h.delegated.ListTags(c.Request.Context(), fwmetadata.ListTagsRequest{
			Namespace: opts.Namespace, ResourceType: opts.ResourceType, Status: opts.Status, Query: opts.Query, Locale: opts.Locale, Page: opts.Page, PageSize: opts.PageSize, RequestID: requestID(c),
		})
		respondPage(c, page, err)
		return
	}
	page, err := h.local.ListTags(c.Request.Context(), opts)
	respondPage(c, page, err)
}

func (h *Handler) CreateTag(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	var req createTagRequest
	if !bindJSON(c, &req) {
		return
	}
	if h.isDelegated() {
		item, err := h.delegated.CreateTag(c.Request.Context(), fwmetadata.CreateTagRequest{
			Namespace: req.Namespace, ResourceType: req.ResourceType, Code: req.Code, Color: req.Color, LabelI18n: req.LabelI18n, DescriptionI18n: req.DescriptionI18n, RequestID: requestID(c),
		})
		respondItem(c, item, err)
		return
	}
	item, err := h.local.CreateTag(c.Request.Context(), metadatasvc.CreateTagInput{
		TenantUUID: opts.TenantUUID, Namespace: req.Namespace, ResourceType: req.ResourceType, Code: req.Code, Color: req.Color, LabelI18n: req.LabelI18n, DescriptionI18n: req.DescriptionI18n,
	})
	respondItem(c, item, err)
}

func (h *Handler) ListResourceTypes(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	if h.isDelegated() {
		page, err := h.delegated.ListResourceTypes(c.Request.Context(), fwmetadata.ListResourceTypesRequest{
			Module: opts.Module, Status: opts.Status, Query: opts.Query, Locale: opts.Locale, Page: opts.Page, PageSize: opts.PageSize, RequestID: requestID(c),
		})
		respondPage(c, page, err)
		return
	}
	page, err := h.local.ListResourceTypes(c.Request.Context(), opts)
	respondPage(c, page, err)
}

func (h *Handler) CreateResourceType(c *gin.Context) {
	opts, ok := h.options(c)
	if !ok {
		return
	}
	var req createResourceTypeRequest
	if !bindJSON(c, &req) {
		return
	}
	if h.isDelegated() {
		item, err := h.delegated.CreateResourceType(c.Request.Context(), fwmetadata.CreateResourceTypeRequest{
			ResourceType: req.ResourceType, Module: req.Module, NameI18n: req.NameI18n, DescriptionI18n: req.DescriptionI18n, ValidatorKey: req.ValidatorKey, BindingEnabled: req.BindingEnabled, RequestID: requestID(c),
		})
		respondItem(c, item, err)
		return
	}
	item, err := h.local.CreateResourceType(c.Request.Context(), metadatasvc.CreateResourceTypeInput{
		TenantUUID: opts.TenantUUID, ResourceType: req.ResourceType, Module: req.Module, NameI18n: req.NameI18n, DescriptionI18n: req.DescriptionI18n, ValidatorKey: req.ValidatorKey, BindingEnabled: req.BindingEnabled,
	})
	respondItem(c, item, err)
}

func (h *Handler) options(c *gin.Context) (metadatasvc.ListOptions, bool) {
	if h.isDelegated() {
		return metadatasvc.ListOptions{
			Module: strings.TrimSpace(c.Query("module")), Namespace: strings.TrimSpace(c.Query("namespace")),
			ResourceType: strings.TrimSpace(c.Query("resource_type")), Status: strings.TrimSpace(c.Query("status")),
			Query: strings.TrimSpace(c.Query("q")), Locale: strings.TrimSpace(c.Query("locale")), Page: intQuery(c, "page", 1), PageSize: intQuery(c, "page_size", fwmetadata.DefaultPageSize),
		}, true
	}
	tenantUUID := admincommon.ResolveTenantUUID(c)
	if tenantUUID == "" {
		contracts.ResponseBadRequest(c, "tenant_uuid is required")
		return metadatasvc.ListOptions{}, false
	}
	if h.local == nil {
		contracts.ResponseServiceUnavailable(c, "local metadata store is unavailable", nil)
		return metadatasvc.ListOptions{}, false
	}
	return metadatasvc.ListOptions{
		TenantUUID: tenantUUID, Module: strings.TrimSpace(c.Query("module")), Namespace: strings.TrimSpace(c.Query("namespace")),
		ResourceType: strings.TrimSpace(c.Query("resource_type")), Status: strings.TrimSpace(c.Query("status")),
		Query: strings.TrimSpace(c.Query("q")), Locale: strings.TrimSpace(c.Query("locale")), Page: intQuery(c, "page", 1), PageSize: intQuery(c, "page_size", fwmetadata.DefaultPageSize),
	}, true
}

func (h *Handler) isDelegated() bool {
	return h != nil && h.mode == iamservice.IAMModeDelegated
}

func respondPage[T any](c *gin.Context, page *fwmetadata.Page[T], err error) {
	if err != nil {
		contracts.ResponseError(c, http.StatusBadGateway, "METADATA_GATEWAY_FAILED", err.Error())
		return
	}
	if page == nil {
		page = &fwmetadata.Page[T]{Items: []T{}, Pagination: fwmetadata.Pagination{Page: 1, PageSize: fwmetadata.DefaultPageSize}}
	}
	contracts.ResponseSuccess(c, gin.H{
		"items":      page.Items,
		"pagination": page.Pagination,
		"total":      page.Pagination.Total,
		"page":       page.Pagination.Page,
		"page_size":  page.Pagination.PageSize,
	})
}

func respondItem[T any](c *gin.Context, item *T, err error) {
	if err != nil {
		if errors.Is(err, metadatasvc.ErrDuplicate) {
			field := metadatasvc.DuplicateField(err)
			message := duplicateMessage(field)
			contracts.ResponseErrorWithDetails(c, http.StatusConflict, "METADATA_DUPLICATE", message, gin.H{"field": field})
			return
		}
		contracts.ResponseError(c, http.StatusBadGateway, "METADATA_GATEWAY_FAILED", err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{"payload": item})
}

func duplicateMessage(field string) string {
	switch strings.TrimSpace(field) {
	case metadatasvc.DuplicateFieldNamespace:
		return "metadata.duplicate.namespace"
	case metadatasvc.DuplicateFieldCode:
		return "metadata.duplicate.code"
	case metadatasvc.DuplicateFieldTag:
		return "metadata.duplicate.tag"
	case metadatasvc.DuplicateFieldResourceType:
		return "metadata.duplicate.resource_type"
	default:
		return "metadata.duplicate"
	}
}

func bindJSON(c *gin.Context, out any) bool {
	if err := c.ShouldBindJSON(out); err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return false
	}
	return true
}

func intQuery(c *gin.Context, key string, fallback int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func requestID(c *gin.Context) string {
	if v := strings.TrimSpace(c.GetHeader("X-Request-ID")); v != "" {
		return v
	}
	return strings.TrimSpace(c.GetString("request_id"))
}
