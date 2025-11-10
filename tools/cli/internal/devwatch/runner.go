package devwatch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/powerx-plugin/cli/internal/audit"
	"github.com/powerx-plugin/cli/internal/build"
	"github.com/powerx-plugin/cli/internal/devapi"
	errors2 "github.com/powerx-plugin/cli/internal/errors"
	"github.com/powerx-plugin/cli/internal/manifest"
	"github.com/powerx-plugin/cli/internal/performance"
	resources "github.com/powerx-plugin/cli/internal/resources"
	"github.com/powerx-plugin/cli/internal/session"
	"github.com/powerx-plugin/cli/internal/watch"
)

var defaultBackoffSchedule = []time.Duration{
	1 * time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	30 * time.Second,
}

// DevAPIClient captures the Dev API operations the runner needs.
type DevAPIClient interface {
	Register(ctx context.Context, req *devapi.RegisterRequest) (*devapi.RegisterResponse, error)
	Reload(ctx context.Context, req *devapi.ReloadRequest) (*devapi.ReloadResponse, error)
	Delete(ctx context.Context, sessionID string) error
	SetReloadToken(token string)
}

// AuditLogger lists the audit logging capabilities required by the runner.
type AuditLogger interface {
	Log(eventType audit.EventType, sessionID, pluginID, version, tenant, entryPath, command string, success bool, duration int64, err error)
}

// RunnerOptions contains the configuration necessary to run dev watch mode.
type RunnerOptions struct {
	EntryPath       string
	Tenant          string
	DevAPIBase      string
	Manifest        *manifest.PluginManifest
	BuildDir        string
	CommandName     string
	Metrics         *performance.MetricsCollector
	Resources       *resources.ResourceMonitor
	BackoffSchedule []time.Duration
}

// Dependencies enumerates the injectable collaborators.
type Dependencies struct {
	Builder        build.Builder
	Watcher        watch.Watcher
	Client         DevAPIClient
	AuditLogger    AuditLogger
	SessionManager *session.Manager
}

// Runner coordinates register → build → reload → delete loops.
type Runner struct {
	opts RunnerOptions
	deps Dependencies

	session              *session.Session
	metrics              *performance.MetricsCollector
	lastMetricsPrint     time.Time
	resources            *resources.ResourceMonitor
	throttleUntil        time.Time
	backoffSchedule      []time.Duration
	backoffIndex         int
	backoffUntil         time.Time
	lastSuccessfulReload *devapi.ReloadRequest
	lastSuccessfulBuild  *build.BuildResult
}

// NewRunner validates options/dependencies and constructs a runner.
func NewRunner(opts RunnerOptions, deps Dependencies) (*Runner, error) {
	if opts.Manifest == nil {
		return nil, fmt.Errorf("manifest is required")
	}
	if opts.EntryPath == "" {
		return nil, fmt.Errorf("entry path is required")
	}
	if opts.BuildDir == "" {
		opts.BuildDir = filepath.Join(opts.EntryPath, ".px-plugin", "build")
	}
	if opts.CommandName == "" {
		opts.CommandName = "dev --watch"
	}
	if opts.Metrics == nil {
		opts.Metrics = performance.NewMetricsCollector()
	}
	if opts.Resources == nil {
		opts.Resources = defaultResourceMonitor()
	}

	switch {
	case deps.Builder == nil:
		return nil, fmt.Errorf("builder dependency is required")
	case deps.Watcher == nil:
		return nil, fmt.Errorf("watcher dependency is required")
	case deps.Client == nil:
		return nil, fmt.Errorf("dev api client dependency is required")
	case deps.AuditLogger == nil:
		return nil, fmt.Errorf("audit logger dependency is required")
	case deps.SessionManager == nil:
		return nil, fmt.Errorf("session manager dependency is required")
	}

	runner := &Runner{
		opts:      opts,
		deps:      deps,
		metrics:   opts.Metrics,
		resources: opts.Resources,
	}

	if len(opts.BackoffSchedule) > 0 {
		runner.backoffSchedule = append([]time.Duration(nil), opts.BackoffSchedule...)
	} else {
		runner.backoffSchedule = append([]time.Duration(nil), defaultBackoffSchedule...)
	}

	return runner, nil
}

