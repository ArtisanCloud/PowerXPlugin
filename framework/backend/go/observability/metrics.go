package observability

import "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"

// InitMetrics 预留指标初始化入口。
func InitMetrics(app *bootstrap.App) error {
	// TODO: 集成 Prometheus/OpenTelemetry 指标
	_ = app
	return nil
}
