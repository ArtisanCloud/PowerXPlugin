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
				Title: "Sample Dashboard",
			},
		},
		Permissions: []string{
			"com.powerx.sample.admin.view",
		},
	}
}
