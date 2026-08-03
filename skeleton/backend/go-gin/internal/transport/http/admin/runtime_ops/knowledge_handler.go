package runtime_ops

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	fwgateway "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	fwmedia "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/media"
	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	capgateway "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	knowledgeSvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/knowledge"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	admincommon "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/admin/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	knowledgeCapabilityListFusionStrategies = "com.corex.rest.admin.gin.get_api_v1_admin_knowledge_spaces_spaceid_fusion_strategies"
	knowledgeCapabilityPlanRetrieval        = "com.corex.rest.admin.gin.post_api_v1_admin_knowledge_spaces_spaceid_playground_retrieval"
	knowledgeCapabilityListIngestionJobs    = "com.corex.rest.admin.gin.get_api_v1_admin_knowledge_spaces_spaceid_ingestion_jobs"
	knowledgeCapabilityCreateIngestionJob   = "com.corex.rest.admin.gin.post_api_v1_admin_knowledge_spaces_spaceid_ingestion_jobs"
)

type KnowledgeHandler struct {
	deps              *app.Deps
	providerOnce      sync.Once
	knowledgeProvider fwknowledge.KnowledgeProvider
	providerErr       error
}

type knowledgeSearchRequest struct {
	Query      string          `json:"query"`
	SpaceID    string          `json:"space_id"`
	TenantUUID string          `json:"tenant_uuid"`
	PluginID   string          `json:"plugin_id"`
	AgentUUID  string          `json:"agent_uuid"`
	SkillID    string          `json:"skill_id"`
	Visibility string          `json:"visibility"`
	Limit      int             `json:"limit"`
	Tags       []string        `json:"tags"`
	Fixture    *fixturePayload `json:"fixture"`
	Filters    map[string]any  `json:"filters"`
}

type knowledgeCreateSpaceRequest struct {
	TenantUUID              string   `json:"tenant_uuid"`
	SpaceName               string   `json:"spaceName"`
	Description             string   `json:"description"`
	DepartmentCode          string   `json:"departmentCode"`
	Visibility              string   `json:"visibility"`
	PolicyTemplateVersionID string   `json:"policyTemplateVersionId"`
	IngestionProfileKey     string   `json:"ingestionProfileKey"`
	IndexProfileKey         string   `json:"indexProfileKey"`
	RAGProfileKey           string   `json:"ragProfileKey"`
	CPUCores                int      `json:"cpuCores"`
	StorageGB               int      `json:"storageGb"`
	IngestionConcurrency    int      `json:"ingestionConcurrency"`
	FeatureFlags            []string `json:"featureFlags"`
	RequestedBy             string   `json:"requestedBy"`
}

type knowledgeRetireSpaceRequest struct {
	Reason      string `json:"reason"`
	RequestedBy string `json:"requestedBy"`
	DropVectors bool   `json:"dropVectors"`
}

type knowledgeDeleteSpaceRequest struct {
	RequestedBy string `json:"requestedBy"`
	Force       bool   `json:"force"`
	DropVectors bool   `json:"dropVectors"`
}

type knowledgeIngestSpaceRequest struct {
	Format           string `json:"format"`
	SourceURI        string `json:"sourceUri"`
	IngestionProfile string `json:"ingestionProfile"`
	RequestedBy      string `json:"requestedBy"`
}

type knowledgeMediaGatewayAdapter struct {
	gateway interface {
		Invoke(ctx context.Context, params capgateway.InvokeParams) (*capgateway.InvokeResult, error)
	}
}

func (a knowledgeMediaGatewayAdapter) Invoke(ctx context.Context, req fwgateway.InvokeRequest) (*fwgateway.Response, error) {
	if a.gateway == nil {
		return nil, fmt.Errorf("PowerX Gateway 客户端未配置")
	}
	result, err := a.gateway.Invoke(ctx, capgateway.InvokeParams{
		CapabilityID:      req.CapabilityID,
		Action:            req.Action,
		PreferredProtocol: req.PreferredProtocol,
		Payload:           req.Payload,
		Headers:           req.Headers,
		RequestID:         req.RequestID,
		TenantUUID:        req.TenantUUID,
		AuthRequired:      !req.DisableAuth,
		TenantScoped:      false,
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("PowerX Gateway 返回空 media 响应")
	}
	return &fwgateway.Response{
		TraceID: result.TraceID,
		Status:  result.Status,
		Data:    result.Data,
		RawData: result.Raw,
	}, nil
}

type knowledgeSpaceDebugRow struct {
	ID                      string         `json:"id"`
	Name                    string         `json:"name"`
	SpaceID                 string         `json:"spaceId"`
	SpaceName               string         `json:"spaceName"`
	TenantUUID              string         `json:"tenant_uuid,omitempty"`
	Department              string         `json:"department,omitempty"`
	DepartmentCode          string         `json:"departmentCode,omitempty"`
	Type                    string         `json:"type"`
	TypeLabel               string         `json:"type_label"`
	Status                  string         `json:"status"`
	StatusLabel             string         `json:"status_label"`
	Provider                string         `json:"provider"`
	ProviderMode            string         `json:"provider_mode"`
	PolicyTemplateVersionID string         `json:"policyTemplateVersionId,omitempty"`
	IngestionProfileKey     string         `json:"ingestionProfileKey,omitempty"`
	IndexProfileKey         string         `json:"indexProfileKey,omitempty"`
	RAGProfileKey           string         `json:"ragProfileKey,omitempty"`
	FeatureFlags            []string       `json:"featureFlags,omitempty"`
	Quotas                  map[string]any `json:"quotas,omitempty"`
	Operations              []string       `json:"operations,omitempty"`
	ContractGaps            []string       `json:"contract_gaps,omitempty"`
	Icon                    string         `json:"icon,omitempty"`
}

func NewKnowledgeHandler(deps *app.Deps) *KnowledgeHandler {
	return &KnowledgeHandler{deps: deps}
}

func (h *KnowledgeHandler) Provider(c *gin.Context) {
	provider, err := h.provider()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	caps := provider.Capabilities(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"provider":     provider.Name(),
			"mode":         provider.Mode(),
			"capabilities": caps,
			"production":   h.deps != nil && h.deps.Config != nil && h.deps.Config.IsProduction(),
		},
	})
}

