package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/shared/app"
	adminconsole "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/transport/http/admin/console"
	adminintegration "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/transport/http/admin/integration"
	adminmarketplace "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/transport/http/admin/marketplace"
	adminoperations "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/transport/http/admin/operations"
	adminruntime "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/transport/http/admin/runtime_ops"
	adminsecurity "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/transport/http/admin/security"
)

// Register 注册 Admin 路由
func RegisterAPIRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	adminHandler := NewAdminHandler(deps)
	admin := rg.Group("/admin")
	{
		// 基础管理功能
		admin.GET("/manifest", adminHandler.GetManifest) // 获取插件清单
		admin.GET("/rbac", adminHandler.GetRBACInfo)     // 获取权限信息

		runtimeOps := admin.Group("/runtime")
		adminruntime.RegisterRoutes(runtimeOps, deps)

		adminmarketplace.RegisterRoutes(admin, deps)
		adminoperations.RegisterRoutes(admin, deps)
		adminconsole.RegisterRoutes(admin, deps)
		adminintegration.RegisterRoutes(admin, deps)
		adminsecurity.RegisterRoutes(admin, deps)
	}
}
