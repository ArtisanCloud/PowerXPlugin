package auth

import (
	"bytes"
	"strings"
	"testing"
)

func TestRecordersAndRender(t *testing.T) {
	Reset()

	RecordLogin("plugin.demo", "delegated", "success")
	RecordLogin("plugin.demo", "delegated", "failure")
	RecordRefresh("plugin.demo", "local", "success")
	RecordLogout("plugin.demo", "local")
	RecordDelegateError("plugin.demo", "network")
	ObserveMode("delegated")

	var buf bytes.Buffer
	RenderMetrics(&buf)

	body := buf.String()
	checks := []string{
		"plugin_auth_login_total{mode=\"delegated\",plugin_id=\"plugin.demo\",result=\"success\"} 1",
		"plugin_auth_login_total{mode=\"delegated\",plugin_id=\"plugin.demo\",result=\"failure\"} 1",
		"plugin_auth_refresh_total{mode=\"local\",plugin_id=\"plugin.demo\",result=\"success\"} 1",
		"plugin_auth_logout_total{mode=\"local\",plugin_id=\"plugin.demo\"} 1",
		"plugin_iam_delegate_errors_total{plugin_id=\"plugin.demo\",type=\"network\"} 1",
		"plugin_iam_mode{mode=\"delegated\"} 1",
	}

	for _, snippet := range checks {
		if !strings.Contains(body, snippet) {
			t.Fatalf("expected metrics output to contain %q, got:\n%s", snippet, body)
		}
	}
}
