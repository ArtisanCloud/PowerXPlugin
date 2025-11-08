package services

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/observability"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/admin/events"
)

// DeploymentState represents the current state of a plugin deployment
type DeploymentState string

const (
	StatePending     DeploymentState = "pending"
	StateInstalling  DeploymentState = "installing"
	StateSuccess     DeploymentState = "success"
	StateFailed      DeploymentState = "failed"
	StateRollingBack DeploymentState = "rolling_back"
	StateRolledBack  DeploymentState = "rolled_back"
)

// DeploymentStatus tracks the status of a plugin deployment
type DeploymentStatus struct {
	DeploymentID string          `json:"deploymentId"`
	TenantID     string          `json:"tenantId"`
	PluginID     string          `json:"pluginId"`
	Version      string          `json:"version"`
	State        DeploymentState `json:"status"`
	PreviousID   string          `json:"previousDeploymentId,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	AutoRollback bool            `json:"autoRollback"`
	RollbackAt   time.Time       `json:"rollbackAt,omitempty"`
}

// PluginDeployer orchestrates install + rollback operations for tenants.
type PluginDeployer struct {
	logger         *slog.Logger
	eventEmitter   *events.InstallEventEmitter
	deployments    map[string]*DeploymentStatus
	deploymentsMu  sync.RWMutex
	rollbackTimer  map[string]*time.Timer
	rollbackMu     sync.Mutex
	autoRollbackAt time.Duration
}

// NewPluginDeployer creates a new PluginDeployer with 5-minute auto rollback
func NewPluginDeployer(logger *slog.Logger) *PluginDeployer {
	if logger == nil {
		logger = slog.Default()
	}
	return &PluginDeployer{
		logger:         logger,
		eventEmitter:   events.NewInstallEventEmitter(logger),
		deployments:    make(map[string]*DeploymentStatus),
		rollbackTimer:  make(map[string]*time.Timer),
		autoRollbackAt: 5 * time.Minute, // 5-minute auto rollback SLA
	}
}

// Deploy initiates a plugin installation with monitoring
func (d *PluginDeployer) Deploy(tenantID, pluginID, version string, previousID string) *DeploymentStatus {
	deploymentID := fmt.Sprintf("%s:%s:%s", tenantID, pluginID, version)

	d.deploymentsMu.Lock()
	defer d.deploymentsMu.Unlock()

	// Check if deployment already exists
	if status, exists := d.deployments[deploymentID]; exists {
		d.logger.Info("deployment already exists", slog.String("deploymentId", deploymentID), slog.String("status", string(status.State)))
		return status
	}

	// Create new deployment status
	status := &DeploymentStatus{
		DeploymentID: deploymentID,
		TenantID:     tenantID,
		PluginID:     pluginID,
		Version:      version,
		State:        StatePending,
		PreviousID:   previousID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		AutoRollback: true,
		RollbackAt:   time.Now().Add(d.autoRollbackAt),
	}

	d.deployments[deploymentID] = status

	// Start the deployment process
	go d.startDeployment(status)

	d.logger.Info("plugin deploy initiated",
		slog.String("deploymentId", deploymentID),
		slog.String("pluginId", pluginID),
		slog.String("version", version),
	)

	return status
}

// GetStatus returns the current status of a deployment
func (d *PluginDeployer) GetStatus(deploymentID string) (*DeploymentStatus, bool) {
	d.deploymentsMu.RLock()
	defer d.deploymentsMu.RUnlock()

	status, exists := d.deployments[deploymentID]
	return status, exists
}

// startDeployment simulates the deployment process
func (d *PluginDeployer) startDeployment(status *DeploymentStatus) {
	// Update state to installing
	d.updateStatus(status.DeploymentID, StateInstalling)

	// Simulate installation process (in real implementation, this would call the actual installation logic)
	// For now, we'll randomly succeed or fail
	success := true // In a real implementation, this would be determined by actual installation result

	if success {
		d.markSuccess(status.DeploymentID)
	} else {
		d.markFailed(status.DeploymentID, "installation failed")
	}
}

// markSuccess marks the deployment as successful and cancels auto-rollback
func (d *PluginDeployer) markSuccess(deploymentID string) {
	d.deploymentsMu.Lock()
	status, exists := d.deployments[deploymentID]
	if !exists {
		d.deploymentsMu.Unlock()
		return
	}

	status.State = StateSuccess
	status.UpdatedAt = time.Now()
	status.AutoRollback = false

	// Cancel auto-rollback timer
	if timer, exists := d.rollbackTimer[deploymentID]; exists {
		timer.Stop()
		delete(d.rollbackTimer, deploymentID)
	}

	d.deploymentsMu.Unlock()

	// Emit success event
	d.eventEmitter.EmitInstallSuccess(deploymentID, status.PluginID, status.Version, time.Since(status.CreatedAt))

	// Record deployment status metric
	observability.IncrementDeploymentStatus("success")

	d.logger.Info("plugin deploy successful",
		slog.String("deploymentId", deploymentID),
		slog.Duration("duration", time.Since(status.CreatedAt)),
	)
}

// markFailed marks the deployment as failed and triggers auto-rollback
func (d *PluginDeployer) markFailed(deploymentID, reason string) {
	d.deploymentsMu.Lock()
	status, exists := d.deployments[deploymentID]
	if !exists {
		d.deploymentsMu.Unlock()
		return
	}
	status.State = StateFailed
	status.UpdatedAt = time.Now()
	d.deploymentsMu.Unlock()

	// Emit failed event
	d.eventEmitter.EmitInstallFailed(deploymentID, status.PluginID, status.Version, reason)

	// Record deployment status metric
	observability.IncrementDeploymentStatus("failed")

	d.logger.Error("plugin deploy failed",
		slog.String("deploymentId", deploymentID),
		slog.String("reason", reason),
	)

	// Trigger rollback
	d.triggerRollback(deploymentID)
}

// updateStatus atomically updates the deployment status
func (d *PluginDeployer) updateStatus(deploymentID string, newState DeploymentState) {
	d.deploymentsMu.Lock()
	defer d.deploymentsMu.Unlock()

	if status, exists := d.deployments[deploymentID]; exists {
		status.State = newState
		status.UpdatedAt = time.Now()

		// Record status change metric (skip pending to avoid double counting)
		if newState != StatePending {
			observability.IncrementDeploymentStatus(string(newState))
		}
	}
}

// triggerRollback initiates the rollback process with timer
func (d *PluginDeployer) triggerRollback(deploymentID string) {
	d.deploymentsMu.Lock()
	status, exists := d.deployments[deploymentID]
	if !exists {
		d.deploymentsMu.Unlock()
		return
	}

	// Check if auto-rollback is enabled
	if !status.AutoRollback {
		d.deploymentsMu.Unlock()
		return
	}

	status.State = StateRollingBack
	status.UpdatedAt = time.Now()
	d.deploymentsMu.Unlock()

	// Emit rollback scheduled event
	d.eventEmitter.EmitRollbackScheduled(deploymentID, d.autoRollbackAt)

	d.rollbackMu.Lock()
	timer := time.AfterFunc(d.autoRollbackAt, func() {
		d.performRollback(deploymentID)
	})
	d.rollbackTimer[deploymentID] = timer
	d.rollbackMu.Unlock()

	d.logger.Warn("rollback scheduled",
		slog.String("deploymentId", deploymentID),
		slog.Duration("rollbackIn", d.autoRollbackAt),
	)
}

// performRollback executes the actual rollback
func (d *PluginDeployer) performRollback(deploymentID string) {
	d.rollbackMu.Lock()
	if timer, exists := d.rollbackTimer[deploymentID]; exists {
		timer.Stop()
		delete(d.rollbackTimer, deploymentID)
	}
	d.rollbackMu.Unlock()

	d.deploymentsMu.Lock()
	status, exists := d.deployments[deploymentID]
	if !exists {
		d.deploymentsMu.Unlock()
		return
	}
	d.deploymentsMu.Unlock()

	d.logger.Warn("executing rollback",
		slog.String("deploymentId", deploymentID),
		slog.String("pluginId", status.PluginID),
		slog.String("version", status.Version),
	)

	// Calculate duration for metrics
	rollbackDuration := time.Since(status.CreatedAt)
	trigger := "auto" // Default to auto for performRollback
	if rollbackDuration > d.autoRollbackAt {
		trigger = "auto" // Auto-triggered after timeout
	}

	// Simulate rollback operation
	time.Sleep(100 * time.Millisecond) // Simulate rollback work

	// Update status
	d.deploymentsMu.Lock()
	if status, exists := d.deployments[deploymentID]; exists {
		status.State = StateRolledBack
		status.UpdatedAt = time.Now()
	}
	d.deploymentsMu.Unlock()

	// Record rollback latency metric
	observability.RecordRollbackLatency(rollbackDuration.Seconds(), status.TenantID, status.PluginID, trigger)

	// Emit rollback completed event (auto-triggered)
	d.eventEmitter.EmitRollbackCompleted(deploymentID, status.PluginID, status.Version, rollbackDuration, true)

	d.logger.Info("rollback completed",
		slog.String("deploymentId", deploymentID),
	)
}

// Rollback manually triggers a rollback for a deployment
func (d *PluginDeployer) Rollback(deploymentID string) error {
	d.deploymentsMu.RLock()
	status, exists := d.deployments[deploymentID]
	d.deploymentsMu.RUnlock()

	if !exists {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	if status.State == StateRolledBack {
		return fmt.Errorf("deployment already rolled back: %s", deploymentID)
	}

	d.logger.Warn("manual rollback triggered", slog.String("deploymentId", deploymentID))

	// Record metric for manual rollback
	rollbackDuration := time.Since(status.CreatedAt)
	observability.RecordRollbackLatency(rollbackDuration.Seconds(), status.TenantID, status.PluginID, "manual")

	// Perform rollback
	d.performRollback(deploymentID)

	// Emit rollback completed event (manual)
	d.eventEmitter.EmitRollbackCompleted(deploymentID, status.PluginID, status.Version, rollbackDuration, false)

	return nil
}

// CancelRollback cancels an automatic rollback timer
func (d *PluginDeployer) CancelRollback(deploymentID string) error {
	d.rollbackMu.Lock()
	defer d.rollbackMu.Unlock()

	if timer, exists := d.rollbackTimer[deploymentID]; exists {
		timer.Stop()
		delete(d.rollbackTimer, deploymentID)

		// Emit rollback cancelled event
		d.eventEmitter.EmitRollbackCancelled(deploymentID)

		d.logger.Info("rollback cancelled", slog.String("deploymentId", deploymentID))
		return nil
	}

	return fmt.Errorf("no rollback timer found for deployment: %s", deploymentID)
}

// GetAllDeployments returns all current deployments
func (d *PluginDeployer) GetAllDeployments() map[string]*DeploymentStatus {
	d.deploymentsMu.RLock()
	defer d.deploymentsMu.RUnlock()

	// Return a copy to avoid race conditions
	deployments := make(map[string]*DeploymentStatus, len(d.deployments))
	for k, v := range d.deployments {
		deployments[k] = v
	}
	return deployments
}
