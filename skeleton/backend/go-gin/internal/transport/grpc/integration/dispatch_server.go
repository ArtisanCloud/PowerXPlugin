package integration

import (
	"context"

	integrationService "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/services/integration"
	pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DispatchGRPCServer 预留的 gRPC Dispatch 服务。
type DispatchGRPCServer struct {
	service *integrationService.DispatchService
	logger  *pxlog.Entry
}

// NewDispatchGRPCServer 构造 gRPC 服务适配器。
func NewDispatchGRPCServer(service *integrationService.DispatchService, logger *pxlog.Entry) *DispatchGRPCServer {
	if logger == nil {
		logger = pxlog.WithField("component", "integration.grpc.dispatch")
	}
	return &DispatchGRPCServer{
		service: service,
		logger:  logger,
	}
}

// Dispatch 目前返回未实现，待 proto 与 gRPC 合约确立后接入 DispatchService。
func (s *DispatchGRPCServer) Dispatch(ctx context.Context, req interface{}) (interface{}, error) {
	s.logger.WithContext(ctx).Warn("gRPC dispatch not implemented yet")
	return nil, status.Error(codes.Unimplemented, "integration dispatch gRPC endpoint is not implemented")
}
