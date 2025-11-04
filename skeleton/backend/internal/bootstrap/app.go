package bootstrap

import (
	"context"

	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/config"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/db"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/entity/models"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/logger"
	"gorm.io/gorm"
)

func BootstrapPlugin(ctx context.Context, cfg *config.Config) (*gorm.DB, error) {
	// 初始化日志
	logger.Init(cfg.LogLevel)
	logger.Info("Starting PowerX Note Plugin...")

	// 初始化 schema
	models.InitSchemaFrom(cfg.Database.Schema)

	// 连接数据库（在进程生命周期内保持打开；在优雅退出时关闭）
	queryDB, err := db.Connect(cfg.Database)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}

	return queryDB, nil
}
