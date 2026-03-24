package runtime_ops

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
)

// RBACEntries exposes route-to-permission mappings for runtime ops admin APIs.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/runtime"
	adminBase := strings.TrimRight(prefix, "/") + "/admin"
	return map[string]authx.Permission{
		"POST:" + adminBase + "/notifications/test":      {Resource: "runtime.ops", Action: "invoke"},
		"POST:" + base + "/bootstrap":                    {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/sessions/register":            {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/sessions/*/ack":               {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/sessions/*/heartbeat":         {Resource: "runtime.ops", Action: "observe"},
		"POST:" + base + "/sessions/*/close":             {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/sessions/*/invoke":            {Resource: "runtime.ops", Action: "invoke"},
		"GET:" + base + "/quota/status":                  {Resource: "runtime.ops", Action: "read"},
		"POST:" + base + "/quota/overrides":              {Resource: "runtime.ops", Action: "manage"},
		"GET:" + base + "/metrics":                       {Resource: "runtime.ops", Action: "observe"},
		"POST:" + base + "/event-bridge/emit":            {Resource: "runtime.ops", Action: "invoke"},
		"POST:" + base + "/internal/event-fabric/topics": {Resource: "runtime.ops", Action: "invoke"},
		"POST:" + base + "/scheduler/mode/validate":      {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/scheduler/dispatches/*/retry": {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/scheduler/dispatches/*/pause": {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/scheduler/tickets/*/resume":   {Resource: "runtime.ops", Action: "manage"},
	}
}
