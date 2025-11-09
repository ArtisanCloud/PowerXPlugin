package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMatcher_ShouldIgnore tests the ShouldIgnore function
func TestMatcher_ShouldIgnore(t *testing.T) {
	matcher := NewMatcher([]string{".git", "node_modules", "dist/**", "*.log"})

	tests := []struct {
		path     string
		expected bool
	}{
		{".git/config", true},
		{"node_modules/package.json", true},
		{"dist/index.js", true},
		{"app.log", true},
		{"src/main.go", false},
		{"README.md", false},
		{"web-admin/app.vue", false},
	}

	for _, tt := range tests {
		result := matcher.ShouldIgnore(tt.path)
		if result != tt.expected {
			t.Errorf("ShouldIgnore(%q) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

// TestDebouncer tests the debouncer
func TestDebouncer(t *testing.T) {
	events := make([][]FileEvent, 0)
	debouncer := NewDebouncer(100*time.Millisecond, func(e []FileEvent) {
		events = append(events, e)
	})
	defer debouncer.Stop()

	// Add multiple events quickly
	for i := 0; i < 5; i++ {
		debouncer.AddEvent(FileEvent{Path: "test.go"})
	}

	// Should not have events yet (debouncing)
	if len(events) != 0 {
		t.Errorf("Expected 0 events during debounce, got %d", len(events))
	}

	// Wait for debounce to fire
	time.Sleep(150 * time.Millisecond)

	// Should have exactly 1 event with 5 file events
	if len(events) != 1 {
		t.Errorf("Expected 1 event batch, got %d", len(events))
	}

	if len(events[0]) != 5 {
		t.Errorf("Expected 5 file events, got %d", len(events[0]))
	}
}

// TestDebouncer_Flush tests the Flush method
func TestDebouncer_Flush(t *testing.T) {
	events := make([][]FileEvent, 0)
	debouncer := NewDebouncer(100*time.Millisecond, func(e []FileEvent) {
		events = append(events, e)
	})
	defer debouncer.Stop()

	// Add events
	debouncer.AddEvent(FileEvent{Path: "test1.go"})
	debouncer.AddEvent(FileEvent{Path: "test2.go"})

	// Flush should trigger events immediately
	debouncer.Flush()

	if len(events) != 1 {
		t.Errorf("Expected 1 event batch after flush, got %d", len(events))
	}

	if len(events[0]) != 2 {
		t.Errorf("Expected 2 file events after flush, got %d", len(events[0]))
	}
}
