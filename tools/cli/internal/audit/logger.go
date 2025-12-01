package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EventType represents the type of audit event
type EventType string

const (
	// Session events
	EventSessionCreate EventType = "session.create"
	EventSessionResume EventType = "session.resume"
	EventSessionStop   EventType = "session.stop"
	EventSessionDelete EventType = "session.delete"
	EventSessionExpire EventType = "session.expire"
	EventSessionLogs   EventType = "session.logs"

	// Build events
	EventBuildStart    EventType = "build.start"
	EventBuildComplete EventType = "build.complete"
	EventBuildFail     EventType = "build.fail"
	EventBuildCancel   EventType = "build.cancel"

	// Reload events
	EventReloadTrigger EventType = "reload.trigger"
	EventReloadSuccess EventType = "reload.success"
	EventReloadFail    EventType = "reload.fail"

	// API events
	EventAPIRegister EventType = "api.register"
	EventAPIDelete   EventType = "api.delete"
	EventAPIErrors   EventType = "api.errors"
)

// Event represents a single audit log entry
type Event struct {
	ID        string          `json:"id"`                 // Unique event ID (UUID)
	Timestamp time.Time       `json:"timestamp"`          // When the event occurred
	EventType EventType       `json:"eventType"`          // Type of event
	SessionID string          `json:"sessionId"`          // Associated session ID
	PluginID  string          `json:"pluginId"`           // Plugin identifier
	Version   string          `json:"version"`            // Plugin version
	Tenant    string          `json:"tenant"`             // Tenant ID
	EntryPath string          `json:"entryPath"`          // Plugin entry path
	User      string          `json:"user"`               // Current user
	IP        string          `json:"ip"`                 // Client IP
	Command   string          `json:"command"`            // CLI command
	Success   bool            `json:"success"`            // Whether operation succeeded
	Error     string          `json:"error"`              // Error message if failed
	Duration  int64           `json:"duration"`           // Duration in milliseconds
	Metadata  json.RawMessage `json:"metadata,omitempty"` // Additional data
}

// Logger handles audit logging
type Logger struct {
	directory string
}

// NewLogger creates a new audit logger
func NewLogger() *Logger {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	baseDir := filepath.Join(homeDir, ".px-plugin", "audit")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		fmt.Printf("Warning: failed to create audit directory: %v\n", err)
	}

	return &Logger{
		directory: baseDir,
	}
}

// Log records an audit event
func (l *Logger) Log(eventType EventType, sessionID, pluginID, version, tenant, entryPath, command string, success bool, duration int64, err error) {
	event := &Event{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		EventType: eventType,
		SessionID: sessionID,
		PluginID:  pluginID,
		Version:   version,
		Tenant:    tenant,
		EntryPath: entryPath,
		User:      getCurrentUser(),
		IP:        getClientIP(),
		Command:   command,
		Success:   success,
		Duration:  duration,
	}

	if err != nil {
		event.Error = err.Error()
	}

	// Write to log file
	l.writeEvent(event)
}

// LogWithMetadata records an audit event with additional metadata
func (l *Logger) LogWithMetadata(eventType EventType, sessionID, pluginID, version, tenant, entryPath, command string, success bool, duration int64, err error, metadata map[string]interface{}) {
	event := &Event{
		ID:        generateEventID(),
		Timestamp: time.Now(),
		EventType: eventType,
		SessionID: sessionID,
		PluginID:  pluginID,
		Version:   version,
		Tenant:    tenant,
		EntryPath: entryPath,
		User:      getCurrentUser(),
		IP:        getClientIP(),
		Command:   command,
		Success:   success,
		Duration:  duration,
	}

	if err != nil {
		event.Error = err.Error()
	}

	// Marshal metadata
	if metadata != nil {
		data, _ := json.Marshal(metadata)
		event.Metadata = data
	}

	// Write to log file
	l.writeEvent(event)
}

// GetEvents retrieves audit events for a session
func (l *Logger) GetEvents(sessionID string) ([]*Event, error) {
	var events []*Event

	// Read all log files
	files, err := os.ReadDir(l.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read audit directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filepath := filepath.Join(l.directory, file.Name())
		f, err := os.Open(filepath)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			var event Event
			if err := json.Unmarshal(line, &event); err != nil {
				continue
			}
			if event.SessionID == sessionID {
				copyEvent := event
				events = append(events, &copyEvent)
			}
		}
		f.Close()
	}

	return events, nil
}

// writeEvent writes an event to a log file
func (l *Logger) writeEvent(event *Event) {
	// Create log file path (one file per day)
	dateStr := event.Timestamp.Format("2006-01-02")
	filename := filepath.Join(l.directory, dateStr+".log")

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("Warning: failed to marshal audit event: %v\n", err)
		return
	}

	// Append to log file
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Printf("Warning: failed to write audit log: %v\n", err)
		return
	}
	defer f.Close()

	// Write JSON + newline
	if _, err := f.Write(append(data, '\n')); err != nil {
		fmt.Printf("Warning: failed to write audit log: %v\n", err)
	}
}

// getCurrentUser returns the current user
func getCurrentUser() string {
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	return user
}

// getClientIP returns the client IP
func getClientIP() string {
	// In a full implementation, would extract from connection
	// For now, return localhost
	return "127.0.0.1"
}

// generateEventID generates a unique event ID
func generateEventID() string {
	// Simple timestamp-based ID for now
	// In production, would use uuid.NewString()
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
