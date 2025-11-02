package manifestx

import "github.com/powerx-plugin/framework/backend/go/manifest"

// Plugin 返回 skeleton 示例的 Manifest 定义。
func Plugin() manifest.Plugin {
	return manifest.Plugin{
		ID:   "com.powerx.sample",
		Name: "PowerX Sample Plugin",
		Menus: []manifest.Menu{
			{
				Path:  "/_p/com.powerx.sample/admin",
				Title: "PowerX Skeleton",
				Children: []manifest.Menu{
					{Path: "/_p/com.powerx.sample/admin/intro", Title: "概览"},
					{Path: "/_p/com.powerx.sample/admin/templates", Title: "模板管理"},
				},
			},
		},
		Permissions: []string{
			"com.powerx.sample.templates.read",
			"com.powerx.sample.templates.write",
		},
	}
}
