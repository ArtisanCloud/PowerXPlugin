package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDoctorEnvReport(t *testing.T) {
	tmpDir := t.TempDir()

	origRuntimeVersion := runtimeVersionFunc
	runtimeVersionFunc = func() string {
		return "go1.25.1"
	}
	t.Cleanup(func() {
		runtimeVersionFunc = origRuntimeVersion
	})

	origNodeRunner := nodeVersionRunner
	nodeVersionRunner = func() ([]byte, error) {
		return []byte("v18.19.0"), nil
	}
	t.Cleanup(func() {
		nodeVersionRunner = origNodeRunner
	})

	if err := runDoctor([]string{"--entry", tmpDir, "--check-env"}); err != nil {
		t.Fatalf("runDoctor failed: %v", err)
	}

	report := readDoctorReport(t, filepath.Join(tmpDir, ".doctor", "report.json"))
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}

	result := report.Results[0]
	if result.Name != "Toolchain" {
		t.Fatalf("unexpected result name %q", result.Name)
	}
	if result.Status != "pass" {
		t.Fatalf("expected pass status, got %s", result.Status)
	}
	if !strings.Contains(result.Details, "Go 1.25.1") {
		t.Fatalf("unexpected details %q", result.Details)
	}
	if !strings.Contains(result.Details, "Node 18.19.0") {
		t.Fatalf("expected node version in details, got %q", result.Details)
	}
}

func TestRunDoctorDevAPICheck(t *testing.T) {
	entryDir := t.TempDir()
	manifest := []byte("id: test-plugin\nversion: 0.1.0\nbackend:\n  entry: ./backend\n")
	if err := os.WriteFile(filepath.Join(entryDir, "plugin.yaml"), manifest, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	var registerCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1")
		switch {
		case r.Method == http.MethodPost && path == "/internal/dev/plugins/register":
			registerCalled = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"code":201,"message":"success","data":{"sessionId":"session-1","reloadToken":"reload-token"}}`)
		case r.Method == http.MethodDelete && strings.HasPrefix(path, "/internal/dev/plugins/register/"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"code":200,"message":"success","data":{}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	reportPath := filepath.Join(entryDir, "doctor-report.json")
	if err := runDoctor([]string{
		"--entry", entryDir,
		"--dev-api", server.URL,
		"--check-devapi",
		"--output", reportPath,
	}); err != nil {
		t.Fatalf("runDoctor failed: %v", err)
	}

	if !registerCalled {
		t.Fatal("expected dev api register to be called")
	}

	report := readDoctorReport(t, reportPath)
	if len(report.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(report.Results))
	}

	result := report.Results[0]
	if result.Name != "Dev API" {
		t.Fatalf("unexpected result name %q", result.Name)
	}
	if result.Status != "pass" {
		t.Fatalf("expected pass status, got %s", result.Status)
	}
	if !strings.Contains(result.Details, "Register/Delete") {
		t.Fatalf("unexpected details %q", result.Details)
	}
}

func readDoctorReport(t *testing.T, path string) *DoctorReport {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}

	var report DoctorReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	return &report
}
