package marketplace

import (
	"context"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	dbm "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models/marketplace"
	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	marketobs "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/marketplace"
)

// ExpiringLicenseLister defines repository capability for fetching expiring licenses.
type ExpiringLicenseLister interface {
	ListExpiringWithin(ctx context.Context, tenantID string, window time.Duration) ([]*dbm.License, error)
}

// RenewalDispatcher abstracts outbound reminder delivery.
type RenewalDispatcher interface {
	DispatchRenewalReminder(ctx context.Context, license *dbm.License, channels []string) error
}

// RenewalNotifier periodically scans expiring licenses and dispatches renewal reminders.
type RenewalNotifier struct {
	cfg        *config.Config
	repo       ExpiringLicenseLister
	tenants    func(context.Context) ([]string, error)
	dispatcher RenewalDispatcher
	logger     *pxlog.Entry
	interval   time.Duration
	clock      func() time.Time
}

// NewLicenseRenewalNotifier constructs the reminder job.
func NewLicenseRenewalNotifier(cfg *config.Config, repo ExpiringLicenseLister, logger *pxlog.Entry, tenantResolver func(context.Context) ([]string, error), dispatcher RenewalDispatcher) *RenewalNotifier {
	if logger == nil {
		logger = pxlog.WithComponent("marketplace_license_renewal_notifier")
	}
	if tenantResolver == nil {
		tenantResolver = func(context.Context) ([]string, error) { return []string{"default"}, nil }
	}
	return &RenewalNotifier{
		cfg:        cfg,
		repo:       repo,
		tenants:    tenantResolver,
		dispatcher: dispatcher,
		logger:     logger,
		interval:   time.Hour,
		clock:      time.Now,
	}
}

// Run starts the reminder loop until the context is canceled.
func (n *RenewalNotifier) Run(ctx context.Context) {
	if n == nil {
		return
	}
	n.execute(ctx)

	ticker := time.NewTicker(n.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n.execute(ctx)
		}
	}
}

func (n *RenewalNotifier) execute(ctx context.Context) {
	if n.cfg == nil || n.repo == nil {
		return
	}
	lead := n.cfg.LicenseReminderLead()
	if lead <= 0 {
		return
	}
	channels := n.cfg.LicenseReminderChannels()

	tenants, err := n.tenants(ctx)
	if err != nil {
		pxlog.WarnCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
			"module":     "marketplace",
			"biz_scene":  "marketplace_license_renewal_notifier",
			"biz_domain": "marketplace",
			"component":  "marketplace_license_renewal_notifier",
			"error":      err.Error(),
		}), "failed to enumerate tenants for renewal notifier")
		return
	}

	window := lead
	if window < time.Hour {
		window = time.Hour
	}

	now := n.clock()
	for _, tenantID := range tenants {
		licenses, err := n.repo.ListExpiringWithin(ctx, tenantID, window)
		if err != nil {
			pxlog.WarnCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
				"module":      "marketplace",
				"biz_scene":   "marketplace_license_renewal_notifier",
				"biz_domain":  "marketplace",
				"component":   "marketplace_license_renewal_notifier",
				"tenant_uuid": tenantID,
				"error":       err.Error(),
			}), "failed to query expiring licenses")
			continue
		}
		for _, license := range licenses {
			deadline := license.ExpiresAt
			if license.OfflineUntil != nil && license.OfflineUntil.Before(deadline) {
				deadline = *license.OfflineUntil
			}
			if deadline.Before(now) {
				deadline = now
			}
			scheduled := deadline.Add(-lead)
			if scheduled.Before(now) {
				scheduled = now
			}
			marketobs.EmitLicenseRenewalDue(n.logger, license, scheduled, channels)
			if n.dispatcher != nil {
				if err := n.dispatcher.DispatchRenewalReminder(ctx, license, channels); err != nil {
					pxlog.WarnCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
						"module":      "marketplace",
						"biz_scene":   "marketplace_license_renewal_notifier",
						"biz_domain":  "marketplace",
						"component":   "marketplace_license_renewal_notifier",
						"tenant_uuid": tenantID,
						"license_id":  license.ID,
						"error":       err.Error(),
					}), "failed to dispatch renewal reminder")
				}
			}
		}
	}
}