func knowledgeSpaceRow(space fwknowledge.KnowledgeSpace, provider fwknowledge.KnowledgeProvider, caps fwknowledge.ProviderCapabilities) knowledgeSpaceDebugRow {
	spaceID := firstNonEmpty(space.SpaceID, "unknown")
	name := firstNonEmpty(space.SpaceName, spaceID)
	visibility := firstNonEmpty(space.Visibility, fwknowledge.VisibilityTenant)
	status := firstNonEmpty(space.Status, "active")
	department := firstNonEmpty(space.DepartmentCode, "-")
	return knowledgeSpaceDebugRow{
		ID:                      spaceID,
		Name:                    name,
		SpaceID:                 spaceID,
		SpaceName:               name,
		TenantUUID:              space.TenantUUID,
		Department:              department,
		DepartmentCode:          department,
		Type:                    visibility,
		TypeLabel:               knowledgeVisibilityLabel(visibility),
		Status:                  status,
		StatusLabel:             knowledgeStatusLabel(status),
		Provider:                provider.Name(),
		ProviderMode:            provider.Mode(),
		PolicyTemplateVersionID: space.PolicyTemplateVersionID,
		IngestionProfileKey:     space.IngestionProfileKey,
		IndexProfileKey:         space.IndexProfileKey,
		RAGProfileKey:           space.RAGProfileKey,
		FeatureFlags:            append([]string(nil), space.FeatureFlags...),
		Quotas:                  copyMap(space.Quotas),
		Operations:              append([]string(nil), caps.Operations...),
		Icon:                    knowledgeSpaceIcon(provider.Mode()),
	}
}

func knowledgeVisibilityLabel(visibility string) string {
	switch visibility {
	case fwknowledge.VisibilityTenant:
		return "租户知识"
	case fwknowledge.VisibilityPlugin:
		return "插件知识"
	case fwknowledge.VisibilityPublic:
		return "公共知识"
	case fwknowledge.VisibilityPrivate:
		return "私有知识"
	default:
		return firstNonEmpty(visibility, "知识空间")
	}
}

func knowledgeStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active", "ready":
		return "可用"
	case "pending_iam":
		return "待权限配置"
	case "pending":
		return "处理中"
	case "retired":
		return "已归档"
	case "unavailable":
		return "不可用"
	default:
		return firstNonEmpty(status, "未知")
	}
}

func knowledgeSpaceIcon(mode string) string {
	if mode == fwknowledge.ProviderModeDelegated {
		return "i-heroicons-cloud"
	}
	return "i-heroicons-circle-stack"
}

func copyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (h *KnowledgeHandler) Spaces(c *gin.Context) {
	provider, err := h.provider()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	tenantUUID, tenantMismatch := resolveKnowledgeTenantUUID(c, c.Query("tenant_uuid"))
	if tenantMismatch {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "tenant_uuid mismatch", "code": fwknowledge.CodeForbidden})
		return
	}
	caps := provider.Capabilities(c.Request.Context())
	spaces, err := provider.ListSpaces(c.Request.Context(), fwknowledge.ListSpacesInput{
		TenantUUID: tenantUUID,
		PluginID:   c.Query("plugin_id"),
		Visibility: c.Query("visibility"),
		Status:     c.Query("status"),
		Limit:      100,
		TraceID:    c.GetString("request_id"),
	})
	if err != nil {
		status := fwknowledge.HTTPStatusForCode(fwknowledge.CodeOf(err))
		c.JSON(status, gin.H{"success": false, "error": err.Error(), "code": fwknowledge.CodeOf(err)})
		return
	}
	rows := make([]knowledgeSpaceDebugRow, 0, len(spaces))
	for _, space := range spaces {
		rows = append(rows, knowledgeSpaceRow(space, provider, caps))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"spaces": rows,
		"source": provider.Mode(),
	}})
}

func (h *KnowledgeHandler) Catalog(c *gin.Context) {
	provider, err := h.provider()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	catalog, err := provider.Catalog(c.Request.Context())
	if err != nil {
		status := fwknowledge.HTTPStatusForCode(fwknowledge.CodeOf(err))
		c.JSON(status, gin.H{"success": false, "error": err.Error(), "code": fwknowledge.CodeOf(err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": catalog})
}

func (h *KnowledgeHandler) CreateSpace(c *gin.Context) {
	var req knowledgeCreateSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid knowledge space create request"})
		return
	}
	if strings.TrimSpace(req.SpaceName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "spaceName is required"})
		return
	}
	if strings.TrimSpace(req.DepartmentCode) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "departmentCode is required"})
		return
	}
	provider, err := h.provider()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if provider.Mode() != fwknowledge.ProviderModeDelegated {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "知识空间创建只支持 PowerX delegated 模式；local 模式请使用本地入库模拟"})
		return
	}
	if !h.gatewayReady() {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": "PowerX Gateway 客户端未配置", "code": fwknowledge.CodeProviderUnavailable})
		return
	}
	tenantUUID, tenantMismatch := resolveKnowledgeTenantUUID(c, req.TenantUUID)
	if tenantMismatch {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "tenant_uuid mismatch", "code": fwknowledge.CodeForbidden})
		return
	}
	record, err := h.deps.CapabilityGateway.CreateKnowledgeSpace(c.Request.Context(), capgateway.KnowledgeSpaceCreateParams{
		TenantUUID:              tenantUUID,
		SpaceName:               req.SpaceName,
		Description:             req.Description,
		DepartmentCode:          req.DepartmentCode,
		Visibility:              firstNonEmpty(req.Visibility, fwknowledge.VisibilityTenant),
		PolicyTemplateVersionID: firstNonEmpty(req.PolicyTemplateVersionID, "default-v1"),
		IngestionProfileKey:     firstNonEmpty(req.IngestionProfileKey, "default"),
		IndexProfileKey:         firstNonEmpty(req.IndexProfileKey, "default"),
		RAGProfileKey:           firstNonEmpty(req.RAGProfileKey, "p1_general"),
		CPUCores:                positiveOrDefault(req.CPUCores, 1),
		StorageGB:               positiveOrDefault(req.StorageGB, 50),
		IngestionConcurrency:    positiveOrDefault(req.IngestionConcurrency, 1),
		FeatureFlags:            append([]string(nil), req.FeatureFlags...),
		RequestedBy:             firstNonEmpty(req.RequestedBy, "plugin-knowledge-lab"),
	})
	if err != nil {
		status, code := knowledgeGatewayErrorStatus(err)
		c.JSON(status, gin.H{"success": false, "error": err.Error(), "code": code})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": record})
}

