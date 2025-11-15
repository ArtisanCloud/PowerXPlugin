package observability

import (
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SC-002: Online publish review pipeline duration
	PublishPipelineDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "plugin_publish_pipeline_duration_ms",
			Help:    "Duration of online publish review pipeline in milliseconds",
			Buckets: prometheus.ExponentialBuckets(1000, 2, 15), // 1s to ~2 hours
		},
		[]string{"channel", "status"},
	)

	// SC-003: Offline approval duration
	OfflineApprovalDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "plugin_offline_approval_duration_minutes",
			Help:    "Duration of offline package approval in minutes",
			Buckets: prometheus.ExponentialBuckets(1, 2, 15), // 1min to ~1 day
		},
		[]string{"plugin", "status"},
	)

	// SC-004: Rollback latency
	RollbackLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "plugin_install_rollback_latency_seconds",
			Help:    "Time to complete plugin rollback in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~8 minutes
		},
		[]string{"tenant", "plugin", "trigger"}, // trigger: "auto" or "manual"
	)

	// Deployment status counter
	DeploymentStatus = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "plugin_deployments_total",
			Help: "Total number of deployment attempts by status",
		},
		[]string{"status"}, // pending, installing, success, failed, rolling_back, rolled_back
	)
)

// InitMetrics initializes Prometheus metrics
func InitMetrics(app *bootstrap.App) error {
	// Metrics are automatically registered via promauto.NewHistogramVec
	// The /metrics endpoint will be available at http://localhost:8080/metrics
	_ = app
	return nil
}

// RecordPublishPipelineDuration records the duration of an online publish review
func RecordPublishPipelineDuration(durationMs float64, channel, status string) {
	PublishPipelineDuration.WithLabelValues(channel, status).Observe(durationMs)
}

// RecordOfflineApprovalDuration records the duration of offline approval
func RecordOfflineApprovalDuration(durationMinutes float64, plugin, status string) {
	OfflineApprovalDuration.WithLabelValues(plugin, status).Observe(durationMinutes)
}

// RecordRollbackLatency records the time to complete a rollback
func RecordRollbackLatency(durationSeconds float64, tenant, plugin, trigger string) {
	RollbackLatency.WithLabelValues(tenant, plugin, trigger).Observe(durationSeconds)
}

// IncrementDeploymentStatus increments the deployment status counter
func IncrementDeploymentStatus(status string) {
	DeploymentStatus.WithLabelValues(status).Inc()
}
