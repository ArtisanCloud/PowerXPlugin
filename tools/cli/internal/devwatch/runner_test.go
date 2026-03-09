package devwatch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/powerx-plugin/cli/internal/audit"
	"github.com/powerx-plugin/cli/internal/build"
	"github.com/powerx-plugin/cli/internal/devapi"
	"github.com/powerx-plugin/cli/internal/manifest"
	"github.com/powerx-plugin/cli/internal/performance"
	"github.com/powerx-plugin/cli/internal/session"
	"github.com/powerx-plugin/cli/internal/watch"
)

func TestRunner_FullLifecycle(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PX_RESOURCE_CPU_THRESHOLD", "101")

	entryDir := filepath.Join(t.TempDir(), "plugin-entry")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("create entry dir: %v", err)
	}

	manifestData := &manifest.PluginManifest{
		ID:      "plugin.full.lifecycle",
		Version: "0.1.0",
	}
	manifestData.Backend.Entry = "cmd/main.go"

	watcher := newFakeWatcher()
	builder := newFakeBuilder()
	client := newMockDevAPI()
	auditLogger := &fakeAuditLogger{}
	metrics := performance.NewMetricsCollector()

	runner, err := NewRunner(RunnerOptions{
		EntryPath:   entryDir,
		Tenant:      "tenant-a",
		TenantUUID:  "tenant-a-uuid",
		DeveloperID: 1,
		DevAPIBase:  "http://127.0.0.1:8077",
		Manifest:    manifestData,
		BuildDir:    filepath.Join(entryDir, ".px-plugin", "build"),
		CommandName: "dev --watch",
		Metrics:     metrics,
	}, Dependencies{
		Builder:        builder,
		Watcher:        watcher,
		Client:         client,
		AuditLogger:    auditLogger,
		SessionManager: session.NewManager(),
	})
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(ctx)
	}()

	waitForBuild(t, builder, build.StrategyFull)

	// Emit a change event to trigger diff build/reload path.
	watcher.Emit([]watch.FileEvent{{
		Type:      watch.EventModify,
		Path:      filepath.Join(entryDir, "main.go"),
		Timestamp: time.Now(),
	}})

	waitForBuild(t, builder, build.StrategyDiff)

	// Stop the runner and wait for cleanup (delete call).
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for runner completion")
	}

	if client.RegisterCount() != 1 {
		t.Fatalf("expected 1 register call, got %d", client.RegisterCount())
	}
	if client.ReloadCount() != 2 {
		t.Fatalf("expected 2 reload calls (full + diff), got %d", client.ReloadCount())
	}
	if client.DeleteCount() != 1 {
		t.Fatalf("expected 1 delete call, got %d", client.DeleteCount())
	}
	if client.LastDeletedSession() != client.ServerSessionID() {
		t.Fatalf("expected delete for %s, got %s", client.ServerSessionID(), client.LastDeletedSession())
	}
	if !watcher.stopped {
		t.Fatalf("expected watcher to be stopped")
	}

	strategies := builder.Strategies()
	if len(strategies) != 2 || strategies[0] != build.StrategyFull || strategies[1] != build.StrategyDiff {
		t.Fatalf("unexpected build strategies sequence: %+v", strategies)
	}

	stats := metrics.GetStats()
	if stats["reloads_total"].(uint64) != 2 {
		t.Fatalf("expected 2 reloads recorded, got %+v", stats["reloads_total"])
	}
	if stats["reload_latency_p95_ms"].(int64) == 0 {
		t.Fatalf("expected non-zero reload latency p95")
	}
}

func TestRunner_BackoffAndRollbackOnReloadFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PX_RESOURCE_CPU_THRESHOLD", "101")

	entryDir := filepath.Join(t.TempDir(), "plugin-entry")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("create entry dir: %v", err)
	}

	manifestData := &manifest.PluginManifest{
		ID:      "plugin.backoff.rollback",
		Version: "0.1.0",
	}
	manifestData.Backend.Entry = "cmd/main.go"

	watcher := newFakeWatcher()
	builder := newFakeBuilder()
	client := newMockDevAPI()
	auditLogger := &fakeAuditLogger{}

	runner, err := NewRunner(RunnerOptions{
		EntryPath:       entryDir,
		Tenant:          "tenant-b",
		TenantUUID:      "tenant-b-uuid",
		DeveloperID:     1,
		DevAPIBase:      "http://127.0.0.1:8077",
		Manifest:        manifestData,
		BuildDir:        filepath.Join(entryDir, ".px-plugin", "build"),
		CommandName:     "dev --watch",
		BackoffSchedule: []time.Duration{25 * time.Millisecond, 50 * time.Millisecond},
	}, Dependencies{
		Builder:        builder,
		Watcher:        watcher,
		Client:         client,
		AuditLogger:    auditLogger,
		SessionManager: session.NewManager(),
	})
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(ctx)
	}()

	waitForBuild(t, builder, build.StrategyFull)
	waitForReloadCount(t, client, 1, 2*time.Second)

	client.FailNextReloads(1, fmt.Errorf("forced network failure"))

	watcher.Emit([]watch.FileEvent{{
		Type:      watch.EventModify,
		Path:      filepath.Join(entryDir, "main.go"),
		Timestamp: time.Now(),
	}})

	waitForBuild(t, builder, build.StrategyDiff)
	waitForReloadCount(t, client, 3, 2*time.Second)

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for runner completion")
	}

	requests := client.ReloadRequests()
	if len(requests) != 3 {
		t.Fatalf("expected 3 reload attempts, got %d", len(requests))
	}
	if requests[2].Strategy != "rollback" {
		t.Fatalf("expected rollback strategy, got %s", requests[2].Strategy)
	}
	if requests[2].BundleHash != requests[0].BundleHash {
		t.Fatalf("expected rollback to reuse original bundle hash, got %s vs %s", requests[2].BundleHash, requests[0].BundleHash)
	}
	if runner.backoffIndex == 0 {
		t.Fatalf("expected backoff index increment after failure")
	}
}

func TestRunner_ReusesExistingSession(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	entryDir := filepath.Join(t.TempDir(), "plugin-entry")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("create entry dir: %v", err)
	}

	manifestData := &manifest.PluginManifest{ID: "plugin.resume", Version: "0.2.0"}
	manifestData.Backend.Entry = "cmd/main.go"

	builder := newFakeBuilder()
	client := newMockDevAPI()
	auditLogger := &fakeAuditLogger{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner, err := NewRunner(RunnerOptions{
		EntryPath:           entryDir,
		Tenant:              "tenant-c",
		TenantUUID:          "tenant-c-uuid",
		DeveloperID:         7,
		DevAPIBase:          "http://127.0.0.1:8077",
		Manifest:            manifestData,
		BuildDir:            filepath.Join(entryDir, ".px-plugin", "build"),
		CommandName:         "dev --resume",
		Mode:                ModeSingle,
		UseExistingSession:  true,
		ExistingSessionID:   "sess-existing",
		ExistingReloadToken: "reload-existing",
	}, Dependencies{
		Builder:        builder,
		Watcher:        nil,
		Client:         client,
		AuditLogger:    auditLogger,
		SessionManager: session.NewManager(),
	})
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("runner returned error: %v", err)
	}

	if client.RegisterCount() != 0 {
		t.Fatalf("expected register to be skipped, got %d", client.RegisterCount())
	}
	if client.DeleteCount() != 1 || client.LastDeletedSession() != "sess-existing" {
		t.Fatalf("expected delete of reused session, got %d (%s)", client.DeleteCount(), client.LastDeletedSession())
	}
}

