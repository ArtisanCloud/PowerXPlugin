package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/powerx-plugin/cli/internal/devapi"
	"github.com/powerx-plugin/cli/internal/manifest"
	"github.com/powerx-plugin/cli/internal/watch"
)

var (
	runtimeVersionFunc = runtime.Version
	nodeVersionRunner  = func() ([]byte, error) {
		cmd := exec.Command("node", "--version")
		return cmd.Output()
	}
)

// DoctorOptions defines doctor command options.
type DoctorOptions struct {
	Entry          string
	DevAPI         string
	Tenant         string
	Output         string
	CheckEnv       bool
	CheckDevAPI    bool
	CheckMTLS      bool
	CheckWatch     bool
	MTLSCert       string
	MTLSKey        string
	MTLSCA         string
	MTLSServerName string
	MTLSSkipVerify bool
}

type DoctorReport struct {
	GeneratedAt time.Time           `json:"generatedAt"`
	EntryPath   string              `json:"entryPath"`
	DevAPIBase  string              `json:"devApiBase"`
	Results     []DoctorCheckResult `json:"results"`
}

type DoctorCheckResult struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Details     string `json:"details,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	DurationMS  int64  `json:"durationMs"`
}

func announceDoctorCheck(name string) {
	fmt.Printf("→ %s...\n", name)
}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ExitOnError)
	opts := &DoctorOptions{}

	fs.StringVar(&opts.Entry, "entry", "", "Path to the plugin directory (default: current)")
	fs.StringVar(&opts.DevAPI, "dev-api", "", "Dev API endpoint URL")
	fs.StringVar(&opts.Tenant, "tenant", "", "Tenant ID for diagnostic Dev API calls")
	fs.StringVar(&opts.Output, "output", "", "Path to write doctor report (default: <entry>/.doctor/report.json)")
	fs.BoolVar(&opts.CheckEnv, "check-env", false, "Run toolchain checks")
	fs.BoolVar(&opts.CheckDevAPI, "check-devapi", false, "Run Dev API connectivity check")
	fs.BoolVar(&opts.CheckMTLS, "check-mtls", false, "Validate mTLS configuration")
	fs.BoolVar(&opts.CheckWatch, "check-watch", false, "Validate file watcher configuration")
	fs.StringVar(&opts.MTLSCert, "mtls-cert", "", "Path to the mTLS client certificate")
	fs.StringVar(&opts.MTLSKey, "mtls-key", "", "Path to the mTLS client key")
	fs.StringVar(&opts.MTLSCA, "mtls-ca", "", "Path to the mTLS CA certificate")
	fs.StringVar(&opts.MTLSServerName, "mtls-server-name", "", "Override mTLS TLS server name")
	fs.BoolVar(&opts.MTLSSkipVerify, "mtls-skip-verify", false, "Skip TLS server verification (insecure)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if opts.Entry == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("determine working directory: %w", err)
		}
		opts.Entry = cwd
	}

	entryPath, err := filepath.Abs(opts.Entry)
	if err != nil {
		return fmt.Errorf("resolve entry path: %w", err)
	}
	if _, err := os.Stat(entryPath); err != nil {
		return fmt.Errorf("entry path invalid: %w", err)
	}

	if opts.DevAPI == "" {
		opts.DevAPI = resolveDevAPIBase("")
	}
	devAPIBase := resolveDevAPIBase(opts.DevAPI)

	if opts.Output == "" {
		opts.Output = filepath.Join(entryPath, ".doctor", "report.json")
	}

	if !opts.CheckEnv && !opts.CheckDevAPI && !opts.CheckMTLS && !opts.CheckWatch {
		opts.CheckEnv = true
		opts.CheckDevAPI = true
		opts.CheckMTLS = true
		opts.CheckWatch = true
	}

	report := &DoctorReport{
		GeneratedAt: time.Now(),
		EntryPath:   entryPath,
		DevAPIBase:  devAPIBase,
	}

	fmt.Println("Running px-plugin doctor diagnostics...")
	fmt.Println()

	hasFail := false

	if opts.CheckEnv {
		announceDoctorCheck("Toolchain")
		res := runDoctorEnvCheck()
		report.Results = append(report.Results, res)
		if res.Status == "fail" {
			hasFail = true
		}
	}
	if opts.CheckMTLS {
		announceDoctorCheck("mTLS configuration")
		res := runDoctorMTLSCheck(opts)
		report.Results = append(report.Results, res)
		if res.Status == "fail" {
			hasFail = true
		}
	}
	if opts.CheckDevAPI {
		announceDoctorCheck("Dev API connectivity")
		res := runDoctorDevAPICheck(opts, entryPath, devAPIBase)
		report.Results = append(report.Results, res)
		if res.Status == "fail" {
			hasFail = true
		}
	}
	if opts.CheckWatch {
		announceDoctorCheck("Watcher readiness")
		res := runDoctorWatchCheck(entryPath)
		report.Results = append(report.Results, res)
		if res.Status == "fail" {
			hasFail = true
		}
	}

	if err := writeDoctorReport(report, opts.Output); err != nil {
		return err
	}
	printDoctorSummary(report, opts.Output)

	if hasFail {
		return fmt.Errorf("doctor detected issues")
	}
	return nil
}

func runDoctorEnvCheck() DoctorCheckResult {
	start := time.Now()
	goStatus, goMsg := checkGoVersion()
	nodeStatus, nodeMsg := checkNodeVersion()
	status := "pass"
	remediation := ""
	if goStatus == "fail" || nodeStatus == "fail" {
		status = "fail"
		remediation = "Install required toolchain versions (Go 1.24+, Node.js 18+)."
	}
	return DoctorCheckResult{
		Name:        "Toolchain",
		Status:      status,
		Details:     fmt.Sprintf("Go: %s; Node: %s", goMsg, nodeMsg),
		Remediation: remediation,
		DurationMS:  time.Since(start).Milliseconds(),
	}
}

func runDoctorMTLSCheck(opts *DoctorOptions) DoctorCheckResult {
	start := time.Now()
	devOpts := &DevOptions{
		MTLSCert:       opts.MTLSCert,
		MTLSKey:        opts.MTLSKey,
		MTLSCA:         opts.MTLSCA,
		MTLSServerName: opts.MTLSServerName,
		MTLSSkipVerify: opts.MTLSSkipVerify,
	}
	client, err := resolveMTLSClient(devOpts, resolveDevAPIBase(opts.DevAPI))
	if err != nil {
		return DoctorCheckResult{
			Name:        "mTLS",
			Status:      "fail",
			Details:     fmt.Sprintf("failed to initialize mTLS: %v", err),
			Remediation: "Verify PX_MTLS_* env or config.json certificate paths.",
			DurationMS:  time.Since(start).Milliseconds(),
		}
	}
	if client == nil {
		return DoctorCheckResult{
			Name:       "mTLS",
			Status:     "warn",
			Details:    "mTLS not configured; using plain HTTP",
			DurationMS: time.Since(start).Milliseconds(),
		}
	}
	defer client.Close()
	if err := client.CheckValidity(); err != nil {
		return DoctorCheckResult{
			Name:        "mTLS",
			Status:      "fail",
			Details:     fmt.Sprintf("certificate issue: %v", err),
			Remediation: "Renew certificates or update PX_MTLS_* paths.",
			DurationMS:  time.Since(start).Milliseconds(),
		}
	}
	info, _ := client.GetCertificateInfo()
	detail := "mTLS enabled"
	if info != nil {
		detail = fmt.Sprintf("mTLS enabled (expires %s)", info.NotAfter.Format(time.RFC3339))
	}
	return DoctorCheckResult{
		Name:       "mTLS",
		Status:     "pass",
		Details:    detail,
		DurationMS: time.Since(start).Milliseconds(),
	}
}

func runDoctorDevAPICheck(opts *DoctorOptions, entryPath, devAPIBase string) DoctorCheckResult {
	start := time.Now()
	devOpts := &DevOptions{
		MTLSCert:       opts.MTLSCert,
		MTLSKey:        opts.MTLSKey,
		MTLSCA:         opts.MTLSCA,
		MTLSServerName: opts.MTLSServerName,
		MTLSSkipVerify: opts.MTLSSkipVerify,
	}
	mtlsClient, err := resolveMTLSClient(devOpts, devAPIBase)
	if err != nil {
		return DoctorCheckResult{
			Name:        "Dev API",
			Status:      "fail",
			Details:     fmt.Sprintf("mTLS error: %v", err),
			Remediation: "Check PX_MTLS_* or certificate paths.",
			DurationMS:  time.Since(start).Milliseconds(),
		}
	}
	if mtlsClient != nil {
		defer mtlsClient.Close()
	}

	client := devapi.NewClient(devapi.ClientOptions{
		BaseURL:    devAPIBase,
		Timeout:    10 * time.Second,
		MTLSClient: mtlsClient,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pluginID := "doctor-check"
	version := "0.0.0"
	backendEntry := ""
	if manifest, err := manifest.Load(entryPath); err == nil {
		pluginID = manifest.ID
		version = manifest.Version
		backendEntry = manifest.Backend.Entry
	}

	resp, err := client.Register(ctx, &devapi.RegisterRequest{
		PluginID:  pluginID,
		Version:   version,
		EntryPath: entryPath,
		Tenant:    opts.Tenant,
		Metadata: map[string]string{
			"backend.entry": backendEntry,
		},
	})
	if err != nil {
		return DoctorCheckResult{
			Name:        "Dev API",
			Status:      "fail",
			Details:     fmt.Sprintf("register failed: %v", err),
			Remediation: "Ensure px-dev-api is reachable and credentials are valid.",
			DurationMS:  time.Since(start).Milliseconds(),
		}
	}
	client.SetReloadToken(resp.ReloadToken)
	if resp.SessionID != "" {
		ctxDelete, cancelDelete := context.WithTimeout(context.Background(), 5*time.Second)
		_ = client.Delete(ctxDelete, resp.SessionID)
		cancelDelete()
	}

	return DoctorCheckResult{
		Name:       "Dev API",
		Status:     "pass",
		Details:    "Register/Delete handshake succeeded",
		DurationMS: time.Since(start).Milliseconds(),
	}
}

func runDoctorWatchCheck(entryPath string) DoctorCheckResult {
	start := time.Now()
	watcher := watch.NewFileWatcher(watch.DefaultConfig(entryPath))
	if err := watcher.Start(); err != nil {
		return DoctorCheckResult{
			Name:        "Watcher",
			Status:      "fail",
			Details:     fmt.Sprintf("failed to start watcher: %v", err),
			Remediation: "Check fsnotify support and file permissions.",
			DurationMS:  time.Since(start).Milliseconds(),
		}
	}
	watcher.Stop()
	return DoctorCheckResult{
		Name:       "Watcher",
		Status:     "pass",
		Details:    "fsnotify watcher initialized successfully",
		DurationMS: time.Since(start).Milliseconds(),
	}
}

func writeDoctorReport(report *DoctorReport, path string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

func printDoctorSummary(report *DoctorReport, output string) {
	fmt.Println("\nDoctor summary:")
	for _, res := range report.Results {
		fmt.Printf("  [%s] %s - %s\n", strings.ToUpper(res.Status), res.Name, res.Details)
		if res.Status == "fail" && res.Remediation != "" {
			fmt.Printf("      Remediation: %s\n", res.Remediation)
		}
	}
	if output != "" {
		fmt.Printf("\nReport saved to: %s\n", output)
	}
}

func checkGoVersion() (string, string) {
	version := runtimeVersionFunc()
	trimmed := strings.TrimPrefix(version, "go")
	if compareSemver(trimmed, "1.24") >= 0 {
		return "pass", fmt.Sprintf("Go %s", trimmed)
	}
	return "fail", fmt.Sprintf("Go %s (need >=1.24)", trimmed)
}

func checkNodeVersion() (string, string) {
	out, err := nodeVersionRunner()
	if err != nil {
		return "fail", "node not found"
	}
	version := strings.TrimSpace(string(out))
	trimmed := strings.TrimPrefix(version, "v")
	if compareSemver(trimmed, "18.0.0") >= 0 {
		return "pass", fmt.Sprintf("Node %s", trimmed)
	}
	return "fail", fmt.Sprintf("Node %s (need >=18)", trimmed)
}

func compareSemver(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for len(as) < 3 {
		as = append(as, "0")
	}
	for len(bs) < 3 {
		bs = append(bs, "0")
	}
	for i := 0; i < 3; i++ {
		ai := parseInt(as[i])
		bi := parseInt(bs[i])
		if ai > bi {
			return 1
		}
		if ai < bi {
			return -1
		}
	}
	return 0
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return 0
}
