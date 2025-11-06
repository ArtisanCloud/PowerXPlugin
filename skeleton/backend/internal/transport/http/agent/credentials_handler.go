package agent

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/contracts"
	repo "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/entity/repository/plugin"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/logger"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/services/agent"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/shared/app"
)

// 凭证投递请求
type UpsertCredentialsRequest struct {
	PluginID     string `json:"plugin_id" binding:"required"`
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	// 可选：rotate/version/issued_at 等字段，后续扩展
}

type UpsertCredentialsResponse struct {
	PluginID string `json:"plugin_id"`
}

type CredentialHandler struct{ deps *app.Deps }

// Upsert 接收宿主发送的明文凭证，落库（加密）
func (h *CredentialHandler) Upsert(c *gin.Context) {
	var req UpsertCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantStr := c.Param("tenantId")
	tenantID, err := strconv.ParseInt(tenantStr, 10, 64)
	if err != nil || tenantID <= 0 {
		contracts.ResponseBadRequest(c, "invalid tenant_id")
		return
	}

	// TODO: 鉴权（生产环境需要 JWT/HMAC/mTLS）；开发模式可放宽，由上层中间件控制

	svc := agent.NewCredentialService(h.deps.Config, repo.NewCredentialsRepository(h.deps.DB))
	if err := svc.SavePlainCredentials(c.Request.Context(), tenantID, req.PluginID, req.ClientID, req.ClientSecret); err != nil {
		logger.WithError(err).Warn("save credentials failed")
		contracts.ResponseInternalError(c, err)
		return
	}
	// 若当前 deps 中的租户就是该租户，立即使内存 token 失效，触发下次调用刷新
	if h.deps != nil && h.deps.PowerXClient != nil && h.deps.Config != nil && h.deps.Config.GRPCUpstream != nil {
		if h.deps.Config.GRPCUpstream.TenantID == tenantID {
			h.deps.PowerXClient.InvalidateSTS()
		}
	}
	contracts.ResponseSuccess(c, &UpsertCredentialsResponse{PluginID: req.PluginID})
}
