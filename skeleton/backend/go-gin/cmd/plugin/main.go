package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	fwbootstrap "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
	fwrouter "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	runtimecap "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/cmd/plugin/runtime"
	pluginbootstrap "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	dbpkg "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/db"
	marketplacerepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/marketplace"
	repository "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/plugin"
	grpcserver "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/grpc/server"
	capgateway "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/gateway"
	powerxclient "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/integrations/powerx"
	integrationjobs "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/jobs/integration"
	marketplacejobs "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/jobs/marketplace"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	manifestx "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/manifestx"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/auth"
	capmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/capability"
	ebmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/event_bridge"
	integrationmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/integration"
	opsmetrics "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/observability/operations"
	pluginrouter "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/router"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/security"
	httpserver "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/server"
	agent "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/agent"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/iam"
	marketplacesvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/marketplace"
	recommendation "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/recommendation"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/utils"
	"golang.org/x/sync/errgroup"

	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
)

type bridgeRecorder struct{}

type startupLog struct {
	level  string
	msg    string
	fields logger.Fields
}

func (bridgeRecorder) RecordEmit(pluginID, tenantUUID, topic, result string) {
	ebmetrics.RecordEmit(pluginID, tenantUUID, topic, result)
}

func (bridgeRecorder) RecordConsume(pluginID, tenantUUID, topic, result string) {
	ebmetrics.RecordConsume(pluginID, tenantUUID, topic, result)
}

func (bridgeRecorder) RecordDrop(pluginID, tenantUUID, topic, reason string) {
	ebmetrics.RecordDrop(pluginID, tenantUUID, topic, reason)
}

func (bridgeRecorder) ObserveLatencyMs(pluginID, tenantUUID, topic, op string, ms float64) {
	ebmetrics.ObserveLatencyMs(pluginID, tenantUUID, topic, op, ms)
}

