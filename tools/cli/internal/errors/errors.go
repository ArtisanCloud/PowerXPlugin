package errors

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// Network errors
	ErrNetwork    ErrorType = "NETWORK"
	ErrTimeout    ErrorType = "TIMEOUT"
	ErrConnection ErrorType = "CONNECTION"

	// API errors
	ErrAPI      ErrorType = "API"
	ErrAuth     ErrorType = "AUTH"
	ErrNotFound ErrorType = "NOT_FOUND"
	ErrConflict ErrorType = "CONFLICT"

	// Build errors
	ErrBuild       ErrorType = "BUILD"
	ErrCompilation ErrorType = "COMPILATION"
	ErrValidation  ErrorType = "VALIDATION"

	// File system errors
	ErrFileSystem ErrorType = "FILESYSTEM"
	ErrPermission ErrorType = "PERMISSION"
	ErrDiskSpace  ErrorType = "DISK_SPACE"

	// Configuration errors
	ErrConfig  ErrorType = "CONFIG"
	ErrInvalid ErrorType = "INVALID"

	// System errors
	ErrSystem   ErrorType = "SYSTEM"
	ErrResource ErrorType = "RESOURCE"
	ErrMemory   ErrorType = "MEMORY"

	// User errors
	ErrUser      ErrorType = "USER"
	ErrCancelled ErrorType = "CANCELLED"
)

// Error represents a structured error
type Error struct {
	Type        ErrorType              `json:"type"`
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Cause       error                  `json:"cause,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Stack       string                 `json:"stack,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Recoverable bool                   `json:"recoverable"`
	Retryable   bool                   `json:"retryable"`
	MaxRetries  int                    `json:"maxRetries,omitempty"`
}

// NewError creates a new Error
func NewError(errType ErrorType, message string, opts ...Option) *Error {
	e := &Error{
		Type:      errType,
		Code:      string(errType),
		Message:   message,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the cause error
func (e *Error) Unwrap() error {
	return e.Cause
}

// Option is a function that modifies an Error
type Option func(*Error)

// WithCause sets the cause error
func WithCause(err error) Option {
	return func(e *Error) {
		e.Cause = err
	}
}

// WithContext adds context to the error
func WithContext(key string, value interface{}) Option {
	return func(e *Error) {
		e.Context[key] = value
	}
}

// WithRetryable marks the error as retryable
func WithRetryable(maxRetries int) Option {
	return func(e *Error) {
		e.Retryable = true
		e.MaxRetries = maxRetries
	}
}

// WithRecoverable marks the error as recoverable
func WithRecoverable() Option {
	return func(e *Error) {
		e.Recoverable = true
	}
}

// WithStack adds stack trace
func WithStack(stack string) Option {
	return func(e *Error) {
		e.Stack = stack
	}
}

// RetryPolicy defines the retry policy
type RetryPolicy struct {
	MaxAttempts    int           `json:"maxAttempts"`
	InitialDelay   time.Duration `json:"initialDelay"`
	MaxDelay       time.Duration `json:"maxDelay"`
	BackoffFactor  float64       `json:"backoffFactor"`
	Jitter         bool          `json:"jitter"`
	RetryableTypes []ErrorType   `json:"retryableTypes"`
}

// RetryConfig default retry configuration
var RetryConfig = RetryPolicy{
	MaxAttempts:   3,
	InitialDelay:  1 * time.Second,
	MaxDelay:      30 * time.Second,
	BackoffFactor: 2.0,
	Jitter:        true,
	RetryableTypes: []ErrorType{
		ErrNetwork,
		ErrTimeout,
		ErrConnection,
		ErrAPI,
		ErrSystem,
	},
}

// Retryable checks if an error is retryable
func Retryable(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		if e.Retryable {
			return true
		}
		for _, t := range RetryConfig.RetryableTypes {
			if e.Type == t {
				return true
			}
		}
		return false
	}

	return false
}

// MaxRetries returns the maximum number of retries for an error
func MaxRetries(err error) int {
	var e *Error
	if errors.As(err, &e) {
		if e.MaxRetries > 0 {
			return e.MaxRetries
		}
	}
	return RetryConfig.MaxAttempts
}

// Retry executes a function with retry logic
func Retry(ctx context.Context, fn func() error, policy *RetryPolicy) (int, error) {
	if policy == nil {
		policy = &RetryConfig
	}

	delay := policy.InitialDelay
	attempt := 0

	for attempt < policy.MaxAttempts {
		attempt++

		err := fn()
		if err == nil {
			return attempt, nil
		}

		// Check if error is retryable
		if !Retryable(err) {
			return attempt, err
		}

		// Check if we've exceeded max retries
		if attempt >= policy.MaxAttempts {
			return attempt, err
		}

		// Check context
		select {
		case <-ctx.Done():
			return attempt, ctx.Err()
		default:
		}

		// Wait before retrying
		waitTime := delay
		if policy.Jitter {
			// Add jitter (±25%)
			factor := 0.75 + (float64(time.Now().UnixNano()%100) / 100.0)
			waitTime = time.Duration(float64(delay) * factor)
		}

		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return attempt, ctx.Err()
		}

		// Exponential backoff
		delay = time.Duration(float64(delay) * policy.BackoffFactor)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
	}

	return attempt, fmt.Errorf("max retries exceeded")
}

