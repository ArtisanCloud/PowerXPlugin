package routes

import (
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"

	"github.com/powerx-plugins/com-powerx-demo/backend/internal/handler"
	"github.com/powerx-plugins/com-powerx-demo/backend/internal/service"
)

// Register 挂载插件业务路由。
func Register(r bootstrap.Router) {
	pingHandler := handler.NewPingHandler(service.NewPingService())
	r.Handle(http.MethodGet, "/ping", pingHandler.Handle())
}
