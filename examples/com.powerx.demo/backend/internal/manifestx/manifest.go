package manifestx

import "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"

// Plugin 返回插件的默认 Manifest 定义。
func Plugin() manifest.Plugin {
	return manifest.Plugin{
		ID:      "com.powerx.demo",
		Name:    "Powerx Demo Plugin",
		Version: "0.1.0",
		Menus: []manifest.Menu{
			{
				Path:  "/_p/com.powerx.demo/admin",
				Title: "Powerx Demo Plugin",
			},
		},
		Permissions: []string{
			"com.powerx.demo.admin.view",
		},
	}
}
