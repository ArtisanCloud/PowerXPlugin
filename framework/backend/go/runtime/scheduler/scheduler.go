package scheduler

import "context"

type Scheduler interface {
	CreateJob(ctx context.Context, job JobSpec) (*Job, error)
	UpdateJob(ctx context.Context, job JobSpec) (*Job, error)
	PauseJob(ctx context.Context, jobID string, tenantUUID string) error
	ResumeJob(ctx context.Context, jobID string, tenantUUID string) error
	TriggerJob(ctx context.Context, jobID string, tenantUUID string) error
	GetJob(ctx context.Context, jobID string, tenantUUID string) (*Job, error)
	ListJobs(ctx context.Context, in ListJobsInput) ([]*Job, error)
}

type Provider interface {
	NewScheduler() (Scheduler, error)
}

type HandlerFunc func(context.Context, TriggeredJob) error

type HandlerRegistry interface {
	RegisterHandler(jobName string, handler HandlerFunc)
	HandleTriggered(ctx context.Context, job TriggeredJob) error
}
