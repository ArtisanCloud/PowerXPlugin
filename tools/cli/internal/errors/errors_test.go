package errors

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewError(t *testing.T) {
	err := NewError(ErrNetwork, "connection failed", WithCause(errors.New("original error")))

	if err.Type != ErrNetwork {
		t.Errorf("Expected type %v, got %v", ErrNetwork, err.Type)
	}

	if err.Message != "connection failed" {
		t.Errorf("Expected message 'connection failed', got %q", err.Message)
	}

	if err.Cause == nil {
		t.Error("Expected cause to be set")
	}
}

func TestNewErrorWithOptions(t *testing.T) {
	err := NewError(ErrBuild, "compilation failed",
		WithCause(errors.New("syntax error")),
		WithContext("file", "main.go"),
		WithRetryable(3),
		WithRecoverable())

	if err.Type != ErrBuild {
		t.Errorf("Expected type %v, got %v", ErrBuild, err.Type)
	}

	if err.Retryable != true {
		t.Error("Expected retryable to be true")
	}

	if err.MaxRetries != 3 {
		t.Errorf("Expected maxRetries 3, got %d", err.MaxRetries)
	}

	if err.Recoverable != true {
		t.Error("Expected recoverable to be true")
	}

	if ctxValue, ok := err.Context["file"]; !ok || ctxValue != "main.go" {
		t.Error("Expected context to have file=main.go")
	}
}

func TestErrorError(t *testing.T) {
	original := errors.New("original error")
	err := NewError(ErrAPI, "API call failed", WithCause(original))

	expected := "API: API call failed (caused by: original error)"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestErrorUnwrap(t *testing.T) {
	original := errors.New("original error")
	err := NewError(ErrNetwork, "connection failed", WithCause(original))

	if !errors.Is(err, original) {
		t.Error("Expected unwrap to return original error")
	}
}

func TestRetryable(t *testing.T) {
	// Test with retryable error
	retryableErr := NewError(ErrNetwork, "connection failed", WithRetryable(3))
	if !Retryable(retryableErr) {
		t.Error("Expected error to be retryable")
	}

	// Test with non-retryable error
	nonRetryableErr := NewError(ErrUser, "invalid input")
	if Retryable(nonRetryableErr) {
		t.Error("Expected error to not be retryable")
	}
}

func TestMaxRetries(t *testing.T) {
	// Test with custom max retries
	err := NewError(ErrNetwork, "connection failed", WithRetryable(5))
	if MaxRetries(err) != 5 {
		t.Errorf("Expected max retries 5, got %d", MaxRetries(err))
	}

	// Test with default max retries
	err2 := NewError(ErrTimeout, "timeout")
	if MaxRetries(err2) != RetryConfig.MaxAttempts {
		t.Errorf("Expected max retries %d, got %d", RetryConfig.MaxAttempts, MaxRetries(err2))
	}
}

func TestRetry(t *testing.T) {
	ctx := context.Background()

	// Test successful operation
	attempt, err := Retry(ctx, func() error {
		return nil
	}, nil)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempt != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempt)
	}

	// Test non-retryable error
	nonRetryable := NewError(ErrUser, "invalid input")
	attempt, err = Retry(ctx, func() error {
		return nonRetryable
	}, nil)

	if err != nonRetryable {
		t.Errorf("Expected error %v, got %v", nonRetryable, err)
	}

	if attempt != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempt)
	}

	// Test retryable error with max attempts
	retryable := NewError(ErrNetwork, "connection failed", WithRetryable(2))
	callCount := 0
	attempt, err = Retry(ctx, func() error {
		callCount++
		return retryable
	}, &RetryPolicy{
		MaxAttempts: 3,
	})

	if err != retryable {
		t.Errorf("Expected error %v, got %v", retryable, err)
	}

	if attempt != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempt)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

func TestRetryWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test context cancellation
	attempt, err := Retry(ctx, func() error {
		return NewError(ErrNetwork, "connection failed")
	}, &RetryPolicy{
		MaxAttempts:  10,
		InitialDelay: 200 * time.Millisecond,
	})

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	if attempt >= 10 {
		t.Errorf("Expected fewer than 10 attempts due to context timeout, got %d", attempt)
	}
}

func TestRecoveryHandler(t *testing.T) {
	handler := NewRecoveryHandler()

	// Test successful recovery
	err := handler.Recover("panic message", "testFunction")
	if err == nil {
		t.Error("Expected error from panic recovery")
	}

	// Test nil panic
	err = handler.Recover(nil, "testFunction2")
	if err != nil {
		t.Errorf("Expected no error for nil panic, got %v", err)
	}
}

