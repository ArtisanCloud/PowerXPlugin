package publish

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

// Handler coordinates publish plan creation/deploy for CLI.
type Handler struct {
	logger *slog.Logger
	store  *planStore
}

// Plan represents a release plan prior to deployment.
type Plan struct {
	PlanID    string        `json:"planId"`
	PublishID string        `json:"publishId"`
	Channel   string        `json:"channel"`
	Manifest  interface{}   `json:"manifest"`
	Rollout   RolloutConfig `json:"rollout"`
	Window    struct {
		Start string `json:"start,omitempty"`
		End   string `json:"end,omitempty"`
	} `json:"window"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	LastUpdated time.Time `json:"updatedAt"`
}

// RolloutConfig describes rollout strategy.
type RolloutConfig struct {
	Strategy string               `json:"strategy"`
	Batches  []RolloutBatchConfig `json:"batches"`
}

// RolloutBatchConfig describes individual batch rollout.
type RolloutBatchConfig struct {
	Percentage int    `json:"percentage"`
	Wait       string `json:"wait,omitempty"`
}

// Deployment captures deploy progress.
type Deployment struct {
	DeploymentID  string            `json:"deploymentId"`
	PlanID        string            `json:"planId"`
	Strategy      string            `json:"strategy"`
	State         string            `json:"state"`
	Batches       []DeploymentBatch `json:"batches"`
	RollbackToken string            `json:"rollbackToken"`
	StartedAt     time.Time         `json:"startedAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	Notes         string            `json:"notes"`
}

// DeploymentBatch stores batch status.
type DeploymentBatch struct {
	Percentage int    `json:"percentage"`
	Status     string `json:"status"`
}

type planStore struct {
	sync.Mutex
	plans       map[string]Plan
	deployments map[string]Deployment
}

func newPlanStore() *planStore {
	return &planStore{
		plans:       map[string]Plan{},
		deployments: map[string]Deployment{},
	}
}

// NewHandler creates publish handler.
func NewHandler(logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		logger: logger,
		store:  newPlanStore(),
	}
}

// Create handles publish plan creation.
func (h *Handler) Create(ctx bootstrap.Context) {
	var payload struct {
		Manifest interface{}   `json:"manifest"`
		Channel  string        `json:"channel"`
		Notes    string        `json:"notes"`
		Rollout  RolloutConfig `json:"rollout"`
		Window   struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"window"`
		AutoRollback bool `json:"autoRollback"`
		DryRun       bool `json:"dryRun"`
	}
	if err := ctx.BindJSON(&payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_PLAN_PAYLOAD", "unable to parse payload", nil)
		return
	}
	if payload.Channel == "" {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_PLAN_PAYLOAD", "channel is required", nil)
		return
	}

	plan := Plan{
		PlanID:      newID("plan"),
		PublishID:   newID("publish"),
		Channel:     payload.Channel,
		Manifest:    payload.Manifest,
		Rollout:     payload.Rollout,
		Status:      "draft",
		CreatedAt:   time.Now().UTC(),
		LastUpdated: time.Now().UTC(),
	}
	plan.Window.Start = payload.Window.Start
	plan.Window.End = payload.Window.End

	h.store.Lock()
	h.store.plans[plan.PlanID] = plan
	h.store.Unlock()

	h.logger.Info("publish plan created", slog.String("planId", plan.PlanID))
	router.RespondSuccess(ctx, http.StatusCreated, map[string]any{
		"planId":       plan.PlanID,
		"publishId":    plan.PublishID,
		"channel":      plan.Channel,
		"status":       plan.Status,
		"window":       plan.Window,
		"autoRollback": payload.AutoRollback,
		"dryRun":       payload.DryRun,
	}, "publish plan created")
}

// Deploy executes rollout for plan.
func (h *Handler) Deploy(ctx bootstrap.Context) {
	var payload struct {
		PlanID   string               `json:"planId"`
		Strategy string               `json:"strategy"`
		Batches  []RolloutBatchConfig `json:"batches"`
		Notes    string               `json:"notes"`
		DryRun   bool                 `json:"dryRun"`
	}
	if err := ctx.BindJSON(&payload); err != nil {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_DEPLOY_PAYLOAD", "unable to parse payload", nil)
		return
	}
	if payload.PlanID == "" {
		router.RespondError(ctx, http.StatusBadRequest, "INVALID_DEPLOY_PAYLOAD", "planId is required", nil)
		return
	}

	h.store.Lock()
	plan, ok := h.store.plans[payload.PlanID]
	if !ok {
		h.store.Unlock()
		router.RespondError(ctx, http.StatusNotFound, "PLAN_NOT_FOUND", "publish plan not found", nil)
		return
	}
	plan.Status = "deploying"
	plan.LastUpdated = time.Now().UTC()
	h.store.plans[payload.PlanID] = plan

	deployment := Deployment{
		DeploymentID:  newID("deploy"),
		PlanID:        plan.PlanID,
		Strategy:      payload.Strategy,
		State:         chooseState(payload.DryRun),
		Batches:       convertBatches(payload.Batches),
		RollbackToken: newID("rollback"),
		StartedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		Notes:         payload.Notes,
	}
	h.store.deployments[deployment.DeploymentID] = deployment
	h.store.Unlock()

	router.RespondSuccess(ctx, http.StatusAccepted, map[string]any{
		"deploymentId":  deployment.DeploymentID,
		"planId":        deployment.PlanID,
		"state":         deployment.State,
		"batches":       deployment.Batches,
		"rollbackToken": deployment.RollbackToken,
	}, "publish deployment started")
}

func chooseState(dryRun bool) string {
	if dryRun {
		return "dry_run"
	}
	return "running"
}

func convertBatches(batches []RolloutBatchConfig) []DeploymentBatch {
	if len(batches) == 0 {
		return []DeploymentBatch{{Percentage: 100, Status: "scheduled"}}
	}
	result := make([]DeploymentBatch, 0, len(batches))
	for _, batch := range batches {
		result = append(result, DeploymentBatch{
			Percentage: batch.Percentage,
			Status:     "scheduled",
		})
	}
	return result
}

func newID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
