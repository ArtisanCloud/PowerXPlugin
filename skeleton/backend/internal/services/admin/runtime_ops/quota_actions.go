package runtime_ops

import (
	"context"
	"time"

	model "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/domain/models/runtime_ops"
	controller "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/mcp/controller"
)

const (
	ActionThrottle     = "throttle"
	ActionCircuitBreak = "circuit_break"
	ActionDisable      = "disable"
	ActionNotifyMarket = "notify_marketplace"
)

// HandleBreach applies configured action when quota exceeds limits.
func (s *QuotaService) HandleBreach(ctx context.Context, pluginID, scopeRef, capability, action string) {
	switch action {
	case ActionThrottle:
		// No-op placeholder; actual throttling handled by token bucket result
	case ActionCircuitBreak:
		// Future implementation: mark capability unavailable
	case ActionDisable:
		// Future implementation: disable plugin capability globally
	case ActionNotifyMarket:
		_, _ = s.ScheduleMarketplaceSummary(ctx, &model.MarketplaceOverage{
			PluginID:    pluginID,
			TenantID:    scopeRef,
			QuotaMetric: capability,
			HourWindow:  time.Now().Truncate(time.Hour),
			BreachCount: 1,
		})
	}
	controller.EmitQuotaEvent(ctx, pluginID, scopeRef, capability, action)
}
