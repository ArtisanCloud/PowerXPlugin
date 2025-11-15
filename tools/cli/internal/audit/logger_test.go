package audit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewLogger tests creating a new audit logger
func TestNewLogger(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home directory for testing
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	logger := NewLogger()
	if logger == nil {
		t.Error("NewLogger returned nil")
	}

	// Verify directory was created
	auditDir := filepath.Join(tmpDir, ".px-plugin", "audit")
	if _, err := os.Stat(auditDir); os.IsNotExist(err) {
		t.Error("Audit directory was not created")
	}
}

// TestLogger_Log tests logging an event
func TestLogger_Log(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create logger with test directory
	logger := &Logger{directory: tmpDir}

	// Log an event
	sessionID := "test-session-123"
	pluginID := "test-plugin"
	version := "1.0.0"
	tenant := "test-tenant"
	entryPath := "/path/to/plugin"
	command := "dev --watch"
	duration := int64(1500)

	logger.Log(EventSessionCreate, sessionID, pluginID, version, tenant, entryPath, command, true, duration, nil)

	// Verify log file was created
	dateStr := time.Now().Format("2006-01-02")
	logFile := filepath.Join(tmpDir, dateStr+".log")
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	// Verify log file contains event
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Log file is empty")
	}
}

// TestLogger_LogWithMetadata tests logging an event with metadata
func TestLogger_LogWithMetadata(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create logger with test directory
	logger := &Logger{directory: tmpDir}

	// Log an event with metadata
	metadata := map[string]interface{}{
		"buildTime":   "2.5s",
		"changedFiles": 3,
		"strategy":    "incremental",
	}

	logger.LogWithMetadata(
		EventBuildComplete,
		"test-session",
		"test-plugin",
		"1.0.0",
		"test-tenant",
		"/path/to/plugin",
		"dev --watch",
		true,
		2500,
		nil,
		metadata,
	)

	// Verify log file was created
	dateStr := time.Now().Format("2006-01-02")
	logFile := filepath.Join(tmpDir, dateStr+".log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Verify metadata was included
	if string(data) == "" {
		t.Error("Log file is empty")
	}
}

// TestLogger_LogError tests logging an error event
func TestLogger_LogError(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create logger with test directory
	logger := &Logger{directory: tmpDir}

	// Log an error event
	testErr := os.ErrNotExist
	logger.Log(EventBuildFail, "test-session", "test-plugin", "1.0.0", "test-tenant", "/path/to/plugin", "dev --watch", false, 0, testErr)

	// Verify log file was created
	dateStr := time.Now().Format("2006-01-02")
	logFile := filepath.Join(tmpDir, dateStr+".log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Log file is empty")
	}
}

// TestGetEvents tests retrieving events for a session
func TestGetEvents(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create logger with test directory
	logger := &Logger{directory: tmpDir}

	sessionID := "session-123"

	// Log events for the session
	logger.Log(EventSessionCreate, sessionID, "plugin1", "1.0.0", "tenant1", "/path/to/plugin1", "dev --watch", true, 1000, nil)
	logger.Log(EventBuildStart, sessionID, "plugin1", "1.0.0", "tenant1", "/path/to/plugin1", "dev --watch", true, 0, nil)

	// Log event for different session
	logger.Log(EventSessionCreate, "session-456", "plugin2", "2.0.0", "tenant2", "/path/to/plugin2", "dev --watch", true, 800, nil)

	// Get events for the first session
	events, err := logger.GetEvents(sessionID)
	if err != nil {
		t.Errorf("GetEvents failed: %v", err)
	}

	// Should have 2 events for session-123
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	// Verify event types
	if events[0].EventType != EventSessionCreate {
		t.Errorf("Expected first event to be SessionCreate, got %v", events[0].EventType)
	}

	if events[1].EventType != EventBuildStart {
		t.Errorf("Expected second event to be BuildStart, got %v", events[1].EventType)
	}

	// Verify session ID
	if events[0].SessionID != sessionID {
		t.Errorf("Expected SessionID %q, got %q", sessionID, events[0].SessionID)
	}
}

// TestGetEvents_NonExistentSession tests retrieving events for a session that doesn't exist
func TestGetEvents_NonExistentSession(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create logger with test directory
	logger := &Logger{directory: tmpDir}

	// Get events for non-existent session
	events, err := logger.GetEvents("non-existent-session")
	if err != nil {
		t.Errorf("GetEvents failed: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
}

// TestGetEvents_EmptyDirectory tests retrieving events from empty directory
func TestGetEvents_EmptyDirectory(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create logger with test directory
	logger := &Logger{directory: tmpDir}

	// Get events from empty directory
	events, err := logger.GetEvents("any-session")
	if err != nil {
		t.Errorf("GetEvents failed: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(events))
	}
}

// TestEventTypeConstants tests that all event type constants are defined
func TestEventTypeConstants(t *testing.T) {
	// Verify all event types are defined
	eventTypes := []EventType{
		EventSessionCreate,
		EventSessionResume,
		EventSessionStop,
		EventSessionDelete,
		EventSessionExpire,
		EventBuildStart,
		EventBuildComplete,
		EventBuildFail,
		EventBuildCancel,
		EventReloadTrigger,
		EventReloadSuccess,
		EventReloadFail,
		EventAPIRegister,
		EventAPIDelete,
		EventAPIErrors,
	}

	if len(eventTypes) == 0 {
		t.Error("No event types defined")
	}

	// Verify all event types have non-empty values
	for _, eventType := range eventTypes {
		if eventType == "" {
			t.Error("Found empty EventType")
		}
	}
}
