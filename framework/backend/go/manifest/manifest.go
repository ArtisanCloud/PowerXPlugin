package manifest

// Menu 描述管理端导航项。
type Menu struct {
	Path  string `json:"path"`
	Title string `json:"title"`
	Icon  string `json:"icon,omitempty"`
	Order int    `json:"order,omitempty"`
}

// Plugin 汇总插件向宿主上报的基础信息。
type Plugin struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	Menus       []Menu   `json:"menus,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// App 定义宿主侧可接受 Manifest 的能力。
type App interface {
	RegisterManifest(p Plugin)
}

// Register 为调用方提供统一入口，方便 skeleton/框架调用。
func Register(app App, p Plugin) {
	if app == nil {
		return
	}
	app.RegisterManifest(p)
}
