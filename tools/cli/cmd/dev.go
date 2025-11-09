package cmd

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/powerx-plugin/cli/internal/audit"
	"github.com/powerx-plugin/cli/internal/devapi"
	"github.com/powerx-plugin/cli/internal/session"
)

// DevOptions holds the configuration for the dev command
type DevOptions struct {
	Watch     bool
	Entry     string
	Tenant    string
	Ignore    []string
	DevAPI    string
	Resume    string
	Stop      string
	List      bool
	Logs      string
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

	// Parse flags
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Execute based on subcommand
	switch {
	case opts.List:
		return runDevListSessions()
	case opts.Resume != "":
		return runDevResumeSession(opts.Resume)
	case opts.Stop != "":
		return runDevStopSession(opts.Stop)
	case opts.Logs != "":
		return runDevShowLogs(opts.Logs)
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

// runDevWatch implements the watch mode
func runDevWatch(opts *DevOptions) error {
	startTime := time.Now()
	auditLogger := audit.NewLogger()

	fmt.Printf("Starting dev watch mode\n")
	fmt.Printf("  Entry: %s\n", opts.Entry)
	if opts.Tenant != "" {
		fmt.Printf("  Tenant: %s\n", opts.Tenant)
	}
	if len(opts.Ignore) > 0 {
		fmt.Printf("  Ignore: %v\n", opts.Ignore)
	}
	if opts.DevAPI != "" {
		fmt.Printf("  Dev API: %s\n", opts.DevAPI)
	}

	// Check if entry path exists
	if _, err := os.Stat(opts.Entry); err != nil {
		duration := time.Since(startTime).Milliseconds()
		auditLogger.Log(audit.EventSessionCreate, "", "my-plugin", "0.1.0", opts.Tenant, opts.Entry, "dev --watch", false, duration, err)
		return fmt.Errorf("entry path does not exist: %w", err)
	}

	// Load plugin.yaml to get plugin ID and version
	// In a full implementation, would parse plugin.yaml here
	pluginID := "my-plugin"
	version := "0.1.0"

	// Create session manager
	manager := session.NewManager()

	// Create a new session
	fmt.Println("\nCreating new session...")
	s, err := manager.CreateSession(pluginID, version, opts.Entry, opts.Tenant)
	if err != nil {
		duration := time.Since(startTime).Milliseconds()
		auditLogger.Log(audit.EventSessionCreate, "", pluginID, version, opts.Tenant, opts.Entry, "dev --watch", false, duration, err)
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Log successful session creation
	duration := time.Since(startTime).Milliseconds()
	auditLogger.Log(audit.EventSessionCreate, s.ID, pluginID, version, s.Tenant, s.EntryPath, "dev --watch", true, duration, nil)

	fmt.Printf("Session created: %s\n", s.ID)
	fmt.Printf("  Plugin:    %s v%s\n", s.PluginID, s.Version)
	fmt.Printf("  Path:      %s\n", s.EntryPath)
	fmt.Printf("  Tenant:    %s\n", s.Tenant)
	fmt.Printf("  Token:     %s\n", s.ReloadToken)
	fmt.Println()

	// Note: Full implementation would:
	// 1. Register with Dev API using the reload token
	// 2. Start file watcher with the ignore patterns
	// 3. Perform initial build
	// 4. Trigger reload on first build
	// 5. Watch for file changes and rebuild incrementally
	// 6. Update session metrics on each reload
	// 7. Handle errors and recovery

	fmt.Println("Note: Full watch mode requires Dev API and file watcher dependencies.")
	fmt.Println("Core components are ready:")
	fmt.Println("  - Dev API client: tools/cli/internal/devapi/")
	fmt.Println("  - File watcher: tools/cli/internal/watch/")
	fmt.Println("  - Session manager: tools/cli/internal/session/")
	fmt.Println("  - Build system: tools/cli/internal/build/")

	return nil
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
			fmt.Printf("  Reloads:   %d (avg: %dms, success: %.1f%%)\n",
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

	// Create Dev API client
	apiClient := devapi.NewClient(devapi.ClientOptions{
		ReloadToken: s.ReloadToken,
	})

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

	// Stop the session
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

// runDevShowLogs shows logs for a session
func runDevShowLogs(sessionID string) error {
	manager := session.NewManager()

	// Get the session
	s, err := manager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session %s: %w", sessionID, err)
	}

	// Display session info
	fmt.Printf("Logs for session: %s\n", sessionID)
	fmt.Printf("  Plugin:    %s v%s\n", s.PluginID, s.Version)
	fmt.Printf("  Status:    %s\n", s.Status)
	fmt.Println()

	// Display session metrics
	fmt.Println("Session Metrics:")
	fmt.Printf("  Total Reloads:     %d\n", s.Metrics.ReloadCount)
	fmt.Printf("  Avg Reload Time:   %dms\n", s.Metrics.AvgReloadTime)
	fmt.Printf("  Total Reload Time: %dms\n", s.Metrics.TotalReloadTime)
	fmt.Printf("  Success Rate:      %.1f%%\n", s.Metrics.SuccessRate*100)
	if s.Metrics.LastError != "" {
		fmt.Printf("  Last Error:        %s\n", s.Metrics.LastError)
	}

	// Note: In full implementation, would:
	// 1. Connect to Dev API log stream
	// 2. Display real-time logs (SSE)
	// 3. Show build output and reload events

	fmt.Println("\nNote: Log streaming requires Dev API connection.")
	return nil
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
