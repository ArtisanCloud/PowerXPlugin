package cmd

import (
	"context"
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
)

// DevOptions holds the configuration for the dev command
type DevOptions struct {
	Watch          bool
	Entry          string
	Tenant         string
	Ignore         []string
	DevAPI         string
	Resume         string
	Stop           string
	List           bool
	Logs           string
	LogsLevel      string
	LogsFile       string
	NoColor        bool
	MTLSCert       string
	MTLSKey        string
	MTLSCA         string
	MTLSServerName string
	MTLSSkipVerify bool
	MaxProcs       int
	MaxMemoryMB    int
	MaxCPUPercent  int
	MaxWatchFiles  int
	userConfig     *config.Config
}

// runDev is the entry point for the dev command
func runDev(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ExitOnError)
	opts := &DevOptions{}

	// Define flags
	fs.BoolVar(&opts.Watch, "watch", false, "Enable file watching and hot reload")
	fs.StringVar(&opts.Entry, "entry", "", "Path to the plugin entry directory")
	fs.StringVar(&opts.Tenant, "tenant", "", "Tenant ID for the dev session")
	fs.Var((*StringSliceFlag)(&opts.Ignore), "ignore", "File patterns to ignore (can be repeated)")
	fs.StringVar(&opts.DevAPI, "dev-api", "", "Dev API endpoint URL")
	fs.StringVar(&opts.Resume, "resume", "", "Resume an existing session by ID")
	fs.StringVar(&opts.Stop, "stop", "", "Stop a running session by ID")
	fs.BoolVar(&opts.List, "list-sessions", false, "List all active sessions")
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
	cfg := applyDevDefaults(opts)
	opts.userConfig = cfg

	// Execute based on subcommand
	switch {
	case opts.List:
		return runDevListSessions()
	case opts.Resume != "":
		return runDevResumeSession(opts.Resume)
	case opts.Stop != "":
		return runDevStopSession(opts.Stop)
	case opts.Logs != "":
		return runDevShowLogs(opts.Logs, opts)
	case opts.Watch:
		if opts.Entry == "" {
			return fmt.Errorf("--entry is required when using --watch")
		}
		return runDevWatch(opts)
	default:
		fs.Usage()
		return nil
	}
}

