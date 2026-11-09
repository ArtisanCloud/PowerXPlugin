package bootstrap

import (
	"context"
	frameworkrealtime "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/realtime"
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
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	localvector "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/knowledge/vectorstore"
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
	if err := initializeConfiguredLogging(cfg.Logging); err != nil {
		return nil, err
	}
	logger.Info("Starting PowerX Plugin...")

	// 初始化 schema
	models.InitSchemaFrom(cfg.Database.Schema)

	// 连接数据库（在进程生命周期内保持打开；在优雅退出时关闭）
	queryDB, err := db.Connect(cfg.Database)
	if err != nil {
		logger.FatalWith(nil, context.Background(), "Failed to connect to database", logger.Fields{
			"module":     "db",
			"biz_scene":  "db_connect",
			"biz_domain": "runtime_ops",
			"component":  "bootstrap.app",
			"error":      err.Error(),
		})
	}
	if !db.IsReadOnly() && cfg.LocalPGVectorEnabled() && cfg.Knowledge.VectorStore.PGVector.EnableMigrations {
		pg := cfg.Knowledge.VectorStore.PGVector
		if err := localvector.EnsurePGVectorTable(ctx, queryDB, localvector.PGVectorConfig{
			Schema:     pg.Schema,
			Table:      pg.Table,
			Dimensions: pg.Dimensions,
			Lists:      pg.Lists,
		}, cfg.Database.Schema); err != nil {
			logger.FatalWith(nil, context.Background(), "Failed to initialize local knowledge pgvector store", logger.Fields{
				"module":    "knowledge",
				"component": "pgvector",
				"error":     err.Error(),
			})
		}
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
		mode   fwiamcontracts.IAMAdapterMode
		bundle fwiamadapters.Bundle
		err    error
	)

	switch deps.IAMAdapterMode {
	case iamservice.IAMAdapterModeDelegated:
		mode = fwiamcontracts.IAMAdapterModeDelegated
		bundle, err = delegatedadapter.NewBundle(deps.AuthProxy)
	default:
		mode = fwiamcontracts.IAMAdapterModeLocal
		bundle, err = localadapter.NewBundle(deps.IAMDirectory)
		bundle.AuthorizationBaselines = deps.LocalAuthorizationBaselines
	}
	if err != nil {
		return err
	}
	if bundle.AuthorizationBaselines != nil {
		bundle.AuthorizationBaselines, err = fwiamadapters.ValidateAuthorizationBaselines(sharedapp.PluginID, bundle.AuthorizationBaselines)
		if err != nil {
			return err
		}
	}
	if err := registry.Bind(mode, bundle); err != nil {
		return err
	}

	deps.IAMRegistry = registry
	deps.WSIdentityLifecycle = frameworkrealtime.NewIdentityLifecycle()
	deps.IAMDirectoryService, _ = registry.Directory()
	deps.IAMAuthzService, _ = registry.Authz()
	deps.IAMContextService, _ = registry.IdentityContext()
	deps.IAMAuthorizationBaselines, _ = registry.AuthorizationBaselines()
	return nil
}