func (h *KnowledgeHandler) RetireSpace(c *gin.Context) {
	spaceID := strings.TrimSpace(c.Param("spaceID"))
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "knowledge space_id is required", "code": fwknowledge.CodeInvalidDocument})
		return
	}
	var req knowledgeRetireSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid knowledge space retire request", "code": fwknowledge.CodeInvalidDocument})
		return
	}
	provider, err := h.provider()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if provider.Mode() != fwknowledge.ProviderModeDelegated {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "知识空间归档只支持 PowerX delegated 模式；local 模式请重置本地 fixture", "code": fwknowledge.CodeUnsupportedCapability})
		return
	}
	if !h.gatewayReady() {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": "PowerX Gateway 客户端未配置", "code": fwknowledge.CodeProviderUnavailable})
		return
	}
	tenantUUID, tenantMismatch := resolveKnowledgeTenantUUID(c, "")
	if tenantMismatch {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "tenant_uuid mismatch", "code": fwknowledge.CodeForbidden})
		return
	}
	record, err := h.deps.CapabilityGateway.RetireKnowledgeSpace(c.Request.Context(), capgateway.KnowledgeSpaceRetireParams{
		TenantUUID:  tenantUUID,
		SpaceID:     spaceID,
		Reason:      firstNonEmpty(req.Reason, "retired from plugin knowledge lab"),
		RequestedBy: firstNonEmpty(req.RequestedBy, "plugin-knowledge-lab"),
		DropVectors: req.DropVectors,
	})
	if err != nil {
		status, code := knowledgeGatewayErrorStatus(err)
		c.JSON(status, gin.H{"success": false, "error": err.Error(), "code": code})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": record})
}

func (h *KnowledgeHandler) DeleteSpace(c *gin.Context) {
	spaceID := strings.TrimSpace(c.Param("spaceID"))
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "knowledge space_id is required", "code": fwknowledge.CodeInvalidDocument})
		return
	}
	var req knowledgeDeleteSpaceRequest
	if c.Request != nil && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid knowledge space delete request", "code": fwknowledge.CodeInvalidDocument})
			return
		}
	}
	provider, err := h.provider()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if provider.Mode() != fwknowledge.ProviderModeDelegated {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "知识空间删除只支持 PowerX delegated 模式；local 模式请重置本地 fixture", "code": fwknowledge.CodeUnsupportedCapability})
		return
	}
	if !h.gatewayReady() {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": "PowerX Gateway 客户端未配置", "code": fwknowledge.CodeProviderUnavailable})
		return
	}
	tenantUUID, tenantMismatch := resolveKnowledgeTenantUUID(c, "")
	if tenantMismatch {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "tenant_uuid mismatch", "code": fwknowledge.CodeForbidden})
		return
	}
	err = h.deps.CapabilityGateway.DeleteKnowledgeSpace(c.Request.Context(), capgateway.KnowledgeSpaceDeleteParams{
		TenantUUID:  tenantUUID,
		SpaceID:     spaceID,
		RequestedBy: firstNonEmpty(req.RequestedBy, "plugin-knowledge-lab"),
		Force:       req.Force,
		DropVectors: req.DropVectors,
	})
	if err != nil {
		status, code := knowledgeGatewayErrorStatus(err)
		c.JSON(status, gin.H{"success": false, "error": err.Error(), "code": code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"deleted": true, "spaceId": spaceID}})
}

func knowledgeGatewayErrorStatus(err error) (int, fwknowledge.ErrorCode) {
	var apiErr *capgateway.PlatformAPIError
	if errors.As(err, &apiErr) && apiErr != nil {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized:
			return http.StatusUnauthorized, fwknowledge.CodeUnauthorized
		case http.StatusForbidden:
			return http.StatusForbidden, fwknowledge.CodeForbidden
		case http.StatusConflict:
			return http.StatusConflict, fwknowledge.CodeConflict
		case http.StatusNotFound:
			return http.StatusNotFound, fwknowledge.CodeNotFound
		case http.StatusTooManyRequests:
			return http.StatusTooManyRequests, fwknowledge.CodeRateLimited
		default:
			if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
				return apiErr.StatusCode, fwknowledge.CodeInvalidDocument
			}
		}
	}
	return http.StatusBadGateway, fwknowledge.CodeProviderUnavailable
}

func (h *KnowledgeHandler) Ingestions(c *gin.Context) {
	provider, err := h.provider()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	spaceID := strings.TrimSpace(c.Param("spaceID"))
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "knowledge space_id is required"})
		return
	}
	if provider.Mode() == fwknowledge.ProviderModeDelegated && h.gatewayReady() {
		limit := strings.TrimSpace(c.DefaultQuery("limit", "20"))
		result, err := h.invokeKnowledgeCapability(c, knowledgeCapabilityListIngestionJobs, "ListIngestionJobs", map[string]any{
			"method":   http.MethodGet,
			"endpoint": "/api/v1/admin/knowledge-spaces/" + url.PathEscape(spaceID) + "/ingestion-jobs",
			"query": map[string]any{
				"limit": limit,
			},
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": err.Error(), "code": fwknowledge.CodeProviderUnavailable})
			return
		}
		records := ingestionJobRecordsFromGatewayResult(result)
		status := knowledgeGatewayResultStatus(result)
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
			"space_id":      spaceID,
			"provider":      provider.Name(),
			"provider_mode": provider.Mode(),
			"source":        "powerx_gateway_capability",
			"capability_id": knowledgeCapabilityListIngestionJobs,
			"trace_id":      result.TraceID,
			"status":        status,
			"records":       records,
			"items":         records,
			"total":         len(records),
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"space_id": spaceID,
		"records": []gin.H{
			{
				"id":            "debug-local-fixture",
				"operation":     fwknowledge.OperationUpsert,
				"status":        "available_in_local_fixture_mode",
				"provider":      provider.Name(),
				"provider_mode": provider.Mode(),
				"updated_at":    time.Now().UTC().Format(time.RFC3339),
			},
		},
		"note": "PowerX 委托入库历史需要等待底座知识 API 继续补齐",
	}})
}

