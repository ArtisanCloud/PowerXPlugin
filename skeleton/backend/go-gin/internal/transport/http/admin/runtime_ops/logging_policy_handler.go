package runtime_ops

import (
	"net/http"
	"sync"
	"time"

	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/contracts"
	"github.com/gin-gonic/gin"
)

type LoggingPolicyHandler struct{}

var (
	loggingPolicyMu sync.RWMutex
	loggingPolicy   = runtimelogging.ResolveWithHostDefaults(runtimelogging.DefaultPolicy())
)

func NewLoggingPolicyHandler() *LoggingPolicyHandler {
	return &LoggingPolicyHandler{}
}

type loggingPolicyRequest struct {
	PolicyVersion        string   `json:"policy_version"`
	Mode                 string   `json:"mode"`
	Sinks                []string `json:"sinks"`
	Format               string   `json:"format"`
	Level                string   `json:"level"`
	AuthorizedExtraSinks []string `json:"authorized_extra_sinks"`
	Retry                struct {
		Enabled     bool `json:"enabled"`
		MaxAttempts int  `json:"max_attempts"`
		BackoffMS   int  `json:"backoff_ms"`
	} `json:"retry"`
}

func (h *LoggingPolicyHandler) Get(c *gin.Context) {
	loggingPolicyMu.RLock()
	policy := loggingPolicy
	loggingPolicyMu.RUnlock()
	contracts.ResponseSuccess(c, gin.H{"policy": policy})
}

func (h *LoggingPolicyHandler) Put(c *gin.Context) {
	var req loggingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid payload")
		return
	}
	p := runtimelogging.Policy{
		PolicyVersion: req.PolicyVersion,
		Mode:          runtimelogging.PolicyMode(req.Mode),
		Format:        req.Format,
		Level:         req.Level,
		Retry: runtimelogging.RetryPolicy{
			Enabled:     req.Retry.Enabled,
			MaxAttempts: req.Retry.MaxAttempts,
			BackoffMS:   req.Retry.BackoffMS,
		},
	}
	for _, sink := range req.Sinks {
		p.Sinks = append(p.Sinks, runtimelogging.SinkType(sink))
	}
	for _, sink := range req.AuthorizedExtraSinks {
		p.AuthorizedExtraSinks = append(p.AuthorizedExtraSinks, runtimelogging.SinkType(sink))
	}
	p = runtimelogging.ResolveWithHostDefaults(p)
	if err := runtimelogging.ValidatePolicy(p); err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}

	loggingPolicyMu.Lock()
	loggingPolicy = p
	loggingPolicyMu.Unlock()
	c.JSON(http.StatusAccepted, contracts.MakeSuccess(gin.H{
		"accepted":     true,
		"policy":       p,
		"effective_at": time.Now().UTC(),
	}, "", c.GetString("request_id")))
}
