package marketplace

import (
	"context"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/marketplace"
	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/recommendation"
)

// MetricsProvider defines the interface to fetch signals for recommendation scoring.
type MetricsProvider interface {
	FetchSignals(ctx context.Context, tenantID string) ([]recommendation.Signal, error)
}

// SyncJob periodically refreshes listing recommendation weights.
type SyncJob struct {
	cfg      *config.Config
	repo     *mrepo.ListingRepository
	provider MetricsProvider
	interval time.Duration
	logger   *pxlog.Entry
	tenants  func(context.Context) ([]string, error)
}

// NewSyncJob constructs a new recommendation synchronization job.
func NewSyncJob(cfg *config.Config, repo *mrepo.ListingRepository, provider MetricsProvider, logger *pxlog.Entry, tenantResolver func(context.Context) ([]string, error)) *SyncJob {
	if logger == nil {
		logger = pxlog.WithComponent("marketplace_recommendation_sync")
	}
	interval := time.Hour
	if cfg != nil {
		interval = cfg.RecommendationFrequency()
	}
	if tenantResolver == nil {
		tenantResolver = func(context.Context) ([]string, error) { return []string{"default"}, nil }
	}
	return &SyncJob{
		cfg:      cfg,
		repo:     repo,
		provider: provider,
		interval: interval,
		logger:   logger,
		tenants:  tenantResolver,
	}
}

// Run starts the background synchronization loop until the context is canceled.
func (j *SyncJob) Run(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	j.execute(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.execute(ctx)
		}
	}
}

func (j *SyncJob) execute(ctx context.Context) {
	if j.cfg != nil && j.cfg.Marketplace != nil && !j.cfg.Marketplace.Recommendation.Enabled {
		return
	}
	tenants, err := j.tenants(ctx)
	if err != nil {
		pxlog.WarnCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
			"module":     "marketplace",
			"biz_scene":  "marketplace_recommendation_sync",
			"biz_domain": "marketplace",
			"component":  "marketplace_recommendation_sync",
			"error":      err.Error(),
		}), "failed to enumerate tenants for recommendation sync")
		return
	}
	engine := recommendation.NewEngine(j.repo, j.provider, j.logger)
	for _, tenantID := range tenants {
		result, err := engine.RefreshRecommendations(ctx, tenantID)
		if err != nil {
			pxlog.ErrorCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
				"module":      "marketplace",
				"biz_scene":   "marketplace_recommendation_sync",
				"biz_domain":  "marketplace",
				"component":   "marketplace_recommendation_sync",
				"tenant_uuid": tenantID,
				"error":       err.Error(),
			}), "recommendation sync failed")
			continue
		}
		pxlog.InfoCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
			"module":            "marketplace",
			"biz_scene":         "marketplace_recommendation_sync",
			"biz_domain":        "marketplace",
			"component":         "marketplace_recommendation_sync",
			"tenant_uuid":       tenantID,
			"updated":           result.UpdatedCount,
			"average_weight":    result.AverageWeight,
			"exploration_share": result.ExplorationShare,
		}), "recommendation weights refreshed")
	}
}
