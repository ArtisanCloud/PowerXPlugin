package sse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fatih/color"
)

// OutputConfig configures the output behavior
type OutputConfig struct {
	ConsoleOutput bool
	FileOutput    bool
	LogFilePath   string
	MaxFileSize   int64
	MaxFiles      int

	// Filtering
	MinLevel          string
	FilterBySessionID string
	DisableColor      bool
}

// DefaultOutputConfig returns default output configuration
func DefaultOutputConfig() *OutputConfig {
	return &OutputConfig{
		ConsoleOutput: true,
		FileOutput:    false,
		MaxFileSize:   10 * 1024 * 1024, // 10MB
		MaxFiles:      5,
		MinLevel:      "info",
		DisableColor:  false,
	}
}

// Output handles parallel output to console and file
type Output struct {
	config *OutputConfig

	// Console output
	consoleColor *color.Color

	// File output
	fileMutex     sync.Mutex
	file          *os.File
	logBuffer     *bytes.Buffer
	writeTimer    *time.Timer
	flushInterval time.Duration

	// State
	mu             sync.RWMutex
	totalEvents    int64
	filteredEvents int64
	levelStats     map[string]int64
}

// NewOutput creates a new output handler
func NewOutput(config *OutputConfig) (*Output, error) {
	if config == nil {
		config = DefaultOutputConfig()
	}

	out := &Output{
		config:        config,
		consoleColor:  color.New(color.FgCyan),
		logBuffer:     new(bytes.Buffer),
		flushInterval: 1 * time.Second,
		levelStats:    make(map[string]int64),
	}

	// Setup file output
	if config.FileOutput {
		if err := out.setupFile(); err != nil {
			return nil, fmt.Errorf("failed to setup file: %w", err)
		}

		// Start flush timer
		out.writeTimer = time.AfterFunc(out.flushInterval, out.flush)
	}

	return out, nil
}

// WriteEvent writes an event to output destinations
func (o *Output) WriteEvent(event Event) {
	o.mu.Lock()
	o.totalEvents++
	o.mu.Unlock()

	// Filter by session ID
	if o.config.FilterBySessionID != "" {
		if sessionID, ok := event.Fields["sessionId"].(string); ok {
			if sessionID != o.config.FilterBySessionID {
				return
			}
		}
	}

	// Filter by log level
	if event.Event == "log" {
		if level, ok := event.Fields["level"].(string); ok {
			if !o.shouldShowLevel(level) {
				return
			}

			o.mu.Lock()
			o.levelStats[level]++
			o.mu.Unlock()
		}
	}

	o.mu.Lock()
	o.filteredEvents++
	o.mu.Unlock()

	// Write to console
	if o.config.ConsoleOutput {
		o.writeToConsole(event)
	}

	// Write to file
	if o.config.FileOutput {
		o.writeToBuffer(event)
	}
}

// shouldShowLevel checks if a log level should be displayed
func (o *Output) shouldShowLevel(level string) bool {
	levels := map[string]int{
		"debug": 0,
		"info":  1,
		"warn":  2,
		"error": 3,
	}

	minLevel, ok := levels[o.config.MinLevel]
	if !ok {
		return true
	}

	eventLevel, ok := levels[level]
	if !ok {
		return true
	}

	return eventLevel >= minLevel
}

// writeToConsole writes event to console with colored output
func (o *Output) writeToConsole(event Event) {
	var c *color.Color

	// Choose color based on level
	if level, ok := event.Fields["level"].(string); ok {
		switch level {
		case "error":
			c = color.New(color.FgRed, color.Bold)
		case "warn":
			c = color.New(color.FgYellow, color.Bold)
		case "info":
			c = color.New(color.FgGreen)
		case "debug":
			c = color.New(color.FgBlue)
		default:
			c = o.consoleColor
		}
	} else {
		c = o.consoleColor
	}

	// Format output
	var output string
	if timestamp, ok := event.Fields["timestamp"].(string); ok {
		output += fmt.Sprintf("[%s] ", timestamp)
	}

	if level, ok := event.Fields["level"].(string); ok {
		output += fmt.Sprintf("[%s] ", level)
	}

	if source, ok := event.Fields["source"].(string); ok {
		output += fmt.Sprintf("(%s) ", source)
	}

	if event.Data != "" {
		output += event.Data
	}

	// Add extra fields as JSON
	if len(event.Fields) > 0 {
		extraFields := make(map[string]interface{})
		for k, v := range event.Fields {
			switch k {
			case "timestamp", "level", "source", "message":
				continue
			default:
				extraFields[k] = v
			}
		}

		if len(extraFields) > 0 {
			if jsonData, err := json.Marshal(extraFields); err == nil {
				output += fmt.Sprintf(" %s", string(jsonData))
			}
		}
	}

	// Print to console
	if o.config.DisableColor {
		fmt.Println(output)
		return
	}
	c.Println(output)
}