func flushStartupLogs(logs []startupLog) {
	for _, item := range logs {
		entry := logger.WithFields(item.fields)
		switch strings.ToLower(strings.TrimSpace(item.level)) {
		case "warn", "warning":
			entry.Warn(item.msg)
		case "error":
			entry.Error(item.msg)
		case "debug":
			entry.Debug(item.msg)
		default:
			entry.Info(item.msg)
		}
	}
}

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
	if err := validateTaskBusProviderConflict(cfg, os.Getenv("POWERX_PROXY")); err != nil {
		fmt.Printf("Failed preflight mode check: %v\n", err)
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
	//   - 宿主模式（POWERX_PROXY=1）：优先 POWERX_HTTP_ADDR，缺失则回退 config/PORT（并告警）
	//   - 本地模式：允许 PORT/配置回退
	hostMode := strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1"
	pendingStartupLogs := make([]startupLog, 0, 2)
	if hostMode {
		httpBindAddr := strings.TrimSpace(os.Getenv("POWERX_HTTP_ADDR"))
		if httpBindAddr != "" {
			cfg.Server.BindAddr = httpBindAddr
		} else {
			cfg.Server.BindAddr = utils.ResolveDynamicAddr(cfg.Server.BindAddr, "PORT")
			pendingStartupLogs = append(pendingStartupLogs, startupLog{
				level: "warn",
				msg:   "POWERX_HTTP_ADDR is empty when POWERX_PROXY=1, fallback to config/PORT bind address",
				fields: logger.Fields{
					"POWERX_PROXY":     strings.TrimSpace(os.Getenv("POWERX_PROXY")),
					"POWERX_HTTP_ADDR": strings.TrimSpace(os.Getenv("POWERX_HTTP_ADDR")),
					"PORT":             strings.TrimSpace(os.Getenv("PORT")),
					"bind_addr":        cfg.Server.BindAddr,
				},
			})
		}
	} else {
		cfg.Server.BindAddr = utils.ResolveDynamicAddr(cfg.Server.BindAddr, "PORT")
	}
	pendingStartupLogs = append(pendingStartupLogs, startupLog{
		level: "info",
		msg:   "Resolved HTTP bind address",
		fields: logger.Fields{
			"bind_addr":            cfg.Server.BindAddr,
			"host_mode":            hostMode,
			"env_POWERX_HTTP_ADDR": strings.TrimSpace(os.Getenv("POWERX_HTTP_ADDR")),
			"env_PORT":             strings.TrimSpace(os.Getenv("PORT")),
		},
	})

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
	flushStartupLogs(pendingStartupLogs)

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
	if err := iamResolver.Err(); err != nil {
		logger.WithFields(logger.Fields{
			"iam_mode":       iamResolver.Mode(),
			"iam_source":     iamResolver.Source(),
			"conflict":       iamResolver.Record().ConflictDetected,
			"decision":       iamResolver.Record().DecisionReason,
			"config_mode":    iamResolver.Record().ConfigMode,
			"env_mode":       iamResolver.Record().EnvMode,
			"effective_mode": iamResolver.Record().EffectiveMode,
		}).WithError(err).Fatal("IAM mode resolution failed")
	}
	mode := strings.ToLower(strings.TrimSpace(iamResolver.Mode().String()))
	if mode == "" {
		mode = "unknown"
	}
	if err := validateHostDelegatedEnvContract(iamResolver); err != nil {
		logger.WithError(err).WithFields(logger.Fields{
			"iam_mode":     iamResolver.Mode(),
			"iam_source":   iamResolver.Source(),
			"POWERX_PROXY": normalizedProxy(strings.TrimSpace(os.Getenv("POWERX_PROXY"))),
			"IAM_MODE":     strings.TrimSpace(os.Getenv("IAM_MODE")),
			"IAMMode":      strings.TrimSpace(os.Getenv("IAMMode")),
		}).Fatal("Host delegated env contract validation failed")
	}
	contractErr := validateDelegatedGatewayContract(cfg, iamResolver.Mode())
	gatewayFields := gatewayContractLogFields(cfg, mode)
	gatewayFields["gateway_contract_valid"] = contractErr == nil
	logger.WithFields(gatewayFields).Info("Gateway contract status")
	integrationmetrics.SetPluginGatewayConfigValid(app.PluginID, mode, contractErr == nil)
	if contractErr != nil && iamResolver.Mode() == iamservice.IAMModeDelegated {
		logger.WithFields(logger.Fields{
			"code":     contractErr.Code,
			"required": strings.Join(contractErr.Required, ","),
			"present":  strings.Join(contractErr.Present, ","),
			"iam_mode": contractErr.IAMMode,
		}).WithError(contractErr).Fatal("Delegated gateway contract validation failed")
	}

	logRuntimeModeMatrix(cfg, iamResolver)
	auth.ObserveMode(iamResolver.Mode().String())
	if cfg != nil && cfg.Logging != nil && cfg.Logging.DebugMode {
		gatewayMode := "local"
		if os.Getenv("POWERX_PROXY") == "1" && cfg != nil && cfg.Gateway != nil {
			if strings.TrimSpace(cfg.Gateway.BaseURL) != "" &&
				strings.TrimSpace(cfg.Gateway.ToolToken) != "" {
				gatewayMode = "host"
			}
		}
		logger.WithFields(logger.Fields{
			"iam_mode":            iamResolver.Mode(),
			"iam_source":          iamResolver.Source(),
			"gateway_mode":        gatewayMode,
			"POWERX_PROXY":        os.Getenv("POWERX_PROXY"),
			"IAMMode":             os.Getenv("IAMMode"),
			"IAM_MODE":            os.Getenv("IAM_MODE"),
			"PX_GATEWAY_BASE_URL": strings.TrimSpace(cfg.Gateway.BaseURL),
		}).Info("Mode decision")
	}

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

	var capabilityGateway *capgateway.Client
	if cfg != nil && cfg.Gateway != nil {
		ensureToolTokenFresh(ctx, cfg)
		capabilityGateway = capgateway.NewClient(cfg, logger.WithField("component", "capability_gateway_client"))
	}

	// 初始化 WS Bus Hub（standalone 可选 Redis，默认内存）
	wsLogger := logger.WithField("component", "ws_bus")
	var wsHub fwwsbus.LocalHub
	if cfg != nil && cfg.WSBus != nil && strings.EqualFold(cfg.WSBus.Provider, "redis") {
		hub, err := fwwsbus.NewRedisHub(fwwsbus.RedisHubConfig{
			RedisURL: cfg.WSBus.RedisURL,
			Channel:  cfg.WSBus.Channel,
			Logger:   nil,
		})
		if err != nil {
			wsLogger.WithError(err).Warn("Failed to initialize redis ws bus; falling back to memory")
			wsHub = fwwsbus.NewMemoryHub()
		} else {
			if err := hub.Start(ctx); err != nil {
				wsLogger.WithError(err).Warn("Failed to start redis ws bus; falling back to memory")
				wsHub = fwwsbus.NewMemoryHub()
			} else {
				wsHub = hub
			}
		}
	} else {
		wsHub = fwwsbus.NewMemoryHub()
	}

	// 初始化 EventBridge Emitter（本地/TaskBus/双写；TaskBus SDK 未就绪时可自动降级到本地 emitter）
	eventLogger := logger.WithField("component", "event_bridge")
	eventCfg := cfg.EventBridge
	if eventCfg != nil && strings.TrimSpace(eventCfg.SourcePlugin) == "" {
		eventCfg.SourcePlugin = app.PluginID
	}
	taskBusRuntime := resolveTaskBusRuntime(cfg, wsHub, eventLogger)

	var bridgeEmitter fweventbridge.Emitter
	if eventCfg == nil {
		bridgeEmitter = fweventbridge.NewLocalEmitter(1024)
	} else {
		factory, err := fweventbridge.NewFactory(fweventbridge.Config{
			Enabled:         eventCfg.Enabled,
			Mode:            eventCfg.Mode,
			FallbackToLocal: eventCfg.FallbackToLocal,
			LocalQueueSize:  eventCfg.LocalQueueSize,
		})
		if err != nil {
			eventLogger.WithError(err).Warn("Invalid event bridge config; falling back to local emitter")
			bridgeEmitter = fweventbridge.NewLocalEmitter(1024)
		} else {
			factory.WithMetrics(bridgeRecorder{})
			factory.WithTaskBusProvider(fweventbridge.NewTaskBusEmitterAdapter(taskBusRuntime.Provider))
			bridgeEmitter, err = factory.NewEmitter()
			if err != nil {
				eventLogger.WithError(err).Warn("Failed to initialize event bridge emitter; falling back to local emitter")
				bridgeEmitter = fweventbridge.NewLocalEmitter(1024)
			}
		}
	}

	// 运行时事件权限：从 plugin.yaml 解析 publish/subscribe（若未声明则默认不强制）
	perms, err := security.LoadEventPermissionsFromManifest("", eventLogger)
	if err != nil {
		eventLogger.WithError(err).Warn("Failed to load event permissions; permissions enforcement disabled")
	} else if perms.Enforced() {
		bridgeEmitter = security.NewPermissionedEmitter(bridgeEmitter, perms, eventLogger)
	}

	deps := &app.Deps{
		DB:                  queryDB,
		Ctx:                 rootCtx,
		PowerXClient:        pxc,
		CapabilityGateway:   capabilityGateway,
		Config:              cfg,
		CapabilitiesManager: capManager,
		CapabilityMetrics:   capMetrics,
		TaxProviderClient:   taxClient,
		MarketplaceBilling:  nil,
		LicenseAuthority:    nil,
		LicenseCache:        licenseCache,
		OperationsMetrics:   opsmetrics.NewMetrics(),
		AdminConsoleMetrics: adminmetrics.NewMetrics(),
		EventEmitter:        bridgeEmitter,
		WSBusHub:            wsHub,
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

	// 调度触发统一走 EventEmitter（与手动 event-bridge/emit 共用语义入口）。
	var integrationScheduler *integrationjobs.Scheduler
	if bridgeEmitter != nil {
		schedulerLogger := logger.WithField("component", "integration.scheduler")
		schedulerDispatcher := integrationjobs.NewSchedulerEventDispatcher(cfg, bridgeEmitter, schedulerLogger.WithField("stage", "dispatcher"))
		integrationScheduler = integrationjobs.NewScheduler(schedulerLogger)
		integrationScheduler.SetDispatcher(schedulerDispatcher)
		integrationScheduler.Register(integrationjobs.NewJobFunc("runtime.scheduler.trigger", time.Minute, func(ctx context.Context) error {
			return nil
		}))
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
		Standalone: os.Getenv("POWERX_PROXY") != "1",
		Gateway: fwbootstrap.GatewayConfig{
			BaseURL:    strings.TrimSpace(cfg.Gateway.BaseURL),
			AuthScheme: strings.TrimSpace(cfg.Gateway.AuthScheme),
			APIKey:     strings.TrimSpace(cfg.Gateway.APIKey),
			TenantID:   strings.TrimSpace(cfg.Gateway.TenantUUID),
			Timeout:    cfg.Gateway.Timeout,
			UserAgent:  strings.TrimSpace(cfg.Gateway.UserAgent),
		},
	}
	fwApp := fwbootstrap.NewApp(appCfg)
	if err := pluginbootstrap.BindFrameworkIAM(deps, fwApp.IAMRegistry()); err != nil {
		logger.WithFields(logger.Fields{
			"iam_mode":        deps.IAMMode,
			"iam_mode_source": deps.IAMModeSource,
		}).WithError(err).Fatal("Failed to bind framework IAM registry")
	}

	if err := fwrouter.AttachHTTPServer(fwApp); err != nil {
		logger.WithError(err).Fatal("Failed to attach HTTP server")
	}
	fwrouter.RegisterFrameworkRoutes(fwApp)
	fwrouter.RegisterPluginRoutes(fwApp, func(r fwbootstrap.Router) {
		httpserver.RegisterGinRoutes(r, engine)
	})
	registerWSRoute(fwApp, engine)

	if err := manifest.Register(fwApp, manifestx.Plugin()); err != nil {
		logger.WithError(err).Fatal("Failed to register manifest")
	}
	// 宿主模式下注册 WS Bus topic（standalone 不触发）
	wsRegisterTopics, err := security.LoadEventFabricTopics(eventLogger)
	if err != nil {
		eventLogger.WithError(err).Warn("failed to load event_fabric topics; fallback to manifest events")
	}
	if len(wsRegisterTopics) == 0 && perms.Enforced() {
		wsRegisterTopics = perms.Topics()
	}
	if len(wsRegisterTopics) == 0 {
		wsRegisterTopics = fwwsbus.AllowedTopics()
	}
	registerResult := fwwsbus.RegisterTopics(fwApp, wsRegisterTopics, fwwsbus.PublishOptions{}, nil)
	if !registerResult.OK {
		logger.WithFields(logger.Fields{
			"code":    registerResult.ErrorCode,
			"message": registerResult.ErrorMessage,
		}).Warn("WS bus topic registration failed")
	}

	// 使用 errgroup 并发启动服务器
	g, groupCtx := errgroup.WithContext(ctx)
	safeGo := func(name string, fn func() error) {
		g.Go(func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.WithFields(logger.Fields{
						"component": "main.goroutine",
						"task":      name,
						"panic":     r,
					}).Error("goroutine panicked")
					err = fmt.Errorf("goroutine panic: %s", name)
				}
			}()
			return fn()
		})
	}

	// 先启动核心 HTTP/gRPC 服务，避免后台任务异常影响 healthz 探活。
	safeGo("http_server", func() error {
		logger.WithField("addr", cfg.Server.BindAddr).Info("Starting HTTP server...")
		return fwApp.Run()
	})
	if gs != nil {
		safeGo("grpc_server", func() error {
			return gs.Serve(groupCtx)
		})
	}

	if integrationScheduler != nil {
		safeGo("integration_scheduler", func() error {
			integrationScheduler.Start(groupCtx)
			<-groupCtx.Done()
			stopCtx, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelStop()
			integrationScheduler.Stop(stopCtx)
			return nil
		})
	}

	if syncJob != nil {
		safeGo("marketplace_sync_job", func() error {
			syncJob.Run(groupCtx)
			return nil
		})
	}
	if renewalJob != nil {
		safeGo("license_renewal_job", func() error {
			renewalJob.Run(groupCtx)
			return nil
		})
	}
	if taskBusRuntime.StartConsumer != nil {
		safeGo("taskbus_consumer", func() error {
			taskBusRuntime.StartConsumer(groupCtx)
			return nil
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

		if capabilityGateway != nil {
			if err := capabilityGateway.Close(); err != nil {
				logger.WithError(err).Warn("Capability gateway client close error")
			}
		}
	}()

	// 等待服务器启动失败或优雅关闭
	if err := g.Wait(); err != nil {
		logger.WithError(err).Error("Server error")
		os.Exit(1)
	}

	logger.Info("All servers shutdown completed")
}

func registerWSRoute(app *fwbootstrap.App, handler http.Handler) {
	if app == nil || app.Router == nil || handler == nil {
		return
	}

	targetPath := path.Join("/api", "ws")

	type httpBridge interface {
		HTTPResponseWriter() http.ResponseWriter
		HTTPRequest() *http.Request
	}

	rewriteToTarget := func(ctx fwbootstrap.Context) {
		bridge, ok := ctx.(httpBridge)
		if !ok {
			ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "http bridge unavailable"})
			return
		}
		req := bridge.HTTPRequest()
		writer := bridge.HTTPResponseWriter()
		if req == nil || writer == nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid http bridge request"})
			return
		}

		cloned := req.Clone(req.Context())
		urlCopy := *cloned.URL
		cloned.URL = &urlCopy
		cloned.URL.Path = targetPath
		cloned.RequestURI = targetPath
		if raw := strings.TrimSpace(cloned.URL.RawQuery); raw != "" {
			cloned.RequestURI += "?" + raw
		}
		handler.ServeHTTP(writer, cloned)
	}

	app.Router.Handle(http.MethodGet, targetPath, rewriteToTarget)
}

