package routes

import (
	"net/http"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
	frameworkmw "github.com/powerx-plugin/framework/backend/go/middleware"

	"github.com/powerx-plugins/com-powerx-demo/backend/internal/handler"
	"github.com/powerx-plugins/com-powerx-demo/backend/internal/service"
	"github.com/powerx-plugins/com-powerx-demo/backend/internal/templates"
)

// Register 挂载插件业务路由。
func Register(r bootstrap.Router) {
	r.Use(frameworkmw.RequestID(), frameworkmw.TenantContext())

	pingHandler := handler.NewPingHandler(service.NewPingService())
	r.Handle(http.MethodGet, "/ping", pingHandler.Handle())

	tRepo := templates.NewTemplateRepository()
	tService := templates.NewService(tRepo)
	tHandler := templates.NewHandler(tService)

	r.Handle(http.MethodGet, "/templates", tHandler.List())
	r.Handle(http.MethodGet, "/templates/:id", tHandler.Get())
	r.Handle(http.MethodPost, "/templates", tHandler.Create())
	r.Handle(http.MethodPut, "/templates/:id", tHandler.Update())
	r.Handle(http.MethodDelete, "/templates/:id", tHandler.Delete())
}
