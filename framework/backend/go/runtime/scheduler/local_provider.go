package scheduler

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
)

type LocalProviderConfig struct {
	Emitter        eventbridge.Emitter
	SourcePlugin   string
	PayloadVersion string
	Now            func() time.Time
}

type LocalProvider struct {
	cfg  LocalProviderConfig
	meta event.MetaBuilder

	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewLocalProvider(cfg LocalProviderConfig) *LocalProvider {
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	return &LocalProvider{
		cfg:  cfg,
		meta: event.NewMetaBuilder(cfg.SourcePlugin, cfg.PayloadVersion),
		jobs: make(map[string]*Job),
	}
}

func (p *LocalProvider) CreateJob(ctx context.Context, spec JobSpec) (*Job, error) {
	_ = ctx
	spec = spec.normalized()
	if err := spec.validate(); err != nil {
		return nil, err
	}
	if spec.JobID == "" {
		spec.JobID = "sch-" + uuid.NewString()
	}
	now := p.now()
	job := &Job{
		JobSpec:   spec,
		Status:    statusFromPaused(spec.Paused),
		CreatedAt: now,
		UpdatedAt: now,
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jobs[job.JobID] = cloneJob(job)
	return cloneJob(job), nil
}

func (p *LocalProvider) UpdateJob(ctx context.Context, spec JobSpec) (*Job, error) {
	_ = ctx
	spec = spec.normalized()
	if strings.TrimSpace(spec.JobID) == "" {
		return nil, ErrJobIDRequired
	}
	if err := spec.validate(); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	existing, ok := p.jobs[spec.JobID]
	if !ok {
		return nil, ErrJobNotFound
	}
	job := &Job{
		JobSpec:   spec,
		Status:    statusFromPaused(spec.Paused),
		CreatedAt: existing.CreatedAt,
		UpdatedAt: p.now(),
	}
	p.jobs[job.JobID] = cloneJob(job)
	return cloneJob(job), nil
}

func (p *LocalProvider) PauseJob(ctx context.Context, jobID string, tenantUUID string) error {
	_ = ctx
	return p.setPaused(jobID, tenantUUID, true)
}

func (p *LocalProvider) ResumeJob(ctx context.Context, jobID string, tenantUUID string) error {
	_ = ctx
	return p.setPaused(jobID, tenantUUID, false)
}

func (p *LocalProvider) TriggerJob(ctx context.Context, jobID string, tenantUUID string) error {
	job, err := p.GetJob(ctx, jobID, tenantUUID)
	if err != nil {
		return err
	}
	if job.Status == StatusPaused || job.Status == StatusCompleted {
		return nil
	}
	if err := p.emitTrigger(ctx, job, "manual"); err != nil {
		return err
	}
	p.markTriggered(job.JobID, job.TenantUUID)
	return nil
}

// TriggerJobPreview emits a manual trigger event without consuming a once job's scheduled run.
func (p *LocalProvider) TriggerJobPreview(ctx context.Context, jobID string, tenantUUID string) error {
	job, err := p.GetJob(ctx, jobID, tenantUUID)
	if err != nil {
		return err
	}
	if job.Status == StatusPaused {
		return nil
	}
	if err := p.emitTrigger(ctx, job, "manual"); err != nil {
		return err
	}
	p.markPreviewTriggered(job.JobID, job.TenantUUID)
	return nil
}

func (p *LocalProvider) GetJob(ctx context.Context, jobID string, tenantUUID string) (*Job, error) {
	_ = ctx
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, ErrJobIDRequired
	}
	tenantUUID = strings.TrimSpace(tenantUUID)
	p.mu.RLock()
	defer p.mu.RUnlock()
	job, ok := p.jobs[jobID]
	if !ok {
		return nil, ErrJobNotFound
	}
	if tenantUUID != "" && job.TenantUUID != tenantUUID {
		return nil, ErrJobNotFound
	}
	return cloneJob(job), nil
}

func (p *LocalProvider) ListJobs(ctx context.Context, in ListJobsInput) ([]*Job, error) {
	_ = ctx
	tenantUUID := strings.TrimSpace(in.TenantUUID)
	ownerType := strings.TrimSpace(in.OwnerType)
	ownerID := strings.TrimSpace(in.OwnerID)
	status := strings.TrimSpace(in.Status)
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*Job, 0, len(p.jobs))
	for _, job := range p.jobs {
		if tenantUUID != "" && job.TenantUUID != tenantUUID {
			continue
		}
		if ownerType != "" && job.OwnerType != ownerType {
			continue
		}
		if ownerID != "" && job.OwnerID != ownerID {
			continue
		}
		if status != "" && job.Status != status {
			continue
		}
		out = append(out, cloneJob(job))
	}
	return out, nil
}