func ingestionJobRecordsFromGatewayResult(result *knowledgeInvokeResult) []any {
	if result == nil {
		return []any{}
	}
	if records, ok := ingestionJobRecordsFromGatewayMap(result.Data); ok {
		return records
	}
	if len(result.Raw) > 0 && string(result.Raw) != "null" {
		var records []any
		if err := json.Unmarshal(result.Raw, &records); err == nil {
			return records
		}
		var envelope map[string]any
		if err := json.Unmarshal(result.Raw, &envelope); err == nil {
			if records, ok := ingestionJobRecordsFromGatewayMap(envelope); ok {
				return records
			}
		}
	}
	return []any{}
}

func ingestionJobRecordsFromGatewayMap(data map[string]any) ([]any, bool) {
	if data == nil {
		return nil, false
	}
	for _, key := range []string{"items", "records", "data"} {
		if records, ok := anySliceFromGatewayValue(data[key]); ok {
			return records, true
		}
	}
	for _, key := range []string{"payload", "result"} {
		if nested, ok := data[key].(map[string]any); ok {
			if records, ok := ingestionJobRecordsFromGatewayMap(nested); ok {
				return records, true
			}
		}
	}
	if records, ok := anySliceFromGatewayValue(data); ok {
		return records, true
	}
	return nil, false
}

func anySliceFromGatewayValue(value any) ([]any, bool) {
	switch typed := value.(type) {
	case []any:
		return typed, true
	case []map[string]any:
		records := make([]any, 0, len(typed))
		for _, record := range typed {
			records = append(records, record)
		}
		return records, true
	default:
		return nil, false
	}
}

func knowledgeGatewayResultStatus(result *knowledgeInvokeResult) string {
	if result == nil {
		return ""
	}
	for _, value := range []any{
		result.Status,
		result.Data["status"],
		result.Data["Status"],
	} {
		if text := strings.TrimSpace(fmt.Sprint(value)); text != "" && text != "<nil>" {
			return text
		}
	}
	for _, key := range []string{"payload", "result"} {
		if nested, ok := result.Data[key].(map[string]any); ok {
			for _, statusKey := range []string{"status", "Status"} {
				if text := strings.TrimSpace(fmt.Sprint(nested[statusKey])); text != "" && text != "<nil>" {
					return text
				}
			}
		}
	}
	return ""
}

func (h *KnowledgeHandler) IngestSpace(c *gin.Context) {
	provider, err := h.provider()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	spaceID := strings.TrimSpace(c.Param("spaceID"))
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "knowledge space_id is required"})
		return
	}
	var req knowledgeIngestSpaceRequest
	if c.Request != nil && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid knowledge ingestion request"})
			return
		}
	}
	if strings.TrimSpace(req.SourceURI) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "knowledge ingestion sourceUri is required"})
		return
	}
	if provider.Mode() == fwknowledge.ProviderModeDelegated && h.gatewayReady() {
		result, err := h.invokeKnowledgeCapability(c, knowledgeCapabilityCreateIngestionJob, "CreateIngestionJob", map[string]any{
			"method":   http.MethodPost,
			"endpoint": "/api/v1/admin/knowledge-spaces/" + url.PathEscape(spaceID) + "/ingestion-jobs",
			"body": map[string]any{
				"sourceUri":        strings.TrimSpace(req.SourceURI),
				"format":           firstNonEmpty(req.Format, "markdown"),
				"ingestionProfile": firstNonEmpty(req.IngestionProfile, "p1_general"),
				"requestedBy":      firstNonEmpty(req.RequestedBy, "plugin-knowledge-lab"),
			},
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": err.Error(), "code": fwknowledge.CodeProviderUnavailable})
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"success": true, "data": gin.H{
			"space_id":      spaceID,
			"provider":      provider.Name(),
			"provider_mode": provider.Mode(),
			"source":        "powerx_gateway_capability",
			"capability_id": knowledgeCapabilityCreateIngestionJob,
			"trace_id":      result.TraceID,
			"status":        result.Status,
			"job":           result.Data,
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"space_id":      spaceID,
		"provider":      provider.Name(),
		"provider_mode": provider.Mode(),
		"status":        "local_fixture_ready",
		"note":          "本地模式入库仍由 Playground 的本地入库模拟完成",
	}})
}

