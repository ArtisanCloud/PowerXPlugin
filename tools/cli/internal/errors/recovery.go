package errors

import (
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	state       CircuitBreakerState
	mu          sync.RWMutex
	failures    int
	lastFailure time.Time

	// Configuration
	failureThreshold int           `json:"failureThreshold"`
	resetTimeout     time.Duration `json:"resetTimeout"`
	halfOpenMax      int           `json:"halfOpenMax"`
	successes        int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 5
	}
	if resetTimeout <= 0 {
		resetTimeout = 60 * time.Second
	}

	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		halfOpenMax:      1,
	}
}

// Allow returns true if the operation should be allowed
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			return true
		}
		return false

	case StateHalfOpen:
		return cb.successes < cb.halfOpenMax
	}

	return false
}

// OnSuccess is called when an operation succeeds
func (cb *CircuitBreaker) OnSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Reset failure count on success
		cb.failures = 0

	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.halfOpenMax {
			// Transition to closed
			cb.state = StateClosed
			cb.failures = 0
		}
	}
}

// OnFailure is called when an operation fails
func (cb *CircuitBreaker) OnFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.failureThreshold {
			cb.state = StateOpen
			cb.lastFailure = time.Now()
		}

	case StateHalfOpen:
		// Go back to open on failure
		cb.state = StateOpen
		cb.lastFailure = time.Now()
	}
}

// State returns the current state
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// FailureCount returns the current failure count
func (cb *CircuitBreaker) FailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastFailure = time.Time{}
}

// BackupManager manages backup operations
type BackupManager struct {
	mu         sync.RWMutex
	backups    map[string]*Backup
	maxBackups int
}

// Backup represents a backup operation
type Backup struct {
	ID        string                 `json:"id"`
	CreatedAt time.Time              `json:"createdAt"`
	Data      map[string]interface{} `json:"data"`
}

// NewBackupManager creates a new backup manager
func NewBackupManager(maxBackups int) *BackupManager {
	if maxBackups <= 0 {
		maxBackups = 10
	}

	return &BackupManager{
		backups:    make(map[string]*Backup),
		maxBackups: maxBackups,
	}
}

// CreateBackup creates a backup
func (bm *BackupManager) CreateBackup(id string, data map[string]interface{}) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.backups[id] = &Backup{
		ID:        id,
		CreatedAt: time.Now(),
		Data:      data,
	}

	// Clean up old backups
	if len(bm.backups) > bm.maxBackups {
		bm.cleanup()
	}
}

// GetBackup retrieves a backup
func (bm *BackupManager) GetBackup(id string) (*Backup, bool) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	backup, ok := bm.backups[id]
	return backup, ok
}

// DeleteBackup deletes a backup
func (bm *BackupManager) DeleteBackup(id string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	delete(bm.backups, id)
}

// ListBackups lists all backups
func (bm *BackupManager) ListBackups() []*Backup {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	backups := make([]*Backup, 0, len(bm.backups))
	for _, backup := range bm.backups {
		backups = append(backups, backup)
	}

	return backups
}

func (bm *BackupManager) cleanup() {
	// Simple cleanup: delete oldest backups
	type backupInfo struct {
		id        string
		createdAt time.Time
	}

	infos := make([]backupInfo, 0, len(bm.backups))
	for id, backup := range bm.backups {
		infos = append(infos, backupInfo{id: id, createdAt: backup.CreatedAt})
	}

	// Sort by creation time (oldest first)
	// Using simple bubble sort for small lists
	for i := 0; i < len(infos); i++ {
		for j := i + 1; j < len(infos); j++ {
			if infos[i].createdAt.After(infos[j].createdAt) {
				infos[i], infos[j] = infos[j], infos[i]
			}
		}
	}

	// Delete oldest backups
	toDelete := len(infos) - bm.maxBackups
	for i := 0; i < toDelete; i++ {
		delete(bm.backups, infos[i].id)
	}
}

// HealthChecker checks the health of a service
type HealthChecker struct {
	mu        sync.RWMutex
	checks    map[string]HealthCheck
	lastCheck time.Time
	status    string
}

