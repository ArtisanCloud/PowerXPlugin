package logger

import (
	"testing"

	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
)

func TestWithRuntimeFieldsRetainsExtensions(t *testing.T) {
	entry := WithRuntimeFields("plugin.demo", "tenant-1", "trace-abc", "ws_bus_gateway_auth", Fields{
		runtimelogging.FieldGatewayAuth: "bearer",
		runtimelogging.FieldTokenSource: "request_bearer_passthrough",
		runtimelogging.FieldPluginID:    "plugin.override",
		runtimelogging.FieldComponent:   "gateway.auth",
	})
	fields := entry.Data

	if fields[runtimelogging.FieldGatewayAuth] != "bearer" {
		t.Fatalf("%s missing", runtimelogging.FieldGatewayAuth)
	}
	if fields[runtimelogging.FieldTokenSource] != "request_bearer_passthrough" {
		t.Fatalf("%s missing", runtimelogging.FieldTokenSource)
	}
	if fields[runtimelogging.FieldPluginID] != "plugin.override" {
		t.Fatalf("%s missing", runtimelogging.FieldPluginID)
	}
	if fields[runtimelogging.FieldComponent] != "gateway.auth" {
		t.Fatalf("%s missing", runtimelogging.FieldComponent)
	}
}