func (h *KnowledgeHandler) UploadIngestionSource(c *gin.Context) {
	if !h.gatewayReady() {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": "PowerX Gateway 客户端未配置", "code": fwknowledge.CodeProviderUnavailable})
		return
	}
	spaceID := strings.TrimSpace(c.PostForm("spaceId"))
	if spaceID == "" {
		spaceID = strings.TrimSpace(c.PostForm("space_id"))
	}
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "knowledge space_id is required"})
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "upload file is required"})
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "read upload file failed"})
		return
	}
	if len(raw) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "upload file is empty"})
		return
	}
	fileName := strings.TrimSpace(header.Filename)
	if fileName == "" {
		fileName = "knowledge-source"
	}
	contentType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = http.DetectContentType(raw)
	}
	sourceFormat := firstNonEmpty(c.PostForm("format"), knowledgeFormatFromFilename(fileName), "markdown")
	requestedBy := firstNonEmpty(c.PostForm("requestedBy"), "plugin-knowledge-lab")
	contentSHA256 := sha256Hex(raw)
	objectKey := deterministicMediaAssetUUID("content_sha256:" + contentSHA256)

	mediaClient := fwmedia.NewClient(knowledgeMediaGatewayAdapter{gateway: h.deps.CapabilityGateway}, http.DefaultClient)
	asset, err := mediaClient.CreateAsset(c.Request.Context(), fwmedia.CreateAssetInput{
		OperatorID:       requestedBy,
		Name:             fileName,
		Description:      "PowerX Plugin knowledge ingestion source",
		Driver:           "local",
		ObjectKey:        objectKey,
		SizeBytes:        int64(len(raw)),
		MimeType:         contentType,
		OwnerSubjectType: "knowledge_space",
		OwnerSubjectID:   spaceID,
		Tags:             []string{"knowledge_space", "knowledge_ingestion_source"},
		UploadChannel:    fwmedia.UploadChannelPresigned,
		ContentSHA256:    contentSHA256,
		Metadata:         map[string]string{"content_sha256": contentSHA256},
		RequestID:        strings.TrimSpace(c.GetHeader("X-Request-ID")),
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": err.Error(), "code": fwknowledge.CodeProviderUnavailable})
		return
	}
	uploadTicket, err := mediaClient.PresignAsset(c.Request.Context(), fwmedia.PresignAssetInput{
		UUID:             asset.UUID,
		OperatorID:       requestedBy,
		Action:           fwmedia.PresignActionUpload,
		Method:           http.MethodPut,
		ExpiresInSeconds: 3600,
		Headers:          map[string]string{"Content-Type": contentType},
		RequestID:        strings.TrimSpace(c.GetHeader("X-Request-ID")),
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": err.Error(), "code": fwknowledge.CodeProviderUnavailable})
		return
	}
	uploadTicket.URL = h.resolveMediaURL(uploadTicket.URL)
	if strings.TrimSpace(uploadTicket.URL) == "" {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": "PowerX Media 上传链接不是有效 URL", "code": fwknowledge.CodeProviderUnavailable})
		return
	}
	if err := mediaClient.UploadBytes(c.Request.Context(), uploadTicket, bytes.NewReader(raw), contentType); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": err.Error(), "code": fwknowledge.CodeProviderUnavailable})
		return
	}
	status := fwmedia.BusinessStatusUnderReview
	_, _ = mediaClient.UpdateAsset(c.Request.Context(), fwmedia.UpdateAssetInput{
		UUID:           asset.UUID,
		OperatorID:     requestedBy,
		BusinessStatus: &status,
		Tags:           []string{"knowledge_space", "knowledge_ingestion_source"},
		RequestID:      strings.TrimSpace(c.GetHeader("X-Request-ID")),
	})
	downloadTicket, err := mediaClient.PresignAsset(c.Request.Context(), fwmedia.PresignAssetInput{
		UUID:             asset.UUID,
		OperatorID:       requestedBy,
		Action:           fwmedia.PresignActionDownload,
		Method:           http.MethodGet,
		ExpiresInSeconds: 24 * 3600,
		RequestID:        strings.TrimSpace(c.GetHeader("X-Request-ID")),
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": err.Error(), "code": fwknowledge.CodeProviderUnavailable})
		return
	}
	downloadTicket.URL = h.resolveMediaURL(downloadTicket.URL)
	if strings.TrimSpace(downloadTicket.URL) == "" {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": "PowerX Media 下载链接不是有效 URL", "code": fwknowledge.CodeProviderUnavailable})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"mediaUuid":   asset.UUID,
		"sourceUri":   downloadTicket.URL,
		"fileName":    fileName,
		"format":      sourceFormat,
		"contentType": contentType,
		"sizeBytes":   len(raw),
	}})
}

func (h *KnowledgeHandler) resolveMediaURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if parsed, err := url.Parse(value); err == nil && parsed.IsAbs() {
		return value
	}
	base := ""
	if h != nil && h.deps != nil && h.deps.Config != nil && h.deps.Config.Gateway != nil {
		base = strings.TrimSpace(h.deps.Config.Gateway.BaseURL)
	}
	if base == "" {
		return ""
	}
	parsedBase, err := url.Parse(base)
	if err != nil || parsedBase.Scheme == "" || parsedBase.Host == "" {
		return ""
	}
	parsedBase.Path = ""
	parsedBase.RawQuery = ""
	parsedBase.Fragment = ""
	if strings.HasPrefix(value, "/") {
		parsedBase.Path = value
	} else {
		parsedBase.Path = "/" + value
	}
	return parsedBase.String()
}

func knowledgeFormatFromFilename(name string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	switch ext {
	case "md", "markdown":
		return "markdown"
	case "txt":
		return "text"
	case "pdf":
		return "pdf"
	case "html", "htm":
		return "html"
	case "csv":
		return "csv"
	default:
		return ""
	}
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func deterministicMediaAssetUUID(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	raw := make([]byte, 16)
	copy(raw, sum[:16])
	raw[6] = (raw[6] & 0x0f) | 0x50
	raw[8] = (raw[8] & 0x3f) | 0x80
	return uuid.UUID(raw).String()
}

func (h *KnowledgeHandler) Policy(c *gin.Context) {
	provider, err := h.provider()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	spaceID := strings.TrimSpace(c.Param("spaceID"))
	if spaceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "knowledge space_id is required"})
		return
	}
	if provider.Mode() == fwknowledge.ProviderModeDelegated && h.gatewayReady() {
		result, err := h.invokeKnowledgeCapability(c, knowledgeCapabilityListFusionStrategies, "ListFusionStrategies", map[string]any{
			"method":   http.MethodGet,
			"endpoint": "/api/v1/admin/knowledge-spaces/" + spaceID + "/fusion-strategies",
			"query":    map[string]string{"limit": "20"},
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": err.Error(), "code": fwknowledge.CodeProviderUnavailable})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
			"space_id":        spaceID,
			"provider":        provider.Name(),
			"provider_mode":   provider.Mode(),
			"source":          "powerx_gateway_capability",
			"capability_id":   knowledgeCapabilityListFusionStrategies,
			"trace_id":        result.TraceID,
			"status":          result.Status,
			"fusion_strategy": result.Data,
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"space_id":        spaceID,
		"provider":        provider.Name(),
		"provider_mode":   provider.Mode(),
		"tenant_required": provider.Mode() != fwknowledge.ProviderModeMock,
		"citation":        "required",
		"redaction":       "enabled",
		"fallback":        "disabled",
	}})
}