func TestWithRecovery(t *testing.T) {
	handler := NewRecoveryHandler()
	callCount := 0

	// Test with panic
	secureFunc := WithRecoveryFunc(func() error {
		panic("test panic")
	}, handler, "test")

	err := secureFunc()
	if err == nil {
		t.Error("Expected error from panic")
	}

	// Test without panic
	callCount = 0
	secureFunc2 := WithRecoveryFunc(func() error {
		callCount++
		return nil
	}, handler, "test2")

	err = secureFunc2()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected call count 1, got %d", callCount)
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Second)

	// Test initial state
	if !cb.Allow() {
		t.Error("Expected circuit breaker to allow operation in closed state")
	}

	// Test failure threshold
	for i := 0; i < 3; i++ {
		cb.OnFailure()
	}

	if cb.State() != StateOpen {
		t.Errorf("Expected state Open, got %v", cb.State())
	}

	// Test open state blocks operations
	if cb.Allow() {
		t.Error("Expected circuit breaker to block operations in open state")
	}

	// Test transition to half-open after timeout
	time.Sleep(1100 * time.Millisecond)
	if !cb.Allow() {
		t.Error("Expected circuit breaker to allow operation in half-open state")
	}

	// Test success in half-open
	cb.OnSuccess()
	if cb.State() != StateClosed {
		t.Errorf("Expected state Closed after success in half-open, got %v", cb.State())
	}
}

func TestBackupManager(t *testing.T) {
	bm := NewBackupManager(2)

	// Create backups
	bm.CreateBackup("backup1", map[string]interface{}{"key": "value1"})
	bm.CreateBackup("backup2", map[string]interface{}{"key": "value2"})
	bm.CreateBackup("backup3", map[string]interface{}{"key": "value3"})

	// Should only keep 2 backups
	if len(bm.ListBackups()) > 2 {
		t.Errorf("Expected at most 2 backups, got %d", len(bm.ListBackups()))
	}

	// Test retrieval
	backup, ok := bm.GetBackup("backup2")
	if !ok {
		t.Error("Expected to find backup2")
	}

	if backup == nil || backup.Data["key"] != "value2" {
		t.Error("Expected backup2 data to be correct")
	}

	// Test deletion
	bm.DeleteBackup("backup2")
	_, ok = bm.GetBackup("backup2")
	if ok {
		t.Error("Expected backup2 to be deleted")
	}
}

func TestHealthChecker(t *testing.T) {
	hc := NewHealthChecker()

	// Add a simple check
	hc.AddCheck("test", func() HealthCheck {
		return HealthCheck{
			Status:  "healthy",
			Message: "test passed",
		}
	})

	// Check status
	if status := hc.GetStatus(); status != "healthy" {
		t.Errorf("Expected status 'healthy', got %q", status)
	}

	// Add a failing check
	hc.AddCheck("critical", func() HealthCheck {
		return HealthCheck{
			Status:  "unhealthy",
			Message: "critical check failed",
		}
	})

	// Status should be critical
	if status := hc.GetStatus(); status != "critical" {
		t.Errorf("Expected status 'critical', got %q", status)
	}
}

func TestFailureAnalysis(t *testing.T) {
	fa := NewFailureAnalysis()

	// Record some failures
	fa.RecordFailure(
		NewError(ErrNetwork, "connection failed"),
		map[string]interface{}{"host": "localhost"},
		0,
	)

	fa.RecordFailure(
		NewError(ErrTimeout, "request timeout"),
		map[string]interface{}{"url": "http://example.com"},
		1,
	)

	// Get recommendations
	recs := fa.GetRecommendations(ErrNetwork)
	if len(recs) == 0 {
		t.Error("Expected recommendations for network errors")
	}

	// Get recent failures
	failures := fa.GetRecentFailures(2)
	if len(failures) != 2 {
		t.Errorf("Expected 2 failures, got %d", len(failures))
	}
}

func TestErrorClassifier(t *testing.T) {
	classifier := NewErrorClassifier()

	classifier.AddPattern(ErrNetwork, "connection refused")
	classifier.AddPattern(ErrTimeout, "timeout")
	classifier.AddPattern(ErrAuth, "unauthorized")

	// Test network error classification
	err := errors.New("connection refused to localhost:8080")
	typ := classifier.Classify(err)
	if typ != ErrNetwork {
		t.Errorf("Expected type %v, got %v", ErrNetwork, typ)
	}

	// Test timeout error classification
	err2 := errors.New("request timeout after 30s")
	typ2 := classifier.Classify(err2)
	if typ2 != ErrTimeout {
		t.Errorf("Expected type %v, got %v", ErrTimeout, typ2)
	}

	// Test auth error classification
	err3 := errors.New("unauthorized access")
	typ3 := classifier.Classify(err3)
	if typ3 != ErrAuth {
		t.Errorf("Expected type %v, got %v", ErrAuth, typ3)
	}
}
