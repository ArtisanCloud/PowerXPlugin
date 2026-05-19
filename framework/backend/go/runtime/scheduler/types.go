package scheduler

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	ModeLocal = "local"
	ModeHost  = "host"
	ModeDual  = "dual"

	OwnerTypePlugin = "plugin"

	ScheduleTypeOnce     = "once"
	ScheduleTypeInterval = "interval"
	ScheduleTypeCron     = "cron"

	StatusActive    = "active"
	StatusPaused    = "paused"
	StatusCompleted = "completed"

	DefaultTriggeredTopic = "powerx.runtime.scheduler.triggered.v1"
	TriggeredTopic        = DefaultTriggeredTopic
)

var (
	ErrJobNameRequired      = errors.New("scheduler job name is required")
	ErrTenantUUIDRequired   = errors.New("scheduler tenant_uuid is required")
	ErrOwnerTypeRequired    = errors.New("scheduler owner_type is required")
	ErrOwnerIDRequired      = errors.New("scheduler owner_id is required")
	ErrScheduleTypeRequired = errors.New("scheduler schedule_type is required")
	ErrScheduleExprRequired = errors.New("scheduler schedule_expr is required")
	ErrUnsupportedSchedule  = errors.New("scheduler schedule_type is unsupported")
	ErrInvalidScheduleExpr  = errors.New("scheduler schedule_expr is invalid")
	ErrJobIDRequired        = errors.New("scheduler job_id is required")
)

type RetryPolicy struct {
	MaxAttempts    int           `json:"max_attempts,omitempty"`
	BackoffSeconds int           `json:"backoff_seconds,omitempty"`
	Backoff        time.Duration `json:"-"`
}

type JobSpec struct {
	JobID          string         `json:"job_id,omitempty"`
	TenantUUID     string         `json:"tenant_uuid,omitempty"`
	OwnerType      string         `json:"owner_type,omitempty"`
	OwnerID        string         `json:"owner_id,omitempty"`
	Name           string         `json:"name,omitempty"`
	ScheduleType   string         `json:"schedule_type,omitempty"`
	ScheduleExpr   string         `json:"schedule_expr,omitempty"`
	Timezone       string         `json:"timezone,omitempty"`
	Topic          string         `json:"topic,omitempty"`
	Payload        map[string]any `json:"payload,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	RetryPolicy    RetryPolicy    `json:"retry_policy,omitempty"`
	Paused         bool           `json:"paused,omitempty"`
}

type Job struct {
	JobSpec
	Status    string    `json:"status,omitempty"`
	NextRunAt time.Time `json:"next_run_at,omitempty"`
	LastRunAt time.Time `json:"last_run_at,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type ListJobsInput struct {
	TenantUUID string `json:"tenant_uuid,omitempty"`
	OwnerType  string `json:"owner_type,omitempty"`
	OwnerID    string `json:"owner_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type TriggeredJob struct {
	JobID          string          `json:"job_id,omitempty"`
	JobName        string          `json:"job_name,omitempty"`
	OwnerType      string          `json:"owner_type,omitempty"`
	OwnerID        string          `json:"owner_id,omitempty"`
	TenantUUID     string          `json:"tenant_uuid,omitempty"`
	TriggerSource  string          `json:"trigger_source,omitempty"`
	ScheduledAt    time.Time       `json:"scheduled_at,omitempty"`
	FiredAt        time.Time       `json:"fired_at,omitempty"`
	TraceID        string          `json:"trace_id,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	BusinessAction string          `json:"business_action,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}

func (s JobSpec) normalized() JobSpec {
	s.JobID = strings.TrimSpace(s.JobID)
	s.TenantUUID = strings.TrimSpace(s.TenantUUID)
	s.OwnerType = strings.TrimSpace(s.OwnerType)
	s.OwnerID = strings.TrimSpace(s.OwnerID)
	s.Name = strings.TrimSpace(s.Name)
	s.ScheduleType = strings.ToLower(strings.TrimSpace(s.ScheduleType))
	s.ScheduleExpr = strings.TrimSpace(s.ScheduleExpr)
	s.Timezone = strings.TrimSpace(s.Timezone)
	s.Topic = strings.TrimSpace(s.Topic)
	s.IdempotencyKey = strings.TrimSpace(s.IdempotencyKey)
	if s.Topic == "" {
		s.Topic = TriggeredTopic
	}
	if s.Payload == nil {
		s.Payload = map[string]any{}
	}
	return s
}

func (s JobSpec) validate() error {
	s = s.normalized()
	if s.TenantUUID == "" {
		return ErrTenantUUIDRequired
	}
	if s.OwnerType == "" {
		return ErrOwnerTypeRequired
	}
	if s.OwnerID == "" {
		return ErrOwnerIDRequired
	}
	if s.Name == "" {
		return ErrJobNameRequired
	}
	if s.ScheduleType == "" {
		return ErrScheduleTypeRequired
	}
	if s.ScheduleExpr == "" {
		return ErrScheduleExprRequired
	}
	switch s.ScheduleType {
	case ScheduleTypeOnce:
		if _, err := time.Parse(time.RFC3339, s.ScheduleExpr); err != nil {
			return ErrInvalidScheduleExpr
		}
	case ScheduleTypeInterval:
		if d, err := time.ParseDuration(s.ScheduleExpr); err != nil || d <= 0 {
			return ErrInvalidScheduleExpr
		}
	case ScheduleTypeCron:
	default:
		return ErrUnsupportedSchedule
	}
	return nil
}
