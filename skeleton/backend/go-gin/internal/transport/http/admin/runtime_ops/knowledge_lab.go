package runtime_ops

import (
	"net/http"
	"strings"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	powerxknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/knowledge"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// NewKnowledgeLabHandler fixes only the administrative Lab to PowerX. The
// startup-selected provider remains authoritative for normal plugin traffic.
func NewKnowledgeLabHandler(deps *app.Deps) *KnowledgeHandler {
	h := &KnowledgeHandler{deps: deps, labOnly: true}
	h.providerErr = fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "knowledgeLab.proxyUnavailable")
	if deps == nil || deps.Config == nil || deps.Config.Gateway == nil {
		return h
	}
	gw := deps.Config.Gateway
	var client fwknowledge.DelegatedClient
	if deps.ProviderMode == fwprovider.ModeDelegated {
		client = deps.KnowledgeDirectory
	} else if deps.ProviderMode == fwprovider.ModeLocal && strings.EqualFold(strings.TrimSpace(gw.AuthScheme), "apikey") {
		var err error
		client, err = powerxknowledge.NewClientWithAPIKey(powerxknowledge.Config{BaseURL: gw.BaseURL, Timeout: gw.Timeout}, gw.APIKey, nil)
		if err != nil {
			return h
		}
	}
	if client == nil || !h.gatewayReady() {
		return h
	}
	h.knowledgeProvider = fwknowledge.NewDelegatedProvider(fwknowledge.DelegatedProviderConfig{Name: "powerx_knowledge_lab", Client: client, Timeout: gw.Timeout})
	h.providerErr = nil
	return h
}

func registerKnowledgeLabRoutes(router *gin.RouterGroup, deps *app.Deps) {
	h := NewKnowledgeLabHandler(deps)
	g := router.Group("/knowledge-lab")
	// All Lab operations fail closed, including upload and legacy admin actions.
	g.Use(func(c *gin.Context) {
		if _, err := h.provider(); err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"success": false, "code": fwknowledge.CodeProviderUnavailable, "error": "knowledgeLab.proxyUnavailable"})
			return
		}
		if spaceID := c.Param("spaceID"); spaceID != "" {
			if _, err := uuid.Parse(spaceID); err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"success": false, "code": fwknowledge.CodeInvalidDocument, "error": "knowledgeLab.invalidSpace"})
				return
			}
		}
		c.Next()
	})
	g.GET("/provider", h.Provider)
	g.GET("/catalog", h.Catalog)
	g.GET("/spaces", h.Spaces)
	g.POST("/spaces", h.CreateSpace)
	g.POST("/media/upload", h.UploadIngestionSource)
	g.POST("/spaces/:spaceID/ingest", h.IngestSpace)
	g.POST("/spaces/:spaceID/retire", h.RetireSpace)
	g.DELETE("/spaces/:spaceID", h.DeleteSpace)
	g.GET("/spaces/:spaceID/ingestions", h.Ingestions)
	g.GET("/spaces/:spaceID/policy", h.Policy)
	g.POST("/search", h.Search)
}
