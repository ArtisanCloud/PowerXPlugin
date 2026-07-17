package bootstrap

import (
	"context"
	"os"
	"strings"

	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	fwprovider "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
)

// ProviderResolver determines whether business providers should use delegated
// (PowerX Core) or local data sources.
type ProviderResolver struct {
	mode   fwprovider.Mode
	source string
	record fwprovider.ModeResolutionRecord
	err    error
}

func NewProviderResolver(cfg *config.Config) *ProviderResolver {
	var configMode string
	if cfg != nil && cfg.Context != nil {
		configMode = cfg.Context.ProviderMode
	}
	envMode := strings.TrimSpace(os.Getenv("POWERX_PROVIDER_MODE"))
	proxy := strings.TrimSpace(os.Getenv("POWERX_PROXY"))
	envName := ""
	if cfg != nil && cfg.Server != nil {
		envName = strings.TrimSpace(cfg.Server.Mode)
	}

	resolver := fwprovider.ModeResolver{}
	mode, record, err := resolver.Resolve(fwprovider.ResolveInput{
		ConfigMode:  configMode,
		EnvMode:     envMode,
		PowerXProxy: proxy,
		Environment: envName,
	})
	if mode == "" {
		mode = fwprovider.ModeLocal
	}
	source := record.Audit.Source
	if strings.TrimSpace(source) == "" {
		source = "auto"
	}

	result := &ProviderResolver{
		mode:   mode,
		source: source,
		record: record,
		err:    err,
	}

	if cfg != nil && cfg.Logging != nil && cfg.Logging.DebugMode {
		logger.InfoCtx(logger.WithLogFields(context.Background(), map[string]interface{}{
			"module":               "provider",
			"biz_scene":            "provider_mode_resolve",
			"biz_domain":           "runtime",
			"component":            "bootstrap.provider_resolver",
			"provider_mode":        mode,
			"source":               source,
			"POWERX_PROXY":         os.Getenv("POWERX_PROXY"),
			"POWERX_PROVIDER_MODE": os.Getenv("POWERX_PROVIDER_MODE"),
			"config_mode":          configMode,
			"conflict":             record.ConflictDetected,
			"decision":             record.DecisionReason,
		}), "Provider mode resolved")
	}
	return result
}

func (r *ProviderResolver) Mode() fwprovider.Mode {
	if r == nil {
		return fwprovider.ModeLocal
	}
	return r.mode
}

func (r *ProviderResolver) Source() string {
	if r == nil {
		return "auto"
	}
	return r.source
}

func (r *ProviderResolver) IsLocal() bool {
	return r != nil && r.mode == fwprovider.ModeLocal
}

func (r *ProviderResolver) IsDelegated() bool {
	return r != nil && r.mode == fwprovider.ModeDelegated
}

func (r *ProviderResolver) Err() error {
	if r == nil {
		return nil
	}
	return r.err
}

func (r *ProviderResolver) Record() fwprovider.ModeResolutionRecord {
	if r == nil {
		return fwprovider.ModeResolutionRecord{}
	}
	return r.record
}

func (r *ProviderResolver) IAMAdapterMode() iamservice.IAMAdapterMode {
	switch r.Mode() {
	case fwprovider.ModeDelegated:
		return iamservice.IAMAdapterModeDelegated
	default:
		return iamservice.IAMAdapterModeLocal
	}
}

func (r *ProviderResolver) FrameworkIAMAdapterMode() fwiamcontracts.IAMAdapterMode {
	switch r.Mode() {
	case fwprovider.ModeDelegated:
		return fwiamcontracts.IAMAdapterModeDelegated
	default:
		return fwiamcontracts.IAMAdapterModeLocal
	}
}

func (r *ProviderResolver) IsConflict() bool {
	return r != nil && fwprovider.IsConflict(r.err)
}

func (r *ProviderResolver) IsInvalid() bool {
	return r != nil && fwprovider.IsInvalid(r.err)
}
