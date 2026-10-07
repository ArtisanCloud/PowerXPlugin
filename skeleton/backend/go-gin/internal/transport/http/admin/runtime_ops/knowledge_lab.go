package runtime_ops

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwiam "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/delegated"
	iamerrors "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/errors"
	"io"
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
		if string(deps.IAMAdapterMode) == "delegated" {
			h.departmentDirectory = deps.IAMDirectoryService
		}
	} else if deps.ProviderMode == fwprovider.ModeLocal && strings.EqualFold(strings.TrimSpace(gw.AuthScheme), "apikey") {
		var err error
		client, err = powerxknowledge.NewClientWithAPIKey(powerxknowledge.Config{BaseURL: gw.BaseURL, Timeout: gw.Timeout}, gw.APIKey, nil)
		if err != nil {
			return h
		}
		directory, directoryErr := fwiam.NewCoreClientWithAPIKey(fwiam.CoreClientConfig{BaseURL: gw.BaseURL, Timeout: gw.Timeout}, gw.APIKey)
		if directoryErr == nil {
			h.departmentDirectory = directory
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
	g.GET("/index-jobs/:jobID", h.LabIndexJob)
	g.GET("/departments", h.LabDepartments)
	g.GET("/spaces", h.Spaces)
	g.POST("/spaces", h.CreateSpace)
	g.POST("/spaces/:spaceID/documents", h.LabUpsertDocument)
	g.POST("/media/upload", h.UploadIngestionSource)
	g.POST("/spaces/:spaceID/ingest", h.IngestSpace)
	g.POST("/spaces/:spaceID/retire", h.RetireSpace)
	g.DELETE("/spaces/:spaceID", h.DeleteSpace)
	g.GET("/spaces/:spaceID/ingestions", h.Ingestions)
	g.GET("/spaces/:spaceID/policy", h.Policy)
	g.POST("/search", h.Search)
}

type knowledgeLabDepartmentDirectory interface {
	ListDepartments(context.Context, string) ([]fwiamcontracts.Department, error)
}

func (h *KnowledgeHandler) LabDepartments(c *gin.Context) {
	if c.Query("tenant_uuid") != "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": iamerrors.CodeInvalidArgument, "error": "knowledgeLab.tenantOverrideRejected"})
		return
	}
	if h.departmentDirectory == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "code": iamerrors.CodeAdapterNotBound, "error": "knowledgeLab.departmentsUnavailable"})
		return
	}
	// The service credential owns Core tenant scope. The local plugin's tenant
	// UUID is not a Core tenant assertion and must never be sent upstream.
	items, err := h.departmentDirectory.ListDepartments(c.Request.Context(), "")
	if err != nil {
		status := iamerrors.StatusCode(err)
		trace := ""
		var coreErr *fwiam.CoreRequestError
		if errors.As(err, &coreErr) {
			status = coreErr.Status
			trace = coreErr.TraceID
		}
		c.JSON(status, gin.H{"success": false, "code": iamerrors.CodeOf(err), "error": "knowledgeLab.gatewayFailed", "trace_id": trace})
		return
	}
	tenant := ""
	for _, item := range items {
		id, idErr := uuid.Parse(item.DepartmentUUID)
		tenantID, tenantErr := uuid.Parse(item.TenantUUID)
		parentErr := error(nil)
		if item.ParentDepartmentUUID != "" {
			_, parentErr = uuid.Parse(item.ParentDepartmentUUID)
		}
		if idErr != nil || id == uuid.Nil || tenantErr != nil || tenantID == uuid.Nil || parentErr != nil || strings.TrimSpace(item.Name) == "" || (tenant != "" && tenant != item.TenantUUID) {
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "code": iamerrors.CodeUpstreamDependency, "error": "knowledgeLab.departmentsInvalid"})
			return
		}
		tenant = item.TenantUUID
	}
	if items == nil {
		items = []fwiamcontracts.Department{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items}})
}

// LabCreateSpace only accepts the typed service DTO, never an admin/proxy payload.
func (h *KnowledgeHandler) LabCreateSpace(c *gin.Context) {
	provider, err := h.provider()
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	provisioning, ok := provider.(fwknowledge.SpaceProvisioningProvider)
	if !ok {
		writeLabKnowledgeError(c, fwknowledge.Unsupported(fwknowledge.OperationCreate))
		return
	}
	var input fwknowledge.CreateSpaceInput
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeLabKnowledgeError(c, fwknowledge.NewError(fwknowledge.CodeInvalidArgument, "knowledgeLab.invalidCreateRequest"))
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeLabKnowledgeError(c, fwknowledge.NewError(fwknowledge.CodeInvalidArgument, "knowledgeLab.invalidCreateRequest"))
		return
	}
	item, err := provisioning.CreateSpace(c.Request.Context(), input)
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": gin.H{"item": item}})
}
func writeLabKnowledgeError(c *gin.Context, err error) {
	trace := ""
	var typed *fwknowledge.Error
	if errors.As(err, &typed) {
		trace = typed.TraceID
	}
	c.JSON(fwknowledge.HTTPStatus(err), gin.H{"success": false, "code": fwknowledge.CodeOf(err), "error": "knowledgeLab.gatewayFailed", "trace_id": trace})
}

// LabUpsertDocument uses only the typed Host service plane. Core owns tenant,
// document persistence and indexing; no Local document is written here.
func (h *KnowledgeHandler) LabUpsertDocument(c *gin.Context) {
	provider, err := h.provider()
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	var input struct {
		Title       string   `json:"title"`
		Content     string   `json:"content"`
		ContentType string   `json:"content_type"`
		Tags        []string `json:"tags"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 8<<20))
	decoder.DisallowUnknownFields()
	var trailing any
	if err := decoder.Decode(&input); err != nil || decoder.Decode(&trailing) != io.EOF || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Content) == "" || (input.ContentType != "text/plain" && input.ContentType != "text/markdown") {
		writeLabKnowledgeError(c, fwknowledge.NewError(fwknowledge.CodeInvalidArgument, "knowledgeLab.workspace.invalidDocument"))
		return
	}
	job, err := provider.UpsertDocument(c.Request.Context(), fwknowledge.KnowledgeDocument{
		SpaceID: c.Param("spaceID"), Title: strings.TrimSpace(input.Title), Content: input.Content,
		ContentType: input.ContentType, Checksum: fmt.Sprintf("%x", sha256.Sum256([]byte(input.Content))),
		URI: "plugin-knowledge://" + uuid.NewString(), Version: "v1", Tags: input.Tags,
	})
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	if job == nil {
		writeLabKnowledgeError(c, fwknowledge.NewError(fwknowledge.CodeInvalidResponse, "knowledgeLab.workspace.invalidResponse"))
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": gin.H{"item": job}})
}

func (h *KnowledgeHandler) LabIndexJob(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("jobID"))
	if err != nil || jobID == uuid.Nil || c.Query("tenant_uuid") != "" {
		writeLabKnowledgeError(c, fwknowledge.NewError(fwknowledge.CodeInvalidArgument, "knowledgeLab.workspace.invalidDocument"))
		return
	}
	provider, err := h.provider()
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	job, err := provider.GetIndexJob(c.Request.Context(), fwknowledge.IndexJobQuery{JobID: jobID.String()})
	if err != nil {
		writeLabKnowledgeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"item": job}})
}