func ensureToolTokenFresh(ctx context.Context, cfg *config.Config) {
	if cfg == nil || cfg.Gateway == nil {
		return
	}
	authScheme := strings.ToLower(strings.TrimSpace(cfg.Gateway.AuthScheme))
	if authScheme == "apikey" || authScheme == "api_key" || authScheme == "api-key" {
		return
	}
	token := strings.TrimSpace(cfg.Gateway.ToolToken)
	if token == "" {
		return
	}
	expiry, err := capgateway.ParseTokenExpiry(token)
	log := logger.WithField("component", "gateway_token_monitor")
	if err != nil {
		log.WithError(err).Debug("PX_PLUGIN_TOOL_TOKEN 无法解析有效期")
		return
	}
	now := time.Now().UTC()
	fields := logger.Fields{
		"expiresAt": expiry.UTC().Format(time.RFC3339),
	}

	refreshToken := strings.TrimSpace(cfg.Gateway.RefreshToken)
	if refreshToken == "" || strings.TrimSpace(cfg.Gateway.BaseURL) == "" {
		logTokenExpiryStatus(log, now, expiry, fields)
		return
	}

	if now.After(expiry) || expiry.Sub(now) < 24*time.Hour {
		log.WithFields(fields).Info("PX_PLUGIN_TOOL_TOKEN 即将过期，尝试使用 PX_TOOL_REFRESH_TOKEN 自动刷新")
		newToken, _, err := capgateway.RefreshToolToken(ctx, cfg)
		if err != nil {
			log.WithError(err).Error("PX_PLUGIN_TOOL_TOKEN 刷新失败，请重新执行 `px-plugin login --manifest ./skeleton/plugin.yaml`")
			return
		}
		if nextExpiry, err := capgateway.ParseTokenExpiry(newToken); err == nil {
			fields["expiresAt"] = nextExpiry.UTC().Format(time.RFC3339)
		}
		log.WithFields(fields).Info("PX_PLUGIN_TOOL_TOKEN 已自动刷新，请同步更新 skeleton/.env.local 中的 PX_PLUGIN_TOOL_TOKEN / PX_TOOL_REFRESH_TOKEN")
		return
	}
	logTokenExpiryStatus(log, now, expiry, fields)
}