func (h *KnowledgeHandler) Search(c *gin.Context) {
	var req knowledgeSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid knowledge search request"})
		return
	}
	provider, err := h.provider()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	tenantUUID, tenantMismatch := resolveKnowledgeTenantUUID(c, req.TenantUUID)
	if tenantMismatch {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "tenant_uuid mismatch", "code": fwknowledge.CodeForbidden})
		return
	}
	req.TenantUUID = tenantUUID
	if local, ok := provider.(*fwknowledge.LocalProvider); ok && req.Fixture != nil {
		if _, err := local.UpsertDocument(c.Request.Context(), req.Fixture.document(req)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
	}
	query := fwknowledge.KnowledgeQuery{
		Query:      req.Query,
		SpaceIDs:   singleSpace(req.SpaceID),
		TenantUUID: req.TenantUUID,
		PluginID:   req.PluginID,
		AgentUUID:  req.AgentUUID,
		SkillID:    req.SkillID,
		CallerType: fwknowledge.CallerTypeAgent,
		Visibility: firstNonEmpty(req.Visibility, fwknowledge.VisibilityTenant),
		Limit:      req.Limit,
		Tags:       req.Tags,
		Filters:    req.Filters,
		TraceID:    c.GetString("request_id"),
	}
	result, err := provider.Search(c.Request.Context(), query)
	if err != nil {
		status := fwknowledge.HTTPStatusForCode(fwknowledge.CodeOf(err))
		c.JSON(status, gin.H{"success": false, "error": err.Error(), "code": fwknowledge.CodeOf(err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *KnowledgeHandler) provider() (fwknowledge.KnowledgeProvider, error) {
	if h == nil {
		return knowledgeSvc.NewProviderFactory(nil, nil, nil).Build()
	}
	h.providerOnce.Do(func() {
		var appCfg *config.Config
		if h.deps != nil {
			appCfg = h.deps.Config
		}
		h.knowledgeProvider, h.providerErr = knowledgeSvc.NewProviderFactory(appCfg, nil, nil).Build()
		if h.providerErr == nil && h.knowledgeProvider.Mode() == fwknowledge.ProviderModeDelegated {
			h.knowledgeProvider, h.providerErr = knowledgeSvc.NewProviderFactory(appCfg, h.delegatedKnowledgeClient(), nil).Build()
		}
	})
	return h.knowledgeProvider, h.providerErr
}

func (h *KnowledgeHandler) delegatedKnowledgeClient() fwknowledge.DelegatedClient {
	if !h.gatewayReady() {
		return nil
	}
	return knowledgeGatewayDelegatedClient{gateway: h.deps.CapabilityGateway}
}

func (h *KnowledgeHandler) gatewayReady() bool {
	return h != nil && h.deps != nil && h.deps.CapabilityGateway != nil && h.deps.CapabilityGateway.Enabled()
}

func (h *KnowledgeHandler) invokeKnowledgeCapability(c *gin.Context, capabilityID string, action string, payload map[string]any) (*knowledgeInvokeResult, error) {
	if !h.gatewayReady() {
		return nil, fmt.Errorf("PowerX Gateway 客户端未配置")
	}
	result, err := h.deps.CapabilityGateway.Invoke(c.Request.Context(), knowledgeInvokeParams(capabilityID, action, payload, c))
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("PowerX Gateway 返回了空的知识响应")
	}
	return &knowledgeInvokeResult{
		TraceID: result.TraceID,
		Status:  result.Status,
		Data:    result.Data,
		Raw:     result.Raw,
	}, nil
}

type knowledgeInvokeResult struct {
	TraceID string
	Status  string
	Data    map[string]any
	Raw     json.RawMessage
}

func knowledgeInvokeParams(capabilityID string, action string, payload map[string]any, c *gin.Context) capgateway.InvokeParams {
	return capgateway.InvokeParams{
		CapabilityID:      capabilityID,
		Action:            action,
		PreferredProtocol: knowledgePreferredProtocol(payload),
		Payload:           payload,
		RequestID:         strings.TrimSpace(c.GetHeader("X-Request-ID")),
		AuthRequired:      true,
		TenantScoped:      false,
	}
}

func knowledgePreferredProtocol(payload map[string]any) string {
	return "rest"
}

type knowledgeGatewayDelegatedClient struct {
	gateway interface {
		Invoke(ctx context.Context, params capgateway.InvokeParams) (*capgateway.InvokeResult, error)
		ListKnowledgeSpaces(ctx context.Context, opts capgateway.KnowledgeSpaceListOptions) ([]capgateway.KnowledgeSpaceRuntimeRecord, error)
	}
}

func (c knowledgeGatewayDelegatedClient) ListKnowledgeSpaces(ctx context.Context, input fwknowledge.ListSpacesInput) ([]fwknowledge.KnowledgeSpace, error) {
	if c.gateway == nil {
		return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "PowerX Gateway 客户端未配置")
	}
	input = input.Normalized()
	records, err := c.gateway.ListKnowledgeSpaces(ctx, capgateway.KnowledgeSpaceListOptions{
		TenantUUID: input.TenantUUID,
		Status:     input.Status,
		Keyword:    stringFromAny(input.Filters["keyword"]),
		Page:       1,
		PageSize:   input.Limit,
	})
	if err != nil {
		return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, err.Error())
	}
	spaces := make([]fwknowledge.KnowledgeSpace, 0, len(records))
	for _, record := range records {
		space := fwknowledge.KnowledgeSpace{
			SpaceID:        strings.TrimSpace(record.UUID),
			SpaceName:      firstNonEmpty(record.SpaceName, record.UUID),
			TenantUUID:     input.TenantUUID,
			DepartmentCode: record.DepartmentCode,
			Visibility:     fwknowledge.VisibilityTenant,
			Status:         firstNonEmpty(record.Status, "active"),
			RAGProfileKey:  record.RAGProfileKey,
			Metadata: map[string]any{
				"source":     "powerx_plugin_runtime",
				"created_at": record.CreatedAt,
				"updated_at": record.UpdatedAt,
			},
		}
		if space.SpaceID != "" {
			spaces = append(spaces, space)
		}
	}
	return spaces, nil
}

func (c knowledgeGatewayDelegatedClient) GetKnowledgeCatalog(context.Context) (*fwknowledge.KnowledgeCatalog, error) {
	catalog := fwknowledge.DefaultKnowledgeCatalog("powerx_delegated_fallback")
	if catalog.Metadata == nil {
		catalog.Metadata = map[string]any{}
	}
	catalog.Metadata["delegated_note"] = "PowerX catalog endpoint is not exposed yet; using framework-compatible fallback"
	return catalog, nil
}

func (c knowledgeGatewayDelegatedClient) SearchKnowledge(ctx context.Context, query fwknowledge.KnowledgeQuery) (*fwknowledge.KnowledgeSearchResult, error) {
	if c.gateway == nil {
		return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "PowerX Gateway 客户端未配置")
	}
	query = query.Normalized()
	result, err := c.gateway.Invoke(ctx, capgateway.InvokeParams{
		CapabilityID:      knowledgeCapabilityPlanRetrieval,
		Action:            "PlanRetrieval",
		PreferredProtocol: "rest",
		Payload: map[string]any{
			"method":   http.MethodPost,
			"endpoint": "/api/v1/admin/knowledge-spaces/" + firstNonEmpty(query.SpaceIDs...) + "/playground/retrieval",
			"body": map[string]any{
				"query":   query.Query,
				"topK":    query.Limit,
				"filters": query.Filters,
			},
		},
		RequestID:    query.TraceID,
		TenantUUID:   query.TenantUUID,
		AuthRequired: true,
		TenantScoped: false,
	})
	if err != nil {
		return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, err.Error())
	}
	return decodeKnowledgeSearchResult(result, query), nil
}