// Run executes the full dev watch workflow until the provided context is cancelled.
func (r *Runner) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	r.setupResourceCallbacks()

	if err := os.MkdirAll(r.opts.BuildDir, 0o755); err != nil {
		return errors2.WrapError(err, errors2.ErrFileSystem, "create build dir",
			errors2.WithContext("dir", r.opts.BuildDir))
	}

	manager := r.deps.SessionManager
	auditLogger := r.deps.AuditLogger

	s, err := manager.CreateSession(r.opts.Manifest.ID, r.opts.Manifest.Version, r.opts.EntryPath, r.opts.Tenant)
	if err != nil {
		return errors2.WrapError(err, errors2.ErrSystem, "create session",
			errors2.WithContext("pluginId", r.opts.Manifest.ID))
	}
	r.session = s
	if r.metrics != nil {
		r.metrics.UpdateActiveSessions(1)
	}

	defer r.cleanup()

	if err := r.register(ctx, auditLogger); err != nil {
		return err
	}

	if err := r.deps.Watcher.Start(); err != nil {
		return errors2.WrapError(err, errors2.ErrFileSystem, "start watcher",
			errors2.WithContext("entry", r.opts.EntryPath))
	}
	defer r.deps.Watcher.Stop()

	if err := r.buildAndReload(ctx, build.StrategyFull, nil, time.Now()); err != nil {
		return err
	}

	fmt.Println("Initial build complete. Watching for changes... (Ctrl+C to stop)")
	for {
		select {
		case <-ctx.Done():
			return nil
		case events, ok := <-r.deps.Watcher.Events():
			if !ok {
				return nil
			}
			if len(events) == 0 {
				continue
			}

			if r.isThrottled() {
				remaining := time.Until(r.throttleUntil).Round(time.Second)
				if remaining > 0 {
					fmt.Printf("Resource guard active (%s remaining), skipping reload\n", remaining)
				}
				time.Sleep(1 * time.Second)
				continue
			}

			if r.isBackoffActive() {
				remaining := time.Until(r.backoffUntil)
				if remaining > 0 {
					if remaining > time.Second {
						fmt.Printf("Backoff active (~%s remaining)\n", remaining.Round(time.Second))
						time.Sleep(time.Second)
					} else {
						fmt.Printf("Backoff active (~%s remaining)\n", remaining)
						time.Sleep(remaining)
					}
				}
				continue
			}

			eventStart := time.Now()
			if r.metrics != nil {
				r.metrics.RecordEvent()
			}
			if err := r.buildAndReload(ctx, build.StrategyDiff, events, eventStart); err != nil {
				fmt.Printf("Reload failed: %v\n", err)
				r.handleFailure(ctx, err)
			} else {
				fmt.Printf("Reload applied (%d files)\n", len(events))
			}
		}
	}
}

func (r *Runner) register(ctx context.Context, auditLogger AuditLogger) error {
	registerStart := time.Now()
	regResp, err := r.deps.Client.Register(ctx, &devapi.RegisterRequest{
		PluginID:  r.opts.Manifest.ID,
		Version:   r.opts.Manifest.Version,
		EntryPath: r.opts.EntryPath,
		Tenant:    r.opts.Tenant,
		Metadata: map[string]string{
			"backend.entry": r.opts.Manifest.Backend.Entry,
		},
	})
	if err != nil {
		auditLogger.Log(audit.EventAPIRegister, "", r.opts.Manifest.ID, r.opts.Manifest.Version, r.opts.Tenant, r.opts.EntryPath, r.opts.CommandName, false, time.Since(registerStart).Milliseconds(), err)
		return errors2.WrapError(err, errors2.ErrAPI, "dev api register",
			errors2.WithContext("pluginId", r.opts.Manifest.ID))
	}

	r.deps.Client.SetReloadToken(regResp.ReloadToken)

	r.session.SessionID = regResp.SessionID
	r.session.ReloadToken = regResp.ReloadToken
	r.session.DevAPIURL = r.opts.DevAPIBase

	if err := r.deps.SessionManager.UpdateSession(r.session); err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	auditLogger.Log(audit.EventAPIRegister, r.session.ID, r.session.PluginID, r.session.Version, r.session.Tenant, r.session.EntryPath, r.opts.CommandName, true, time.Since(registerStart).Milliseconds(), nil)
	return nil
}

