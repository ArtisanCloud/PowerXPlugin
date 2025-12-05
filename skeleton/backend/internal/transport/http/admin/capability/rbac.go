package capability

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

// RBACEntries describes the minimal permissions required by capability routes.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/capabilities/register"
	resource := app.PluginID + ":capability"
	reviewBase := strings.TrimRight(prefix, "/") + "/admin/capabilities/reviews"
	reviewRes := app.PluginID + ":capability.review"
	return map[string]authx.Permission{
		"GET:" + base + "/template":                {Resource: resource, Action: "read"},
		"POST:" + base + "/validate":               {Resource: resource, Action: "create"},
		"POST:" + base:                             {Resource: resource, Action: "create"},
		"GET:" + reviewBase + "/*":                 {Resource: reviewRes, Action: "read"},
		"POST:" + reviewBase + "/*/resubmit":       {Resource: reviewRes, Action: "update"},
		"POST:" + reviewBase + "/tasks/*/comments": {Resource: reviewRes, Action: "update"},
		"POST:" + reviewBase + "/tasks/*/decision": {Resource: reviewRes, Action: "review"},
	}
}
