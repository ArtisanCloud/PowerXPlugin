package eventbridge

type MetricsRecorder interface {
	RecordEmit(pluginID, tenantUUID, topic, result string)
	RecordConsume(pluginID, tenantUUID, topic, result string)
	ObserveLatencyMs(pluginID, tenantUUID, topic, op string, ms float64)
}

