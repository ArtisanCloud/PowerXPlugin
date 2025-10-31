package observability

import "github.com/powerx-plugin/framework/backend/go/bootstrap"

// InitTracing 预留链路追踪初始化入口。
func InitTracing(app *bootstrap.App) error {
	// TODO: 接入 OpenTelemetry / Jaeger
	_ = app
	return nil
}
