package observability

// MetricDescriptor describes a telemetry signal Publish Hub needs to expose.
type MetricDescriptor struct {
	Name        string
	Help        string
	Unit        string
	Labels      []string
	Recommended string // counter|gauge|histogram
}

var (
	MetricDevHotloadReloadDuration = MetricDescriptor{
		Name:        "dev_hotload_reload_duration_ms",
		Help:        "Time spent to push a single px-plugin dev reload round-trip (CLI → Dev API → Admin)",
		Unit:        "milliseconds",
		Labels:      []string{"tenant", "plugin", "status"},
		Recommended: "histogram",
	}
	MetricPublishPipelineDuration = MetricDescriptor{
		Name:        "plugin_publish_pipeline_duration_ms",
		Help:        "Total runtime of px-plugin publish command until Marketplace enqueue",
		Unit:        "milliseconds",
		Labels:      []string{"channel", "result"},
		Recommended: "histogram",
	}
	MetricOfflineApprovalDuration = MetricDescriptor{
		Name:        "plugin_offline_approval_duration_minutes",
		Help:        "Duration from offline upload to reviewer approval/ rejection",
		Unit:        "minutes",
		Labels:      []string{"status"},
		Recommended: "histogram",
	}
	MetricInstallRollbackLatency = MetricDescriptor{
		Name:        "plugin_install_rollback_latency_seconds",
		Help:        "Latency between install failure detection and successful rollback",
		Unit:        "seconds",
		Labels:      []string{"tenant", "plugin", "result"},
		Recommended: "histogram",
	}
	MetricPublishErrorsTotal = MetricDescriptor{
		Name:        "plugin_publish_errors_total",
		Help:        "Counter for failed publish attempts grouped by failure reason",
		Unit:        "count",
		Labels:      []string{"reason"},
		Recommended: "counter",
	}
)

// PublishHubMetrics lists all base metrics to register with the host runtime.
func PublishHubMetrics() []MetricDescriptor {
	return []MetricDescriptor{
		MetricDevHotloadReloadDuration,
		MetricPublishPipelineDuration,
		MetricOfflineApprovalDuration,
		MetricInstallRollbackLatency,
		MetricPublishErrorsTotal,
	}
}
