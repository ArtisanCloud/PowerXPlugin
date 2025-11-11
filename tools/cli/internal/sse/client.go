package sse

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/powerx-plugin/cli/internal/mtls"
)

// Event represents a Server-Sent Event
type Event struct {
	ID     string                 `json:"id,omitempty"`
	Event  string                 `json:"event,omitempty"`
	Data   string                 `json:"data"`
	Retry  time.Duration          `json:"retry,omitempty"`
	Fields map[string]interface{} `json:"-"`
}

// Client handles Server-Sent Events connections
type Client struct {
	baseURL     string
	headers     map[string]string
	httpClient  *http.Client
	mtlsEnabled bool
	mtlsConfig  *mtls.Config

	// Event handling
	eventCh chan Event
	errorCh chan error
	done    chan struct{}

	// Options
	reconnectAttempts int
	reconnectDelay    time.Duration
	maxReconnectDelay time.Duration
	heartbeatInterval time.Duration

	// State
	mu              sync.RWMutex
	lastEventID     string
	isConnected     bool
	shouldReconnect bool
	lastEventTime   time.Time

	mtlsClient *mtls.Client
	mtlsOwned  bool
}

// ClientOptions configures the SSE client
type ClientOptions struct {
	BaseURL     string
	Headers     map[string]string
	HTTPClient  *http.Client
	MTLSEnabled bool
	MTLSConfig  *mtls.Config
	MTLSClient  *mtls.Client

	// Reconnection options
	MaxReconnectAttempts  int
	InitialReconnectDelay time.Duration
	MaxReconnectDelay     time.Duration

	// Heartbeat
	HeartbeatInterval time.Duration
}

// DefaultClientOptions returns default client options
func DefaultClientOptions() *ClientOptions {
	return &ClientOptions{
		MaxReconnectAttempts:  10,
		InitialReconnectDelay: 1 * time.Second,
		MaxReconnectDelay:     30 * time.Second,
		HeartbeatInterval:     30 * time.Second,
		HTTPClient:            &http.Client{},
	}
}

// NewClient creates a new SSE client
func NewClient(opts *ClientOptions) (*Client, error) {
	if opts == nil {
		opts = DefaultClientOptions()
	}

	// Set up HTTP client with mTLS if enabled
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
					MaxVersion: tls.VersionTLS13,
				},
			},
		}
	}

	client := &Client{
		baseURL:    opts.BaseURL,
		headers:    opts.Headers,
		httpClient: httpClient,
		mtlsConfig: opts.MTLSConfig,

		eventCh: make(chan Event, 100),
		errorCh: make(chan error, 1),
		done:    make(chan struct{}),

		reconnectAttempts: opts.MaxReconnectAttempts,
		reconnectDelay:    opts.InitialReconnectDelay,
		maxReconnectDelay: opts.MaxReconnectDelay,
		heartbeatInterval: opts.HeartbeatInterval,

		shouldReconnect: true,
		lastEventTime:   time.Now(),
	}

	// Set up mTLS if enabled
	if opts.MTLSClient != nil {
		client.mtlsClient = opts.MTLSClient
		client.mtlsEnabled = true
		client.httpClient.Transport = &http.Transport{
			TLSClientConfig: client.mtlsClient.GetTLSConfig(),
		}
	} else if opts.MTLSEnabled && client.mtlsConfig != nil {
		mtlsClient, err := mtls.NewClient(client.mtlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create mTLS client: %w", err)
		}

		client.mtlsClient = mtlsClient
		client.mtlsOwned = true
		client.mtlsEnabled = true
		client.httpClient.Transport = &http.Transport{
			TLSClientConfig: mtlsClient.GetTLSConfig(),
		}
	}

	return client, nil
}

// EventChan returns the event channel for receiving SSE events
func (c *Client) EventChan() <-chan Event {
	return c.eventCh
}

// ErrorChan returns the error channel
func (c *Client) ErrorChan() <-chan error {
	return c.errorCh
}

// Connect establishes the SSE connection
func (c *Client) Connect(ctx context.Context, path string) error {
	c.mu.Lock()
	c.isConnected = true
	c.mu.Unlock()

	go c.establishConnection(ctx, path)
	return nil
}

