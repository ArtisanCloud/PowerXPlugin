package events

import "log/slog"

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

func (e *InstallEventEmitter) EmitInstallStarted(tenantID, pluginID, version string) {
	e.logger.Info("tenant.install.started", slog.String("tenant", tenantID), slog.String("plugin", pluginID), slog.String("version", version))
}

func (e *InstallEventEmitter) EmitRollbackCompleted(tenantID, pluginID, version string) {
	e.logger.Warn("tenant.install.rolled_back", slog.String("tenant", tenantID), slog.String("plugin", pluginID), slog.String("version", version))
}
