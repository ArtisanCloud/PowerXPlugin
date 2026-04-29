package integration

import (
	"context"
	"time"

	repo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/integration"
	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	obs "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/integration"
	service "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/integration"
)

// SecretRotationWorker scans for secrets nearing rotation and schedules reminders.
type SecretRotationWorker struct {
	service  *service.SecretService
	repo     *repo.SecretRepository
	logger   *pxlog.Entry
	interval time.Duration
}

// NewSecretRotationWorker constructs the worker.
func NewSecretRotationWorker(secretSvc *service.SecretService, secretRepo *repo.SecretRepository, interval time.Duration, logger *pxlog.Entry) *SecretRotationWorker {
	if interval <= 0 {
		interval = time.Hour
	}
	if logger == nil {
		logger = pxlog.WithComponent("integration.secret_rotation_worker")
	}
	return &SecretRotationWorker{
		service:  secretSvc,
		repo:     secretRepo,
		logger:   logger,
		interval: interval,
	}
}

// Name returns job name.
func (w *SecretRotationWorker) Name() string {
	return "integration.secret_rotation"
}

// Interval returns run interval.
func (w *SecretRotationWorker) Interval() time.Duration {
	return w.interval
}

// Run fetches due secrets and schedules rotation reminders.
func (w *SecretRotationWorker) Run(ctx context.Context) error {
	if w.repo == nil || w.service == nil {
		pxlog.WarnCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
			"module":     "integration",
			"biz_scene":  "secret_rotation_run",
			"biz_domain": "integration",
			"component":  "integration.secret_rotation_worker",
		}), "secret rotation worker dependencies not configured")
		return w.service.RefreshRotationMetrics(ctx)
	}

	now := time.Now().UTC()
	soon := now.Add(24 * time.Hour)

	dueSecrets, err := w.repo.ListDueForRotation(ctx, now, 0)
	if err != nil {
		return err
	}
	obs.SetSecretsDue("due_now", float64(len(dueSecrets)))

	soonSecrets, err := w.repo.ListDueForRotation(ctx, soon, 0)
	if err != nil {
		return err
	}
	obs.SetSecretsDue("due_24h", float64(len(soonSecrets)))

	for _, secret := range dueSecrets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, err := w.service.RotateSecret(ctx, service.RotateSecretParams{
			TenantUuid: secret.TenantUuid,
			SecretID:   secret.ID,
			Generate:   true,
			Actor:      "system-rotation-worker",
		})
		if err != nil {
			pxlog.WarnCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
				"module":      "integration",
				"tenant_uuid": secret.TenantUuid,
				"secret_id":   secret.ID,
				"biz_scene":   "secret_rotation_schedule",
				"biz_domain":  "integration",
				"component":   "integration.secret_rotation_worker",
				"error":       err.Error(),
			}), "failed to schedule automatic rotation")
			continue
		}
	}

	return nil
}