// establishConnection manages the SSE connection lifecycle
func (c *Client) establishConnection(ctx context.Context, path string) {
	defer func() {
		c.mu.Lock()
		c.isConnected = false
		c.mu.Unlock()
		select {
		case <-c.done:
		default:
			close(c.done)
		}
	}()

	attempt := 0
	delay := c.reconnectDelay

	for {
		select {
		case <-ctx.Done():
			c.shouldReconnect = false
			return
		default:
		}

		// Attempt to connect
		err := c.connectOnce(ctx, path)
		if err != nil {
			attempt++

			// Check if we should retry
			if attempt >= c.reconnectAttempts {
				select {
				case c.errorCh <- fmt.Errorf("max reconnect attempts reached: %w", err):
				default:
				}
				return
			}

			// Wait before retrying
			select {
			case <-time.After(delay):
				// Exponential backoff with jitter
				delay = time.Duration(float64(delay) * 1.5)
				if delay > c.maxReconnectDelay {
					delay = c.maxReconnectDelay
				}
				continue
			case <-ctx.Done():
				return
			}
		}

		// Connected successfully, reset attempt counter
		attempt = 0
		delay = c.reconnectDelay
	}
}

// connectOnce establishes a single SSE connection
func (c *Client) connectOnce(ctx context.Context, path string) error {
	url := c.baseURL + path

	// Prepare request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Add Last-Event-ID header if we have one
	c.mu.RLock()
	if c.lastEventID != "" {
		req.Header.Set("Last-Event-ID", c.lastEventID)
	}
	c.mu.RUnlock()

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	// Verify content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/event-stream" && contentType != "text/event-stream; charset=utf-8" {
		return fmt.Errorf("unexpected content type: %s", contentType)
	}

	stopHeartbeat := make(chan struct{})
	c.startHeartbeatMonitor(stopHeartbeat)
	defer close(stopHeartbeat)

	reader := bufio.NewReader(resp.Body)
	c.recordEvent()
	for {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Parse event
		event, err := c.parseEvent(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to parse event: %w", err)
		}

		if event.ID != "" {
			c.mu.Lock()
			c.lastEventID = event.ID
			c.mu.Unlock()
		}

		c.recordEvent()

		select {
		case c.eventCh <- event:
		case <-c.done:
			return nil
		}
	}
}

// parseEvent parses a single SSE event from the reader
func (c *Client) parseEvent(reader *bufio.Reader) (Event, error) {
	event := Event{
		Fields: make(map[string]interface{}),
	}

	// Read lines until empty line
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return event, err
		}

		line = strings.TrimRight(line, "\r\n")

		// Empty line indicates end of event
		if line == "" {
			break
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		field := parts[0]
		value := ""
		if len(parts) == 2 {
			value = strings.TrimLeft(parts[1], " ")
		}

		switch field {
		case "id":
			event.ID = value
		case "event":
			event.Event = value
		case "data":
			if event.Data != "" {
				event.Data += "\n"
			}
			event.Data += value
		case "retry":
			if duration, err := time.ParseDuration(value + "ms"); err == nil {
				event.Retry = duration
			}
		default:
			event.Fields[field] = value
		}
	}

	event.Data = strings.TrimSuffix(event.Data, "\n")

	// Handle event parsing
	if event.Event == "log" {
		// Parse JSON data
		var logData map[string]interface{}
		if err := json.Unmarshal([]byte(event.Data), &logData); err == nil {
			// Update fields with parsed data
			for k, v := range logData {
				event.Fields[k] = v
			}
		}
	}

	return event, nil
}

// Close closes the SSE connection
func (c *Client) Close() error {
	c.mu.Lock()
	c.shouldReconnect = false
	c.mu.Unlock()

	select {
	case <-c.done:
	default:
		close(c.done)
	}

	if c.mtlsClient != nil && c.mtlsOwned {
		c.mtlsClient.Close()
	}

	return nil
}

// IsConnected returns connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// GetLastEventID returns the last received event ID
func (c *Client) GetLastEventID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastEventID
}

func (c *Client) recordEvent() {
	c.mu.Lock()
	c.lastEventTime = time.Now()
	c.mu.Unlock()
}

func (c *Client) startHeartbeatMonitor(stop <-chan struct{}) {
	if c.heartbeatInterval <= 0 {
		return
	}

	ticker := time.NewTicker(c.heartbeatInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				c.mu.RLock()
				last := c.lastEventTime
				c.mu.RUnlock()
				if time.Since(last) >= c.heartbeatInterval {
					select {
					case c.eventCh <- Event{Event: "heartbeat"}:
					default:
					}
					c.recordEvent()
				}
			}
		}
	}()
}

// Reconnect forces a reconnection
func (c *Client) Reconnect(ctx context.Context, path string) error {
	if !c.IsConnected() {
		return c.Connect(ctx, path)
	}

	// Close existing connection
	c.Close()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Reconnect
	return c.Connect(ctx, path)
}
