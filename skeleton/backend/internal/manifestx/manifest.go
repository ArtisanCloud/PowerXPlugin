package manifestx

import "github.com/powerx-plugin/framework/backend/go/manifest"

// Plugin returns the manifest definition consumed by the framework/router layer.
func Plugin() manifest.Plugin {
	return manifest.Plugin{
		ID:      "com.powerx.plugin.base",
		Name:    "PowerX Base Plugin",
		Version: "0.1.0",
		Menus: []manifest.Menu{
			{
				Path:  "/_p/com.powerx.plugin.base/admin/templates/intro",
				Title: "模板介绍",
			},
			{
				Path:  "/_p/com.powerx.plugin.base/admin/templates/crud",
				Title: "模板管理",
			},
		},
	}
}
