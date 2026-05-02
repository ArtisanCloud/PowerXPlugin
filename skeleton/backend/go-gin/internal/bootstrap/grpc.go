package bootstrap

import (
	"context"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/grpc/client"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"
)

func BootstrapGRPCClient(ctx context.Context, cfg *config.GRPCUpstream) *client.PowerXServiceClient {

	pxc, err := client.NewPowerXServiceClient(ctx, cfg)
	if err != nil {
		logger.FatalWith(nil, ctx, "Failed to initialize PowerX gRPC client", logger.Fields{
			"module":     "grpc",
			"biz_scene":  "grpc_client_bootstrap",
			"biz_domain": "integration",
			"component":  "bootstrap.grpc",
			"error":      err.Error(),
		})
	}

	logger.Info("PowerX gRPC client initialized successfully")

	return pxc
}
