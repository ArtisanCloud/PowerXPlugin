package runtime

import (
	"context"
	"fmt"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
)

// SyncCapabilities validates capability catalog and optionally registers it with host runtime.
func SyncCapabilities(ctx context.Context, mgr capabilities.Manager, client capabilities.HostSyncClient) error {
	if mgr == nil {
		return nil
	}
	if client != nil {
		if err := mgr.RegisterWithHost(ctx, client); err != nil {
			return fmt.Errorf("capability sync failed: %w", err)
		}
		return nil
	}
	if err := capabilities.EnsureManager(ctx, mgr, logger.WithField("component", "capabilities_sync")); err != nil {
		return fmt.Errorf("capability warmup failed: %w", err)
	}
	return nil
}
