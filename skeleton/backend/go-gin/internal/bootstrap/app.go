package bootstrap

import (
	"context"

	fwiamadapters "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	fwiamcontracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	delegatedadapter "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/adapters/delegated"
	localadapter "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam/adapters/local"
	sharedapp "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"gorm.io/gorm"
)

func BootstrapPlugin(ctx context.Context, cfg *config.Config) (*gorm.DB, error) {
	// 初始化日志
	logCfg := cfg.Logging
	if logCfg == nil {
		logCfg = &config.LoggingConfig{}
	}
	logger.Init(
		logCfg.Level,
		logCfg.Format,
		logCfg.Output,
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

	return queryDB, nil
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