// RecoveryHandler handles panic recovery
type RecoveryHandler struct {
	recoveries int64
	mu         struct {
		sync.Mutex
		recoveries map[string]int64
	}
}

// NewRecoveryHandler creates a new recovery handler
func NewRecoveryHandler() *RecoveryHandler {
	return &RecoveryHandler{
		recoveries: 0,
		mu: struct {
			sync.Mutex
			recoveries map[string]int64
		}{
			recoveries: make(map[string]int64),
		},
	}
}

// Recover recovers from a panic
func (r *RecoveryHandler) Recover(panicValue interface{}, functionName string) (err error) {
	r.recoveries++
	r.mu.Lock()
	r.mu.recoveries[functionName]++
	r.mu.Unlock()

	if panicValue == nil {
		return nil
	}

	// Convert panic to error
	switch v := panicValue.(type) {
	case error:
		return NewError(ErrSystem, "panic recovered", WithCause(v), WithContext("function", functionName))
	case string:
		return NewError(ErrSystem, "panic recovered", WithCause(errors.New(v)), WithContext("function", functionName))
	default:
		return NewError(ErrSystem, "panic recovered", WithCause(fmt.Errorf("%v", v)), WithContext("function", functionName))
	}
}

// WithRecovery wraps a function with panic recovery
func WithRecovery(fn func() error, handler *RecoveryHandler, functionName string) func() error {
	return func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = handler.Recover(r, functionName)
			}
		}()

		return fn()
	}
}

// WithRecoveryFunc wraps a function with panic recovery
func WithRecoveryFunc(fn func() error, handler *RecoveryHandler, functionName string) func() error {
	return WithRecovery(fn, handler, functionName)
}

// ErrorClassifier classifies errors by type
type ErrorClassifier struct {
	patterns map[ErrorType][]string
}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{
		patterns: make(map[ErrorType][]string),
	}
}

// AddPattern adds a pattern for error classification
func (c *ErrorClassifier) AddPattern(errType ErrorType, pattern string) {
	c.patterns[errType] = append(c.patterns[errType], pattern)
}

// Classify classifies an error by type
func (c *ErrorClassifier) Classify(err error) ErrorType {
	errMsg := err.Error()

	for errType, patterns := range c.patterns {
		for _, pattern := range patterns {
			if contains(errMsg, pattern) {
				return errType
			}
		}
	}

	// Default classification based on error content
	switch {
	case contains(errMsg, "network"),
		contains(errMsg, "connection refused"),
		contains(errMsg, "dial tcp"):
		return ErrNetwork
	case contains(errMsg, "timeout"),
		contains(errMsg, "deadline exceeded"):
		return ErrTimeout
	case contains(errMsg, "build"),
		contains(errMsg, "compilation"):
		return ErrBuild
	case contains(errMsg, "permission denied"):
		return ErrPermission
	case contains(errMsg, "no space left"),
		contains(errMsg, "disk full"):
		return ErrDiskSpace
	case contains(errMsg, "unauthorized"),
		contains(errMsg, "authentication"):
		return ErrAuth
	case contains(errMsg, "not found"):
		return ErrNotFound
	case contains(errMsg, "conflict"):
		return ErrConflict
	case contains(errMsg, "cancelled"),
		contains(errMsg, "canceled"):
		return ErrCancelled
	default:
		return ErrSystem
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

// equalFold compares two strings case-insensitively
func equalFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if lower(s[i]) != lower(t[i]) {
			return false
		}
	}
	return true
}

// lower converts a byte to lowercase
func lower(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// WrapError wraps an error with additional context
func WrapError(err error, errType ErrorType, message string, opts ...Option) *Error {
	if err == nil {
		return NewError(errType, message, opts...)
	}

	e := NewError(errType, message, append(opts, WithCause(err))...)
	return e
}

// IsRetryable checks if an error should be retried
func IsRetryable(err error) bool {
	return Retryable(err)
}

// IsRecoverable checks if an error is recoverable
func IsRecoverable(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Recoverable
	}
	return true // Assume recoverable if not a structured error
}