func (c knowledgeGatewayDelegatedClient) UpsertKnowledgeDocument(ctx context.Context, document fwknowledge.KnowledgeDocument) (*fwknowledge.KnowledgeIndexJob, error) {
	sourceURI := strings.TrimSpace(document.URI)
	if sourceURI == "" {
		return nil, fwknowledge.NewError(fwknowledge.CodeInvalidDocument, "knowledge ingestion sourceUri is required; upload the source through PowerX Media first")
	}
	return c.invokeIndexJob(ctx, knowledgeCapabilityCreateIngestionJob, "CreateIngestionJob", document.TenantUUID, map[string]any{
		"method":   http.MethodPost,
		"endpoint": "/api/v1/admin/knowledge-spaces/" + url.PathEscape(strings.TrimSpace(document.SpaceID)) + "/ingestion-jobs",
		"body": map[string]any{
			"sourceUri":   sourceURI,
			"format":      firstNonEmpty(document.ContentType, "markdown"),
			"requestedBy": "plugin-knowledge-lab",
		},
	})
}

func (c knowledgeGatewayDelegatedClient) DeleteKnowledgeDocument(context.Context, fwknowledge.DeleteDocumentInput) (*fwknowledge.KnowledgeIndexJob, error) {
	return nil, fwknowledge.NewError(fwknowledge.CodeUnsupportedCapability, "PowerX delegated knowledge delete is not exposed by the current gateway contract")
}

func (c knowledgeGatewayDelegatedClient) ReindexKnowledgeDocument(context.Context, fwknowledge.ReindexInput) (*fwknowledge.KnowledgeIndexJob, error) {
	return nil, fwknowledge.NewError(fwknowledge.CodeUnsupportedCapability, "PowerX delegated knowledge reindex is not exposed by the current gateway contract")
}

func (c knowledgeGatewayDelegatedClient) invokeIndexJob(ctx context.Context, capabilityID, action, tenantUUID string, payload map[string]any) (*fwknowledge.KnowledgeIndexJob, error) {
	if c.gateway == nil {
		return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "PowerX Gateway 客户端未配置")
	}
	result, err := c.gateway.Invoke(ctx, capgateway.InvokeParams{
		CapabilityID:      capabilityID,
		Action:            action,
		PreferredProtocol: knowledgePreferredProtocol(payload),
		Payload:           payload,
		TenantUUID:        tenantUUID,
		AuthRequired:      true,
		TenantScoped:      false,
	})
	if err != nil {
		return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, err.Error())
	}
	job := &fwknowledge.KnowledgeIndexJob{Operation: fwknowledge.IndexOperationUpsert, Status: fwknowledge.IndexStatusQueued}
	_ = mapToStruct(firstMap(resultData(result), "job", "index_job", "data"), job)
	if job.Status == "" {
		job.Status = firstNonEmpty(result.Status, fwknowledge.IndexStatusQueued)
	}
	return job, nil
}

func decodeKnowledgeSpaces(result *capgateway.InvokeResult) ([]fwknowledge.KnowledgeSpace, error) {
	data := resultData(result)
	rawSpaces := firstSlice(data, "spaces", "items", "knowledge_spaces")
	if len(rawSpaces) == 0 {
		rawSpaces = firstSlice(firstMap(data, "data", "result", "response"), "spaces", "items", "knowledge_spaces")
	}
	spaces := make([]fwknowledge.KnowledgeSpace, 0, len(rawSpaces))
	for _, raw := range rawSpaces {
		var space fwknowledge.KnowledgeSpace
		if err := mapToStruct(raw, &space); err != nil {
			return nil, fwknowledge.NewError(fwknowledge.CodeProviderUnavailable, "PowerX knowledge space response is invalid")
		}
		normalizeDelegatedSpace(&space, raw)
		if strings.TrimSpace(space.SpaceID) != "" {
			spaces = append(spaces, space)
		}
	}
	return spaces, nil
}

