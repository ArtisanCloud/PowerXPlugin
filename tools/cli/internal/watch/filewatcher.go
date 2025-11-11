package watch

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/powerx-plugin/cli/internal/performance"
)

// FileWatcher implements file watching with debouncing and ignore patterns
type FileWatcher struct {
	config     *Config
	watcher    *fsnotify.Watcher
	events     chan []FileEvent
	stopCh     chan struct{}
	mu         sync.Mutex
	isWatching bool
	debouncer  *Debouncer
	watchCount int
	maxFiles   int

	// Performance optimizations
	hashCache   *performance.HashCache
	fastHasher  *performance.FastHasher
	metrics     *performance.MetricsCollector
	stringPool  *performance.StringPool
	concurrency *performance.ConcurrencyLimiter

	// Ignore matcher
	matcher *Matcher
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(config *Config) *FileWatcher {
	if config == nil {
		config = DefaultConfig(".")
	}
	if config.MaxFiles <= 0 {
		config.MaxFiles = 10000
	}

	return &FileWatcher{
		config:      config,
		events:      make(chan []FileEvent, 100),
		stopCh:      make(chan struct{}),
		hashCache:   performance.NewHashCache(),
		fastHasher:  performance.NewFastHasher(),
		metrics:     performance.NewMetricsCollector(),
		stringPool:  performance.NewStringPool(),
		concurrency: performance.NewConcurrencyLimiter(10),
		matcher:     NewMatcher(config.Ignore),
		maxFiles:    config.MaxFiles,
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

	w.debouncer = NewDebouncer(w.config.Debounce, func(events []FileEvent) {
		select {
		case w.events <- events:
		case <-w.stopCh:
		}
	})

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

	if w.debouncer != nil {
		w.debouncer.Stop()
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
		return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if w.matcher.ShouldIgnore(p) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			if d.IsDir() {
				if err := w.watcher.Add(p); err != nil {
					return fmt.Errorf("failed to watch %s: %w", p, err)
				}
				if err := w.registerWatch(p); err != nil {
					return err
				}
			}
			return nil
		})
	}

	// Add single file (non-recursive use-case)
	if err := w.watcher.Add(path); err != nil {
		return err
	}
	return w.registerWatch(path)
}

// eventListener listens for file events
func (w *FileWatcher) eventListener() {
	for {
		select {
		case <-w.stopCh:
			return
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
				Timestamp: time.Now(),
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
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = w.addPath(event.Name)
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				w.hashCache.Delete(event.Name)
			}

			// Add to debouncer
			if w.debouncer != nil {
				w.debouncer.AddEvent(fileEvent)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

func (w *FileWatcher) registerWatch(path string) error {
	if w.maxFiles <= 0 {
		return nil
	}
	w.watchCount++
	if w.watchCount > w.maxFiles {
		return fmt.Errorf("watch limit exceeded (%d). Use --max-watch-files or PX_MAX_WATCH_FILES to increase the limit", w.maxFiles)
	}
	return nil
}

// computeHash computes SHA256 hash of a file
func (w *FileWatcher) computeHash(path string) (string, error) {
	// Check cache first
	if hash, ok := w.hashCache.Get(path); ok {
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
	w.hashCache.Set(path, hash)

	return hash, nil
}
