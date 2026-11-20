package metadata

import (
	"strings"

	fwmetadata "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/metadata"
	"github.com/gin-gonic/gin"
)

// Framework Lab uses these typed Service operations for both debug routes.
// The selected runtime and trusted local tenant context come from debugService.
func (h *Handler) DebugListDictionaryNamespaces(c *gin.Context) {
	s, ctx, ok := h.debugService(c)
	if !ok {
		return
	}
	page, err := s.ListDictionaryNamespaces(ctx, fwmetadata.ListDictionaryNamespacesRequest{Module: c.Query("module"), Status: c.Query("status"), Query: c.Query("q"), Locale: c.Query("locale"), Page: intQuery(c, "page", 1), PageSize: intQuery(c, "page_size", fwmetadata.DefaultPageSize), RequestID: requestID(c)})
	respondPage(c, page, err)
}

func (h *Handler) DebugCreateDictionaryNamespace(c *gin.Context) {
	s, ctx, ok := h.debugService(c)
	if !ok {
		return
	}
	var req createDictionaryNamespaceRequest
	if !bindJSON(c, &req) {
		return
	}
	item, err := s.CreateDictionaryNamespace(ctx, fwmetadata.CreateDictionaryNamespaceRequest{Namespace: req.Namespace, Module: req.Module, NameI18n: req.NameI18n, DescriptionI18n: req.DescriptionI18n, RequestID: requestID(c)})
	respondItem(c, item, err)
}

func (h *Handler) DebugListDictionaryItems(c *gin.Context) {
	s, ctx, ok := h.debugService(c)
	if !ok {
		return
	}
	page, err := s.ListDictionaryItems(ctx, fwmetadata.ListDictionaryItemsRequest{NamespaceUUID: strings.TrimSpace(c.Param("namespace_uuid")), Status: c.Query("status"), Query: c.Query("q"), Locale: c.Query("locale"), Page: intQuery(c, "page", 1), PageSize: intQuery(c, "page_size", fwmetadata.DefaultPageSize), RequestID: requestID(c)})
	respondPage(c, page, err)
}

func (h *Handler) DebugCreateDictionaryItem(c *gin.Context) {
	s, ctx, ok := h.debugService(c)
	if !ok {
		return
	}
	var req createDictionaryItemRequest
	if !bindJSON(c, &req) {
		return
	}
	item, err := s.CreateDictionaryItem(ctx, fwmetadata.CreateDictionaryItemRequest{NamespaceUUID: strings.TrimSpace(c.Param("namespace_uuid")), Code: req.Code, LabelI18n: req.LabelI18n, DescriptionI18n: req.DescriptionI18n, SortOrder: req.SortOrder, RequestID: requestID(c)})
	respondItem(c, item, err)
}

func (h *Handler) DebugListTaxonomies(c *gin.Context) {
	s, ctx, ok := h.debugService(c)
	if !ok {
		return
	}
	page, err := s.ListTaxonomies(ctx, fwmetadata.ListTaxonomiesRequest{Module: c.Query("module"), Status: c.Query("status"), Query: c.Query("q"), Locale: c.Query("locale"), Page: intQuery(c, "page", 1), PageSize: intQuery(c, "page_size", fwmetadata.DefaultPageSize), RequestID: requestID(c)})
	respondPage(c, page, err)
}

func (h *Handler) DebugCreateTaxonomy(c *gin.Context) {
	s, ctx, ok := h.debugService(c)
	if !ok {
		return
	}
	var req createTaxonomyRequest
	if !bindJSON(c, &req) {
		return
	}
	item, err := s.CreateTaxonomy(ctx, fwmetadata.CreateTaxonomyRequest{Namespace: req.Namespace, Module: req.Module, NameI18n: req.NameI18n, DescriptionI18n: req.DescriptionI18n, MaxDepth: req.MaxDepth, RequestID: requestID(c)})
	respondItem(c, item, err)
}

func (h *Handler) DebugCreateTaxonomyNode(c *gin.Context) {
	s, ctx, ok := h.debugService(c)
	if !ok {
		return
	}
	var req createTaxonomyNodeRequest
	if !bindJSON(c, &req) {
		return
	}
	item, err := s.CreateTaxonomyNode(ctx, fwmetadata.CreateTaxonomyNodeRequest{TaxonomyUUID: strings.TrimSpace(c.Param("taxonomy_uuid")), ParentUUID: req.ParentUUID, Code: req.Code, LabelI18n: req.LabelI18n, DescriptionI18n: req.DescriptionI18n, SortOrder: req.SortOrder, RequestID: requestID(c)})
	respondItem(c, item, err)
}

func (h *Handler) DebugListResourceTypes(c *gin.Context) {
	s, ctx, ok := h.debugService(c)
	if !ok {
		return
	}
	page, err := s.ListResourceTypes(ctx, fwmetadata.ListResourceTypesRequest{Module: c.Query("module"), Status: c.Query("status"), Query: c.Query("q"), Locale: c.Query("locale"), Page: intQuery(c, "page", 1), PageSize: intQuery(c, "page_size", fwmetadata.DefaultPageSize), RequestID: requestID(c)})
	respondPage(c, page, err)
}

func (h *Handler) DebugCreateResourceType(c *gin.Context) {
	s, ctx, ok := h.debugService(c)
	if !ok {
		return
	}
	var req createResourceTypeRequest
	if !bindJSON(c, &req) {
		return
	}
	item, err := s.CreateResourceType(ctx, fwmetadata.CreateResourceTypeRequest{ResourceType: req.ResourceType, Module: req.Module, NameI18n: req.NameI18n, DescriptionI18n: req.DescriptionI18n, ValidatorKey: req.ValidatorKey, BindingEnabled: req.BindingEnabled, RequestID: requestID(c)})
	respondItem(c, item, err)
}
