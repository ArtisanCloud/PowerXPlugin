package iam

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	federatedrepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/repository/iam"
	srviam "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	federatedsvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/federated"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(admin *gin.RouterGroup, deps *app.Deps) {
	if admin == nil || deps == nil {
		return
	}
	group := admin.Group("/iam")
	group.GET("/mode", modeHandler(deps))

	var audit *srviam.AuditService
	var stsSvc *srviam.STSService
	var roleSvc *srviam.RoleService
	var tenantSvc *srviam.TenantService
	var departmentSvc *srviam.DepartmentService
	var userSvc *srviam.UserService
	if deps.DB != nil {
		audit = srviam.NewAuditService(deps.DB)
		stsSvc = srviam.NewSTSService(deps.Config, audit, app.PluginID, "")
		roleSvc = srviam.NewRoleService(deps.DB, audit, app.PluginID)
		tenantSvc = srviam.NewTenantService(deps.DB, audit)
		departmentSvc = srviam.NewDepartmentService(deps.DB, audit)
		userSvc = srviam.NewUserService(deps.DB, audit)
	}

	tenantHandler := NewTenantHandler(
		tenantSvc,
		deps.IAMDirectoryService,
		deps.IAMAuthzService,
		deps.IAMAdapterMode,
	)
	departmentHandler := NewDepartmentHandler(departmentSvc, deps.IAMAdapterMode, deps.IAMDirectoryService)
	memberHandler := NewMemberHandler(userSvc, deps.IAMAdapterMode, deps.IAMDirectoryService)
	roleHandler := NewRoleHandler(roleSvc, deps.IAMAdapterMode, deps.IAMDirectoryService)
	rolePermissionsHandler := NewRolePermissionsHandler(roleSvc, deps.IAMAdapterMode)
	roleMembersHandler := NewRoleMembersHandler(roleSvc, deps.IAMAdapterMode)
	permissionHandler := NewPermissionHandler(roleSvc, deps.IAMAdapterMode, deps.IAMDirectoryService)
	auditHandler := NewAuditHandler(audit)
	stsHandler := NewSTSHandler(deps.IAMAdapterMode, stsSvc)
	sessionSvc := federatedsvc.NewSessionService()
	var bindingSvc *federatedsvc.BindingService
	if deps.DB != nil {
		fedRepo := federatedrepo.NewFederatedBindingRepository(deps.DB)
		bindingSvc = federatedsvc.NewBindingService(fedRepo, deps.DB, sessionSvc)
	}
	jitPolicySvc := federatedsvc.NewJITPolicyService()
	mappingSvc := federatedsvc.NewMappingService()
	federatedBindingHandler := NewFederatedBindingHandler(bindingSvc, jitPolicySvc, mappingSvc)
	wecomChannelHandler := NewChannelWeComHandlerWithDeps(deps)
	dingtalkChannelHandler := NewChannelDingTalkHandlerWithDeps(deps)
	larkChannelHandler := NewChannelLarkHandlerWithDeps(deps)

	group.GET("/tenants", tenantHandler.List)
	group.POST("/tenants", tenantHandler.Create)
	group.PATCH("/tenants/:id", tenantHandler.Update)

	group.GET("/departments", departmentHandler.List)
	group.GET("/departments/tree", departmentHandler.Tree)
	group.POST("/departments", departmentHandler.Create)
	group.PATCH("/departments/:department_uuid", departmentHandler.Update)
	group.DELETE("/departments/:department_uuid", departmentHandler.Delete)

	group.GET("/members", memberHandler.List)
	group.POST("/members", memberHandler.Create)
	group.PATCH("/members/:member_uuid", memberHandler.Update)
	group.POST("/members/import", memberHandler.BulkImport)

	group.GET("/roles", roleHandler.List)
	group.POST("/roles", roleHandler.Create)
	group.GET("/roles/:role_uuid", roleHandler.Get)
	group.PATCH("/roles/:role_uuid", roleHandler.Update)
	group.DELETE("/roles/:role_uuid", roleHandler.Delete)
	group.PUT("/roles/:role_uuid/permissions", rolePermissionsHandler.Replace)
	group.POST("/roles/:role_uuid/members", roleMembersHandler.Add)
	group.DELETE("/roles/:role_uuid/members", roleMembersHandler.Remove)

	group.GET("/permissions", permissionHandler.List)
	if deps.DB != nil {
		group.GET("/audit/logs", auditHandler.List)
		group.POST("/auth/local/sts", stsHandler.Mint)
		group.GET("/federated/bindings", federatedBindingHandler.List)
		group.POST("/federated/bindings", federatedBindingHandler.Create)
		group.DELETE("/federated/bindings", federatedBindingHandler.Delete)
		group.PUT("/federated/jit-policy", federatedBindingHandler.UpdateJITPolicy)
		group.PUT("/federated/mapping-policy", federatedBindingHandler.UpdateMappingPolicy)
	}
	group.GET("/channels/wecom/config", wecomChannelHandler.GetConfig)
	group.PUT("/channels/wecom/config", wecomChannelHandler.SaveConfig)
	group.GET("/channels/wecom/sync-tasks", wecomChannelHandler.ListSyncTasks)
	group.POST("/channels/wecom/sync-tasks", wecomChannelHandler.TriggerSyncTask)
	group.DELETE("/channels/wecom/sync-tasks", wecomChannelHandler.ClearSyncTasks)
	group.GET("/channels/dingtalk/config", dingtalkChannelHandler.GetConfig)
	group.PUT("/channels/dingtalk/config", dingtalkChannelHandler.SaveConfig)
	group.GET("/channels/dingtalk/sync-tasks", dingtalkChannelHandler.ListSyncTasks)
	group.POST("/channels/dingtalk/sync-tasks", dingtalkChannelHandler.TriggerSyncTask)
	group.DELETE("/channels/dingtalk/sync-tasks", dingtalkChannelHandler.ClearSyncTasks)
	group.GET("/channels/lark/config", larkChannelHandler.GetConfig)
	group.PUT("/channels/lark/config", larkChannelHandler.SaveConfig)
	group.GET("/channels/lark/sync-tasks", larkChannelHandler.ListSyncTasks)
	group.POST("/channels/lark/sync-tasks", larkChannelHandler.TriggerSyncTask)
	group.DELETE("/channels/lark/sync-tasks", larkChannelHandler.ClearSyncTasks)

}

func modeHandler(deps *app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		mode := strings.TrimSpace(deps.IAMAdapterMode.String())
		if mode == "" {
			mode = "local"
		}
		registryBound := deps.IAMRegistry != nil && deps.IAMRegistry.IsBound()
		contracts.ResponseSuccess(c, gin.H{
			"mode":                mode,
			"source":              strings.TrimSpace(deps.IAMAdapterModeSource),
			"provider":            mode,
			"delegated_available": deps.IAMDirectoryService != nil,
			"local_available":     deps.IAMDirectory != nil || deps.DB != nil,
			"read_only":           mode == "delegated",
			"registry_bound":      registryBound,
			"directory_available": deps.IAMDirectoryService != nil,
			"authz_available":     deps.IAMAuthzService != nil,
			"context_available":   deps.IAMContextService != nil,
		})
	}
}
