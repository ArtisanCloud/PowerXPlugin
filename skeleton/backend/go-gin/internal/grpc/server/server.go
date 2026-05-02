package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"

	cfgpkg "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	marketplacerepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/marketplace"
	srvtemplates "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/admin/templates"
	integrationService "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/integration"
	marketplacesvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/marketplace"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	grpcTransport "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/grpc"
	integrationTransport "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/grpc/integration"
	marketplaceTransport "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/grpc/marketplace"
	templateTransport "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/grpc/template"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	defaultGRPCPort        = 9101
	defaultGRPCPortRetries = 10
)

// Server 插件 gRPC 服务器
type Server struct {
	*grpc.Server
	lis    net.Listener
	config *cfgpkg.GRPCServer
}

// NewGRPCServer 创建新的插件 gRPC 服务器
func NewGRPCServer(ctx context.Context, deps *app.Deps, c *cfgpkg.GRPCServer) (*Server, error) {
	logCtx := logger.WithLogFields(ctx, map[string]interface{}{
		"module":     "grpc",
		"biz_scene":  "grpc_server_bootstrap",
		"biz_domain": "integration",
		"component":  "grpc.server",
	})
	if c == nil {
		logger.InfoCtx(logCtx, "gRPC server config not provided; skipping")
		return nil, nil
	}
	if !c.Enable {
		logger.InfoCtx(logCtx, "gRPC server is disabled")
		return nil, nil
	}

	lis, err := acquireListener(c)
	if err != nil {
		return nil, err
	}

	var opts []grpc.ServerOption

	if c.UseTLS {
		if c.Cert == "" || c.Key == "" {
			return nil, fmt.Errorf("TLS is enabled but cert or key is missing")
		}
		creds, err := credentials.NewServerTLSFromFile(c.Cert, c.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
		logger.InfoCtx(logCtx, "gRPC server TLS enabled")
	} else {
		// 明确声明：开发期不加 TLS
		logger.WarnCtx(logCtx, "gRPC server running without TLS (development mode)")
		_ = insecure.NewCredentials()
	}

	s := grpc.NewServer(opts...)

	// 注册健康检查服务
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(s, healthServer)

	// 设置服务健康状态
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("template-plugin", healthpb.HealthCheckResponse_SERVING)

	// 注册反射服务（开发和调试用）
	reflection.Register(s)

	dispatchService := integrationService.BuildDispatchService(deps, logger.WithComponent("integration.dispatch_factory"))

	var marketplaceServer marketplaceTransport.LicenseServiceServer
	if deps != nil && deps.DB != nil {
		pricingRepo := marketplacerepo.NewPricingRepository(deps.DB)
		licenseRepo := marketplacerepo.NewLicenseRepository(deps.DB)
		licenseLogger := logger.WithComponent("marketplace.grpc.license")
		licenseService := marketplacesvc.NewLicenseService(
			deps.Config,
			pricingRepo,
			licenseRepo,
			deps.TaxProviderClient,
			deps.MarketplaceBilling,
			deps.LicenseAuthority,
			deps.LicenseCache,
			licenseLogger,
		)
		marketplaceServer = marketplaceTransport.NewLicenseServer(licenseService)
	}

	var templateServer templateTransport.TemplateServiceServer
	if deps != nil && deps.DB != nil {
		templateServer = templateTransport.NewServer(srvtemplates.NewTemplateService(deps.DB))
	}

	grpcTransport.Register(s, grpcTransport.Registrar{
		Integration: integrationTransport.NewServer(dispatchService, logger.WithComponent("integration.grpc")),
		Marketplace: marketplaceServer,
		Template:    templateServer,
	})

	logger.InfoCtx(logger.WithLogFields(logCtx, map[string]interface{}{
		"address": lis.Addr().String(),
	}), "gRPC server configured")

	return &Server{
		Server: s,
		lis:    lis,
		config: c,
	}, nil
}

func acquireListener(c *cfgpkg.GRPCServer) (net.Listener, error) {
	logCtx := logger.WithLogFields(context.Background(), map[string]interface{}{
		"module":     "grpc",
		"biz_scene":  "grpc_acquire_listener",
		"biz_domain": "integration",
		"component":  "grpc.server",
	})
	envSources := []string{"POWERX_GRPC_ADDR", "GRPC_ADDR"}
	for _, key := range envSources {
		if addr := strings.TrimSpace(os.Getenv(key)); addr != "" {
			lis, err := net.Listen("tcp", addr)
			if err != nil {
				return nil, fmt.Errorf("failed to listen on %s=%s: %w", key, addr, err)
			}
			c.Addr = addr
			if port := extractPort(addr); port > 0 {
				c.Port = port
			}
			logger.InfoCtx(logger.WithLogFields(logCtx, map[string]interface{}{
				"address": addr,
				"source":  key,
			}), "gRPC server address resolved from environment")
			return lis, nil
		}
	}

	host, basePort := deriveHostPort(c)
	attempts := c.PortMaxRetries
	if attempts <= 0 {
		attempts = defaultGRPCPortRetries
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		candidatePort := basePort + i
		addr := net.JoinHostPort(host, strconv.Itoa(candidatePort))
		lis, err := net.Listen("tcp", addr)
		if err == nil {
			c.Addr = addr
			c.Port = candidatePort
			if i > 0 {
				logger.InfoCtx(logger.WithLogFields(logCtx, map[string]interface{}{
					"address":  addr,
					"attempts": i + 1,
				}), "gRPC server port auto-incremented to avoid conflicts")
			}
			return lis, nil
		}
		lastErr = err
		logger.WarnCtx(logger.WithLogFields(logCtx, map[string]interface{}{
			"error":    err.Error(),
			"address":  addr,
			"attempt":  i + 1,
			"attempts": attempts,
		}), "failed to bind gRPC address")
	}

	return nil, fmt.Errorf("failed to bind gRPC port starting at %d after %d attempts: %w", basePort, attempts, lastErr)
}

func deriveHostPort(c *cfgpkg.GRPCServer) (string, int) {
	host := ""
	basePort := c.Port
	addr := strings.TrimSpace(c.Addr)
	if addr != "" {
		if strings.Contains(addr, ":") {
			if h, p, err := net.SplitHostPort(addr); err == nil {
				host = h
				if basePort == 0 {
					if portVal, err := strconv.Atoi(p); err == nil {
						basePort = portVal
					}
				}
			}
		}
	}
	if basePort <= 0 {
		basePort = defaultGRPCPort
	}
	return host, basePort
}

func extractPort(addr string) int {
	addr = strings.TrimSpace(addr)
	if addr == "" || !strings.Contains(addr, ":") {
		return 0
	}
	if _, p, err := net.SplitHostPort(addr); err == nil {
		if portVal, err := strconv.Atoi(p); err == nil {
			return portVal
		}
	}
	return 0
}

// Serve 启动 gRPC 服务器
func (s *Server) Serve(ctx context.Context) error {
	logCtx := logger.WithLogFields(ctx, map[string]interface{}{
		"module":     "grpc",
		"biz_scene":  "grpc_server_serve",
		"biz_domain": "integration",
		"component":  "grpc.server",
		"address":    s.lis.Addr().String(),
	})
	logger.InfoCtx(logCtx, "starting gRPC server")

	// 在单独的 goroutine 中监听上下文取消
	go func() {
		<-ctx.Done()
		logger.InfoCtx(logCtx, "shutting down gRPC server")
		s.GracefulStop()
	}()

	return s.Server.Serve(s.lis)
}

// GetListenAddr 获取监听地址
func (s *Server) GetListenAddr() string {
	if s.lis != nil {
		return s.lis.Addr().String()
	}
	return s.config.Addr
}

// IsServing 检查服务器是否在运行
func (s *Server) IsServing() bool {
	return s.lis != nil
}

// TODO: 当定义插件自己的 proto 服务时，在这里实现服务逻辑
// 例如：
//
// type TemplateServer struct {
// 	pluginv1.UnimplementedTemplatePluginServiceServer
// 	templateService *services.TemplateService
// 	// 其他依赖
// }
//
// func NewTemplateServer(deps *SomeDependencies) *TemplateServer {
// 	return &TemplateServer{
// 		templateService: deps.TemplateService,
// 	}
// }
//
// func (s *TemplateServer) CreateTemplate(ctx context.Context, req *pluginv1.CreateTemplateRequest) (*pluginv1.CreateTemplateResponse, error) {
// 	// 实现创建模板逻辑
// 	return &pluginv1.CreateTemplateResponse{}, nil
// }
//
// func (s *TemplateServer) GetTemplate(ctx context.Context, req *pluginv1.GetTemplateRequest) (*pluginv1.GetTemplateResponse, error) {
// 	// 实现获取模板逻辑
// 	return &pluginv1.GetTemplateResponse{}, nil
// }