func logTokenExpiryStatus(log *logger.Entry, now, expiry time.Time, fields logger.Fields) {
	if now.After(expiry) {
		log.WithFields(fields).Error("PX_PLUGIN_TOOL_TOKEN 已过期，请重新执行 `px-plugin login --manifest ./skeleton/plugin.yaml` 刷新凭证")
		return
	}
	if expiry.Sub(now) < 24*time.Hour {
		log.WithFields(fields).Warn("PX_PLUGIN_TOOL_TOKEN 将在 24 小时内过期，请尽快运行 `px-plugin login` 刷新凭证，或设置 PX_TOOL_REFRESH_TOKEN 以便自动刷新")
		return
	}
	log.WithFields(fields).Info("PX_PLUGIN_TOOL_TOKEN 有效")
}

func logRuntimeModeMatrix(cfg *config.Config, iamResolver *pluginbootstrap.IAMResolver) {
	mode := strings.ToLower(strings.TrimSpace(iamResolver.Mode().String()))
	if mode == "" {
		mode = string(iamservice.IAMModeLocal)
	}

	proxyRaw := strings.TrimSpace(os.Getenv("POWERX_PROXY"))
	proxyEnabled := proxyRaw == "1"

	gatewayBaseURL := ""
	gatewayToken := ""
	gatewayAPIKey := ""
	gatewayAuthScheme := ""
	if cfg != nil && cfg.Gateway != nil {
		gatewayBaseURL = strings.TrimSpace(cfg.Gateway.BaseURL)
		gatewayToken = strings.TrimSpace(cfg.Gateway.ToolToken)
		gatewayAPIKey = strings.TrimSpace(cfg.Gateway.APIKey)
		gatewayAuthScheme = strings.ToLower(strings.TrimSpace(cfg.Gateway.AuthScheme))
	}

	missingGateway := make([]string, 0, 3)
	if gatewayBaseURL == "" {
		missingGateway = append(missingGateway, "PX_GATEWAY_BASE_URL")
	}
	switch gatewayAuthScheme {
	case "apikey", "api_key", "api-key":
		if gatewayAPIKey == "" {
			missingGateway = append(missingGateway, "PX_GATEWAY_API_KEY")
		}
	default:
		if gatewayAuthScheme != "bearer" {
			missingGateway = append(missingGateway, "PX_GATEWAY_AUTH_SCHEME=bearer")
		}
		if gatewayToken == "" {
			missingGateway = append(missingGateway, "PX_PLUGIN_TOOL_TOKEN")
		}
	}
	gatewayReady := len(missingGateway) == 0

	wsTarget := "local"
	wsDisplay := "本地"
	if proxyEnabled {
		wsTarget = "host"
		wsDisplay = "宿主（需 PX_GATEWAY_*）"
	}

	fields := logger.Fields{
		"matrix_row":              fmt.Sprintf("%s | %s", mode, normalizedProxy(proxyRaw)),
		"iam_mode":                mode,
		"iam_source":              iamResolver.Source(),
		"POWERX_PROXY":            normalizedProxy(proxyRaw),
		"iam_result":              iamResultLabel(mode),
		"ws_capability_target":    wsTarget,
		"ws_capability_target_zh": wsDisplay,
		"scenario":                modeScenarioLabel(mode, proxyEnabled),
		"priority_note":           modePriorityNote(iamResolver.Source(), mode, proxyEnabled),
	}

	if proxyEnabled {
		fields["gateway_ready"] = gatewayReady
		if !gatewayReady {
			fields["gateway_missing"] = strings.Join(missingGateway, ",")
		}
	}

	logger.WithFields(logger.Fields{
		"matrix_row": fields["matrix_row"],
	}).Info("Runtime mode resolved (2x2)")
	logger.WithFields(logger.Fields{
		"iam_mode":   fields["iam_mode"],
		"iam_source": fields["iam_source"],
		"iam_result": fields["iam_result"],
	}).Info("IAM decision")
	logger.WithFields(logger.Fields{
		"POWERX_PROXY":  fields["POWERX_PROXY"],
		"priority_note": fields["priority_note"],
	}).Info("Decision inputs")
	logger.WithFields(logger.Fields{
		"ws_capability_target":    fields["ws_capability_target"],
		"ws_capability_target_zh": fields["ws_capability_target_zh"],
		"scenario":                fields["scenario"],
	}).Info("WS/Capability routing")

	if proxyEnabled {
		gatewayFields := logger.Fields{
			"gateway_ready": fields["gateway_ready"],
		}
		if !gatewayReady {
			gatewayFields["gateway_missing"] = fields["gateway_missing"]
		}
		logger.WithFields(gatewayFields).Info("Gateway readiness")
	}

	if proxyEnabled && !gatewayReady {
		logger.WithFields(logger.Fields{
			"missing": strings.Join(missingGateway, ","),
		}).Warn("宿主链路已启用，但 delegated gateway contract 未满足；WS/能力注册可能失败")
	}
}