func (p *LocalProvider) EmitDueTrigger(ctx context.Context, jobID string, tenantUUID string, triggerSource string) error {
	job, err := p.GetJob(ctx, jobID, tenantUUID)
	if err != nil {
		return err
	}
	if job.Status == StatusPaused || job.Status == StatusCompleted {
		return nil
	}
	if err := p.emitTrigger(ctx, job, triggerSource); err != nil {
		return err
	}
	p.markTriggered(job.JobID, job.TenantUUID)
	return nil
}

func (p *LocalProvider) markTriggered(jobID string, tenantUUID string) {
	jobID = strings.TrimSpace(jobID)
	tenantUUID = strings.TrimSpace(tenantUUID)
	p.mu.Lock()
	defer p.mu.Unlock()
	job, ok := p.jobs[jobID]
	if !ok {
		return
	}
	if tenantUUID != "" && job.TenantUUID != tenantUUID {
		return
	}
	now := p.now()
	job.LastRunAt = now
	job.UpdatedAt = now
	if job.ScheduleType == ScheduleTypeOnce {
		job.Status = StatusCompleted
		job.Paused = false
	}
}

func (p *LocalProvider) markPreviewTriggered(jobID string, tenantUUID string) {
	jobID = strings.TrimSpace(jobID)
	tenantUUID = strings.TrimSpace(tenantUUID)
	p.mu.Lock()
	defer p.mu.Unlock()
	job, ok := p.jobs[jobID]
	if !ok {
		return
	}
	if tenantUUID != "" && job.TenantUUID != tenantUUID {
		return
	}
	now := p.now()
	job.LastRunAt = now
	job.UpdatedAt = now
}

func (p *LocalProvider) setPaused(jobID string, tenantUUID string, paused bool) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return ErrJobIDRequired
	}
	tenantUUID = strings.TrimSpace(tenantUUID)
	p.mu.Lock()
	defer p.mu.Unlock()
	job, ok := p.jobs[jobID]
	if !ok {
		return ErrJobNotFound
	}
	if tenantUUID != "" && job.TenantUUID != tenantUUID {
		return ErrJobNotFound
	}
	job.Paused = paused
	job.Status = statusFromPaused(paused)
	job.UpdatedAt = p.now()
	return nil
}

func (p *LocalProvider) emitTrigger(ctx context.Context, job *Job, triggerSource string) error {
	if p == nil || p.cfg.Emitter == nil {
		return ErrProviderNotConfigured
	}
	triggerSource = strings.TrimSpace(triggerSource)
	if triggerSource == "" {
		triggerSource = job.ScheduleType
	}
	traceID := payloadString(job.Payload, "trace_id")
	if traceID == "" {
		traceID = uuid.NewString()
	}
	meta, err := p.meta.Build(job.TenantUUID, traceID, traceID)
	if err != nil {
		return err
	}
	body := map[string]any{
		"job_id":          job.JobID,
		"job_name":        job.Name,
		"owner_type":      job.OwnerType,
		"owner_id":        job.OwnerID,
		"tenant_uuid":     job.TenantUUID,
		"trigger_source":  triggerSource,
		"scheduled_at":    job.ScheduleExpr,
		"fired_at":        p.now().Format(time.RFC3339),
		"trace_id":        traceID,
		"idempotency_key": job.IdempotencyKey,
		"payload":         job.Payload,
	}
	for k, v := range job.Payload {
		if _, exists := body[k]; !exists {
			body[k] = v
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return p.cfg.Emitter.Emit(ctx, event.Event{
		Topic:   event.Topic(job.Topic),
		Meta:    meta,
		Payload: raw,
	})
}

func (p *LocalProvider) now() time.Time {
	if p == nil || p.cfg.Now == nil {
		return time.Now().UTC()
	}
	return p.cfg.Now().UTC()
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

func statusFromPaused(paused bool) string {
	if paused {
		return StatusPaused
	}
	return StatusActive
}

func cloneJob(in *Job) *Job {
	if in == nil {
		return nil
	}
	out := *in
	if in.Payload != nil {
		out.Payload = make(map[string]any, len(in.Payload))
		for k, v := range in.Payload {
			out.Payload[k] = v
		}
	}
	return &out
}