func (r *Runner) buildAndReload(ctx context.Context, strategy build.BuildStrategy, events []watch.FileEvent, changeStart time.Time) error {
	if len(events) == 0 {
		changeStart = time.Time{}
	}
	buildStart := time.Now()
	result, err := r.deps.Builder.Build(ctx, &build.BuildOptions{
		EntryPath:    r.opts.EntryPath,
		OutDir:       r.opts.BuildDir,
		Strategy:     strategy,
		ChangedFiles: convertEvents(events),
	})
	if err != nil {
		_ = r.deps.SessionManager.RecordReload(r.session.ID, 0, false, err.Error())
		r.deps.AuditLogger.Log(audit.EventReloadFail, r.session.ID, r.session.PluginID, r.session.Version, r.session.Tenant, r.session.EntryPath, r.opts.CommandName, false, 0, err)
		wrapped := errors2.WrapError(err, errors2.ErrBuild, "builder failure",
			errors2.WithContext("strategy", strategy.String()))
		if r.metrics != nil {
			r.metrics.RecordBuild(false)
			r.metrics.RecordBuildLatency(time.Since(buildStart))
		}
		return wrapped
	}
	if r.metrics != nil {
		r.metrics.RecordBuild(true)
		r.metrics.RecordBuildLatency(result.BuildDuration)
	}

	reloadReq := &devapi.ReloadRequest{
		SessionID:     r.session.SessionID,
		BundleHash:    result.BundleHash,
		BundleSize:    result.BundleSize,
		BuildDuration: result.BuildDuration.Milliseconds(),
		Strategy:      strategy.String(),
		ChangedFiles:  events,
	}

	reloadStart := time.Now()
	if _, err := r.deps.Client.Reload(ctx, reloadReq); err != nil {
		_ = r.deps.SessionManager.RecordReload(r.session.ID, reloadReq.BuildDuration, false, err.Error())
		r.deps.AuditLogger.Log(audit.EventReloadFail, r.session.ID, r.session.PluginID, r.session.Version, r.session.Tenant, r.session.EntryPath, r.opts.CommandName, false, time.Since(reloadStart).Milliseconds(), err)
		if r.metrics != nil {
			r.metrics.RecordReload(false)
		}
		return errors2.WrapError(err, errors2.ErrAPI, "dev api reload",
			errors2.WithContext("sessionId", r.session.SessionID))
	}

	_ = r.deps.SessionManager.RecordReload(r.session.ID, reloadReq.BuildDuration, true, "")
	r.deps.AuditLogger.Log(audit.EventReloadSuccess, r.session.ID, r.session.PluginID, r.session.Version, r.session.Tenant, r.session.EntryPath, r.opts.CommandName, true, time.Since(reloadStart).Milliseconds(), nil)

	if r.metrics != nil {
		total := time.Since(changeStart)
		r.metrics.RecordReload(true)
		if !changeStart.IsZero() {
			r.metrics.RecordEventLatency(total)
			r.metrics.RecordReloadLatency(time.Since(reloadStart))
		}
		r.sampleRuntimeMetrics()
		r.maybeReportMetrics()
	}

	r.recordSuccessfulReload(reloadReq, result)
	return nil
}

func (r *Runner) cleanup() {
	if r.session == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if r.session.SessionID != "" {
		_ = r.deps.Client.Delete(ctx, r.session.SessionID)
	}

	_ = r.deps.SessionManager.StopSession(r.session.ID)
	if r.metrics != nil {
		r.metrics.UpdateActiveSessions(-1)
	}
}

func convertEvents(events []watch.FileEvent) []build.FileEvent {
	if len(events) == 0 {
		return nil
	}

	converted := make([]build.FileEvent, 0, len(events))
	for _, evt := range events {
		converted = append(converted, build.FileEvent{
			Type:      string(evt.Type),
			Path:      evt.Path,
			Hash:      evt.Hash,
			Timestamp: evt.Timestamp,
		})
	}
	return converted
}

