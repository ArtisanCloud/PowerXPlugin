package capability

import (
	"errors"
	"net/http"
	"strings"

	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	capservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/capability"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// CatalogHandler exposes read-only capability catalog endpoints.
type CatalogHandler struct {
	service *capservice.CatalogService
}

type sourceOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type grantStatusRequest struct {
	CapabilityIDs []string `json:"capability_ids"`
}

// NewCatalogHandler wires catalog handler when the service is available.
func NewCatalogHandler(deps *app.Deps) *CatalogHandler {
	svc := capservice.NewCatalogService(deps)
	if svc == nil {
		return nil
	}
	return &CatalogHandler{service: svc}
}

// List returns all capabilities collected in the catalog snapshot.
func (h *CatalogHandler) List(c *gin.Context) {
	source := strings.TrimSpace(c.Query("source"))
	entries, err := h.service.List(c.Request.Context(), capservice.ListOptions{Source: source})
	if err != nil {
		logger.ErrorCtx(logger.WithLogFields(c.Request.Context(), map[string]interface{}{
			"module":     "capability",
			"biz_scene":  "capability_catalog_list",
			"biz_domain": "capability",
			"component":  "capability_catalog_handler",
			"error":      err.Error(),
		}), "failed to load capability catalog")
		contracts.ResponseErrorWithDetails(
			c,
			http.StatusInternalServerError,
			contracts.ErrCodeInternalError,
			"failed to load capability catalog",
			gin.H{"error": err.Error()},
		)
		return
	}
	logger.InfoCtx(logger.WithLogFields(c.Request.Context(), map[string]interface{}{
		"component":    "capability_catalog_handler",
		"entry_count":  len(entries),
		"request_path": c.FullPath(),
		"module":       "capability",
		"biz_scene":    "capability_catalog_list",
		"biz_domain":   "capability",
	}), "capability catalog request handled")
	contracts.ResponseSuccess(c, entries)
}

// Sources returns capability source enums and alias mapping.
func (h *CatalogHandler) Sources(c *gin.Context) {
	contracts.ResponseSuccess(c, gin.H{
		"default": "all",
		"aliases": gin.H{
			"all":      "all",
			"any":      "all",
			"platform": "corex",
		},
		"sources": []sourceOption{
			{ID: "all", Label: "all", Description: "查询全部来源（不传 source 或 source=all）"},
			{ID: "corex", Label: "corex", Description: "PowerX 底座能力"},
			{ID: "plugin", Label: "plugin", Description: "插件/租户注册能力"},
		},
	})
}

// GrantStatus proxies the typed Framework client. The browser never receives
// a Gateway credential and cannot choose a tenant or service actor.
func (h *CatalogHandler) GrantStatus(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability grant-status service not available", nil)
		return
	}
	var request grantStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil || len(request.CapabilityIDs) == 0 {
		contracts.ResponseErrorWithDetails(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, "capability_ids is required", nil)
		return
	}
	items, err := h.service.GrantStatus(c.Request.Context(), request.CapabilityIDs)
	if err != nil {
		var upstream *powerxcapability.HTTPError
		if errors.As(err, &upstream) && upstream != nil {
			contracts.ResponseErrorWithDetails(c, upstream.StatusCode, contracts.ErrCodeInternalError, "capability grant-status request rejected", gin.H{"upstream": upstream.Body})
			return
		}
		contracts.ResponseErrorWithDetails(c, http.StatusBadGateway, contracts.ErrCodeInternalError, "failed to read capability grant status", gin.H{"error": err.Error()})
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}
