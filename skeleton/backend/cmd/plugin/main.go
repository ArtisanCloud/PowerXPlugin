package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	fwbootstrap "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
	fwrouter "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
	runtimecap "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/cmd/plugin/runtime"
	pluginbootstrap "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	dbpkg "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	marketplacerepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/marketplace"
	repository "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/plugin"
	grpcserver "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/grpc/server"
	powerxclient "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/powerx"
	marketplacejobs "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/jobs/marketplace"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	manifestx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/manifestx"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/auth"
	capmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/capability"
	opsmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/operations"
	pluginrouter "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/router"
	httpserver "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/server"
	agent "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/agent"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	marketplacesvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/marketplace"
	recommendation "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/recommendation"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/utils"
	"golang.org/x/sync/errgroup"
)

func main() {
	rootCtx := context.Background()
	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()

	if os.Getenv("CONFIG_PATH") == "" && os.Getenv("POWERX_PLUGIN_CONFIG_DIR") != "" {
		os.Setenv("CONFIG_PATH", os.Getenv("POWERX_PLUGIN_CONFIG_DIR"))
	}

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	if err := pluginbootstrap.EnsureLocalIAMSecret(cfg); err != nil {
		logger.WithError(err).Fatal("Failed to ensure local IAM secret")
	}

	// 初始化日志隐私掩码规则
	masking := cfg.SecurityBaselineConfig().MaskingRules
	if len(masking.PIIFields) > 0 {
		placeholder := masking.LogRedaction.Placeholder
		logger.ConfigurePrivacyMasker(masking.PIIFields, placeholder)
	}

	// ★ 在这里把 HTTP/GRPC 的占位符先解析掉（一定要在起服务之前）
	//   - HTTP 用 PORT（由 PowerX 的 supervisor 注入）
	cfg.Server.BindAddr = utils.ResolveDynamicAddr(cfg.Server.BindAddr, "PORT")

	//   - gRPC 用 POWERX_GRPC_PORT（由 PowerX 的 Enable 阶段注入）
	if cfg.GRPCServer != nil {
		// 如果你的字段叫 Addr，就把下一行改成：cfg.GRPCServer.Addr = resolveDynamicAddr(cfg.GRPCServer.Addr, "POWERX_GRPC_PORT")
		cfg.GRPCServer.Addr = utils.ResolveDynamicAddr(cfg.GRPCServer.Addr, "POWERX_GRPC_PORT")
	}

	// 初始化插件
	queryDB, err := pluginbootstrap.BootstrapPlugin(ctx, cfg)
	if err != nil {
		logger.WithError(err).Fatal("Failed to bootstrap plugin")
	}

	// 在初始化 gRPC 客户端之前，尝试从本地数据库加载租户凭证（若存在），以便通过 STS 获取短期令牌
	if cfg.GRPCUpstream != nil && strings.TrimSpace(cfg.GRPCUpstream.TenantUUID) != "" {
		// 延迟依赖：仅当配置未提供 STS client 时，尝试 DB 加载；若配置已有，则优先生效
		if cfg.GRPCUpstream.STSClientID == "" || cfg.GRPCUpstream.STSClientSecret == "" {
			repo := repository.NewCredentialsRepository(queryDB)
			svc := agent.NewCredentialService(cfg, repo)
			if cid, sec, err := svc.LoadDecryptedCredentials(rootCtx, cfg.GRPCUpstream.TenantUUID, app.PluginID); err == nil {
				cfg.GRPCUpstream.STSClientID = cid
				cfg.GRPCUpstream.STSClientSecret = sec
				logger.Info("Loaded STS credentials for tenant from DB")
			} else {
				logger.WithError(err).Warn("No DB-stored credentials found or failed to decrypt; will rely on config/env if provided")
			}
		}
	}

	iamResolver := pluginbootstrap.NewIAMResolver(cfg)
	logger.WithFields(logger.Fields{
		"mode":   iamResolver.Mode(),
		"source": iamResolver.Source(),
	}).Info("IAM mode resolved")
	auth.ObserveMode(iamResolver.Mode().String())

	var authClient *authproxy.DelegatedClient
	var localIAM iamservice.IAMDirectory
	if iamResolver.Mode() == iamservice.IAMModeDelegated {
		client, err := authproxy.NewDelegatedClient("", "")
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize delegated auth proxy; auth endpoints will be unavailable")
		} else {
			authClient = client
		}
	} else {
		dir, err := iamservice.NewLocalDirectory(queryDB, cfg)
		if err != nil {
			logger.WithError(err).Fatal("Failed to initialize local IAM directory")
		}
		localIAM = dir
	}

	// 初始化 PowerX gRPC Client 客户端
	pxc := pluginbootstrap.BootstrapGRPCClient(rootCtx, cfg.GRPCUpstream)

	taxLogger := logger.WithField("component", "tax_provider_client")
	taxClient, err := marketplacesvc.NewTaxProviderClient(cfg, nil, taxLogger)
	if err != nil {
		taxLogger.WithError(err).Warn("Tax provider client initialization failed")
	}

	var licenseCache marketplacesvc.LicenseCache
	cacheCfg := cfg.LicenseCacheConfig()
	cacheLogger := logger.WithField("component", "marketplace_license_cache")
	if strings.EqualFold(strings.TrimSpace(cacheCfg.Provider), "redis") {
		if lc, err := marketplacesvc.NewRedisLicenseCache(cacheCfg.RedisURL, cacheCfg.KeyPrefix, cacheLogger); err != nil {
			cacheLogger.WithError(err).Warn("license cache initialization failed")
		} else {
			licenseCache = lc
		}
	}

	capLog := logger.WithField("component", "capabilities_manager")
	capManager := capabilities.NewManager(cfg, capLog)
	capClient := powerxclient.NewCapabilityClientFromEnv(capLog)
	capMetrics := capmetrics.NewMetrics()
	if err := runtimecap.SyncCapabilities(ctx, capManager, capClient, capMetrics); err != nil {
		logger.WithError(err).Fatal("Failed to initialize capability catalog")
	}

	deps := &app.Deps{
		DB:                  queryDB,
		Ctx:                 rootCtx,
		PowerXClient:        pxc,
		Config:              cfg,
		CapabilitiesManager: capManager,
		CapabilityMetrics:   capMetrics,
		TaxProviderClient:   taxClient,
		MarketplaceBilling:  nil,
		LicenseAuthority:    nil,
		LicenseCache:        licenseCache,
		OperationsMetrics:   opsmetrics.NewMetrics(),
		AdminConsoleMetrics: adminmetrics.NewMetrics(),
		IAMMode:             iamResolver.Mode(),
		IAMModeSource:       iamResolver.Source(),
		AuthProxy:           authClient,
		IAMDirectory:        localIAM,
	}

	listingRepo := marketplacerepo.NewListingRepository(queryDB)
	licenseRepoGlobal := marketplacerepo.NewLicenseRepository(queryDB)
	metricsProvider := recommendation.NewListingMetricsProvider(listingRepo)
	var syncJob *marketplacejobs.SyncJob
	if cfg == nil || cfg.Marketplace == nil || cfg.Marketplace.Recommendation.Enabled {
		syncJob = marketplacejobs.NewSyncJob(cfg, listingRepo, metricsProvider, logger.WithField("component", "marketplace_recommendation_sync"), listingRepo.ListTenantUuids)
	}

	var renewalJob *marketplacejobs.RenewalNotifier
	if cfg != nil && cfg.LicenseReminderLead() > 0 {
		renewalJob = marketplacejobs.NewLicenseRenewalNotifier(cfg, licenseRepoGlobal, logger.WithField("component", "marketplace_license_renewal_notifier"), listingRepo.ListTenantUuids, nil)
	}

	// 设置 gin engine 路由
	r := pluginrouter.NewRouter(cfg, deps)
	engine := r.Setup()

	// 创建 gRPC 服务器（可选）
	gs, err := grpcserver.NewGRPCServer(ctx, deps, cfg.GRPCServer)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create gRPC server")
	}

	appCfg := &fwbootstrap.Config{
		Listen:     cfg.Server.BindAddr,
		Env:        cfg.Server.Mode,
		Standalone: true,
	}
	fwApp := fwbootstrap.NewApp(appCfg)

	if err := fwrouter.AttachHTTPServer(fwApp); err != nil {
		logger.WithError(err).Fatal("Failed to attach HTTP server")
	}
	fwrouter.RegisterFrameworkRoutes(fwApp)
	fwrouter.RegisterPluginRoutes(fwApp, func(r fwbootstrap.Router) {
		httpserver.RegisterGinRoutes(r, engine)
	})

	if err := manifest.Register(fwApp, manifestx.Plugin()); err != nil {
		logger.WithError(err).Fatal("Failed to register manifest")
	}

	// 使用 errgroup 并发启动服务器
	g, groupCtx := errgroup.WithContext(ctx)

	if syncJob != nil {
		g.Go(func() error {
			syncJob.Run(groupCtx)
			return nil
		})
	}
	if renewalJob != nil {
		g.Go(func() error {
			renewalJob.Run(groupCtx)
			return nil
		})
	}

	g.Go(func() error {
		logger.WithField("addr", cfg.Server.BindAddr).Info("Starting HTTP server...")
		return fwApp.Run()
	})

	if gs != nil {
		g.Go(func() error {
			return gs.Serve(groupCtx)
		})
	}

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 在单独的 goroutine 中等待信号
	go func() {
		<-quit
		logger.Info("Shutting down servers...")

		cancel()

		if err := fwApp.Shutdown(); err != nil {
			logger.WithError(err).Error("HTTP server shutdown error")
		} else {
			logger.Info("HTTP server shutdown completed")
		}

		// 关闭数据库连接
		if err := dbpkg.Close(); err != nil {
			logger.WithError(err).Error("DB close error")
		} else {
			logger.Info("Database connection closed")
		}

		if gs != nil {
			gs.GracefulStop()
		}
	}()

	// 等待服务器启动失败或优雅关闭
	if err := g.Wait(); err != nil {
		logger.WithError(err).Error("Server error")
		os.Exit(1)
	}

	logger.Info("All servers shutdown completed")
}
