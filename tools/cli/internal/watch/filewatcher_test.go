package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMatcher(t *testing.T) {
	matcher := NewMatcher([]string{".git/**", "node_modules/**", "**/*.log"})
	cases := []struct {
		path string
		want bool
	}{
		{".git/config", true},
		{"node_modules/pkg/index.js", true},
		{"app/debug.log", true},
		{"src/main.go", false},
	}
	for _, c := range cases {
		if got := matcher.ShouldIgnore(c.path); got != c.want {
			t.Fatalf("matcher.ShouldIgnore(%q)=%v want %v", c.path, got, c.want)
		}
	}
}

func TestDebouncer(t *testing.T) {
	var batches [][]FileEvent
	d := NewDebouncer(50*time.Millisecond, func(events []FileEvent) {
		batches = append(batches, events)
	})
	defer d.Stop()

	for i := 0; i < 5; i++ {
		d.AddEvent(FileEvent{Path: "file", Type: EventModify})
	}
	time.Sleep(80 * time.Millisecond)

	if len(batches) != 1 || len(batches[0]) != 5 {
		t.Fatalf("unexpected debounced batches %+v", batches)
	}
}

func TestFileWatcherEmitsEvents(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		EntryPath:   dir,
		Ignore:      []string{},
		Debounce:    20 * time.Millisecond,
		Recursive:   true,
		ComputeHash: true,
	}
	w := NewFileWatcher(cfg)
	if err := w.Start(); err != nil {
		t.Fatalf("start watcher: %v", err)
	}
	defer w.Stop()

	file := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(file, []byte("hello world"), 0o644); err != nil {
		t.Fatalf("update file: %v", err)
	}

	select {
	case events := <-w.Events():
		if len(events) == 0 {
			t.Fatalf("expected at least one event")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for watcher events")
	}
}
