package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"

	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"github.com/google/uuid"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
)

const (
	// SchedulerTriggeredTopic is the canonical scheduler trigger topic.
	SchedulerTriggeredTopic = "powerx.runtime.scheduler.triggered.v1"

	defaultSchedulerTenantUUID = "00000000-0000-0000-0000-000000000001"
)

// EventDispatcher represents unified scheduler event dispatch behavior.
type EventDispatcher interface {
	DispatchCronTrigger(ctx context.Context, jobName string, payload map[string]any) (traceID string, err error)
}

// SchedulerEventDispatcher emits scheduler events through EventBridge.
type SchedulerEventDispatcher struct {
	emitter    fweventbridge.Emitter
	meta       event.MetaBuilder
	tenantUUID string
	logger     *pxlog.Entry
}

// NewSchedulerEventDispatcher constructs a dispatcher backed by EventEmitter.
func NewSchedulerEventDispatcher(cfg *config.Config, emitter fweventbridge.Emitter, logger *pxlog.Entry) *SchedulerEventDispatcher {
	sourcePlugin := app.PluginID
	payloadVersion := "v1"
	tenantUUID := defaultSchedulerTenantUUID

	if cfg != nil {
		if cfg.EventBridge != nil {
			if v := strings.TrimSpace(cfg.EventBridge.SourcePlugin); v != "" {
				sourcePlugin = v
			}
			if v := strings.TrimSpace(cfg.EventBridge.PayloadVersion); v != "" {
				payloadVersion = v
			}
		}
		if cfg.Gateway != nil {
			if v := strings.TrimSpace(cfg.Gateway.TenantUUID); v != "" {
				tenantUUID = v
			}
		}
	}
	if v := strings.TrimSpace(os.Getenv("POWERX_TENANT_UUID")); v != "" {
		tenantUUID = v
	}

	if logger == nil {
		logger = pxlog.WithComponent("integration.scheduler_dispatcher")
	}

	return &SchedulerEventDispatcher{
		emitter:    emitter,
		meta:       event.NewMetaBuilder(sourcePlugin, payloadVersion),
		tenantUUID: tenantUUID,
		logger:     logger,
	}
}

// DispatchCronTrigger emits scheduler trigger events to unified event chain.
func (d *SchedulerEventDispatcher) DispatchCronTrigger(ctx context.Context, jobName string, payload map[string]any) (string, error) {
	if d == nil || d.emitter == nil {
		return "", errors.New("scheduler dispatcher emitter is not configured")
	}

	traceID := uuid.NewString()
	meta, err := d.meta.Build(d.tenantUUID, traceID, traceID)
	if err != nil {
		return "", err
	}

	body := map[string]any{
		"source":         "scheduler",
		"trigger_source": "cron",
		"job_name":       strings.TrimSpace(jobName),
		"trace_id":       traceID,
	}
	for k, v := range payload {
		body[k] = v
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	err = d.emitter.Emit(ctx, event.Event{
		Topic:   event.Topic(SchedulerTriggeredTopic),
		Meta:    meta,
		Payload: raw,
	})
	if err != nil {
		if d.logger != nil {
			pxlog.WarnCtx(pxlog.WithLogFields(ctx, map[string]interface{}{
				"module":     "integration",
				"topic":      SchedulerTriggeredTopic,
				"trace_id":   traceID,
				"job_name":   jobName,
				"biz_scene":  "scheduler_trigger_emit",
				"biz_domain": "integration",
				"component":  "integration.scheduler_dispatcher",
				"error":      err.Error(),
			}), "failed to emit scheduler trigger event")
		}
		return "", err
	}
	return traceID, nil
}
