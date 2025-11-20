package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/powerx-plugin/cli/internal/audit"
	"github.com/powerx-plugin/cli/internal/build"
	"github.com/powerx-plugin/cli/internal/config"
	"github.com/powerx-plugin/cli/internal/devapi"
	"github.com/powerx-plugin/cli/internal/devwatch"
	"github.com/powerx-plugin/cli/internal/manifest"
	"github.com/powerx-plugin/cli/internal/mtls"
	"github.com/powerx-plugin/cli/internal/resources"
	"github.com/powerx-plugin/cli/internal/session"
	"github.com/powerx-plugin/cli/internal/sse"
	"github.com/powerx-plugin/cli/internal/watch"
	"gopkg.in/yaml.v3"
)

// DevOptions holds the configuration for the dev command
type DevOptions struct {
	Watch              bool
	Once               bool
	AuthSetup          bool
	Entry              string
	Tenant             string
	TenantID           uint64
	DeveloperID        uint64
	Ignore             []string
	DevAPI             string
	DevAPIToken        string
	Resume             string
	Stop               string
	ForceStop          string
	List               bool
	Logs               string
	LogsLevel          string
	LogsFile           string
	NoColor            bool
	ListStatus         string
	MTLSCert           string
	MTLSKey            string
	MTLSCA             string
	MTLSServerName     string
	MTLSSkipVerify     bool
	MaxProcs           int
	MaxMemoryMB        int
	MaxCPUPercent      int
	MaxWatchFiles      int
	DeleteSession      string
	ClearSessions      bool
	ClearSessionsForce bool
	NoConfirm          bool
	Yes                bool
	userConfig         *config.Config
}

// runDev is the entry point for the dev command
func runDev(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ExitOnError)
	opts := &DevOptions{}

	// Define flags
	fs.BoolVar(&opts.Once, "once", false, "Run a single build and reload, then exit (no watch)")
	fs.BoolVar(&opts.Watch, "watch", false, "Enable file watching and hot reload")
	fs.BoolVar(&opts.AuthSetup, "auth", false, "Set up px-plugin auth defaults (~/.px-plugin/config.json & certs) and exit")
	fs.StringVar(&opts.Entry, "entry", "", "Path to the plugin entry directory")
	fs.StringVar(&opts.Tenant, "tenant", "", "Tenant slug for the dev session")
	fs.Uint64Var(&opts.TenantID, "tenant-id", 0, "Numeric tenant ID used when registering Dev API session")
	fs.Uint64Var(&opts.DeveloperID, "developer-id", 0, "Developer/member ID used for Dev API session ownership")
	fs.Var((*StringSliceFlag)(&opts.Ignore), "ignore", "File patterns to ignore (can be repeated)")
	fs.StringVar(&opts.DevAPI, "dev-api", "", "Dev API endpoint URL")
	fs.StringVar(&opts.DevAPIToken, "dev-api-token", "", "Dev API bearer/API token (optional)")
	fs.StringVar(&opts.Resume, "resume", "", "Resume an existing session by ID")
	fs.StringVar(&opts.Stop, "stop", "", "Stop a running session by ID")
	fs.StringVar(&opts.ForceStop, "force-stop", "", "Force stop a remote session by ID (bypasses local cache)")
	fs.StringVar(&opts.DeleteSession, "delete-session", "", "Delete a remote Dev API session by ID and remove local cache")
	fs.BoolVar(&opts.ClearSessions, "clear-sessions", false, "Clear all remote Dev API sessions (terminated by default)")
	fs.BoolVar(&opts.ClearSessionsForce, "clear-sessions-force", false, "Force clear all remote Dev API sessions (include active)")
	fs.BoolVar(&opts.NoConfirm, "no-confirm", false, "Skip confirmation prompt for destructive actions")
	fs.BoolVar(&opts.Yes, "yes", false, "Alias for --no-confirm")
	fs.BoolVar(&opts.List, "list-sessions", false, "List all active sessions")
	fs.StringVar(&opts.ListStatus, "list-status", "active", "Status filter for --list-sessions (active|pending|terminated|all)")
	fs.StringVar(&opts.Logs, "logs", "", "Show logs for a specific session")
	fs.StringVar(&opts.LogsLevel, "logs-level", "info", "Minimum log level to display (debug, info, warn, error)")
	fs.StringVar(&opts.LogsFile, "logs-file", "", "Write logs to a file (optional)")
	fs.BoolVar(&opts.NoColor, "no-color", false, "Disable colored output")
	fs.StringVar(&opts.MTLSCert, "mtls-cert", "", "Path to the mTLS client certificate")
	fs.StringVar(&opts.MTLSKey, "mtls-key", "", "Path to the mTLS client key")
	fs.StringVar(&opts.MTLSCA, "mtls-ca", "", "Path to the mTLS CA certificate")
	fs.StringVar(&opts.MTLSServerName, "mtls-server-name", "", "Override mTLS TLS server name")
	fs.BoolVar(&opts.MTLSSkipVerify, "mtls-skip-verify", false, "Skip TLS server certificate verification (insecure)")
	fs.IntVar(&opts.MaxProcs, "max-procs", 0, "Maximum number of OS threads for Go runtime (default derived from config)")
	fs.IntVar(&opts.MaxMemoryMB, "max-memory-mb", 0, "Memory budget for dev --watch before throttling (MB)")
	fs.IntVar(&opts.MaxCPUPercent, "max-cpu-percent", 0, "CPU usage threshold before throttling (percent)")
	fs.IntVar(&opts.MaxWatchFiles, "max-watch-files", 0, "Maximum number of files/directories to watch recursively")

	// Parse flags
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	if opts.Entry == "" {
		if cwd, err := os.Getwd(); err == nil {
			opts.Entry = cwd
		}
	}
	if opts.Once && opts.Watch {
		return fmt.Errorf("--once and --watch are mutually exclusive")
	}
	cfg := applyDevDefaults(opts)
	opts.userConfig = cfg

	// Execute based on subcommand
	switch {
	case opts.AuthSetup:
		return runDevAuthSetup(opts)
	case opts.List:
		return runDevListSessions(opts)
	case opts.Resume != "":
		return runDevResumeSession(opts.Resume, opts)
	case opts.Stop != "":
		return runDevStopSession(opts.Stop, opts)
	case opts.ForceStop != "":
		return runDevForceStopSession(opts.ForceStop, opts)
	case opts.DeleteSession != "":
		return runDevDeleteSession(opts.DeleteSession, opts)
	case opts.ClearSessions:
		return runDevClearSessions(opts)
	case opts.Logs != "":
		return runDevShowLogs(opts.Logs, opts)
	case opts.Watch:
		if opts.Entry == "" {
			return fmt.Errorf("--entry is required when using --watch")
		}
		return runDevWatch(opts)
	case opts.Once || (opts.Entry != "" && !opts.Watch):
		if opts.Entry == "" {
			return fmt.Errorf("--entry is required for single-run mode")
		}
		return runDevOnce(opts)
	default:
		fs.Usage()
		return nil
	}
}

