package iam

import (
	"strconv"
	"strings"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	repo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/repository/iam"
	federatedsvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/federated"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type ChannelWeComHandler struct {
	configSvc *federatedsvc.WeComConfigService
	syncSvc   *federatedsvc.WeComSyncTaskService
}

func NewChannelWeComHandler(configSvc *federatedsvc.WeComConfigService) *ChannelWeComHandler {
	return &ChannelWeComHandler{configSvc: configSvc}
}

func NewChannelWeComHandlerWithDeps(deps *app.Deps) *ChannelWeComHandler {
	if deps == nil || deps.DB == nil {
		return &ChannelWeComHandler{}
	}
	configSvc := federatedsvc.NewWeComConfigService(deps.DB)
	syncRepo := repo.NewChannelSyncTaskRepository(deps.DB)
	publisher := fwwsbus.NewAdapter(
		fwwsbus.NewLocalPublisher(deps.WSBusHub, nil),
		"",
		nil,
		"wecom.sync.progress",
	)
	if adapter, ok := publisher.(*fwwsbus.Adapter); ok {
		adapter.EnableHubBridge(deps.WSBusHub)
	}
	return &ChannelWeComHandler{
		configSvc: configSvc,
		syncSvc:   federatedsvc.NewWeComSyncTaskService(syncRepo, configSvc, publisher, deps.DB),
	}
}

type wecomConfigRequest struct {
	Status       string `json:"status"`
	RotationDays int    `json:"rotation_days"`
	CallbackHost string `json:"callback_host"`
	CorpID       string `json:"corp_id" binding:"required"`
	AgentID      int    `json:"agent_id" binding:"required"`
	Secret       string `json:"secret" binding:"required"`
	Token        string `json:"token"`
	AESKey       string `json:"aes_key"`
	HttpDebug    bool   `json:"http_debug"`
}

type syncTaskRequest struct {
	Action string `json:"action" binding:"required"`
}

func (h *ChannelWeComHandler) GetConfig(c *gin.Context) {
	if h.configSvc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom config service not available", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	cfg, err := h.configSvc.GetByTenant(c.Request.Context(), tenantUUID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			contracts.ResponseNotFound(c, "wecom config not found")
			return
		}
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	contracts.ResponseSuccess(c, cfg)
}

func (h *ChannelWeComHandler) SaveConfig(c *gin.Context) {
	if h.configSvc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom config service not available", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req wecomConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	cfg, err := h.configSvc.UpsertTenantConfig(c.Request.Context(), federatedsvc.WeComConfig{
		TenantUUID:   tenantUUID,
		Status:       req.Status,
		RotationDays: req.RotationDays,
		CallbackHost: req.CallbackHost,
		CorpID:       req.CorpID,
		AgentID:      req.AgentID,
		Secret:       req.Secret,
		Token:        req.Token,
		AESKey:       req.AESKey,
		HttpDebug:    req.HttpDebug,
	})
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	contracts.ResponseSuccess(c, cfg)
}

func (h *ChannelWeComHandler) TriggerSyncTask(c *gin.Context) {
	if h.syncSvc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom sync task service not available", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req syncTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	task, err := h.syncSvc.Trigger(c.Request.Context(), tenantUUID, req.Action)
	if err != nil {
		if isMissingSyncTaskTable(err) {
			contracts.ResponseBadRequest(c, "同步任务表未初始化，请先执行数据库迁移（cd skeleton/backend/go-gin && go run ./cmd/database migrate --include-iam）")
			return
		}
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	contracts.ResponseSuccess(c, task)
}

func (h *ChannelWeComHandler) ListSyncTasks(c *gin.Context) {
	if h.syncSvc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom sync task service not available", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	limit := 20
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	tasks, err := h.syncSvc.List(c.Request.Context(), tenantUUID, limit)
	if err != nil {
		if isMissingSyncTaskTable(err) {
			contracts.ResponseSuccess(c, gin.H{"items": []any{}, "warning": "同步任务表未初始化，请先执行数据库迁移"})
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": tasks})
}

func (h *ChannelWeComHandler) ClearSyncTasks(c *gin.Context) {
	if h.syncSvc == nil {
		contracts.ResponseServiceUnavailable(c, "wecom sync task service not available", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	affected, err := h.syncSvc.Clear(c.Request.Context(), tenantUUID)
	if err != nil {
		if isMissingSyncTaskTable(err) {
			contracts.ResponseSuccess(c, gin.H{"deleted": 0, "warning": "同步任务表未初始化"})
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"deleted": affected})
}

func isMissingSyncTaskTable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "iam_channel_sync_tasks") &&
		(strings.Contains(msg, "does not exist") || strings.Contains(msg, "no such table"))
}
