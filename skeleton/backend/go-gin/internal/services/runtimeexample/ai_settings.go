package runtimeexample

import (
	"context"
	"sort"

	fw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/aisettings"
)

var _ fw.Service = (*LocalAI)(nil)

// Settings expose the same immutable model configuration used for execution.
// Reading configuration never probes providers or reports fabricated health.
func (s *LocalAI) Summary(ctx context.Context, _ string) (fw.Summary, error) {
	if _, err := mediaTenant(ctx); err != nil {
		return nil, aiError(401, "UNAUTHORIZED")
	}
	return fw.Summary{"model_count": len(s.models), "configuration_source": "local_ai.models", "execution_mode": "local"}, nil
}
func (s *LocalAI) ModelProfiles(ctx context.Context, _ string) ([]fw.ModelProfile, error) {
	if _, err := mediaTenant(ctx); err != nil {
		return nil, aiError(401, "UNAUTHORIZED")
	}
	out := []fw.ModelProfile{}
	for _, m := range s.models {
		out = append(out, fw.ModelProfile{"name": m.Key, "model_key": m.Key, "model": m.Model, "provider": m.Provider, "status": "configured", "modalities": append([]string(nil), m.Modalities...), "endpoint": m.Endpoint, "timeout_seconds": m.TimeoutSeconds})
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["model_key"].(string) < out[j]["model_key"].(string) })
	return out, nil
}
func (s *LocalAI) ProviderProfiles(ctx context.Context, _ string) ([]fw.ProviderProfile, error) {
	models, err := s.ModelProfiles(ctx, "")
	if err != nil {
		return nil, err
	}
	out := []fw.ProviderProfile{}
	seen := map[string]bool{}
	for _, m := range models {
		p := m["provider"].(string)
		endpoint := m["endpoint"].(string)
		key := p + "/" + endpoint
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, fw.ProviderProfile{"name": p, "provider": p, "endpoint": endpoint, "status": "configured"})
	}
	return out, nil
}
func (s *LocalAI) Routing(ctx context.Context, _ string) (fw.RoutingConfig, error) {
	if _, err := mediaTenant(ctx); err != nil {
		return nil, aiError(401, "UNAUTHORIZED")
	}
	return fw.RoutingConfig{"selection": "explicit_model_key", "fallback_enabled": false}, nil
}
func (s *LocalAI) Health(ctx context.Context, _ string) (fw.HealthStatus, error) {
	if _, err := mediaTenant(ctx); err != nil {
		return nil, aiError(401, "UNAUTHORIZED")
	}
	status := "not_verified"
	if len(s.models) == 0 {
		status = "not_configured"
	}
	return fw.HealthStatus{"status": status, "connectivity_verified": false}, nil
}
