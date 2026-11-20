package runtimeexample

import (
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/cache"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/taskcenter"
	"net/http"
)

// Build is called once at startup; only the selected mode constructs adapters.
func Build(mode provider.Mode, cfg hostapi.Config, tokens hostapi.TokenProvider, h *http.Client) (*cache.Runtime, *taskcenter.Runtime, error) {
	var lc, dc cache.Service
	var lt, dt taskcenter.Service
	var err error
	switch mode {
	case provider.ModeLocal:
		lc = NewMemoryCache()
		lt = NewMemoryTasks()
	case provider.ModeDelegated:
		dc, err = cache.NewHostProvider(cfg, tokens, h)
		if err != nil {
			return nil, nil, err
		}
		dt, err = taskcenter.NewHostProvider(cfg, tokens, h)
		if err != nil {
			return nil, nil, err
		}
	}
	cr, err := cache.NewRuntime(mode, lc, dc)
	if err != nil {
		return nil, nil, err
	}
	tr, err := taskcenter.NewRuntime(mode, lt, dt)
	if err != nil {
		return nil, nil, err
	}
	return cr, tr, nil
}