func validateDelegatedGatewayContract(cfg *config.Config, mode iamservice.IAMMode) *capgateway.GatewayConfigError {
	if mode != iamservice.IAMModeDelegated {
		return nil
	}
	return capgateway.ValidateDelegatedConfig(cfg)
}

func validateHostDelegatedEnvContract(iamResolver *pluginbootstrap.IAMResolver) error {
	if iamResolver == nil {
		return nil
	}
	mode := strings.ToLower(strings.TrimSpace(iamResolver.Mode().String()))
	if mode != string(iamservice.IAMModeDelegated) {
		return nil
	}
	// 仅宿主 delegated 场景做 fail-fast；允许 standalone_mock_delegated 本地联调。
	if strings.TrimSpace(os.Getenv("POWERX_PROXY")) != "1" {
		return nil
	}

	missing := make([]string, 0, 2)
	invalid := make([]string, 0, 1)
	iamModeEnv := strings.ToLower(strings.TrimSpace(os.Getenv("IAM_MODE")))
	iamModeLegacy := strings.ToLower(strings.TrimSpace(os.Getenv("IAMMode")))
	if iamModeEnv == "" && iamModeLegacy == "" {
		missing = append(missing, "IAM_MODE=delegated")
	} else if iamModeEnv != "delegated" && iamModeLegacy != "delegated" {
		invalid = append(invalid, fmt.Sprintf("IAM_MODE=%s IAMMode=%s", iamModeEnv, iamModeLegacy))
	}

	scheme := strings.ToLower(strings.TrimSpace(os.Getenv("PX_GATEWAY_AUTH_SCHEME")))
	if scheme == "" {
		missing = append(missing, "PX_GATEWAY_AUTH_SCHEME=bearer")
	} else if scheme != "bearer" {
		invalid = append(invalid, "PX_GATEWAY_AUTH_SCHEME must be bearer")
	}

	if len(missing) == 0 && len(invalid) == 0 {
		return nil
	}

	parts := make([]string, 0, 2)
	if len(missing) > 0 {
		parts = append(parts, "missing: "+strings.Join(missing, ", "))
	}
	if len(invalid) > 0 {
		parts = append(parts, "invalid: "+strings.Join(invalid, ", "))
	}
	return errors.New(strings.Join(parts, "; "))
}