func TestRunner_SingleRunMode(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	entryDir := filepath.Join(t.TempDir(), "plugin-entry")
	if err := os.MkdirAll(entryDir, 0o755); err != nil {
		t.Fatalf("create entry dir: %v", err)
	}

	manifestData := &manifest.PluginManifest{
		ID:      "plugin.single.run",
		Version: "0.1.0",
	}
	manifestData.Backend.Entry = "cmd/main.go"

	builder := newFakeBuilder()
	client := newMockDevAPI()
	auditLogger := &fakeAuditLogger{}

	runner, err := NewRunner(RunnerOptions{
		EntryPath:   entryDir,
		Tenant:      "tenant-c",
		TenantUUID:  "tenant-c-uuid",
		DeveloperID: 2,
		DevAPIBase:  "http://127.0.0.1:8077",
		Manifest:    manifestData,
		BuildDir:    filepath.Join(entryDir, ".px-plugin", "build"),
		CommandName: "dev",
		Mode:        ModeSingle,
	}, Dependencies{
		Builder:        builder,
		Watcher:        nil,
		Client:         client,
		AuditLogger:    auditLogger,
		SessionManager: session.NewManager(),
	})
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("runner returned error: %v", err)
	}

	if client.RegisterCount() != 1 {
		t.Fatalf("expected 1 register call, got %d", client.RegisterCount())
	}
	if client.ReloadCount() != 1 {
		t.Fatalf("expected 1 reload call, got %d", client.ReloadCount())
	}
	if client.DeleteCount() != 1 {
		t.Fatalf("expected 1 delete call, got %d", client.DeleteCount())
	}
	strategies := builder.Strategies()
	if len(strategies) != 1 || strategies[0] != build.StrategyFull {
		t.Fatalf("expected single full build, got %+v", strategies)
	}
}

func waitForBuild(t *testing.T, builder *fakeBuilder, expected build.BuildStrategy) {
	t.Helper()

	select {
	case got := <-builder.callCh:
		if got != expected {
			t.Fatalf("expected build strategy %s, got %s", expected, got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for build strategy %s", expected)
	}
}

func waitForReloadCount(t *testing.T, client *mockDevAPI, expected int, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if client.ReloadCount() >= expected {
			return
		}
		select {
		case <-ticker.C:
		case <-deadline:
			t.Fatalf("timeout waiting for reload count %d (current %d)", expected, client.ReloadCount())
		}
	}
}

type fakeBuilder struct {
	mu         sync.Mutex
	strategies []build.BuildStrategy
	callCh     chan build.BuildStrategy
}

func newFakeBuilder() *fakeBuilder {
	return &fakeBuilder{
		callCh: make(chan build.BuildStrategy, 8),
	}
}

func (b *fakeBuilder) Build(ctx context.Context, opts *build.BuildOptions) (*build.BuildResult, error) {
	if opts == nil {
		return nil, fmt.Errorf("build options required")
	}

	b.mu.Lock()
	callIndex := len(b.strategies) + 1
	b.strategies = append(b.strategies, opts.Strategy)
	b.mu.Unlock()

	result := build.NewBuildResult(time.Now())
	result.BundleHash = fmt.Sprintf("bundle-%d", callIndex)
	result.BundleSize = int64(callIndex)
	result.Complete(time.Now().Add(5*time.Millisecond), true, nil)

	select {
	case b.callCh <- opts.Strategy:
	default:
	}

	return result, nil
}

func (b *fakeBuilder) Name() string {
	return "fake"
}

func (b *fakeBuilder) Description() string {
	return "fake builder for tests"
}

func (b *fakeBuilder) Strategies() []build.BuildStrategy {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]build.BuildStrategy, len(b.strategies))
	copy(out, b.strategies)
	return out
}

type fakeWatcher struct {
	events  chan []watch.FileEvent
	started bool
	stopped bool
	mu      sync.Mutex
}

func newFakeWatcher() *fakeWatcher {
	return &fakeWatcher{
		events: make(chan []watch.FileEvent, 8),
	}
}