func decodeKnowledgeSearchResult(result *capgateway.InvokeResult, query fwknowledge.KnowledgeQuery) *fwknowledge.KnowledgeSearchResult {
	data := resultData(result)
	search := &fwknowledge.KnowledgeSearchResult{
		Provider:  "powerx_delegated",
		SpaceID:   firstNonEmpty(query.SpaceIDs...),
		Chunks:    []fwknowledge.KnowledgeChunk{},
		Citations: []fwknowledge.KnowledgeCitation{},
		Total:     0,
		TraceID:   firstNonEmpty(resultTraceID(result), query.TraceID),
		Diagnostics: map[string]any{
			"source":        "powerx_gateway_capability",
			"capability_id": knowledgeCapabilityPlanRetrieval,
			"status":        resultStatus(result),
			"qa_plan":       data,
		},
	}
	_ = mapToStruct(firstMap(data, "search_result", "result", "data"), search)
	if search.Chunks == nil {
		search.Chunks = []fwknowledge.KnowledgeChunk{}
	}
	if search.Citations == nil {
		search.Citations = []fwknowledge.KnowledgeCitation{}
	}
	if search.Provider == "" {
		search.Provider = "powerx_delegated"
	}
	if search.TraceID == "" {
		search.TraceID = firstNonEmpty(resultTraceID(result), query.TraceID)
	}
	if search.Diagnostics == nil {
		search.Diagnostics = map[string]any{}
	}
	search.Diagnostics["source"] = "powerx_gateway_capability"
	search.Diagnostics["capability_id"] = knowledgeCapabilityPlanRetrieval
	search.Diagnostics["status"] = resultStatus(result)
	search.Diagnostics["qa_plan"] = data
	return search
}

func normalizeDelegatedSpace(space *fwknowledge.KnowledgeSpace, raw any) {
	if space == nil {
		return
	}
	values, _ := raw.(map[string]any)
	space.SpaceID = firstNonEmpty(space.SpaceID, stringFromAny(values["spaceId"]), stringFromAny(values["id"]), stringFromAny(values["uuid"]))
	space.SpaceName = firstNonEmpty(space.SpaceName, stringFromAny(values["spaceName"]), stringFromAny(values["name"]), space.SpaceID)
	space.TenantUUID = firstNonEmpty(space.TenantUUID, stringFromAny(values["tenantUUID"]), stringFromAny(values["tenantUuid"]))
	space.DepartmentCode = firstNonEmpty(space.DepartmentCode, stringFromAny(values["departmentCode"]), stringFromAny(values["department"]))
	space.Visibility = firstNonEmpty(space.Visibility, stringFromAny(values["type"]), stringFromAny(values["visibility"]), fwknowledge.VisibilityTenant)
	space.Status = firstNonEmpty(space.Status, stringFromAny(values["status"]), "active")
	space.PolicyTemplateVersionID = firstNonEmpty(space.PolicyTemplateVersionID, stringFromAny(values["policyTemplateVersionId"]))
	space.IngestionProfileKey = firstNonEmpty(space.IngestionProfileKey, stringFromAny(values["ingestionProfileKey"]))
	space.IndexProfileKey = firstNonEmpty(space.IndexProfileKey, stringFromAny(values["indexProfileKey"]))
	space.RAGProfileKey = firstNonEmpty(space.RAGProfileKey, stringFromAny(values["ragProfileKey"]))
	if space.Quotas == nil {
		space.Quotas = map[string]any{}
	}
	if space.Metadata == nil {
		space.Metadata = map[string]any{}
	}
}

func resultData(result *capgateway.InvokeResult) map[string]any {
	if result == nil || result.Data == nil {
		return map[string]any{}
	}
	return result.Data
}

func resultTraceID(result *capgateway.InvokeResult) string {
	if result == nil {
		return ""
	}
	return result.TraceID
}

func resultStatus(result *capgateway.InvokeResult) string {
	if result == nil {
		return ""
	}
	return result.Status
}

func firstMap(values map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if child, ok := values[key].(map[string]any); ok {
			return child
		}
	}
	return map[string]any{}
}

func firstSlice(values map[string]any, keys ...string) []any {
	for _, key := range keys {
		switch typed := values[key].(type) {
		case []any:
			return typed
		case []map[string]any:
			out := make([]any, 0, len(typed))
			for _, item := range typed {
				out = append(out, item)
			}
			return out
		}
	}
	return nil
}

func mapToStruct(raw any, dest any) error {
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func resolveKnowledgeTenantUUID(c *gin.Context, requested string) (string, bool) {
	return admincommon.ResolveTenantUUIDStrict(c, firstNonEmpty(requested, knowledgeRequestedTenantUUID(c)))
}

func knowledgeRequestedTenantUUID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	for _, value := range []string{
		c.Query("tenant_uuid"),
		c.GetHeader("tenant_uuid"),
		c.GetHeader("X-Tenant-UUID"),
	} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if value, ok := c.Get("tenant_uuid"); ok {
		if str, ok := value.(string); ok && strings.TrimSpace(str) != "" {
			return strings.TrimSpace(str)
		}
	}
	return ""
}

type fixturePayload struct {
	DocumentID string   `json:"document_id"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
}

func (f fixturePayload) document(req knowledgeSearchRequest) fwknowledge.KnowledgeDocument {
	return fwknowledge.KnowledgeDocument{
		SpaceID:     firstNonEmpty(req.SpaceID, "debug"),
		DocumentID:  firstNonEmpty(f.DocumentID, "debug-fixture"),
		Title:       firstNonEmpty(f.Title, "调试文档"),
		Content:     firstNonEmpty(f.Content, req.Query),
		ContentType: "text/markdown",
		Tags:        append([]string(nil), f.Tags...),
		Visibility:  firstNonEmpty(req.Visibility, fwknowledge.VisibilityTenant),
		TenantUUID:  strings.TrimSpace(req.TenantUUID),
	}
}

func singleSpace(spaceID string) []string {
	if strings.TrimSpace(spaceID) == "" {
		return nil
	}
	return []string{strings.TrimSpace(spaceID)}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func positiveOrDefault(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
