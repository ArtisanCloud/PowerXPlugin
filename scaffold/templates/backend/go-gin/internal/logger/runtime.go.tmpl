package logger

import (
	"strings"

	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	"github.com/sirupsen/logrus"
)

var runtimeFieldMasker func(Fields) Fields

// RegisterRuntimeMasker sets a hook that can rewrite runtime log fields before
// they are emitted, enabling downstream privacy masking.
func RegisterRuntimeMasker(masker func(Fields) Fields) {
	runtimeFieldMasker = masker
}

// WithRuntimeFields enriches the log entry with standard runtime metadata.
func WithRuntimeFields(pluginID, tenantID, traceID, component string, extra Fields) *logrus.Entry {
	fields := Fields{
		runtimelogging.FieldPluginID:   strings.TrimSpace(pluginID),
		runtimelogging.FieldTenantUUID: strings.TrimSpace(tenantID),
		runtimelogging.FieldTenantKey:  runtimelogging.TenantKeyFromUUID(tenantID),
		runtimelogging.FieldTraceID:    strings.TrimSpace(traceID),
		runtimelogging.FieldComponent:  strings.TrimSpace(component),
	}
	for k, v := range extra {
		fields[k] = v
	}
	tenantUUID, _ := fields[runtimelogging.FieldTenantUUID].(string)
	tenantUUID = strings.TrimSpace(tenantUUID)
	fields[runtimelogging.FieldTenantUUID] = tenantUUID
	fields[runtimelogging.FieldTenantKey] = runtimelogging.TenantKeyFromUUID(tenantUUID)

	if runtimeFieldMasker != nil {
		fields = runtimeFieldMasker(fields)
	}
	return WithFields(fields)
}