// writeToBuffer adds event to buffer for file writing
func (o *Output) writeToBuffer(event Event) {
	// Create log entry
	logEntry := LogEntry{
		Timestamp: time.Now(),
		Event:     event,
	}

	// Marshal to JSON
	data, err := json.Marshal(logEntry)
	if err != nil {
		return
	}

	// Add to buffer
	o.logBuffer.Write(data)
	o.logBuffer.WriteByte('\n')
}

// flush writes buffer to file
func (o *Output) flush() {
	if o.logBuffer.Len() == 0 {
		// Reset timer
		o.writeTimer.Reset(o.flushInterval)
		return
	}

	o.fileMutex.Lock()
	defer o.fileMutex.Unlock()

	if o.file == nil {
		// Reset timer
		o.writeTimer.Reset(o.flushInterval)
		return
	}

	// Check if we need to rotate
	if o.shouldRotate() {
		if err := o.rotateFile(); err != nil {
			// Try to continue anyway
		}
	}

	// Write to file
	_, err := o.file.Write(o.logBuffer.Bytes())
	if err == nil {
		o.logBuffer.Reset()
	} else {
		// On error, log to stderr
		fmt.Fprintf(os.Stderr, "Failed to write to log file: %v\n", err)
	}

	// Reset timer
	o.writeTimer.Reset(o.flushInterval)
}

// shouldRotate checks if file rotation is needed
func (o *Output) shouldRotate() bool {
	if o.file == nil {
		return false
	}

	stat, err := o.file.Stat()
	if err != nil {
		return false
	}

	return stat.Size() > o.config.MaxFileSize
}

// rotateFile rotates the log file
func (o *Output) rotateFile() error {
	if o.file == nil {
		return nil
	}

	// Close current file
	o.file.Close()

	// Rename current file
	timestamp := time.Now().Format("20060102-150405")
	newPath := fmt.Sprintf("%s.%s", o.config.LogFilePath, timestamp)

	if err := os.Rename(o.config.LogFilePath, newPath); err != nil {
		return err
	}

	// Remove old files if we have too many
	return o.removeOldFiles()
}

// removeOldFiles removes old log files
func (o *Output) removeOldFiles() error {
	dir := filepath.Dir(o.config.LogFilePath)
	base := filepath.Base(o.config.LogFilePath)

	// Get all matching files
	pattern := fmt.Sprintf("%s.*", filepath.Join(dir, base))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	sort.Slice(files, func(i, j int) bool {
		infoI, errI := os.Stat(files[i])
		infoJ, errJ := os.Stat(files[j])
		if errI != nil || errJ != nil {
			return files[i] < files[j]
		}
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	for len(files) >= o.config.MaxFiles {
		os.Remove(files[0])
		files = files[1:]
	}

	return nil
}

// setupFile sets up file output
func (o *Output) setupFile() error {
	// Ensure directory exists
	dir := filepath.Dir(o.config.LogFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Open file in append mode
	file, err := os.OpenFile(o.config.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	o.file = file
	return nil
}

// Close closes the output handler
func (o *Output) Close() error {
	// Flush remaining data
	if o.writeTimer != nil {
		o.writeTimer.Stop()
	}

	if o.logBuffer.Len() > 0 {
		o.flush()
	}

	// Close file
	if o.file != nil {
		o.file.Close()
	}

	return nil
}

// GetStats returns output statistics
func (o *Output) GetStats() map[string]interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()

	stats := map[string]interface{}{
		"total_events":    o.totalEvents,
		"filtered_events": o.filteredEvents,
		"level_stats":     o.levelStats,
	}

	return stats
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Event     Event     `json:"event"`
}
