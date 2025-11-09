package watch

import "time"

// EventType represents the type of file event
type EventType string

const (
	EventCreate EventType = "create"
	EventModify EventType = "modify"
	EventDelete EventType = "delete"
)

// FileEvent represents a file change event
type FileEvent struct {
	Type      EventType `json:"type"`
	Path      string    `json:"path"`
	Hash      string    `json:"hash,omitempty"` // SHA256 hash of file content
	Timestamp time.Time `json:"timestamp"`
}

// Watcher interface for file watching
type Watcher interface {
	Start() error
	Stop() error
	Events() <-chan []FileEvent
	IsWatching() bool
}

// Config holds watcher configuration
type Config struct {
	EntryPath   string
	Ignore      []string
	Debounce    time.Duration
	Recursive   bool
	ComputeHash bool
}

// DefaultConfig returns a default watcher configuration
func DefaultConfig(entryPath string) *Config {
	return &Config{
		EntryPath:   entryPath,
		Ignore:      []string{".git", "node_modules", "dist/**", "build/**", "*.log"},
		Debounce:    250 * time.Millisecond,
		Recursive:   true,
		ComputeHash: true,
	}
}
