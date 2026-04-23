package logging

import "strings"

// NormalizeContextFields merges runtime fields with extra fields and ensures
// trace_id / component / plugin_id stay consistent across call sites.
func NormalizeContextFields(base, extra Fields) Fields {
	merged := MergeFields(base, extra)
	merged = NormalizeRuntimeFields(merged)

	// Keep canonical component and plugin keys trimmed if provided.
	if component := trimString(merged[FieldComponent]); component != "" {
		merged[FieldComponent] = component
	}
	if pluginID := trimString(merged[FieldPluginID]); pluginID != "" {
		merged[FieldPluginID] = strings.TrimSpace(pluginID)
	}

	// request_id can be used as trace fallback if caller did not provide trace_id.
	traceID := trimString(merged[FieldTraceID])
	if traceID == "" {
		if reqID := trimString(merged["request_id"]); reqID != "" {
			merged[FieldTraceID] = reqID
		}
	}
	return merged
}
