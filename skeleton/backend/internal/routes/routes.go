package routes

import (
	"net/http"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
	frameworkmw "github.com/powerx-plugin/framework/backend/go/middleware"

	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/handler"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/service"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/templates"
)

// Register 将业务路由挂载到框架提供的路由器上。
func Register(r bootstrap.Router) {
	r.Use(frameworkmw.CORS(), frameworkmw.RequestID(), frameworkmw.TenantContext())

	pingHandler := handler.NewPingHandler(service.NewPingService())
	r.Handle(http.MethodGet, "/ping", pingHandler.Handle())

	tRepo := templates.NewTemplateRepository()
	tService := templates.NewService(tRepo)
	th := templates.NewHandler(tService)

	r.Handle(http.MethodGet, "/templates", th.List())
	r.Handle(http.MethodGet, "/templates/:id", th.Get())
	r.Handle(http.MethodPost, "/templates", th.Create())
	r.Handle(http.MethodPut, "/templates/:id", th.Update())
	r.Handle(http.MethodDelete, "/templates/:id", th.Delete())
}
