package auth

import (
	"bytes"
	"strings"
	"testing"
)

func TestRecordersAndRender(t *testing.T) {
	Reset()

	RecordLogin("delegated", "success")
	RecordLogin("delegated", "failure")
	RecordRefresh("local", "success")
	RecordLogout("local")
	RecordDelegateError("network")
	ObserveMode("delegated")

	var buf bytes.Buffer
	RenderMetrics(&buf)

	body := buf.String()
	checks := []string{
		"plugin_auth_login_total{mode=\"delegated\",result=\"success\"} 1",
		"plugin_auth_login_total{mode=\"delegated\",result=\"failure\"} 1",
		"plugin_auth_refresh_total{mode=\"local\",result=\"success\"} 1",
		"plugin_auth_logout_total{mode=\"local\"} 1",
		"plugin_iam_delegate_errors_total{type=\"network\"} 1",
		"plugin_iam_mode{mode=\"delegated\"} 1",
	}

	for _, snippet := range checks {
		if !strings.Contains(body, snippet) {
			t.Fatalf("expected metrics output to contain %q, got:\n%s", snippet, body)
		}
	}
}
