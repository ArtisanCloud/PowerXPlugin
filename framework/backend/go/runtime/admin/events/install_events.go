package events

import (
	"log/slog"
	"time"
)

// InstallEventEmitter emits tenant notifications for installs/rollback.
type InstallEventEmitter struct {
	logger *slog.Logger
}

func NewInstallEventEmitter(logger *slog.Logger) *InstallEventEmitter {
	if logger == nil {
		logger = slog.Default()
	}
	return &InstallEventEmitter{logger: logger}
}

// EmitInstallStarted logs when a plugin installation begins
func (e *InstallEventEmitter) EmitInstallStarted(tenantID, pluginID, version string) {
	e.logger.Info("tenant.install.started",
		slog.String("tenant", tenantID),
		slog.String("plugin", pluginID),
		slog.String("version", version),
	)
}

// EmitInstallSuccess logs when a plugin installation completes successfully
func (e *InstallEventEmitter) EmitInstallSuccess(deploymentID, pluginID, version string, duration time.Duration) {
	e.logger.Info("tenant.install.success",
		slog.String("deploymentId", deploymentID),
		slog.String("plugin", pluginID),
		slog.String("version", version),
		slog.Duration("duration", duration),
	)
}

// EmitInstallFailed logs when a plugin installation fails
func (e *InstallEventEmitter) EmitInstallFailed(deploymentID, pluginID, version, reason string) {
	e.logger.Error("tenant.install.failed",
		slog.String("deploymentId", deploymentID),
		slog.String("plugin", pluginID),
		slog.String("version", version),
		slog.String("reason", reason),
	)
}

// EmitRollbackScheduled logs when an automatic rollback is scheduled
func (e *InstallEventEmitter) EmitRollbackScheduled(deploymentID string, rollbackIn time.Duration) {
	e.logger.Warn("tenant.install.rollback_scheduled",
		slog.String("deploymentId", deploymentID),
		slog.Duration("rollbackIn", rollbackIn),
	)
}

// EmitRollbackCompleted logs when a rollback operation completes
func (e *InstallEventEmitter) EmitRollbackCompleted(deploymentID, pluginID, version string, duration time.Duration, autoTriggered bool) {
	level := slog.LevelWarn
	if !autoTriggered {
		level = slog.Info
	}
	e.logger.Log(level, "tenant.install.rolled_back",
		slog.String("deploymentId", deploymentID),
		slog.String("plugin", pluginID),
		slog.String("version", version),
		slog.Duration("duration", duration),
		slog.Bool("autoTriggered", autoTriggered),
	)
}

// EmitRollbackCancelled logs when a rollback is manually cancelled
func (e *InstallEventEmitter) EmitRollbackCancelled(deploymentID string) {
	e.logger.Info("tenant.install.rollback_cancelled",
		slog.String("deploymentId", deploymentID),
	)
}