func runDevAuthSetup(opts *DevOptions) error {
	fmt.Println("Configuring px-plugin auth defaults...")

	entryPath := opts.Entry
	if entryPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			entryPath = cwd
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("determine home directory: %w", err)
	}

	srcCert := opts.MTLSCert
	srcKey := opts.MTLSKey
	srcCA := opts.MTLSCA
	if srcCert == "" {
		srcCert = filepath.Join(home, ".powerx", "cli", "client.crt")
	}
	if srcKey == "" {
		srcKey = filepath.Join(home, ".powerx", "cli", "client.key")
	}
	if srcCA == "" {
		srcCA = filepath.Join(home, ".powerx", "cli", "ca.crt")
	}

	if err := ensureFileExists(srcCert, "client certificate", "px auth configure"); err != nil {
		return err
	}
	if err := ensureFileExists(srcKey, "client key", "px auth configure"); err != nil {
		return err
	}
	if err := ensureFileExists(srcCA, "CA certificate", "px auth configure"); err != nil {
		return err
	}

	destCertDir := filepath.Join(home, ".px-plugin", "certs")
	if err := os.MkdirAll(destCertDir, 0o700); err != nil {
		return fmt.Errorf("ensure cert dir: %w", err)
	}

	destCert := filepath.Join(destCertDir, "client.crt")
	destKey := filepath.Join(destCertDir, "client.key")
	destCA := filepath.Join(destCertDir, "ca.crt")

	if err := copyFileSecure(srcCert, destCert); err != nil {
		return fmt.Errorf("copy client certificate: %w", err)
	}
	if err := copyFileSecure(srcKey, destKey); err != nil {
		return fmt.Errorf("copy client key: %w", err)
	}
	if err := copyFileSecure(srcCA, destCA); err != nil {
		return fmt.Errorf("copy CA certificate: %w", err)
	}

	devAPIBase := resolveDevAPIBaseWithPrefix(opts.DevAPI, entryPath)
	defaultBase := config.DefaultConfig().DevAPI.BaseURL
	cfgPath := config.DefaultConfigPath()
	cfg := config.DefaultConfig()
	if existing, err := config.LoadConfigFile(cfgPath); err == nil {
		cfg = existing
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load existing config: %w", err)
	}

	if opts.DevAPI != "" || cfg.DevAPI.BaseURL == "" || cfg.DevAPI.BaseURL == defaultBase {
		cfg.DevAPI.BaseURL = devAPIBase
	}
	if opts.Tenant != "" {
		cfg.Dev.Tenant = opts.Tenant
	}
	if cfg.Dev.EntryPath == "" && opts.Entry != "" {
		cfg.Dev.EntryPath = opts.Entry
	}

	token := opts.DevAPIToken
	if token == "" {
		token = resolveDevAPIToken(opts, devAPIBase)
	}
	if token != "" {
		cfg.DevAPI.APIKey = token
	}

	defaultIgnore := []string{".git/**", "node_modules/**", ".nuxt/**", ".output/**", ".px-plugin/**"}
	cfg.Dev.Ignore = appendUniqueStrings(cfg.Dev.Ignore, defaultIgnore)

	cfg.DevAPI.EnableMTLS = true
	cfg.DevAPI.CertPath = destCert
	cfg.DevAPI.KeyPath = destKey
	cfg.DevAPI.CACertPath = destCA

	cfg.Security.EnableMTLS = true
	cfg.Security.CertDir = destCertDir
	if opts.MTLSSkipVerify {
		cfg.Security.InsecureSkipVerify = true
	}
	if cfg.Security.RotationCheck == 0 {
		cfg.Security.RotationCheck = 5
	}
	if cfg.Security.AutoRotate == false {
		cfg.Security.AutoRotate = true
	}

	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Printf("mTLS assets copied to %s\n", destCertDir)
	fmt.Printf("Config written to %s\n", cfgPath)
	fmt.Println("Run 'px-plugin doctor --check-mtls' to verify, or start dev with --watch/--once.")
	return nil
}

func ensureFileExists(path, label, remediation string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s not found at %s (remediation: %s): %w", label, path, remediation, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s path is a directory: %s", label, path)
	}
	return nil
}

func copyFileSecure(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o600)
}

const (
	defaultDevAPIBase = "http://127.0.0.1:8077"
	defaultAPIPrefix  = "/api/v1"
)

// runDevWatch implements the watch mode
func runDevWatch(opts *DevOptions) error {
	auditLogger := audit.NewLogger()

	entryPath, err := filepath.Abs(opts.Entry)
	if err != nil {
		return fmt.Errorf("resolve entry path: %w", err)
	}
	if _, err := os.Stat(entryPath); err != nil {
		return fmt.Errorf("entry path does not exist: %w", err)
	}

	m, err := manifest.Load(entryPath)
	if err != nil {
		return fmt.Errorf("load plugin manifest: %w", err)
	}

	devAPIBase := resolveDevAPIBaseWithPrefix(opts.DevAPI, entryPath)
	applyPowerXCredentialDefaults(opts, devAPIBase)
	fmt.Printf("Starting dev watch mode\n  Entry: %s\n  Plugin: %s@%s\n  Dev API: %s\n", entryPath, m.ID, m.Version, devAPIBase)
	if opts.Tenant != "" {
		fmt.Printf("  Tenant: %s\n", opts.Tenant)
	}
	if opts.MaxProcs > 0 {
		prev := runtime.GOMAXPROCS(opts.MaxProcs)
		fmt.Printf("  Max Procs: %d (prev %d)\n", opts.MaxProcs, prev)
	}
	if opts.MaxMemoryMB > 0 || opts.MaxCPUPercent > 0 || opts.MaxWatchFiles > 0 {
		fmt.Printf("  Limits:  Memory≤%dMB  CPU≤%d%%  Watch≤%d files\n",
			opts.MaxMemoryMB, opts.MaxCPUPercent, opts.MaxWatchFiles)
	}

	mtlsClient, err := resolveMTLSClient(opts, devAPIBase)
	if err != nil {
		return err
	}
	if mtlsClient != nil {
		defer mtlsClient.Close()
	}

	manager := session.NewManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived stop signal, cleaning up...")
		cancel()
	}()

	apiToken := resolveDevAPIToken(opts, devAPIBase)
	client := devapi.NewClient(devapi.ClientOptions{
		BaseURL:    devAPIBase,
		APIKey:     apiToken,
		Timeout:    30 * time.Second,
		MTLSClient: mtlsClient,
	})
	defer client.Close()

	watcher, err := newFileWatcher(entryPath, opts.Ignore, opts.MaxWatchFiles)
	if err != nil {
		return err
	}
	builder := build.NewSimpleBuilder()
	resourceMonitor := resourceMonitorFromOptions(opts)

	runner, err := devwatch.NewRunner(devwatch.RunnerOptions{
		EntryPath:   entryPath,
		Tenant:      opts.Tenant,
		TenantID:    resolveTenantID(opts),
		DeveloperID: resolveDeveloperID(opts),
		DevAPIBase:  devAPIBase,
		APIToken:    apiToken,
		Manifest:    m,
		BuildDir:    filepath.Join(entryPath, ".px-plugin", "build"),
		CommandName: "dev --watch",
		Resources:   resourceMonitor,
	}, devwatch.Dependencies{
		Builder:        builder,
		Watcher:        watcher,
		Client:         client,
		AuditLogger:    auditLogger,
		SessionManager: manager,
	})
	if err != nil {
		return err
	}

	return runner.Run(ctx)
}

