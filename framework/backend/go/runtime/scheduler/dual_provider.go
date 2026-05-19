package scheduler

import "context"

type DualProvider struct {
	Primary   Scheduler
	Secondary Scheduler
}

func NewDualProvider(primary, secondary Scheduler) *DualProvider {
	return &DualProvider{Primary: primary, Secondary: secondary}
}

func (p *DualProvider) CreateJob(ctx context.Context, job JobSpec) (*Job, error) {
	if p == nil || p.Primary == nil {
		return nil, ErrProviderNotConfigured
	}
	created, err := p.Primary.CreateJob(ctx, job)
	if err != nil {
		return nil, err
	}
	if p.Secondary != nil {
		_, _ = p.Secondary.CreateJob(ctx, job)
	}
	return created, nil
}

func (p *DualProvider) UpdateJob(ctx context.Context, job JobSpec) (*Job, error) {
	if p == nil || p.Primary == nil {
		return nil, ErrProviderNotConfigured
	}
	updated, err := p.Primary.UpdateJob(ctx, job)
	if err != nil {
		return nil, err
	}
	if p.Secondary != nil {
		_, _ = p.Secondary.UpdateJob(ctx, job)
	}
	return updated, nil
}

func (p *DualProvider) PauseJob(ctx context.Context, jobID string, tenantUUID string) error {
	if p == nil || p.Primary == nil {
		return ErrProviderNotConfigured
	}
	if err := p.Primary.PauseJob(ctx, jobID, tenantUUID); err != nil {
		return err
	}
	if p.Secondary != nil {
		_ = p.Secondary.PauseJob(ctx, jobID, tenantUUID)
	}
	return nil
}

func (p *DualProvider) ResumeJob(ctx context.Context, jobID string, tenantUUID string) error {
	if p == nil || p.Primary == nil {
		return ErrProviderNotConfigured
	}
	if err := p.Primary.ResumeJob(ctx, jobID, tenantUUID); err != nil {
		return err
	}
	if p.Secondary != nil {
		_ = p.Secondary.ResumeJob(ctx, jobID, tenantUUID)
	}
	return nil
}

func (p *DualProvider) TriggerJob(ctx context.Context, jobID string, tenantUUID string) error {
	if p == nil || p.Primary == nil {
		return ErrProviderNotConfigured
	}
	return p.Primary.TriggerJob(ctx, jobID, tenantUUID)
}

func (p *DualProvider) GetJob(ctx context.Context, jobID string, tenantUUID string) (*Job, error) {
	if p == nil || p.Primary == nil {
		return nil, ErrProviderNotConfigured
	}
	return p.Primary.GetJob(ctx, jobID, tenantUUID)
}

func (p *DualProvider) ListJobs(ctx context.Context, in ListJobsInput) ([]*Job, error) {
	if p == nil || p.Primary == nil {
		return nil, ErrProviderNotConfigured
	}
	return p.Primary.ListJobs(ctx, in)
}
