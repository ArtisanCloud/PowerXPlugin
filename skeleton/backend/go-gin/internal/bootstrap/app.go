package bootstrap

import (
	"context"
	"fmt"
	"os"
	"sync"

	fwiamadapters "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	federatedChallenge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/challenge"
	federatedContracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
	federatedProviders "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers"
	providerDingTalk "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers/dingtalk"
	providerLark "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers/lark"
	providerWeCom "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers/wecom"
	federatedRisk "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/risk"
	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	delegatedadapter "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/adapters/delegated"
	localadapter "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/adapters/local"
	federatedService "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/federated"
	sharedapp "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"gorm.io/gorm"
)

type FederatedRuntime struct {
	Factory   federatedContracts.ProviderFactory
	Challenge federatedContracts.ChallengeManager
	Risk      federatedContracts.RiskEvaluator
}

var (
	federatedRuntimeMu sync.RWMutex
	federatedRuntime   *FederatedRuntime
)

func BootstrapPlugin(ctx context.Context, cfg *config.Config) (*gorm.DB, error) {
	// 初始化日志
	logCfg := cfg.Logging
	if logCfg == nil {
		logCfg = &config.LoggingConfig{}
	}
	policy := runtimelogging.ResolveWithHostDefaults(runtimelogging.Policy{
		Mode:   runtimelogging.ModeStandalone,
		Sinks:  []runtimelogging.SinkType{runtimelogging.SinkType(logCfg.Output)},
		Format: logCfg.Format,
		Level:  logCfg.Level,
		Retry: runtimelogging.RetryPolicy{
			Enabled:     true,
			MaxAttempts: 3,
			BackoffMS:   200,
		},
	})
	if runtimelogging.IsHostProxyMode() {
		policy.Mode = runtimelogging.ModeHost
	}
	if err := runtimelogging.ValidatePolicy(policy); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "logging policy validation failed, fallback to default: %v\n", err)
		policy = runtimelogging.ResolveWithHostDefaults(runtimelogging.DefaultPolicy())
	}

	logger.Init(
		policy.Level,
		policy.Format,
		runtimelogging.PrimaryOutput(policy),
		logCfg.FilePath,
		logCfg.MaxSize,
		logCfg.MaxBackups,
		logCfg.MaxAge,
		logCfg.HTTPAccess,
	)
	logger.Info("Starting PowerX Plugin...")

	// 初始化 schema
	models.InitSchemaFrom(cfg.Database.Schema)

	// 连接数据库（在进程生命周期内保持打开；在优雅退出时关闭）
	queryDB, err := db.Connect(cfg.Database)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}

	initFederatedRuntime(queryDB)

	return queryDB, nil
}

func initFederatedRuntime(queryDB *gorm.DB) {
	registry := federatedProviders.NewRegistry()
	wecomConfigSvc := federatedService.NewWeComConfigService(queryDB)
	dingtalkConfigSvc := federatedService.NewDingTalkConfigService(queryDB)
	larkConfigSvc := federatedService.NewLarkConfigService(queryDB)
	_ = registry.Register(providerWeCom.NewWithResolver(func(ctx context.Context, tenantUUID string) (providerWeCom.Config, error) {
		return wecomConfigSvc.ResolveProviderConfig(ctx, tenantUUID)
	}))
	_ = registry.Register(providerDingTalk.NewWithResolver(func(ctx context.Context, tenantUUID string) (providerDingTalk.Config, error) {
		return dingtalkConfigSvc.ResolveProviderConfig(ctx, tenantUUID)
	}))
	_ = registry.Register(providerLark.NewWithResolver(func(ctx context.Context, tenantUUID string) (providerLark.Config, error) {
		return larkConfigSvc.ResolveProviderConfig(ctx, tenantUUID)
	}))

	federatedRuntimeMu.Lock()
	federatedRuntime = &FederatedRuntime{
		Factory:   registry,
		Challenge: federatedChallenge.NewManager(),
		Risk:      federatedRisk.NewEvaluator(0),
	}
	federatedRuntimeMu.Unlock()
}

func Federated() *FederatedRuntime {
	federatedRuntimeMu.RLock()
	defer federatedRuntimeMu.RUnlock()
	return federatedRuntime
}

// SetFederatedForTests 允许测试覆盖联邦运行时依赖。
func SetFederatedForTests(rt *FederatedRuntime) {
	federatedRuntimeMu.Lock()
	federatedRuntime = rt
	federatedRuntimeMu.Unlock()
}

// BindFrameworkIAM 在启动期将 skeleton IAM 能力绑定到 framework registry。
func BindFrameworkIAM(deps *sharedapp.Deps, registry *fwiamadapters.Registry) error {
	if deps == nil || registry == nil {
		return nil
	}

	var (
		mode   fwiamcontracts.IAMMode
		bundle fwiamadapters.Bundle
		err    error
	)

	switch deps.IAMMode {
	case iamservice.IAMModeDelegated:
		mode = fwiamcontracts.IAMModeDelegated
		bundle, err = delegatedadapter.NewBundle(deps.AuthProxy)
	default:
		mode = fwiamcontracts.IAMModeLocal
		bundle, err = localadapter.NewBundle(deps.IAMDirectory)
	}
	if err != nil {
		return err
	}
	if err := registry.Bind(mode, bundle); err != nil {
		return err
	}

	deps.IAMRegistry = registry
	deps.IAMDirectoryService, _ = registry.Directory()
	deps.IAMAuthzService, _ = registry.Authz()
	deps.IAMContextService, _ = registry.IdentityContext()
	return nil
}
