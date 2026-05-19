package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	iamadapters "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/adapters"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
	federatedProviders "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/providers"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/internal/integration/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
	runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"
)

// App 封装后端运行时依赖，供 skeleton 与框架层共享。
type App struct {
	Ctx    context.Context
	Config *Config
	Logger *slog.Logger
	DB     *sql.DB
	Router Router

	server   *http.Server
	shutdown func(context.Context) error
	closeFn  func() error

	gatewayClient    *gateway.Client
	federatedFactory contracts.ProviderFactory
	iamRegistry      *iamadapters.Registry

	mu       sync.RWMutex
	manifest *manifest.Plugin
}

// Config 描述服务的基础运行配置。
type Config struct {
	Listen     string
	Env        string
	Standalone bool
	Gateway    GatewayConfig
}

// GatewayConfig 描述 Integration Gateway 所需的凭证。
type GatewayConfig struct {
	BaseURL            string
	APIPrefix          string
	AuthScheme         string
	APIKey             string
	TenantID           string
	GRPCTarget         string
	Timeout            time.Duration
	UserAgent          string
	ContractVersion    string
	ContractDigestPath string
}

// Option 用于在构造 App 时覆盖默认配置。
type Option func(*Config)

// WithListen 设置监听地址。
func WithListen(addr string) Option {
	return func(cfg *Config) {
		if addr != "" {
			cfg.Listen = addr
		}
	}
}

// WithEnv 设置运行环境标识。
func WithEnv(env string) Option {
	return func(cfg *Config) {
		if env != "" {
			cfg.Env = env
		}
	}
}

// WithStandaloneDefaults 将配置切换为本地独立运行模式。
func WithStandaloneDefaults() Option {
	return func(cfg *Config) {
		cfg.Standalone = true
		if cfg.Listen == "" {
			cfg.Listen = ":8078"
		}
		if cfg.Env == "" {
			cfg.Env = "development"
		}
	}
}

// WithGatewayConfig 用于显式设置 Gateway 凭证。
func WithGatewayConfig(g GatewayConfig) Option {
	return func(cfg *Config) {
		cfg.Gateway = g
	}
}

// NewApp 根据显式配置构造 App。
func NewApp(cfg *Config) *App {
	if cfg == nil {
		cfg = &Config{
			Listen:     ":8078",
			Env:        "development",
			Standalone: true,
		}
	}
	ctx := context.Background()
	app := &App{
		Ctx:    ctx,
		Config: cfg,
		Logger: withRuntimeDefaults(slog.Default(), cfg),
	}
	app.initGatewayClient()
	app.initFederatedFactory()
	app.initIAMRegistry()
	return app
}

// NewAppFromEnv 读取环境变量并应用可选项构造 App。
func NewAppFromEnv(opts ...Option) *App {
	cfg := &Config{
		Listen:     getEnvOrDefault("POWERX_LISTEN", ":8078"),
		Env:        getEnvOrDefault("POWERX_ENV", "development"),
		Standalone: parseBoolEnv("STANDALONE", true),
	}
	cfg.Gateway = GatewayConfig{
		BaseURL:         getEnvOrDefault("PX_GATEWAY_BASE_URL", ""),
		APIPrefix:       getEnvOrDefault("PX_GATEWAY_API_PREFIX", "/api/v1"),
		AuthScheme:      strings.TrimSpace(os.Getenv("PX_GATEWAY_AUTH_SCHEME")),
		APIKey:          firstNonEmpty(strings.TrimSpace(os.Getenv("PX_GATEWAY_API_KEY")), strings.TrimSpace(os.Getenv("PX_PLUGIN_API_KEY"))),
		GRPCTarget:      strings.TrimSpace(os.Getenv("PX_GATEWAY_GRPC_TARGET")),
		ContractVersion: strings.TrimSpace(os.Getenv("PX_GATEWAY_CONTRACT_VERSION")),
	}
	cfg.Gateway.AuthScheme = normalizeGatewayAuthScheme(cfg.Gateway.AuthScheme, cfg.Gateway.APIKey)
	if timeoutStr := strings.TrimSpace(os.Getenv("PX_GATEWAY_TIMEOUT")); timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.Gateway.Timeout = d
		}
	}
	if ua := strings.TrimSpace(os.Getenv("PX_GATEWAY_USER_AGENT")); ua != "" {
		cfg.Gateway.UserAgent = ua
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return NewApp(cfg)
}