// runDevOnce performs a single build + reload without starting a file watcher.
func runDevOnce(opts *DevOptions) error {
	auditLogger := audit.NewLogger()

	entryPath, err := filepath.Abs(opts.Entry)
	if err != nil {
		return fmt.Errorf("resolve entry path: %w", err)
	}
	if _, err := os.Stat(entryPath); err != nil {
		return fmt.Errorf("entry path does not exist: %w", err)
	}

	m, err := manifest.Load(entryPath)
	if err != nil {
		return fmt.Errorf("load plugin manifest: %w", err)
	}

	devAPIBase := resolveDevAPIBaseWithPrefix(opts.DevAPI, entryPath)
	applyPowerXCredentialDefaults(opts, devAPIBase)
	fmt.Printf("Starting dev (single run mode)\n  Entry: %s\n  Plugin: %s@%s\n  Dev API: %s\n", entryPath, m.ID, m.Version, devAPIBase)
	if opts.Tenant != "" {
		fmt.Printf("  Tenant: %s\n", opts.Tenant)
	}
	if opts.MaxProcs > 0 {
		prev := runtime.GOMAXPROCS(opts.MaxProcs)
		fmt.Printf("  Max Procs: %d (prev %d)\n", opts.MaxProcs, prev)
	}

	mtlsClient, err := resolveMTLSClient(opts, devAPIBase)
	if err != nil {
		return err
	}
	if mtlsClient != nil {
		defer mtlsClient.Close()
	}

	manager := session.NewManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived stop signal, cleaning up...")
		cancel()
	}()

	apiToken := resolveDevAPIToken(opts, devAPIBase)
	client := devapi.NewClient(devapi.ClientOptions{
		BaseURL:    devAPIBase,
		APIKey:     apiToken,
		Timeout:    30 * time.Second,
		MTLSClient: mtlsClient,
	})
	defer client.Close()

	builder := build.NewSimpleBuilder()
	resourceMonitor := resourceMonitorFromOptions(opts)

	runner, err := devwatch.NewRunner(devwatch.RunnerOptions{
		EntryPath:   entryPath,
		Tenant:      opts.Tenant,
		TenantID:    resolveTenantID(opts),
		DeveloperID: resolveDeveloperID(opts),
		DevAPIBase:  devAPIBase,
		APIToken:    apiToken,
		Manifest:    m,
		BuildDir:    filepath.Join(entryPath, ".px-plugin", "build"),
		CommandName: "dev --once",
		Resources:   resourceMonitor,
		Mode:        devwatch.ModeSingle,
	}, devwatch.Dependencies{
		Builder:        builder,
		Watcher:        nil,
		Client:         client,
		AuditLogger:    auditLogger,
		SessionManager: manager,
	})
	if err != nil {
		return err
	}

	return runner.Run(ctx)
}

