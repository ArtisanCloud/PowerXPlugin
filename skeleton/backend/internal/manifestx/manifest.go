package manifestx

import "github.com/powerx-plugin/framework/backend/go/manifest"

// Plugin returns the manifest definition consumed by the framework/router layer.
func Plugin() manifest.Plugin {
	return manifest.Plugin{
		ID:      "com.powerx.plugin.demo",
		Name:    "PowerX Demo Plugin",
		Version: "0.1.0",
		Menus: []manifest.Menu{
			{
				Path:  "/templates/intro",
				Title: "Plugin Intro",
			},
			{
				Path:  "/templates/crud",
				Title: "Template CRUD",
			},
		},
	}
}