func gatewayContractLogFields(cfg *config.Config, mode string) logger.Fields {
	authScheme := ""
	baseURLPresent := false
	toolTokenPresent := false
	if cfg != nil && cfg.Gateway != nil {
		authScheme = strings.ToLower(strings.TrimSpace(cfg.Gateway.AuthScheme))
		baseURLPresent = strings.TrimSpace(cfg.Gateway.BaseURL) != ""
		toolTokenPresent = strings.TrimSpace(cfg.Gateway.ToolToken) != ""
	}
	return logger.Fields{
		"iam_mode":                 mode,
		"gateway_base_url_present": baseURLPresent,
		"tool_token_present":       toolTokenPresent,
		"auth_scheme":              authScheme,
	}
}

func normalizedProxy(value string) string {
	if strings.TrimSpace(value) == "1" {
		return "1"
	}
	return "0"
}

func iamResultLabel(mode string) string {
	if mode == string(iamservice.IAMModeDelegated) {
		return "委派 IAM"
	}
	return "本地 IAM"
}

func modeScenarioLabel(mode string, proxyEnabled bool) string {
	switch {
	case mode == string(iamservice.IAMModeLocal) && !proxyEnabled:
		return "standalone_local"
	case mode == string(iamservice.IAMModeLocal) && proxyEnabled:
		return "local + proxy（调试态）"
	case mode == string(iamservice.IAMModeDelegated) && !proxyEnabled:
		return "standalone_mock_delegated"
	case mode == string(iamservice.IAMModeDelegated) && proxyEnabled:
		return "host_delegated"
	default:
		return "自定义组合"
	}
}

func modePriorityNote(source, mode string, proxyEnabled bool) string {
	switch source {
	case "config":
		if mode == string(iamservice.IAMModeLocal) && proxyEnabled {
			return "IAMMode=local 显式配置，覆盖 POWERX_PROXY"
		}
		if mode == string(iamservice.IAMModeDelegated) && !proxyEnabled {
			return "IAMMode=delegated 显式配置，覆盖 POWERX_PROXY"
		}
		return "IAMMode 显式配置优先级最高"
	case "env:POWERX_PROXY":
		return "POWERX_PROXY=1 生效（未设置 IAMMode）"
	default:
		return "未显式配置，使用默认 local"
	}
}
