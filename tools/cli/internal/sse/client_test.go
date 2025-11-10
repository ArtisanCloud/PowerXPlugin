package sse

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	opts := DefaultClientOptions()
	opts.BaseURL = "http://localhost:8080"

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.baseURL != "http://localhost:8080" {
		t.Errorf("Expected baseURL to be 'http://localhost:8080', got %q", client.baseURL)
	}

	if client.reconnectAttempts != opts.MaxReconnectAttempts {
		t.Errorf("Expected reconnect attempts %d, got %d", opts.MaxReconnectAttempts, client.reconnectAttempts)
	}

	if client.heartbeatInterval != opts.HeartbeatInterval {
		t.Errorf("Expected heartbeat interval %v, got %v", opts.HeartbeatInterval, client.heartbeatInterval)
	}
}

func TestClient_Connect(t *testing.T) {
	// Create test SSE server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send a test event
		w.Write([]byte("id: 1\n"))
		w.Write([]byte("event: log\n"))
		w.Write([]byte("data: {\"message\":\"test\"}\n\n"))
	}))
	defer server.Close()

	// Create client
	client, err := NewClient(&ClientOptions{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Connect
	err = client.Connect(ctx, "/events")
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Wait for event
	select {
	case event := <-client.EventChan():
		if event.ID != "1" {
			t.Errorf("Expected event ID '1', got %q", event.ID)
		}
		if event.Event != "log" {
			t.Errorf("Expected event type 'log', got %q", event.Event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestClient_parseEvent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Event
	}{
		{
			name:  "simple event",
			input: "data: hello world\n\n",
			expected: Event{
				Data: "hello world",
			},
		},
		{
			name:  "event with id and type",
			input: "id: 123\nevent: log\ndata: test message\n\n",
			expected: Event{
				ID:    "123",
				Event: "log",
				Data:  "test message",
			},
		},
		{
			name:  "event with multiline data",
			input: "data: line 1\ndata: line 2\ndata: line 3\n\n",
			expected: Event{
				Data: "line 1\nline 2\nline 3",
			},
		},
		{
			name:  "event with retry",
			input: "retry: 5000\ndata: test\n\n",
			expected: Event{
				Data:  "test",
				Retry: 5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(DefaultClientOptions())
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			reader := bufio.NewReader(strings.NewReader(tt.input))
			event, err := client.parseEvent(reader)
			if err != nil {
				t.Fatalf("parseEvent failed: %v", err)
			}

			if event.ID != tt.expected.ID {
				t.Errorf("Expected ID %q, got %q", tt.expected.ID, event.ID)
			}
			if event.Event != tt.expected.Event {
				t.Errorf("Expected event %q, got %q", tt.expected.Event, event.Event)
			}
			if event.Data != tt.expected.Data {
				t.Errorf("Expected data %q, got %q", tt.expected.Data, event.Data)
			}
		})
	}
}

func TestClient_parseEvent_jsonData(t *testing.T) {
	client, err := NewClient(DefaultClientOptions())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test parsing JSON data
	input := `event: log
data: {"level":"info","message":"test message","timestamp":"2025-11-09T12:00:00Z"}

`
	reader := bufio.NewReader(strings.NewReader(input))
	event, err := client.parseEvent(reader)
	if err != nil {
		t.Fatalf("parseEvent failed: %v", err)
	}

	if event.Event != "log" {
		t.Errorf("Expected event 'log', got %q", event.Event)
	}

	// Check if JSON was parsed into fields
	if level, ok := event.Fields["level"]; !ok || level != "info" {
		t.Error("Expected level field to be parsed")
	}

	if msg, ok := event.Fields["message"]; !ok || msg != "test message" {
		t.Error("Expected message field to be parsed")
	}
}

func TestClient_Close(t *testing.T) {
	client, err := NewClient(DefaultClientOptions())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify connection is closed
	if client.IsConnected() {
		t.Error("Expected connection to be closed")
	}
}

func TestClient_IsConnected(t *testing.T) {
	client, err := NewClient(DefaultClientOptions())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Initially not connected
	if client.IsConnected() {
		t.Error("Expected client to not be connected initially")
	}
}

func TestClient_GetLastEventID(t *testing.T) {
	client, err := NewClient(DefaultClientOptions())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// No event ID initially
	if id := client.GetLastEventID(); id != "" {
		t.Errorf("Expected empty event ID, got %q", id)
	}
}

func TestClient_withMTLSConfigMissing(t *testing.T) {
	opts := DefaultClientOptions()
	opts.MTLSEnabled = true
	opts.MTLSConfig = nil

	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("NewClient should ignore nil mTLS config: %v", err)
	}
	defer client.Close()

	if client.mtlsEnabled {
		t.Error("Expected mTLS to remain disabled when config is nil")
	}
}

func TestClient_HeartbeatEmitsEvent(t *testing.T) {
	opts := DefaultClientOptions()
	opts.HeartbeatInterval = 10 * time.Millisecond
	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	client.mu.Lock()
	client.lastEventTime = time.Now().Add(-50 * time.Millisecond)
	client.mu.Unlock()

	stop := make(chan struct{})
	client.startHeartbeatMonitor(stop)
	defer close(stop)

	select {
	case evt := <-client.EventChan():
		if evt.Event != "heartbeat" {
			t.Fatalf("Expected heartbeat event, got %s", evt.Event)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Expected heartbeat event before timeout")
	}
}

func TestClient_ReconnectsAfterEOF(t *testing.T) {
	var connCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		n := atomic.AddInt32(&connCount, 1)
		fmt.Fprintf(w, "id: %d\ndata: message-%d\n\n", n, n)
		flusher.Flush()
		if n == 1 {
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()

	opts := DefaultClientOptions()
	opts.BaseURL = server.URL
	opts.HeartbeatInterval = 0
	client, err := NewClient(opts)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Connect(ctx, "/"); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	var received []string
	timeout := time.After(2 * time.Second)
	for len(received) < 2 {
		select {
		case evt := <-client.EventChan():
			received = append(received, evt.Data)
		case <-timeout:
			t.Fatalf("Timed out waiting for reconnect events (received %d)", len(received))
		}
	}

	if atomic.LoadInt32(&connCount) < 2 {
		t.Fatalf("Expected at least 2 connections, got %d", atomic.LoadInt32(&connCount))
	}
}

func TestOutput_NewOutput(t *testing.T) {
	config := DefaultOutputConfig()
	config.ConsoleOutput = true

	output, err := NewOutput(config)
	if err != nil {
		t.Fatalf("Failed to create output: %v", err)
	}
	defer output.Close()

	if output.config.ConsoleOutput != true {
		t.Error("Expected console output to be enabled")
	}
}

func TestOutput_WriteEvent(t *testing.T) {
	output, err := NewOutput(DefaultOutputConfig())
	if err != nil {
		t.Fatalf("Failed to create output: %v", err)
	}
	defer output.Close()

	// Write a test event
	event := Event{
		ID:    "1",
		Event: "log",
		Data:  "test message",
		Fields: map[string]interface{}{
			"level":     "info",
			"timestamp": "2025-11-09T12:00:00Z",
			"source":    "test",
		},
	}

	// This should not panic
	output.WriteEvent(event)

	// Get stats
	stats := output.GetStats()
	if stats["total_events"].(int64) != 1 {
		t.Errorf("Expected 1 total event, got %v", stats["total_events"])
	}

	if stats["filtered_events"].(int64) != 1 {
		t.Errorf("Expected 1 filtered event, got %v", stats["filtered_events"])
	}
}

func TestOutput_shouldShowLevel(t *testing.T) {
	output, err := NewOutput(DefaultOutputConfig())
	if err != nil {
		t.Fatalf("Failed to create output: %v", err)
	}
	defer output.Close()

	// Set minimum level to warn
	output.config.MinLevel = "warn"

	// Test level filtering
	if !output.shouldShowLevel("error") {
		t.Error("Expected error level to be shown")
	}

	if !output.shouldShowLevel("warn") {
		t.Error("Expected warn level to be shown")
	}

	if output.shouldShowLevel("info") {
		t.Error("Expected info level to not be shown")
	}

	if output.shouldShowLevel("debug") {
		t.Error("Expected debug level to not be shown")
	}
}

func TestOutput_filterBySessionID(t *testing.T) {
	output, err := NewOutput(DefaultOutputConfig())
	if err != nil {
		t.Fatalf("Failed to create output: %v", err)
	}
	defer output.Close()

	// Set session filter
	output.config.FilterBySessionID = "session-123"

	// Write event with matching session ID
	event1 := Event{
		Event: "log",
		Data:  "message 1",
		Fields: map[string]interface{}{
			"sessionId": "session-123",
			"level":     "info",
		},
	}
	output.WriteEvent(event1)

	// Write event with different session ID
	event2 := Event{
		Event: "log",
		Data:  "message 2",
		Fields: map[string]interface{}{
			"sessionId": "session-456",
			"level":     "info",
		},
	}
	output.WriteEvent(event2)

	// Get stats
	stats := output.GetStats()
	// Only 1 event should be filtered (the one matching session ID)
	if stats["filtered_events"].(int64) != 1 {
		t.Errorf("Expected 1 filtered event, got %v", stats["filtered_events"])
	}
}
