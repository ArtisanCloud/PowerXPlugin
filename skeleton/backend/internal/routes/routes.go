package routes

import (
	"net/http"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"

	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/handler"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/service"
)

// Register 将业务路由挂载到框架提供的路由器上。
func Register(r bootstrap.Router) {
	pingHandler := handler.NewPingHandler(service.NewPingService())
	r.Handle(http.MethodGet, "/ping", pingHandler.Handle())
}