// Run 启动 HTTP 服务。
func (a *App) Run() error {
	if a.server == nil {
		return errors.New("bootstrap: http server not attached")
	}
	if a.Logger != nil {
		a.Logger.Info("starting HTTP server",
			slog.String("listen", a.server.Addr),
			slog.String("env", a.Config.Env),
			slog.Bool("standalone", a.Config.Standalone),
		)
	}
	err := a.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown 优雅关闭已附加的服务。
func (a *App) Shutdown() error {
	defer a.closeGateway()
	if a.shutdown == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(a.Ctx, 10*time.Second)
	defer cancel()
	return a.shutdown(ctx)
}

// AttachServer 由 router 包调用，注入底层 HTTP 服务器。
func (a *App) AttachServer(server *http.Server, shutdown func(context.Context) error) {
	a.server = server
	a.shutdown = shutdown
}

// AttachRouter 允许 router 包设置抽象路由器实例。
func (a *App) AttachRouter(r Router) {
	a.Router = r
}

// RegisterManifest 存储插件清单，可供宿主/工具链读取。
func (a *App) RegisterManifest(p manifest.Plugin) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.manifest = &p
}

// Manifest 读取当前注册的 Manifest。
func (a *App) Manifest() *manifest.Plugin {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.manifest == nil {
		return nil
	}
	cp := *a.manifest
	return &cp
}

// GatewayClient 返回已注入的 Gateway Client（可能为空）。
func (a *App) GatewayClient() *gateway.Client {
	return a.gatewayClient
}

// FederatedProviderFactory 返回联邦登录 provider factory。
func (a *App) FederatedProviderFactory() contracts.ProviderFactory {
	return a.federatedFactory
}

// IAMRegistry 返回 framework IAM 注册中心。
func (a *App) IAMRegistry() *iamadapters.Registry {
	return a.iamRegistry
}

func (a *App) initGatewayClient() {
	if a.Config == nil || !a.Config.Gateway.enabled() {
		return
	}
	authScheme := normalizeGatewayAuthScheme(
		a.Config.Gateway.AuthScheme,
		a.Config.Gateway.APIKey,
	)
	tenantID := strings.TrimSpace(a.Config.Gateway.TenantID)
	gcfg := gateway.Config{
		BaseURL:    a.Config.Gateway.BaseURL,
		APIPrefix:  a.Config.Gateway.APIPrefix,
		AuthScheme: authScheme,
		APIKey:     a.Config.Gateway.APIKey,
		TenantUUID: tenantID,
	}
	if a.Config.Gateway.Timeout > 0 {
		gcfg.RequestTimeout = a.Config.Gateway.Timeout
	}
	if ua := strings.TrimSpace(a.Config.Gateway.UserAgent); ua != "" {
		gcfg.UserAgent = ua
	}
	if target := strings.TrimSpace(a.Config.Gateway.GRPCTarget); target != "" {
		gcfg.GRPCTarget = target
	}
	if cv := strings.TrimSpace(a.Config.Gateway.ContractVersion); cv != "" {
		gcfg.ContractVersion = cv
	}
	if digestPath := strings.TrimSpace(a.Config.Gateway.ContractDigestPath); digestPath != "" {
		gcfg.ContractDigestPath = digestPath
	}
	client, err := gateway.NewClient(gcfg)
	if err != nil {
		if a.Logger != nil {
			a.Logger.Warn("failed to initialize gateway client", slog.String("error", err.Error()))
		}
		return
	}
	a.gatewayClient = client
}

func (a *App) closeGateway() {
	if a.gatewayClient == nil {
		return
	}
	if err := a.gatewayClient.Close(); err != nil && a.Logger != nil {
		a.Logger.Warn("failed to close gateway client", slog.String("error", err.Error()))
	}
	a.gatewayClient = nil
}

func (a *App) initFederatedFactory() {
	a.federatedFactory = federatedProviders.NewRegistry()
}

func (a *App) initIAMRegistry() {
	if a == nil {
		return
	}
	a.iamRegistry = iamadapters.NewRegistry()
}

func (cfg GatewayConfig) enabled() bool {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return false
	}
	scheme := normalizeGatewayAuthScheme(cfg.AuthScheme, cfg.APIKey)
	if scheme == "apikey" {
		return strings.TrimSpace(cfg.APIKey) != ""
	}
	return false
}

func getEnvOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func parseBoolEnv(key string, fallback bool) bool {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeGatewayAuthScheme(raw, apiKey string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "apikey", "api_key", "api-key":
		return "apikey"
	case "bearer":
		return "bearer"
	}
	if strings.TrimSpace(apiKey) != "" {
		return "apikey"
	}
	return "bearer"
}

func withRuntimeDefaults(base *slog.Logger, cfg *Config) *slog.Logger {
	if base == nil {
		base = slog.Default()
	}
	tenantUUID := runtimelogging.FallbackUnknown
	if cfg != nil {
		if tid := strings.TrimSpace(cfg.Gateway.TenantID); tid != "" {
			tenantUUID = tid
		}
	}

	return base.With(
		slog.String(runtimelogging.FieldTenantUUID, tenantUUID),
		slog.String(runtimelogging.FieldTenantKey, runtimelogging.TenantKeyFromUUID(tenantUUID)),
		slog.String(runtimelogging.FieldSubscriber, "bootstrap.app"),
		slog.String(runtimelogging.FieldComponent, "bootstrap.app"),
	)
}
