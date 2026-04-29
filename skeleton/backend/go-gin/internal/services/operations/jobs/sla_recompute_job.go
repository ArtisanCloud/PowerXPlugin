package jobs

import (
	"context"
	"time"

	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	operationsvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/operations"
)

// SLARecomputeJob triggers SLA score recomputation across profiles.
type SLARecomputeJob struct {
	service *operationsvc.SLAService
	log     *pxlog.Entry
}

// NewSLARecomputeJob constructs a job instance.
func NewSLARecomputeJob(service *operationsvc.SLAService, log *pxlog.Entry) *SLARecomputeJob {
	if log == nil {
		log = pxlog.WithComponent("operations.sla_recompute_job")
	}
	return &SLARecomputeJob{service: service, log: log}
}

// Run executes the recomputation workflow.
func (j *SLARecomputeJob) Run(ctx context.Context) error {
	if j.service == nil {
		return nil
	}
	start := time.Now()
	profiles, err := j.service.RecomputeScores(ctx)
	if err != nil {
		return err
	}
	pxlog.InfoWith(j.log, ctx, "recomputed SLA profiles", pxlog.Fields{
		"module":     "operations",
		"biz_scene":  "sla_recompute",
		"biz_domain": "operations",
		"component":  "operations.sla_recompute_job",
		"profiles":   len(profiles),
		"duration":   time.Since(start).String(),
	})
	return nil
}
