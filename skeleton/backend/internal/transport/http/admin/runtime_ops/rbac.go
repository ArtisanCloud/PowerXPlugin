package runtime_ops

import (
	"strings"

	authx "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/middleware"
)

// RBACEntries exposes route-to-permission mappings for runtime ops admin APIs.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/runtime"
	return map[string]authx.Permission{
		"POST:" + base + "/bootstrap":         {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/sessions/register": {Resource: "runtime.ops", Action: "manage"},
		"GET:" + base + "/quota/status":       {Resource: "runtime.ops", Action: "read"},
		"POST:" + base + "/quota/overrides":   {Resource: "runtime.ops", Action: "manage"},
		"GET:" + base + "/metrics":            {Resource: "runtime.ops", Action: "observe"},
	}
}