func (r *Runner) sampleRuntimeMetrics() {
	if r.metrics == nil {
		return
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	r.metrics.UpdateMemoryUsage(int64(mem.Alloc / 1024 / 1024))
	cpuPercent := int64(float64(runtime.NumGoroutine()) / float64(runtime.NumCPU()) * 100)
	if cpuPercent > 100 {
		cpuPercent = 100
	}
	r.metrics.UpdateCPUUsage(cpuPercent)
	if r.resources != nil {
		r.resources.SetUsage(resources.Memory, int64(mem.Alloc))
		r.resources.SetUsage(resources.CPU, cpuPercent)
	}
}

func (r *Runner) maybeReportMetrics() {
	if r.metrics == nil {
		return
	}
	if !r.lastMetricsPrint.IsZero() && time.Since(r.lastMetricsPrint) < 30*time.Second {
		return
	}
	stats := r.metrics.GetStats()
	var (
		reloads    uint64
		successPct float64
		p95        int64
		memUsage   int64
	)
	if v, ok := stats["reloads_total"].(uint64); ok {
		reloads = v
	}
	if v, ok := stats["reload_success_rate"].(float64); ok {
		successPct = v * 100
	}
	if v, ok := stats["reload_latency_p95_ms"].(int64); ok {
		p95 = v
	}
	if v, ok := stats["memory_usage_mb"].(int64); ok {
		memUsage = v
	}

	fmt.Printf("Metrics: reloads=%d success=%.1f%% p95=%dms mem=%dMB\n",
		reloads, successPct, p95, memUsage)
	r.lastMetricsPrint = time.Now()
}

func (r *Runner) setupResourceCallbacks() {
	if r.resources == nil {
		return
	}
	r.resources.AddCallback(resources.Memory, func(usage int64, limit resources.Limit) {
		percent := 0.0
		if limit.Value > 0 {
			percent = float64(usage) / float64(limit.Value) * 100
		}
		fmt.Printf("Warning: memory usage high (%.1f%%). Throttling reloads...\n", percent)
		r.throttleFor(10 * time.Second)
	})
	r.resources.AddCallback(resources.CPU, func(usage int64, limit resources.Limit) {
		fmt.Printf("Warning: CPU usage high (%d%%). Throttling reloads...\n", usage)
		r.throttleFor(5 * time.Second)
	})
}

func (r *Runner) throttleFor(d time.Duration) {
	until := time.Now().Add(d)
	if until.After(r.throttleUntil) {
		r.throttleUntil = until
	}
}

func (r *Runner) isThrottled() bool {
	if r.throttleUntil.IsZero() {
		return false
	}
	return time.Now().Before(r.throttleUntil)
}

func (r *Runner) isBackoffActive() bool {
	if r.backoffUntil.IsZero() {
		return false
	}
	return time.Now().Before(r.backoffUntil)
}

func defaultResourceMonitor() *resources.ResourceMonitor {
	monitor := resources.NewResourceMonitor()
	memLimitMB := envInt("PX_RESOURCE_MEMORY_MB", 2048)
	memThreshold := envInt("PX_RESOURCE_MEMORY_THRESHOLD", 85)
	memBytes := int64(memLimitMB) * 1024 * 1024
	monitor.SetLimit(resources.Limit{
		Type:      resources.Memory,
		Value:     memBytes,
		Unit:      "bytes",
		Threshold: float64(memThreshold),
	})

	cpuThreshold := envInt("PX_RESOURCE_CPU_THRESHOLD", 90)
	monitor.SetLimit(resources.Limit{
		Type:      resources.CPU,
		Value:     100,
		Unit:      "percent",
		Threshold: float64(cpuThreshold),
	})
	return monitor
}

func envInt(key string, def int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return def
}

func (r *Runner) recordSuccessfulReload(req *devapi.ReloadRequest, result *build.BuildResult) {
	if req != nil {
		r.lastSuccessfulReload = cloneReloadRequest(req)
	}
	if result != nil {
		cloned := *result
		r.lastSuccessfulBuild = &cloned
	}
	r.resetBackoff()
}

func (r *Runner) handleFailure(ctx context.Context, cause error) {
	delay := r.scheduleBackoff()
	fmt.Printf("Applying backoff %s after failure (%v)\n", delay, cause)
	if err := r.rollbackLastBundle(ctx, cause); err != nil {
		if err != context.Canceled && err != context.DeadlineExceeded {
			fmt.Printf("Rollback attempt skipped: %v\n", err)
		}
	}
}

func (r *Runner) scheduleBackoff() time.Duration {
	if len(r.backoffSchedule) == 0 {
		r.backoffSchedule = append([]time.Duration(nil), defaultBackoffSchedule...)
	}
	idx := r.backoffIndex
	if idx >= len(r.backoffSchedule) {
		idx = len(r.backoffSchedule) - 1
	}
	delay := r.backoffSchedule[idx]
	if r.backoffIndex < len(r.backoffSchedule)-1 {
		r.backoffIndex++
	}
	r.backoffUntil = time.Now().Add(delay)
	return delay
}

func (r *Runner) resetBackoff() {
	r.backoffIndex = 0
	r.backoffUntil = time.Time{}
}

func (r *Runner) rollbackLastBundle(ctx context.Context, cause error) error {
	if r.lastSuccessfulReload == nil {
		return fmt.Errorf("no previous bundle to rollback to")
	}

	rollbackReq := cloneReloadRequest(r.lastSuccessfulReload)
	if rollbackReq.Metadata == nil {
		rollbackReq.Metadata = map[string]interface{}{}
	}
	rollbackReq.Metadata["rollback"] = true
	rollbackReq.Metadata["rollbackReason"] = cause.Error()
	rollbackReq.Metadata["rollbackAt"] = time.Now().UTC().Format(time.RFC3339)
	rollbackReq.Strategy = "rollback"

	rollbackCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := r.deps.Client.Reload(rollbackCtx, rollbackReq); err != nil {
		return fmt.Errorf("rollback reload failed: %w", err)
	}

	fmt.Println("Rollback applied using last successful bundle")
	return nil
}

func cloneReloadRequest(req *devapi.ReloadRequest) *devapi.ReloadRequest {
	if req == nil {
		return nil
	}

	copied := *req
	if len(req.ChangedFiles) > 0 {
		copied.ChangedFiles = append([]watch.FileEvent(nil), req.ChangedFiles...)
	}
	if req.Metadata != nil {
		meta := make(map[string]interface{}, len(req.Metadata))
		for k, v := range req.Metadata {
			meta[k] = v
		}
		copied.Metadata = meta
	}
	return &copied
}
