package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"

	cfgpkg "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	marketplacerepo "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/repository/marketplace"
	integrationService "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/integration"
	marketplacesvc "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/marketplace"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/shared/app"
	grpcTransport "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/grpc"
	integrationTransport "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/grpc/integration"
	marketplaceTransport "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/transport/grpc/marketplace"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server 插件 gRPC 服务器
type Server struct {
	*grpc.Server
	lis    net.Listener
	config *cfgpkg.GRPCServer
}

// NewGRPCServer 创建新的插件 gRPC 服务器
func NewGRPCServer(ctx context.Context, deps *app.Deps, c *cfgpkg.GRPCServer) (*Server, error) {
	if !c.Enable {
		logger.Info("gRPC server is disabled")
		return nil, nil
	}

	addr := os.Getenv("POWERX_GRPC_ADDR")
	if strings.TrimSpace(addr) == "" {
		addr = os.Getenv("GRPC_ADDR")
	}
	if strings.TrimSpace(addr) == "" {
		addr = ":9101" // 兜底
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", c.Addr, err)
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
		logger.Info("gRPC server TLS enabled")
	} else {
		// 明确声明：开发期不加 TLS
		logger.Warn("gRPC server running without TLS (development mode)")
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

	dispatchService := integrationService.BuildDispatchService(deps, logger.WithField("component", "integration.dispatch_factory"))

	var marketplaceServer marketplaceTransport.LicenseServiceServer
	if deps != nil && deps.DB != nil {
		pricingRepo := marketplacerepo.NewPricingRepository(deps.DB)
		licenseRepo := marketplacerepo.NewLicenseRepository(deps.DB)
		licenseLogger := logger.WithField("component", "marketplace.grpc.license")
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

	grpcTransport.Register(s, grpcTransport.Registrar{
		Integration: integrationTransport.NewServer(dispatchService, logger.WithField("component", "integration.grpc")),
		Marketplace: marketplaceServer,
	})

	//logger.WithField("address", lis.Addr().String()).Info("gRPC server configured")
	logger.Info("gRPC server configured")

	return &Server{
		Server: s,
		lis:    lis,
		config: c,
	}, nil
}

// Serve 启动 gRPC 服务器
func (s *Server) Serve(ctx context.Context) error {
	logger.WithField("address", s.lis.Addr().String()).Info("Starting gRPC server")

	// 在单独的 goroutine 中监听上下文取消
	go func() {
		<-ctx.Done()
		logger.Info("Shutting down gRPC server...")
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
