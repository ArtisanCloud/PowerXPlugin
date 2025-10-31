package manifestx

import "github.com/powerx-plugin/framework/backend/go/manifest"

// Plugin 返回插件的默认 Manifest 定义。
func Plugin() manifest.Plugin {
	return manifest.Plugin{
		ID:      "com.powerx.starter",
		Name:    "Powerx Starter Plugin",
		Version: "0.1.0",
		Menus: []manifest.Menu{
			{
				Path:  "/_p/com.powerx.starter/admin",
				Title: "Powerx Starter Plugin",
			},
		},
		Permissions: []string{
			"com.powerx.starter.admin.view",
		},
	}
}
