package manifestx

import (
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

// Plugin returns the manifest definition consumed by the framework/router layer.
func Plugin() manifest.Plugin {
	return manifest.Plugin{
		ID:      app.PluginID,
		Name:    app.PluginName,
		Version: app.PluginVersion,
		Menus: []manifest.Menu{
			{
				Path:  "/_p/com.powerx.plugins.base/admin/templates/intro",
				Title: "模板介绍",
			},
			{
				Path:  "/_p/com.powerx.plugins.base/admin/templates/crud",
				Title: "模板管理",
			},
			{
				Path:  "/_p/com.powerx.plugins.base/admin/iam/overview",
				Title: "组织与权限",
			},
			{
				Path:  "/_p/com.powerx.plugins.base/admin/iam/members",
				Title: "成员管理",
			},
			{
				Path:  "/_p/com.powerx.plugins.base/admin/iam/roles",
				Title: "角色与权限",
			},
		},
	}
}
