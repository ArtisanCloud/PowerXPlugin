package scheduler

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type HostProviderConfig struct {
	BaseURL       string
	GRPCTarget    string
	APIPrefix     string
	Token         string
	TokenProvider TokenProvider
	APIKey        string
	AuthScheme    string
	TenantUUID    string
	UserAgent     string
	Timeout       time.Duration
	HTTPClient    *http.Client
}

type TokenProvider func(ctx context.Context) (string, error)

// HostClient is the minimal adapter expected from the PowerX SchedulerService
// client. It keeps the framework facade stable while the concrete generated
// client is provided by the host SDK.
type HostClient interface {
	CreateJob(ctx context.Context, job JobSpec) (*Job, error)
	UpdateJob(ctx context.Context, job JobSpec) (*Job, error)
	PauseJob(ctx context.Context, jobID string, tenantUUID string) error
	ResumeJob(ctx context.Context, jobID string, tenantUUID string) error
	TriggerJob(ctx context.Context, jobID string, tenantUUID string) error
	GetJob(ctx context.Context, jobID string, tenantUUID string) (*Job, error)
	ListJobs(ctx context.Context, in ListJobsInput) ([]*Job, error)
}

type HostProvider struct {
	cfg    HostProviderConfig
	client HostClient
}

func NewHostProvider(cfg HostProviderConfig, client HostClient) *HostProvider {
	return &HostProvider{cfg: cfg, client: client}
}

func (p *HostProvider) CreateJob(ctx context.Context, job JobSpec) (*Job, error) {
	if p == nil || p.client == nil {
		return nil, ErrHostProviderUnavailable
	}
	job = p.applyDefaults(job)
	if err := job.validateForHost(); err != nil {
		return nil, err
	}
	return p.client.CreateJob(ctx, job)
}

func (p *HostProvider) UpdateJob(ctx context.Context, job JobSpec) (*Job, error) {
	if p == nil || p.client == nil {
		return nil, ErrHostProviderUnavailable
	}
	job = p.applyDefaults(job)
	if strings.TrimSpace(job.JobID) == "" {
		return nil, ErrJobIDRequired
	}
	if err := job.validateForHost(); err != nil {
		return nil, err
	}
	return p.client.UpdateJob(ctx, job)
}

func (p *HostProvider) PauseJob(ctx context.Context, jobID string, tenantUUID string) error {
	if p == nil || p.client == nil {
		return ErrHostProviderUnavailable
	}
	return p.client.PauseJob(ctx, strings.TrimSpace(jobID), "")
}

func (p *HostProvider) ResumeJob(ctx context.Context, jobID string, tenantUUID string) error {
	if p == nil || p.client == nil {
		return ErrHostProviderUnavailable
	}
	return p.client.ResumeJob(ctx, strings.TrimSpace(jobID), "")
}

func (p *HostProvider) TriggerJob(ctx context.Context, jobID string, tenantUUID string) error {
	if p == nil || p.client == nil {
		return ErrHostProviderUnavailable
	}
	return p.client.TriggerJob(ctx, strings.TrimSpace(jobID), "")
}

func (p *HostProvider) GetJob(ctx context.Context, jobID string, tenantUUID string) (*Job, error) {
	if p == nil || p.client == nil {
		return nil, ErrHostProviderUnavailable
	}
	return p.client.GetJob(ctx, strings.TrimSpace(jobID), "")
}

func (p *HostProvider) ListJobs(ctx context.Context, in ListJobsInput) ([]*Job, error) {
	if p == nil || p.client == nil {
		return nil, ErrHostProviderUnavailable
	}
	in.TenantUUID = ""
	return p.client.ListJobs(ctx, in)
}

func (p *HostProvider) applyDefaults(job JobSpec) JobSpec {
	job = job.normalized()
	job.TenantUUID = ""
	return job
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if v := strings.TrimSpace(value); v != "" {
			return v
		}
	}
	return ""
}
