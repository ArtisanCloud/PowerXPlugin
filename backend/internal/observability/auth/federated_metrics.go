package auth

import (
	"sync"
)

const (
	metricFederatedLoginSuccess = "plugin_federated_login_success_total"
	metricPasswordLoginSuccess  = "plugin_password_login_success_total"
)

var (
	federatedMetricsMu sync.RWMutex
	federatedCounters  = map[string]map[string]float64{}
)

func ensureFederatedCounter(metric string) map[string]float64 {
	if federatedCounters[metric] == nil {
		federatedCounters[metric] = make(map[string]float64)
	}
	return federatedCounters[metric]
}

func RecordFederatedLoginSuccess(pluginID, tenantUUID string) {
	federatedMetricsMu.Lock()
	defer federatedMetricsMu.Unlock()
	labels := map[string]string{
		"plugin_id":   normalizedPluginID(pluginID),
		"tenant_uuid": normalizedTenantUUID(tenantUUID),
	}
	ensureFederatedCounter(metricFederatedLoginSuccess)[labelKey(labels)]++
}

func RecordPasswordLoginSuccess(pluginID, mode string) {
	federatedMetricsMu.Lock()
	defer federatedMetricsMu.Unlock()
	labels := map[string]string{
		"plugin_id": normalizedPluginID(pluginID),
		"mode":      normalizedMode(mode),
	}
	ensureFederatedCounter(metricPasswordLoginSuccess)[labelKey(labels)]++
}

// LoginMethodSnapshot 返回当前统计快照，用于回归记录与报表导出。
func LoginMethodSnapshot() map[string]map[string]float64 {
	federatedMetricsMu.RLock()
	defer federatedMetricsMu.RUnlock()
	out := make(map[string]map[string]float64, len(federatedCounters))
	for metric, values := range federatedCounters {
		cloned := make(map[string]float64, len(values))
		for k, v := range values {
			cloned[k] = v
		}
		out[metric] = cloned
	}
	return out
}

func resetFederatedLoginMetricsForTests() {
	federatedMetricsMu.Lock()
	defer federatedMetricsMu.Unlock()
	federatedCounters = map[string]map[string]float64{}
}