// HealthCheck represents a health check
type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      string                 `json:"status"`
	Message     string                 `json:"message"`
	LastChecked time.Time              `json:"lastChecked"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]HealthCheck),
		status: "unknown",
	}
}

// AddCheck adds a health check
func (hc *HealthChecker) AddCheck(name string, checkFunc func() HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[name] = HealthCheck{
		Name: name,
	}

	// Run the check
	hc.runCheck(name, checkFunc)
	hc.updateOverallStatus()
}

// RunChecks runs all health checks
func (hc *HealthChecker) RunChecks(checkFuncs map[string]func() HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	for name, checkFunc := range checkFuncs {
		hc.runCheck(name, checkFunc)
	}

	hc.lastCheck = time.Now()
	hc.updateOverallStatus()
}

func (hc *HealthChecker) runCheck(name string, checkFunc func() HealthCheck) {
	check := checkFunc()
	check.Name = name
	check.LastChecked = time.Now()

	hc.checks[name] = check
}

func (hc *HealthChecker) updateOverallStatus() {
	allHealthy := true

	for _, check := range hc.checks {
		if check.Status != "healthy" {
			allHealthy = false
			break
		}
	}

	if allHealthy {
		hc.status = "healthy"
	} else {
		// Check if any critical checks are failing
		for name, check := range hc.checks {
			if name == "critical" && check.Status != "healthy" {
				hc.status = "critical"
				return
			}
		}
		hc.status = "degraded"
	}
}

// GetStatus returns the overall health status
func (hc *HealthChecker) GetStatus() string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.status
}

// GetChecks returns all health checks
func (hc *HealthChecker) GetChecks() map[string]HealthCheck {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	checks := make(map[string]HealthCheck)
	for name, check := range hc.checks {
		checks[name] = check
	}

	return checks
}

// FailureAnalysis analyzes failures and provides recommendations
type FailureAnalysis struct {
	mu              sync.RWMutex
	failures        []FailureRecord
	recommendations map[string][]string
}

// FailureRecord represents a failure record
type FailureRecord struct {
	Timestamp  time.Time              `json:"timestamp"`
	Error      *Error                 `json:"error"`
	Context    map[string]interface{} `json:"context"`
	RetryCount int                    `json:"retryCount"`
}

// NewFailureAnalysis creates a new failure analyzer
func NewFailureAnalysis() *FailureAnalysis {
	return &FailureAnalysis{
		failures:        make([]FailureRecord, 0),
		recommendations: make(map[string][]string),
	}
}

// RecordFailure records a failure
func (fa *FailureAnalysis) RecordFailure(err *Error, context map[string]interface{}, retryCount int) {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	record := FailureRecord{
		Timestamp:  time.Now(),
		Error:      err,
		Context:    context,
		RetryCount: retryCount,
	}

	fa.failures = append(fa.failures, record)

	// Keep only last 100 failures
	if len(fa.failures) > 100 {
		fa.failures = fa.failures[1:]
	}

	// Generate recommendations
	fa.generateRecommendations()
}

func (fa *FailureAnalysis) generateRecommendations() {
	// Simple recommendation system
	// In a real implementation, this would be more sophisticated

	for _, record := range fa.failures {
		switch record.Error.Type {
		case ErrNetwork:
			fa.recommendations[string(ErrNetwork)] = []string{
				"Check network connectivity",
				"Verify firewall settings",
				"Check DNS resolution",
			}
		case ErrTimeout:
			fa.recommendations[string(ErrTimeout)] = []string{
				"Increase timeout values",
				"Check network latency",
				"Reduce request payload size",
			}
		case ErrAuth:
			fa.recommendations[string(ErrAuth)] = []string{
				"Verify authentication credentials",
				"Check token expiration",
				"Ensure proper permissions",
			}
		case ErrDiskSpace:
			fa.recommendations[string(ErrDiskSpace)] = []string{
				"Free up disk space",
				"Clean temporary files",
				"Move data to another disk",
			}
		}
	}
}

// GetRecommendations returns recommendations for an error type
func (fa *FailureAnalysis) GetRecommendations(errType ErrorType) []string {
	fa.mu.RLock()
	defer fa.mu.RUnlock()

	return fa.recommendations[string(errType)]
}

// GetRecentFailures returns recent failures
func (fa *FailureAnalysis) GetRecentFailures(count int) []FailureRecord {
	fa.mu.RLock()
	defer fa.mu.RUnlock()

	if count > len(fa.failures) {
		count = len(fa.failures)
	}

	failures := make([]FailureRecord, count)
	copy(failures, fa.failures[len(fa.failures)-count:])

	return failures
}