// runDevListSessions lists remote Dev API sessions.
func runDevListSessions(opts *DevOptions) error {
	devAPIBase := resolveDevAPIBaseWithPrefix(opts.DevAPI, opts.Entry)
	mtlsClient, err := resolveMTLSClient(opts, devAPIBase)
	if err != nil {
		return fmt.Errorf("failed to initialize mTLS client: %w", err)
	}
	if mtlsClient != nil {
		defer mtlsClient.Close()
	}

	client := devapi.NewClient(devapi.ClientOptions{
		BaseURL:    devAPIBase,
		APIKey:     resolveDevAPIToken(opts, devAPIBase),
		Timeout:    15 * time.Second,
		MTLSClient: mtlsClient,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := &devapi.ListSessionsFilter{}
	if pid := detectPluginID(opts); pid != "" {
		filter.PluginID = pid
	}
	if opts.TenantID != 0 {
		filter.TenantID = opts.TenantID
	}
	if opts.DeveloperID != 0 {
		filter.DeveloperID = opts.DeveloperID
	}
	statusFilter := strings.ToLower(strings.TrimSpace(opts.ListStatus))
	if statusFilter == "" {
		statusFilter = "active"
	}
	switch statusFilter {
	case "active", "pending", "terminated", "all":
	default:
		fmt.Printf("Unknown list-status %q, defaulting to active\n", opts.ListStatus)
		statusFilter = "active"
	}
	if statusFilter != "active" && statusFilter != "all" {
		filter.Status = statusFilter
	}

	sessions, err := client.ListSessions(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list remote sessions: %w", err)
	}
	sessions = filterSessionsByStatus(sessions, statusFilter)

	if len(sessions) == 0 {
		fmt.Println("No remote sessions found.")
		if statusFilter == "active" {
			fmt.Println("Use --list-status all to include terminated sessions.")
		}
		return nil
	}

	fmt.Printf("Remote sessions (%d):\n\n", len(sessions))
	for _, s := range sessions {
		fmt.Printf("  Session:  %s\n", s.SessionID)
		fmt.Printf("  Plugin:   %s@%s\n", s.PluginID, s.Version)
		if s.Tenant != "" || s.TenantID != 0 {
			label := s.Tenant
			if label == "" {
				label = fmt.Sprintf("#%d", s.TenantID)
			}
			fmt.Printf("  Tenant:   %s\n", label)
		}
		if !s.CreatedAt.IsZero() {
			fmt.Printf("  Created:  %s\n", s.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		if !s.LastReload.IsZero() {
			fmt.Printf("  Reloaded: %s\n", s.LastReload.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("  Status:   %s\n", s.Status)
		fmt.Println()
	}

	fmt.Println("Use 'px-plugin dev --force-stop <session-id>' to delete a session or rerun dev once it is cleared.")
	return nil
}

// runDevResumeSession resumes a session
func runDevResumeSession(sessionID string, opts *DevOptions) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	entryPath, err := filepath.Abs(opts.Entry)
	if err != nil {
		return fmt.Errorf("resolve entry path: %w", err)
	}
	if _, err := os.Stat(entryPath); err != nil {
		return fmt.Errorf("entry path does not exist: %w", err)
	}

	m, err := manifest.Load(entryPath)
	if err != nil {
		return fmt.Errorf("load plugin manifest: %w", err)
	}

	devAPIBase := resolveDevAPIBaseWithPrefix(opts.DevAPI, entryPath)
	applyPowerXCredentialDefaults(opts, devAPIBase)
	fmt.Printf("Resuming dev watch\n  Entry: %s\n  Plugin: %s@%s\n  Dev API: %s\n", entryPath, m.ID, m.Version, devAPIBase)

	mtlsClient, err := resolveMTLSClient(opts, devAPIBase)
	if err != nil {
		return err
	}
	if mtlsClient != nil {
		defer mtlsClient.Close()
	}

	apiToken := resolveDevAPIToken(opts, devAPIBase)
	client := devapi.NewClient(devapi.ClientOptions{
		BaseURL:    devAPIBase,
		APIKey:     apiToken,
		Timeout:    30 * time.Second,
		MTLSClient: mtlsClient,
	})
	ctxLookup, cancelLookup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelLookup()
	sessions, err := client.ListSessions(ctxLookup, &devapi.ListSessionsFilter{SessionID: sessionID})
	if err != nil {
		return fmt.Errorf("failed to inspect remote session %s: %w", sessionID, err)
	}
	if len(sessions) == 0 {
		return fmt.Errorf("session %s not found on Dev API", sessionID)
	}
	info := sessions[0]
	if info.ReloadToken == "" {
		return fmt.Errorf("session %s is missing reload token; please stop it via Dev API and rerun dev", sessionID)
	}
	if info.PluginID != "" && info.PluginID != m.ID {
		fmt.Printf("Warning: remote session pluginId=%s does not match manifest=%s\n", info.PluginID, m.ID)
	}
	if opts.Tenant == "" && info.Tenant != "" {
		opts.Tenant = info.Tenant
	}
	if opts.TenantID == 0 && info.TenantID != 0 {
		opts.TenantID = info.TenantID
	}
	if opts.DeveloperID == 0 && info.DeveloperID != 0 {
		opts.DeveloperID = info.DeveloperID
	}

	client.SetReloadToken(info.ReloadToken)

	auditLogger := audit.NewLogger()
	manager := session.NewManager()
	resourceMonitor := resourceMonitorFromOptions(opts)

	var watcher *watch.FileWatcher
	var runnerMode devwatch.RunnerMode = devwatch.ModeSingle
	if opts.Watch {
		var watchErr error
		watcher, watchErr = newFileWatcher(entryPath, opts.Ignore, opts.MaxWatchFiles)
		if watchErr != nil {
			return watchErr
		}
		runnerMode = devwatch.ModeWatch
	}
	builder := build.NewSimpleBuilder()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived stop signal, cleaning up...")
		cancel()
	}()

	runner, err := devwatch.NewRunner(devwatch.RunnerOptions{
		EntryPath:           entryPath,
		Tenant:              opts.Tenant,
		TenantID:            resolveTenantID(opts),
		DeveloperID:         resolveDeveloperID(opts),
		DevAPIBase:          devAPIBase,
		APIToken:            apiToken,
		Manifest:            m,
		BuildDir:            filepath.Join(entryPath, ".px-plugin", "build"),
		CommandName:         "dev --resume",
		Resources:           resourceMonitor,
		Mode:                runnerMode,
		UseExistingSession:  true,
		ExistingSessionID:   sessionID,
		ExistingReloadToken: info.ReloadToken,
	}, devwatch.Dependencies{
		Builder:        builder,
		Watcher:        watcher,
		Client:         client,
		AuditLogger:    auditLogger,
		SessionManager: manager,
	})
	if err != nil {
		return err
	}

	fmt.Printf("  Session: %s (tenant=%s status=%s)\n", sessionID, info.Tenant, info.Status)
	return runner.Run(ctx)
}

// runDevStopSession stops a session
func runDevStopSession(sessionID string, opts *DevOptions) error {
	startTime := time.Now()
	auditLogger := audit.NewLogger()
	manager := session.NewManager()

	// Get the session first to confirm it exists
	s, err := manager.GetSession(sessionID)
	if err != nil {
		auditLogger.Log(audit.EventSessionStop, sessionID, "", "", "", "", "dev --stop", false, 0, err)
		return fmt.Errorf("failed to get session %s: %w", sessionID, err)
	}

	devAPIHint := s.DevAPIURL
	if devAPIHint == "" {
		devAPIHint = opts.DevAPI
	}
	if devAPIHint == "" {
		if creds, _ := loadPowerXCredentials(); creds != nil && creds.APIBase != "" {
			devAPIHint = creds.APIBase
		}
	}
	if opts.DevAPI == "" && devAPIHint != "" {
		opts.DevAPI = devAPIHint
	}
	devAPIBase := resolveDevAPIBaseWithPrefix(devAPIHint, s.EntryPath)
	mtlsClient, err := resolveMTLSClient(opts, devAPIBase)
	if err != nil {
		fmt.Printf("Warning: failed to initialize mTLS client: %v\n", err)
	}

	client := devapi.NewClient(devapi.ClientOptions{
		BaseURL:    devAPIBase,
		APIKey:     resolveDevAPIToken(opts, devAPIBase),
		MTLSClient: mtlsClient,
	})
	if mtlsClient != nil {
		defer mtlsClient.Close()
	}

	apiToken := strings.TrimSpace(resolveDevAPIToken(opts, devAPIBase))
	reloadToken := strings.TrimSpace(s.ReloadToken)
	// Prefer API token for stop operations; only fall back to reload token if it looks like a JWT.
	if apiToken != "" {
		// API token will be used via APIKey; avoid setting reload token to a non-JWT string that backend rejects.
		client.SetReloadToken("")
	} else if reloadToken != "" && strings.Count(reloadToken, ".") == 2 {
		client.SetReloadToken(reloadToken)
	}

	if s.SessionID != "" {
		if err := client.Delete(context.Background(), s.SessionID); err != nil {
			var apiErr *devapi.DevAPIError
			if errors.As(err, &apiErr) && apiErr.Original != nil && strings.Contains(apiErr.Original.Error(), "status 404") {
				fmt.Println("Dev API session already removed remotely (404).")
			} else {
				fmt.Printf("Warning: failed to delete Dev API session: %v\n", err)
			}
		}
	}

	err = manager.StopSession(sessionID)
	if err != nil {
		duration := time.Since(startTime).Milliseconds()
		auditLogger.Log(audit.EventSessionStop, sessionID, s.PluginID, s.Version, s.Tenant, s.EntryPath, "dev --stop", false, duration, err)
		return fmt.Errorf("failed to stop session: %w", err)
	}
	// remove from local store to avoid lingering list entries
	_ = manager.DeleteSession(sessionID)

	// Log successful stop
	duration := time.Since(startTime).Milliseconds()
	auditLogger.Log(audit.EventSessionStop, sessionID, s.PluginID, s.Version, s.Tenant, s.EntryPath, "dev --stop", true, duration, nil)

	// Display session info
	fmt.Printf("Session stopped: %s\n", sessionID)
	fmt.Printf("  Plugin:    %s v%s\n", s.PluginID, s.Version)
	fmt.Printf("  Path:      %s\n", s.EntryPath)
	fmt.Printf("  Status:    %s\n", s.Status)

	// Note: In full implementation, would also:
	// 1. Notify Dev API that session is stopped
	// 2. Clean up file watcher resources
	// 3. Close any open connections

	return nil
}

// runDevForceStopSession attempts to delete a remote Dev API session directly.
func runDevForceStopSession(remoteSessionID string, opts *DevOptions) error {
	if strings.TrimSpace(remoteSessionID) == "" {
		return fmt.Errorf("remote session id is required")
	}

	devAPIBase := resolveDevAPIBaseWithPrefix(opts.DevAPI, opts.Entry)
	mtlsClient, err := resolveMTLSClient(opts, devAPIBase)
	if err != nil {
		return fmt.Errorf("failed to initialize mTLS client: %w", err)
	}
	if mtlsClient != nil {
		defer mtlsClient.Close()
	}

	client := devapi.NewClient(devapi.ClientOptions{
		BaseURL:    devAPIBase,
		APIKey:     resolveDevAPIToken(opts, devAPIBase),
		MTLSClient: mtlsClient,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Delete(ctx, remoteSessionID); err != nil {
		return fmt.Errorf("failed to delete remote session %s: %w", remoteSessionID, err)
	}

	fmt.Printf("Remote session deleted: %s\n", remoteSessionID)
	return nil
}

// runDevDeleteSession deletes a remote session and removes its local cache entry.
func runDevDeleteSession(sessionID string, opts *DevOptions) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	devAPIBase := resolveDevAPIBaseWithPrefix(opts.DevAPI, opts.Entry)
	mtlsClient, err := resolveMTLSClient(opts, devAPIBase)
	if err != nil {
		return fmt.Errorf("failed to initialize mTLS client: %w", err)
	}
	if mtlsClient != nil {
		defer mtlsClient.Close()
	}

	client := devapi.NewClient(devapi.ClientOptions{
		BaseURL:    devAPIBase,
		APIKey:     resolveDevAPIToken(opts, devAPIBase),
		MTLSClient: mtlsClient,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete remote session %s: %w", sessionID, err)
	}
	if _, err := client.DeleteSessions(ctx, &devapi.DeleteSessionsRequest{
		SessionID: sessionID,
		Status:    "all",
		Force:     true,
		Confirm:   true,
	}); err != nil {
		fmt.Printf("Warning: failed to purge session record %s: %v\n", sessionID, err)
	}

	manager := session.NewManager()
	removeLocalSessions(manager, []string{sessionID})
	fmt.Printf("Remote session deleted and local cache removed: %s\n", sessionID)
	return nil
}

// runDevClearSessions clears remote sessions (terminated by default) and removes local cache entries.
func runDevClearSessions(opts *DevOptions) error {
	entry := opts.Entry
	if entry == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("determine working directory: %w", err)
		}
		entry = cwd
	}
	if entry == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("determine working directory: %w", err)
		}
		entry = cwd
	}
	entryPath, err := filepath.Abs(entry)
	if err != nil {
		return fmt.Errorf("resolve entry path: %w", err)
	}
	manifest, err := manifest.Load(entryPath)
	if err != nil {
		return fmt.Errorf("load plugin manifest: %w", err)
	}

	devAPIBase := resolveDevAPIBaseWithPrefix(opts.DevAPI, opts.Entry)
	mtlsClient, err := resolveMTLSClient(opts, devAPIBase)
	if err != nil {
		return fmt.Errorf("failed to initialize mTLS client: %w", err)
	}
	if mtlsClient != nil {
		defer mtlsClient.Close()
	}

	client := devapi.NewClient(devapi.ClientOptions{
		BaseURL:    devAPIBase,
		APIKey:     resolveDevAPIToken(opts, devAPIBase),
		MTLSClient: mtlsClient,
	})

	req := &devapi.DeleteSessionsRequest{
		PluginID: manifest.ID,
		Force:    opts.ClearSessionsForce,
	}
	if opts.ClearSessionsForce {
		req.Status = "all"
		req.Confirm = true
	} else {
		req.Status = "terminated"
	}
	if tenantID := resolveTenantID(opts); tenantID != 0 {
		req.TenantID = tenantID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.DeleteSessions(ctx, req)
	if err != nil {
		return fmt.Errorf("clear sessions failed: %w", err)
	}

	manager := session.NewManager()
	removeLocalSessions(manager, resp.SessionIDs)

	deleted := resp.Deleted
	if deleted == 0 {
		deleted = len(resp.SessionIDs)
	}
	fmt.Printf("Cleared %d remote sessions (force=%v)\n", deleted, resp.Force)
	if resp.Deleted <= 0 && len(resp.SessionIDs) > 0 {
		fmt.Printf("  (Dev API returned sessionIds=%d)\n", len(resp.SessionIDs))
	}
	return nil
}

// removeLocalSessions removes local cache files matching remote session IDs.
func removeLocalSessions(manager *session.Manager, remoteIDs []string) {
	if len(remoteIDs) == 0 {
		return
	}
	remoteSet := make(map[string]struct{}, len(remoteIDs))
	for _, id := range remoteIDs {
		if id == "" {
			continue
		}
		remoteSet[id] = struct{}{}
	}

	sessions, err := manager.ListSessions()
	if err != nil {
		return
	}
	for _, s := range sessions {
		if s == nil {
			continue
		}
		if _, ok := remoteSet[s.ID]; ok {
			_ = manager.DeleteSession(s.ID)
			continue
		}
		if s.SessionID != "" {
			if _, ok := remoteSet[s.SessionID]; ok {
				_ = manager.DeleteSession(s.ID)
			}
		}
	}
}

// runDevShowLogs shows logs for a session with SSE streaming
func runDevShowLogs(sessionID string, opts *DevOptions) error {
	startTime := time.Now()
	auditLogger := audit.NewLogger()
	manager := session.NewManager()

	// Get the session
	s, err := manager.GetSession(sessionID)
	if err != nil {
		auditLogger.Log(audit.EventSessionLogs, sessionID, "", "", "", "", "dev --logs", false, 0, err)
		return fmt.Errorf("failed to get session %s: %w", sessionID, err)
	}

	// Display session info
	fmt.Printf("Streaming logs for session: %s\n", sessionID)
	fmt.Printf("  Plugin:    %s v%s\n", s.PluginID, s.Version)
	fmt.Printf("  Status:    %s\n", s.Status)
	fmt.Println()

	// Check if session is active
	if s.Status != session.StatusActive {
		fmt.Println("Warning: Session is not active. Logs may not be available.")
	}

	devAPIURL := resolveDevAPIBaseWithPrefix(s.DevAPIURL, s.EntryPath)

	mtlsClient, err := resolveMTLSClient(opts, devAPIURL)
	if err != nil {
		return err
	}
	if mtlsClient != nil {
		defer mtlsClient.Close()
	}

	// Create headers for authentication
	headers := make(map[string]string)
	if s.ReloadToken != "" {
		headers["Authorization"] = "Bearer " + s.ReloadToken
	}

	// Create SSE client
	sseOpts := sse.DefaultClientOptions()
	sseOpts.BaseURL = devAPIURL
	sseOpts.Headers = headers
	if mtlsClient != nil {
		sseOpts.MTLSEnabled = true
		sseOpts.MTLSClient = mtlsClient
	}

	sseClient, err := sse.NewClient(sseOpts)
	if err != nil {
		duration := time.Since(startTime).Milliseconds()
		auditLogger.Log(audit.EventSessionLogs, sessionID, s.PluginID, s.Version, s.Tenant, s.EntryPath, "dev --logs", false, duration, err)
		return fmt.Errorf("failed to create SSE client: %w", err)
	}
	defer sseClient.Close()

	// Set up output handler
	outputConfig := sse.DefaultOutputConfig()
	outputConfig.FilterBySessionID = sessionID
	outputConfig.ConsoleOutput = true
	outputConfig.MinLevel = normalizeLogLevel(opts.LogsLevel)
	outputConfig.DisableColor = opts.NoColor

	if opts.LogsFile != "" {
		if err := os.MkdirAll(filepath.Dir(opts.LogsFile), 0o755); err != nil {
			return fmt.Errorf("create logs directory: %w", err)
		}
		outputConfig.FileOutput = true
		outputConfig.LogFilePath = opts.LogsFile
	}

	output, err := sse.NewOutput(outputConfig)
	if err != nil {
		duration := time.Since(startTime).Milliseconds()
		auditLogger.Log(audit.EventSessionLogs, sessionID, s.PluginID, s.Version, s.Tenant, s.EntryPath, "dev --logs", false, duration, err)
		return fmt.Errorf("failed to create output handler: %w", err)
	}
	defer output.Close()

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Start SSE connection
	logsPath := fmt.Sprintf("/api/v1/dev/%s/logs", sessionID)
	err = sseClient.Connect(ctx, logsPath)
	if err != nil {
		duration := time.Since(startTime).Milliseconds()
		auditLogger.Log(audit.EventSessionLogs, sessionID, s.PluginID, s.Version, s.Tenant, s.EntryPath, "dev --logs", false, duration, err)
		fmt.Fprintf(os.Stderr, "Failed to connect to log stream: %v\n", err)
		return err
	}

	fmt.Println("Connected to log stream. Press Ctrl+C to stop.")
	fmt.Println()

	// Create channels for events and errors
	eventCh := sseClient.EventChan()
	errorCh := sseClient.ErrorChan()

	// Display session metrics
	fmt.Println("Session Metrics:")
	fmt.Printf("  Total Reloads:     %d\n", s.Metrics.ReloadCount)
	fmt.Printf("  Avg Reload Time:   %.0fms\n", s.Metrics.AvgReloadTime)
	fmt.Printf("  Total Reload Time: %dms\n", s.Metrics.TotalReloadTime)
	fmt.Printf("  Success Rate:      %.1f%%\n", s.Metrics.SuccessRate*100)
	if s.Metrics.LastError != "" {
		fmt.Printf("  Last Error:        %s\n", s.Metrics.LastError)
	}
	fmt.Println()
	fmt.Println("--- Log Stream ---")
	fmt.Println()

	// Log successful connection
	duration := time.Since(startTime).Milliseconds()
	auditLogger.Log(audit.EventSessionLogs, sessionID, s.PluginID, s.Version, s.Tenant, s.EntryPath, "dev --logs", true, duration, nil)

	// Stream logs
	for {
		select {
		case event := <-eventCh:
			output.WriteEvent(event)

		case err := <-errorCh:
			fmt.Fprintf(os.Stderr, "Log stream error: %v\n", err)
			return err

		case <-ctx.Done():
			fmt.Println("\nStopping log stream...")
			return nil
		}
	}
}

// StringSliceFlag implements flag.Value for string slices
type StringSliceFlag []string

func (s *StringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *StringSliceFlag) String() string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%v", []string(*s))
}

func applyDevDefaults(opts *DevOptions) *config.Config {
	cfg := loadUserConfig()
	if cfg != nil {
		if opts.Entry == "" && cfg.Dev.EntryPath != "" {
			opts.Entry = cfg.Dev.EntryPath
		}
		if opts.Tenant == "" && cfg.Dev.Tenant != "" {
			opts.Tenant = cfg.Dev.Tenant
		}
		if opts.DevAPI == "" && cfg.DevAPI.BaseURL != "" {
			opts.DevAPI = cfg.DevAPI.BaseURL
		}
		if opts.DevAPIToken == "" && cfg.DevAPI.APIKey != "" {
			opts.DevAPIToken = cfg.DevAPI.APIKey
		}
		if len(cfg.Dev.Ignore) > 0 {
			opts.Ignore = appendUniqueStrings(cfg.Dev.Ignore, opts.Ignore)
		}
		if opts.MTLSCert == "" && cfg.DevAPI.CertPath != "" {
			opts.MTLSCert = cfg.DevAPI.CertPath
		}
		if opts.MTLSKey == "" && cfg.DevAPI.KeyPath != "" {
			opts.MTLSKey = cfg.DevAPI.KeyPath
		}
		if opts.MTLSCA == "" && cfg.DevAPI.CACertPath != "" {
			opts.MTLSCA = cfg.DevAPI.CACertPath
		}
		if !opts.MTLSSkipVerify && cfg.Security.InsecureSkipVerify {
			opts.MTLSSkipVerify = true
		}
		if opts.MaxProcs == 0 && cfg.Performance.MaxConcurrency > 0 {
			opts.MaxProcs = cfg.Performance.MaxConcurrency
		}
		if opts.MaxMemoryMB == 0 && cfg.Performance.MemoryLimit > 0 {
			opts.MaxMemoryMB = int((cfg.Performance.MemoryLimit + (1024*1024 - 1)) / (1024 * 1024))
		}
		if opts.MaxCPUPercent == 0 && cfg.Performance.CPUThreshold > 0 {
			opts.MaxCPUPercent = cfg.Performance.CPUThreshold
		}
		if opts.MaxWatchFiles == 0 && cfg.Watch.MaxFiles > 0 {
			opts.MaxWatchFiles = cfg.Watch.MaxFiles
		}
		if opts.MTLSServerName == "" && cfg.DevAPI.BaseURL != "" {
			if host := hostFromURL(cfg.DevAPI.BaseURL); host != "" {
				opts.MTLSServerName = host
			}
		}
	}
	if opts.Tenant == "" {
		if env := os.Getenv("PX_DEV_TENANT"); env != "" {
			opts.Tenant = env
		}
	}
	if opts.MaxProcs == 0 {
		opts.MaxProcs = envIntDefault("PX_MAX_PROCS", 0)
	}
	if opts.MaxMemoryMB == 0 {
		opts.MaxMemoryMB = envIntDefault("PX_RESOURCE_MEMORY_MB", 100)
	}
	if opts.MaxCPUPercent == 0 {
		opts.MaxCPUPercent = envIntDefault("PX_RESOURCE_CPU_THRESHOLD", 10)
	}
	if opts.MaxWatchFiles == 0 {
		opts.MaxWatchFiles = envIntDefault("PX_MAX_WATCH_FILES", 20000)
	}
	return cfg
}

func loadUserConfig() *config.Config {
	path := config.DefaultConfigPath()
	cfg, err := config.LoadConfigFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", path, err)
		return nil
	}
	return cfg
}

func appendUniqueStrings(prefix []string, suffix []string) []string {
	if len(prefix) == 0 {
		return append([]string(nil), suffix...)
	}
	result := make([]string, 0, len(prefix)+len(suffix))
	seen := make(map[string]struct{}, len(prefix)+len(suffix))
	for _, v := range prefix {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	for _, v := range suffix {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}

func hostFromURL(raw string) string {
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func envIntDefault(key string, def int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return def
}

func resourceMonitorFromOptions(opts *DevOptions) *resources.ResourceMonitor {
	monitor := resources.NewResourceMonitor()

	memLimit := opts.MaxMemoryMB
	if memLimit <= 0 {
		memLimit = 100
	}
	memBytes := int64(memLimit) * 1024 * 1024
	monitor.SetLimit(resources.Limit{
		Type:      resources.Memory,
		Value:     memBytes,
		Unit:      "bytes",
		Threshold: 90,
	})

	cpuThreshold := opts.MaxCPUPercent
	if cpuThreshold <= 0 {
		cpuThreshold = 10
	}
	monitor.SetLimit(resources.Limit{
		Type:      resources.CPU,
		Value:     100,
		Unit:      "percent",
		Threshold: float64(cpuThreshold),
	})

	return monitor
}

func resolveDevAPIBaseWithPrefix(flagVal, entryPath string) string {
	base := resolveDevAPIBase(flagVal)
	prefix := deriveAPIPrefix(entryPath)
	urlWithPrefix := applyAPIPrefix(base, prefix)
	if needsPrefix(urlWithPrefix) {
		urlWithPrefix = applyAPIPrefix(base, defaultAPIPrefix)
	}
	return urlWithPrefix
}

func resolveDevAPIBase(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if env := os.Getenv("PX_DEV_API_BASE"); env != "" {
		return env
	}
	return defaultDevAPIBase
}

func deriveAPIPrefix(entryPath string) string {
	if entryPath == "" {
		return defaultAPIPrefix
	}
	configPath := filepath.Join(entryPath, "backend", "etc", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultAPIPrefix
	}
	var file struct {
		Server struct {
			APIPrefix string `yaml:"api_prefix"`
		} `yaml:"server"`
	}
	if err := yaml.Unmarshal(data, &file); err != nil {
		return defaultAPIPrefix
	}
	prefix := strings.TrimSpace(file.Server.APIPrefix)
	if prefix == "" {
		return defaultAPIPrefix
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return prefix
}

func applyAPIPrefix(baseURL, prefix string) string {
	if baseURL == "" || prefix == "" {
		return baseURL
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	if u.Path != "" && u.Path != "/" {
		return baseURL
	}
	u.Path = prefix
	return u.String()
}

func needsPrefix(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Path == "" || u.Path == "/"
}

func resolveDevAPIToken(opts *DevOptions, devAPIBase string) string {
	if opts.DevAPIToken != "" {
		return opts.DevAPIToken
	}
	if env := os.Getenv("PX_DEV_API_TOKEN"); env != "" {
		return env
	}
	if tok := tokenFromPowerXCredentials(devAPIBase); tok != "" {
		return tok
	}
	return ""
}

func detectPluginID(opts *DevOptions) string {
	entry := opts.Entry
	if entry == "" && opts.userConfig != nil && opts.userConfig.Dev.EntryPath != "" {
		entry = opts.userConfig.Dev.EntryPath
	}
	if entry == "" {
		if cwd, err := os.Getwd(); err == nil {
			entry = cwd
		}
	}
	if entry == "" {
		return ""
	}
	entryPath, err := filepath.Abs(entry)
	if err != nil {
		return ""
	}
	if m, err := manifest.Load(entryPath); err == nil {
		return m.ID
	}
	return ""
}

func filterSessionsByStatus(sessions []devapi.SessionRecord, filter string) []devapi.SessionRecord {
	if filter == "all" {
		return sessions
	}
	filtered := make([]devapi.SessionRecord, 0, len(sessions))
	for _, s := range sessions {
		status := strings.ToLower(strings.TrimSpace(s.Status))
		if filter == "active" {
			if status == "terminated" {
				continue
			}
			filtered = append(filtered, s)
			continue
		}
		if status == filter {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func resolveTenantID(opts *DevOptions) uint64 {
	if opts.TenantID != 0 {
		return opts.TenantID
	}
	if env := os.Getenv("PX_DEV_TENANT_ID"); env != "" {
		if v, err := strconv.ParseUint(env, 10, 64); err == nil {
			return v
		}
	}
	if creds, _ := loadPowerXCredentials(); creds != nil && creds.TenantID > 0 {
		return creds.TenantID
	}
	return 0
}

func resolveDeveloperID(opts *DevOptions) uint64 {
	if opts.DeveloperID != 0 {
		return opts.DeveloperID
	}
	if env := os.Getenv("PX_DEV_DEVELOPER_ID"); env != "" {
		if v, err := strconv.ParseUint(env, 10, 64); err == nil {
			return v
		}
	}
	if creds, _ := loadPowerXCredentials(); creds != nil && creds.DeveloperID > 0 {
		return creds.DeveloperID
	}
	return 0
}

// tokenFromPowerXCredentials tries to read ~/.powerx/credentials.json and return access_token
// when the api base matches (prefix match) the current dev-api base.
func tokenFromPowerXCredentials(devAPIBase string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".powerx", "credentials.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var creds struct {
		API         string `json:"api"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return ""
	}
	return strings.TrimSpace(creds.AccessToken)
}

type powerXCredentials struct {
	APIBase     string
	AccessToken string
	TenantID    uint64
	DeveloperID uint64
}

func loadPowerXCredentials() (*powerXCredentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".powerx", "credentials.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var file struct {
		API         string `json:"api"`
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	token := strings.TrimSpace(file.AccessToken)
	if token == "" {
		return nil, nil
	}
	claims, err := decodePowerXClaims(token)
	if err != nil {
		return nil, err
	}
	return &powerXCredentials{
		APIBase:     strings.TrimSpace(file.API),
		AccessToken: token,
		TenantID:    claims.TenantID,
		DeveloperID: claims.DeveloperID,
	}, nil
}

type powerXClaims struct {
	TenantID    uint64
	DeveloperID uint64
}

func decodePowerXClaims(token string) (*powerXClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid token format")
	}
	payload := parts[1]
	payload += strings.Repeat("=", (4-len(payload)%4)%4)
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}
	var raw struct {
		TenantNumber float64 `json:"tid_n"`
		MemberNumber float64 `json:"mid_n"`
		UserNumber   float64 `json:"uid_n"`
	}
	if err := json.Unmarshal(decoded, &raw); err != nil {
		return nil, err
	}
	claims := &powerXClaims{}
	if raw.TenantNumber > 0 {
		claims.TenantID = uint64(raw.TenantNumber)
	}
	if raw.MemberNumber > 0 {
		claims.DeveloperID = uint64(raw.MemberNumber)
	} else if raw.UserNumber > 0 {
		claims.DeveloperID = uint64(raw.UserNumber)
	}
	return claims, nil
}

func matchesAPIBase(expected, actual string) bool {
	if strings.TrimSpace(expected) == "" {
		return true
	}
	exp := strings.TrimRight(strings.TrimSpace(expected), "/")
	act := strings.TrimRight(strings.TrimSpace(actual), "/")
	return strings.HasPrefix(act, exp)
}

func applyPowerXCredentialDefaults(opts *DevOptions, devAPIBase string) {
	creds, err := loadPowerXCredentials()
	if err != nil || creds == nil {
		return
	}
	if matchesAPIBase(creds.APIBase, devAPIBase) && opts.DevAPIToken == "" {
		opts.DevAPIToken = creds.AccessToken
	}
	if opts.TenantID == 0 && creds.TenantID > 0 {
		opts.TenantID = creds.TenantID
	}
	if opts.DeveloperID == 0 && creds.DeveloperID > 0 {
		opts.DeveloperID = creds.DeveloperID
	}
}

func newFileWatcher(entry string, extraIgnore []string, maxFiles int) (*watch.FileWatcher, error) {
	cfg := watch.DefaultConfig(entry)
	defaultIgnore := []string{
		".px-plugin/**",
		".git/**",
		"node_modules/**",
		".nuxt/**",
		".output/**",
	}
	cfg.Ignore = append(cfg.Ignore, defaultIgnore...)
	cfg.Ignore = append(cfg.Ignore, extraIgnore...)
	if maxFiles > 0 {
		cfg.MaxFiles = maxFiles
	} else if env := os.Getenv("PX_MAX_WATCH_FILES"); env != "" {
		if val, err := strconv.Atoi(env); err == nil && val > 0 {
			cfg.MaxFiles = val
		}
	}
	// Fallback default if not set anywhere.
	if cfg.MaxFiles == 0 {
		cfg.MaxFiles = 20000
	}
	return watch.NewFileWatcher(cfg), nil
}

func normalizeLogLevel(raw string) string {
	level := strings.TrimSpace(strings.ToLower(raw))
	switch level {
	case "":
		return "info"
	case "debug", "info", "warn", "error":
		return level
	default:
		fmt.Fprintf(os.Stderr, "Warning: unknown logs-level %q, defaulting to info\n", raw)
		return "info"
	}
}

func resolveMTLSClient(opts *DevOptions, devAPIBase string) (*mtls.Client, error) {
	cfg, err := determineMTLSConfig(opts, devAPIBase)
	if err != nil || cfg == nil {
		return nil, err
	}

	client, err := mtls.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	warnIfCertificateExpiring(client)
	return client, nil
}

func determineMTLSConfig(opts *DevOptions, devAPIBase string) (*mtls.Config, error) {
	if cfg, err := configFromFlags(opts, devAPIBase); err != nil || cfg != nil {
		return cfg, err
	}

	if cfg, ok, err := mtls.LoadConfigFromEnv(); err != nil {
		return nil, err
	} else if ok {
		applyMTLSDefaults(cfg, opts, devAPIBase)
		return cfg, nil
	}

	cfg, err := loadMTLSConfigFromFile(devAPIBase)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Warning: failed to load config.json for mTLS: %v\n", err)
		}
	} else if cfg != nil {
		applyMTLSDefaults(cfg, opts, devAPIBase)
		return cfg, nil
	}

	if cfg, err := mtls.TryLoadConfigFromDefaultDir(); err != nil {
		return nil, err
	} else if cfg != nil {
		applyMTLSDefaults(cfg, opts, devAPIBase)
		return cfg, nil
	}

	return nil, nil
}

func configFromFlags(opts *DevOptions, devAPIBase string) (*mtls.Config, error) {
	if opts.MTLSCert == "" && opts.MTLSKey == "" && opts.MTLSCA == "" {
		return nil, nil
	}
	cfg, err := mtls.ConfigFromPaths(opts.MTLSCert, opts.MTLSKey, opts.MTLSCA)
	if err != nil {
		return nil, err
	}
	applyMTLSDefaults(cfg, opts, devAPIBase)
	return cfg, nil
}

func loadMTLSConfigFromFile(devAPIBase string) (*mtls.Config, error) {
	path := config.DefaultConfigPath()
	cfg, err := config.LoadConfigFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	certPath := firstNonEmpty(
		cfg.DevAPI.CertPath,
		filepath.Join(cfg.Security.CertDir, "client.crt"),
	)
	keyPath := firstNonEmpty(
		cfg.DevAPI.KeyPath,
		filepath.Join(cfg.Security.CertDir, "client.key"),
	)
	caPath := firstNonEmpty(
		cfg.DevAPI.CACertPath,
		filepath.Join(cfg.Security.CertDir, "ca.crt"),
	)

	if certPath == "" || keyPath == "" || caPath == "" {
		return nil, nil
	}

	mtlsCfg, err := mtls.ConfigFromPaths(certPath, keyPath, caPath)
	if err != nil {
		return nil, err
	}

	mtlsCfg.AutoRotate = cfg.Security.AutoRotate
	if cfg.Security.RotationCheck > 0 {
		mtlsCfg.RotationCheck = time.Duration(cfg.Security.RotationCheck) * time.Minute
	}
	mtlsCfg.InsecureSkipVerify = cfg.Security.InsecureSkipVerify
	mtlsCfg.ServerName = deriveServerName(firstNonEmpty(cfg.DevAPI.BaseURL, devAPIBase))
	return mtlsCfg, nil
}

func applyMTLSDefaults(cfg *mtls.Config, opts *DevOptions, devAPIBase string) {
	if cfg.ServerName == "" {
		cfg.ServerName = deriveServerName(devAPIBase)
	}
	if opts.MTLSServerName != "" {
		cfg.ServerName = opts.MTLSServerName
	}
	if opts.MTLSSkipVerify {
		cfg.InsecureSkipVerify = true
	}
}

func deriveServerName(base string) string {
	if base == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func warnIfCertificateExpiring(client *mtls.Client) {
	const warnBefore = 7 * 24 * time.Hour
	days, err := client.GetDaysUntilExpiry()
	if err != nil {
		return
	}

	if time.Duration(days)*24*time.Hour <= warnBefore {
		if info, err := client.GetCertificateInfo(); err == nil {
			fmt.Fprintf(os.Stderr, "Warning: mTLS certificate expires on %s. Please rotate soon.\n", info.NotAfter.Format(time.RFC3339))
		}
	}
}