const defaultDevAPIBase = "http://127.0.0.1:8077"

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

	devAPIBase := resolveDevAPIBase(opts.DevAPI)
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

	client := devapi.NewClient(devapi.ClientOptions{
		BaseURL:    devAPIBase,
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
		DevAPIBase:  devAPIBase,
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

// runDevListSessions lists all active sessions
func runDevListSessions() error {
	manager := session.NewManager()

	sessions, err := manager.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Println("Active sessions:")
	fmt.Println()
	for _, s := range sessions {
		fmt.Printf("  ID:        %s\n", s.ID)
		fmt.Printf("  Plugin:    %s v%s\n", s.PluginID, s.Version)
		fmt.Printf("  Path:      %s\n", s.EntryPath)
		fmt.Printf("  Tenant:    %s\n", s.Tenant)
		fmt.Printf("  Status:    %s\n", s.Status)
		fmt.Printf("  Created:   %s\n", s.CreatedAt.Format("2006-01-02 15:04:05"))
		if s.Status == session.StatusActive {
			fmt.Printf("  Reloads:   %d (avg: %.0fms, success: %.1f%%)\n",
				s.Metrics.ReloadCount,
				s.Metrics.AvgReloadTime,
				s.Metrics.SuccessRate*100)
		}
		if s.Metrics.LastError != "" {
			fmt.Printf("  Last Err:  %s\n", s.Metrics.LastError)
		}
		fmt.Println()
	}

	return nil
}

// runDevResumeSession resumes a session
func runDevResumeSession(sessionID string) error {
	startTime := time.Now()
	auditLogger := audit.NewLogger()
	manager := session.NewManager()

	// Get the session
	s, err := manager.GetSession(sessionID)
	if err != nil {
		auditLogger.Log(audit.EventSessionResume, sessionID, "", "", "", "", "dev --resume", false, 0, err)
		return fmt.Errorf("failed to get session %s: %w", sessionID, err)
	}

	// Log session resume attempt
	duration := time.Since(startTime).Milliseconds()
	success := true
	var resumeErr error

	// Display session info
	fmt.Printf("Resuming session: %s\n", sessionID)
	fmt.Printf("  Plugin:    %s v%s\n", s.PluginID, s.Version)
	fmt.Printf("  Path:      %s\n", s.EntryPath)
	fmt.Printf("  Tenant:    %s\n", s.Tenant)
	fmt.Printf("  Status:    %s\n", s.Status)

	// Check if session is expired
	if s.IsExpired() {
		fmt.Println("\nWarning: Session has expired.")
		auditLogger.Log(audit.EventSessionResume, sessionID, s.PluginID, s.Version, s.Tenant, s.EntryPath, "dev --resume", false, duration, fmt.Errorf("session expired"))
		return nil
	}

	// Re-register with Dev API
	if s.ReloadToken == "" {
		resumeErr = fmt.Errorf("session has no reload token")
		auditLogger.Log(audit.EventSessionResume, sessionID, s.PluginID, s.Version, s.Tenant, s.EntryPath, "dev --resume", false, duration, resumeErr)
		return resumeErr
	}

	// Log successful resume
	auditLogger.Log(audit.EventSessionResume, sessionID, s.PluginID, s.Version, s.Tenant, s.EntryPath, "dev --resume", success, duration, resumeErr)

	devAPIBase := resolveDevAPIBase(s.DevAPIURL)
	apiClient := devapi.NewClient(devapi.ClientOptions{
		BaseURL: devAPIBase,
	})
	apiClient.SetReloadToken(s.ReloadToken)

	// Note: In full implementation, would:
	// 1. Connect to Dev API
	// 2. Re-register the session
	// 3. Start file watching
	// 4. Begin hot reload loop

	fmt.Println("\nNote: Full re-connection requires Dev API and file watcher dependencies.")
	fmt.Println("Session is ready to resume.")

	return nil
}

// runDevStopSession stops a session
func runDevStopSession(sessionID string) error {
	startTime := time.Now()
	auditLogger := audit.NewLogger()
	manager := session.NewManager()

	// Get the session first to confirm it exists
	s, err := manager.GetSession(sessionID)
	if err != nil {
		auditLogger.Log(audit.EventSessionStop, sessionID, "", "", "", "", "dev --stop", false, 0, err)
		return fmt.Errorf("failed to get session %s: %w", sessionID, err)
	}

	if s.SessionID != "" && s.ReloadToken != "" {
		client := devapi.NewClient(devapi.ClientOptions{
			BaseURL: resolveDevAPIBase(s.DevAPIURL),
		})
		client.SetReloadToken(s.ReloadToken)
		if err := client.Delete(context.Background(), s.SessionID); err != nil {
			fmt.Printf("Warning: failed to delete Dev API session: %v\n", err)
		}
	}

	err = manager.StopSession(sessionID)
	if err != nil {
		duration := time.Since(startTime).Milliseconds()
		auditLogger.Log(audit.EventSessionStop, sessionID, s.PluginID, s.Version, s.Tenant, s.EntryPath, "dev --stop", false, duration, err)
		return fmt.Errorf("failed to stop session: %w", err)
	}

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

	devAPIURL := resolveDevAPIBase(s.DevAPIURL)

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
		opts.MaxWatchFiles = envIntDefault("PX_MAX_WATCH_FILES", 10000)
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

func resolveDevAPIBase(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if env := os.Getenv("PX_DEV_API_BASE"); env != "" {
		return env
	}
	return defaultDevAPIBase
}

func newFileWatcher(entry string, extraIgnore []string, maxFiles int) (*watch.FileWatcher, error) {
	cfg := watch.DefaultConfig(entry)
	cfg.Ignore = append(cfg.Ignore, ".px-plugin/**")
	cfg.Ignore = append(cfg.Ignore, extraIgnore...)
	if maxFiles > 0 {
		cfg.MaxFiles = maxFiles
	} else if env := os.Getenv("PX_MAX_WATCH_FILES"); env != "" {
		if val, err := strconv.Atoi(env); err == nil && val > 0 {
			cfg.MaxFiles = val
		}
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
