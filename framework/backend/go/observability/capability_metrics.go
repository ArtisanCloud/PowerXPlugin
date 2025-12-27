package observability

import (
	"strings"
	"time"

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

	capabilityInvocationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "capability_invocation_duration_ms",
			Help:    "Duration of capability invocations routed via Gateway (milliseconds)",
			Buckets: []float64{20, 50, 100, 250, 500, 1000, 2000, 5000},
		},
		[]string{"capability", "tenant", "result"},
	)

	capabilityInvocationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "capability_invocation_total",
			Help: "Total capability invocations grouped by capability, tenant, and result",
		},
		[]string{"capability", "tenant", "result"},
	)

	capabilityRateLimitEvents = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "capability_rate_limit_events_total",
			Help: "Rate limit events grouped by capability and tenant",
		},
		[]string{"capability", "tenant"},
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

// ObserveCapabilityInvocation 记录一次 Gateway 能力调用的耗时与结果。
func ObserveCapabilityInvocation(capability, tenant string, result CapabilityInvocationResult, duration time.Duration) {
	valueMs := duration.Seconds() * 1000
	if valueMs < 0 {
		valueMs = 0
	}
	cap := normalizeLabel(capability)
	tenantLabel := normalizeLabel(tenant)
	resultLabel := string(result)
	capabilityInvocationDuration.WithLabelValues(cap, tenantLabel, resultLabel).Observe(valueMs)
	capabilityInvocationTotal.WithLabelValues(cap, tenantLabel, resultLabel).Inc()
}

// IncrementCapabilityRateLimit 记录一次限流事件。
func IncrementCapabilityRateLimit(capability, tenant string) {
	capabilityRateLimitEvents.WithLabelValues(normalizeLabel(capability), normalizeLabel(tenant)).Inc()
}

func normalizeLabel(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}