func (w *fakeWatcher) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.started = true
	return nil
}

func (w *fakeWatcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return nil
	}
	w.stopped = true
	close(w.events)
	return nil
}

func (w *fakeWatcher) Events() <-chan []watch.FileEvent {
	return w.events
}

func (w *fakeWatcher) IsWatching() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.started && !w.stopped
}

func (w *fakeWatcher) Emit(events []watch.FileEvent) {
	w.events <- events
}

type mockDevAPI struct {
	mu               sync.Mutex
	registerCalls    []*devapi.RegisterRequest
	reloadCalls      []*devapi.ReloadRequest
	deleteCalls      []string
	reloadToken      string
	registerResponse *devapi.RegisterResponse
	reloadFailCount  int
	reloadFailErr    error
}

func newMockDevAPI() *mockDevAPI {
	return &mockDevAPI{
		registerResponse: &devapi.RegisterResponse{
			SessionID:   "server-session",
			ReloadToken: "server-token",
		},
	}
}

func (m *mockDevAPI) Register(ctx context.Context, req *devapi.RegisterRequest) (*devapi.RegisterResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registerCalls = append(m.registerCalls, req)
	return m.registerResponse, nil
}

func (m *mockDevAPI) Reload(ctx context.Context, req *devapi.ReloadRequest) (*devapi.ReloadResponse, error) {
	m.mu.Lock()
	fail := m.reloadFailCount > 0
	if fail {
		m.reloadFailCount--
		if m.reloadFailErr == nil {
			m.reloadFailErr = fmt.Errorf("mock reload failure")
		}
	}
	cloned := cloneReloadRequestForTest(req)
	m.reloadCalls = append(m.reloadCalls, cloned)
	count := len(m.reloadCalls)
	err := m.reloadFailErr
	m.mu.Unlock()

	if fail {
		return nil, err
	}

	return &devapi.ReloadResponse{
		Status:   "ok",
		ReloadID: fmt.Sprintf("reload-%d", count),
	}, nil
}

func (m *mockDevAPI) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteCalls = append(m.deleteCalls, sessionID)
	return nil
}

func (m *mockDevAPI) SetReloadToken(token string) {
	m.mu.Lock()
	m.reloadToken = token
	m.mu.Unlock()
}

func (m *mockDevAPI) RegisterCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.registerCalls)
}

func (m *mockDevAPI) ReloadCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.reloadCalls)
}

func (m *mockDevAPI) ReloadRequests() []*devapi.ReloadRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*devapi.ReloadRequest, len(m.reloadCalls))
	copy(out, m.reloadCalls)
	return out
}

func (m *mockDevAPI) DeleteCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.deleteCalls)
}

func (m *mockDevAPI) LastDeletedSession() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.deleteCalls) == 0 {
		return ""
	}
	return m.deleteCalls[len(m.deleteCalls)-1]
}

func (m *mockDevAPI) ServerSessionID() string {
	return m.registerResponse.SessionID
}

func (m *mockDevAPI) FailNextReloads(count int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloadFailCount = count
	m.reloadFailErr = err
}

func (m *mockDevAPI) ListSessions(ctx context.Context, filter *devapi.ListSessionsFilter) ([]devapi.SessionRecord, error) {
	return []devapi.SessionRecord{}, nil
}

type fakeAuditLogger struct {
	mu     sync.Mutex
	events []AuditEventRecord
}

// AuditEventRecord keeps a lightweight copy of audit logs for verification.
type AuditEventRecord struct {
	Type    audit.EventType
	Command string
	Success bool
}

func (l *fakeAuditLogger) Log(eventType audit.EventType, sessionID, pluginID, version, tenant, entryPath, command string, success bool, duration int64, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, AuditEventRecord{
		Type:    eventType,
		Command: command,
		Success: success,
	})
}

func cloneReloadRequestForTest(req *devapi.ReloadRequest) *devapi.ReloadRequest {
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
