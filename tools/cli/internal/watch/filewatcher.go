package watch

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher implements file watching with debouncing and ignore patterns
type FileWatcher struct {
	config    *Config
	watcher   *fsnotify.Watcher
	events    chan []FileEvent
	stopCh    chan struct{}
	mu        sync.Mutex
	isWatching bool

	// File hash cache
	hashCache map[string]string

	// Ignore matcher
	matcher *Matcher
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(config *Config) *FileWatcher {
	if config == nil {
		config = DefaultConfig(".")
	}

	return &FileWatcher{
		config:    config,
		events:    make(chan []FileEvent, 100),
		stopCh:    make(chan struct{}),
		hashCache: make(map[string]string),
		matcher:   NewMatcher(config.Ignore),
	}
}

// Start starts watching files
func (w *FileWatcher) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isWatching {
		return fmt.Errorf("watcher is already running")
	}

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	w.watcher = watcher

	// Add all files and directories
	if err := w.addPath(w.config.EntryPath); err != nil {
		w.watcher.Close()
		return fmt.Errorf("failed to add path: %w", err)
	}

	w.isWatching = true

	// Start listening for events
	go w.eventListener()

	return nil
}

// Stop stops watching files
func (w *FileWatcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isWatching {
		return nil
	}

	close(w.stopCh)
	w.isWatching = false

	if w.watcher != nil {
		w.watcher.Close()
	}

	return nil
}

// IsWatching returns whether the watcher is currently watching
func (w *FileWatcher) IsWatching() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.isWatching
}

// Events returns the events channel
func (w *FileWatcher) Events() <-chan []FileEvent {
	return w.events
}

// addPath adds a file or directory to the watcher
func (w *FileWatcher) addPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat path %s: %w", path, err)
	}

	if info.IsDir() {
		// Walk directory and add all files
		return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip ignored paths
			if w.matcher.ShouldIgnore(p) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Add file to watcher
			if !d.IsDir() {
				return w.watcher.Add(p)
			}

			// Add directory to watcher
			return w.watcher.Add(p)
		})
	}

	// Add single file
	return w.watcher.Add(path)
}

// eventListener listens for file events
func (w *FileWatcher) eventListener() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Ignore if path is in ignore list
			if w.matcher.ShouldIgnore(event.Name) {
				continue
			}

			// Determine event type
			var eventType EventType
			if event.Op&fsnotify.Create == fsnotify.Create {
				eventType = EventCreate
			} else if event.Op&fsnotify.Write == fsnotify.Write {
				eventType = EventModify
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				eventType = EventDelete
			} else {
				continue
			}

			// Create file event
			fileEvent := FileEvent{
				Type:      eventType,
				Path:      event.Name,
				Timestamp: event.Timestamp,
			}

			// Compute hash for non-delete events
			if eventType != EventDelete && w.config.ComputeHash {
				hash, err := w.computeHash(event.Name)
				if err == nil {
					fileEvent.Hash = hash
				}
			}

			// Check if file/directory was added or removed from watch
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					// Add new directory to watcher
					w.watcher.Add(event.Name)
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				// File/directory was removed
				w.mu.Lock()
				delete(w.hashCache, event.Name)
				w.mu.Unlock()
			}

			// Add to debouncer
			w.debouncer().AddEvent(fileEvent)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

// debouncer gets or creates a debouncer
func (w *FileWatcher) debouncer() *Debouncer {
	// This is a simple implementation
	// In production, you might want to manage this more carefully
	return NewDebouncer(w.config.Debounce, func(events []FileEvent) {
		select {
		case w.events <- events:
		case <-w.stopCh:
		}
	})
}

// computeHash computes SHA256 hash of a file
func (w *FileWatcher) computeHash(path string) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check cache first
	if hash, ok := w.hashCache[path]; ok {
		return hash, nil
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	// Skip directories
	if info.IsDir() {
		return "", fmt.Errorf("cannot hash directory")
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Compute hash
	hash := fmt.Sprintf("%x", sha256.Sum256(data))

	// Cache the hash
	w.hashCache[path] = hash

	return hash, nil
}
