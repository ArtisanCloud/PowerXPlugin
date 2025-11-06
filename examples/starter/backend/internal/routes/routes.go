package routes

import (
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"

	"github.com/ArtisanCloud/PowerXPlugin/examples/com-powerx-starter/backend/internal/handler"
	"github.com/ArtisanCloud/PowerXPlugin/examples/com-powerx-starter/backend/internal/service"
)

// Register 挂载插件业务路由。
func Register(r bootstrap.Router) {
	pingHandler := handler.NewPingHandler(service.NewPingService())
	r.Handle(http.MethodGet, "/ping", pingHandler.Handle())
}
