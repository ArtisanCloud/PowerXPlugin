package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const serviceSchedulerJobsPath = "/tenant/scheduler/jobs"

// ServiceHTTPHostClient uses Core's tenant/owner scoped service read contract.
// Mutations retain the published STS scheduler bindings. A denied service read
// never retries the admin surface or changes credentials.
type ServiceHTTPHostClient struct{ *HTTPHostClient }

func NewServiceHTTPHostClient(cfg HostProviderConfig) (*ServiceHTTPHostClient, error) {
	client, err := NewHTTPHostClient(cfg)
	if err != nil {
		return nil, err
	}
	return &ServiceHTTPHostClient{HTTPHostClient: client}, nil
}

func (c *ServiceHTTPHostClient) ListJobs(ctx context.Context, in ListJobsInput) ([]*Job, error) {
	if c == nil || c.HTTPHostClient == nil {
		return nil, ErrHostProviderUnavailable
	}
	if in.OwnerType != OwnerTypePlugin || strings.TrimSpace(in.OwnerID) == "" {
		return nil, errors.New("SCHEDULER_SERVICE_OWNER_REQUIRED")
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 1000
	}
	size := 50
	if limit < size {
		size = limit
	}
	jobs := make([]*Job, 0)
	for page := 1; ; page++ {
		query := url.Values{"owner_type": {in.OwnerType}, "owner_id": {strings.TrimSpace(in.OwnerID)}, "page": {strconv.Itoa(page)}, "page_size": {strconv.Itoa(size)}}
		if in.Status != "" {
			query.Set("status", in.Status)
		}
		data, err := requestScheduler[struct {
			Items      []json.RawMessage `json:"items"`
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		}](c.HTTPHostClient, ctx, http.MethodGet, serviceSchedulerJobsPath, query, nil)
		if err != nil {
			return nil, err
		}
		for _, raw := range data.Items {
			job, err := decodeServiceJob(raw)
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, job)
		}
		if len(jobs) >= data.Pagination.Total {
			return jobs, nil
		}
		if len(jobs) >= limit || len(data.Items) == 0 {
			return nil, errors.New("SCHEDULER_JOB_LOOKUP_TRUNCATED")
		}
	}
}

func (c *ServiceHTTPHostClient) GetJob(ctx context.Context, jobID, tenantUUID string) (*Job, error) {
	if c == nil || c.HTTPHostClient == nil {
		return nil, ErrHostProviderUnavailable
	}
	data, err := requestScheduler[struct {
		Item json.RawMessage `json:"item"`
	}](c.HTTPHostClient, ctx, http.MethodGet, serviceSchedulerJobsPath+"/"+url.PathEscape(strings.TrimSpace(jobID)), nil, nil)
	if err != nil {
		return nil, err
	}
	return decodeServiceJob(data.Item)
}

func decodeServiceJob(raw json.RawMessage) (*Job, error) {
	var identity struct {
		JobUUID string `json:"job_uuid"`
	}
	if err := json.Unmarshal(raw, &identity); err != nil {
		return nil, err
	}
	if identity.JobUUID == "" {
		return nil, ErrJobIDRequired
	}
	var job Job
	if err := json.Unmarshal(raw, &job); err != nil {
		return nil, err
	}
	job.JobID, job.UUID = identity.JobUUID, identity.JobUUID
	return &job, nil
}
