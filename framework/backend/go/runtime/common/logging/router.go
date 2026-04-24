package logging

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type Router struct {
	registry *SinkRegistry
	policy   Policy
}

func NewRouter(policy Policy, registry *SinkRegistry) (*Router, error) {
	resolved := ResolvePolicy(policy)
	if err := ValidatePolicy(resolved); err != nil {
		return nil, err
	}
	if registry == nil {
		registry = NewSinkRegistry()
	}
	return &Router{registry: registry, policy: resolved}, nil
}

func (r *Router) Route(ctx context.Context, event Event) []SinkOutcome {
	if r == nil {
		return []SinkOutcome{{Status: OutcomeDropped, ErrorCode: "router_not_configured", Error: "router is nil"}}
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	outcomes := make([]SinkOutcome, 0, len(r.policy.Sinks))
	if r.policy.Mode == ModeHost {
		if err := ValidateHostSinkAuthorization(r.policy); err != nil {
			slog.Warn("logging router rejected unauthorized sink in host mode", "error", err.Error())
			for _, sink := range r.policy.Sinks {
				if sink == SinkStdout {
					continue
				}
				outcomes = append(outcomes, SinkOutcome{
					Sink:      sink,
					Status:    OutcomeFailed,
					Attempt:   1,
					ErrorCode: "unauthorized_sink",
					Error:     err.Error(),
				})
			}
		}
	}
	for _, sinkName := range r.policy.Sinks {
		if r.policy.Mode == ModeHost && sinkName != SinkStdout {
			authorized := false
			for _, allowed := range r.policy.AuthorizedExtraSinks {
				if allowed == sinkName {
					authorized = true
					break
				}
			}
			if !authorized {
				continue
			}
		}
		sink, ok := r.registry.Resolve(sinkName)
		if !ok {
			outcomes = append(outcomes, SinkOutcome{Sink: sinkName, Status: OutcomeDropped, Attempt: 1, ErrorCode: "sink_not_registered", Error: "sink is not registered"})
			continue
		}
		outcomes = append(outcomes, executeWithRetry(ctx, sinkName, r.policy.Retry, func() error {
			return sink.Emit(ctx, event)
		})...)
	}
	return outcomes
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ValidateHostSinkAuthorization(policy Policy) error {
	for _, sink := range policy.Sinks {
		if sink == SinkStdout {
			continue
		}
		authorized := false
		for _, allowed := range policy.AuthorizedExtraSinks {
			if sink == allowed {
				authorized = true
				break
			}
		}
		if !authorized {
			return fmt.Errorf("sink %s is not authorized in host mode", sink)
		}
	}
	return nil
}
