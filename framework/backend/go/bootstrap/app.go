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

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
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

	mu       sync.RWMutex
	manifest *manifest.Plugin
}

// Config 描述服务的基础运行配置。
type Config struct {
	Listen     string
	Env        string
	Standalone bool
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
	return &App{
		Ctx:    ctx,
		Config: cfg,
		Logger: slog.Default(),
	}
}

// NewAppFromEnv 读取环境变量并应用可选项构造 App。
func NewAppFromEnv(opts ...Option) *App {
	cfg := &Config{
		Listen:     getEnvOrDefault("POWERX_LISTEN", ":8078"),
		Env:        getEnvOrDefault("POWERX_ENV", "development"),
		Standalone: parseBoolEnv("STANDALONE", true),
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
