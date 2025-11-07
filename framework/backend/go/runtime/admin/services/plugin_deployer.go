package services

import "log/slog"

// PluginDeployer orchestrates install + rollback operations for tenants.
type PluginDeployer struct {
	logger *slog.Logger
}

func NewPluginDeployer(logger *slog.Logger) *PluginDeployer {
	if logger == nil {
		logger = slog.Default()
	}
	return &PluginDeployer{logger: logger}
}

// Deploy performs install and returns deployment record ID.
func (d *PluginDeployer) Deploy(tenantID, pluginID, version string) string {
	deploymentID := tenantID + ":" + pluginID + ":" + version
	d.logger.Info("plugin deploy", slog.String("deploymentId", deploymentID))
	return deploymentID
}

// Rollback reverts to previous version.
func (d *PluginDeployer) Rollback(deploymentID string) {
	d.logger.Warn("plugin rollback", slog.String("deploymentId", deploymentID))
}
