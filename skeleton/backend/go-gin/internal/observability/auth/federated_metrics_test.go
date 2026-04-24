package auth

import "testing"

func TestLoginMethodMetricsSnapshot(t *testing.T) {
	resetFederatedLoginMetricsForTests()
	RecordFederatedLoginSuccess("com.powerx.plugins.demo", "tenant-a")
	RecordPasswordLoginSuccess("com.powerx.plugins.demo", "local")

	snapshot := LoginMethodSnapshot()
	if len(snapshot[metricFederatedLoginSuccess]) != 1 {
		t.Fatalf("federated metric series=%d, want 1", len(snapshot[metricFederatedLoginSuccess]))
	}
	if len(snapshot[metricPasswordLoginSuccess]) != 1 {
		t.Fatalf("password metric series=%d, want 1", len(snapshot[metricPasswordLoginSuccess]))
	}
}
