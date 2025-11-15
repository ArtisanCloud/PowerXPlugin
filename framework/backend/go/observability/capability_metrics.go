package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CapabilityRegistryDuration measures how long registry / submit pipelines take.
	CapabilityRegistryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "capability_registry_duration_ms",
			Help:    "Duration of capability registry operations in milliseconds",
			Buckets: prometheus.ExponentialBuckets(100, 2, 10), // 100ms ~ 51s
		},
		[]string{"operation", "result"},
	)

	// CapabilityExposureActivateRate captures the rolling success ratio for exposure activation.
	CapabilityExposureActivateRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "capability_exposure_activate_rate",
			Help: "Successful activation ratio (0-1) for capability exposure channels",
		},
		[]string{"capability", "channel"},
	)
)

// ObserveCapabilityRegistryDuration records a registry workflow duration.
func ObserveCapabilityRegistryDuration(operation, result string, durationMs float64) {
	CapabilityRegistryDuration.WithLabelValues(operation, result).Observe(durationMs)
}

// SetCapabilityExposureActivateRate updates the activation ratio for a capability/channel pair.
func SetCapabilityExposureActivateRate(capability, channel string, ratio float64) {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	CapabilityExposureActivateRate.WithLabelValues(capability, channel).Set(ratio)
}
