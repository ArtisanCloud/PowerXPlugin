package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
	"github.com/powerx-plugin/framework/backend/go/manifest"
	"github.com/powerx-plugin/framework/backend/go/observability"
	"github.com/powerx-plugin/framework/backend/go/router"

	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/manifestx"
	"github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/routes"
)

func main() {
	app, err := setupApp()
	if err != nil {
		log.Fatalf("setup app: %v", err)
	}

	go func() {
		if err := app.Run(); err != nil {
			log.Fatalf("run server: %v", err)
		}
	}()

	waitForShutdown(app)
}

func setupApp() (*bootstrap.App, error) {
	app := bootstrap.NewAppFromEnv()

	if err := router.AttachHTTPServer(app); err != nil {
		return nil, err
	}

	if err := observability.InitMetrics(app); err != nil {
		log.Printf("init metrics: %v", err)
	}
	if err := observability.InitTracing(app); err != nil {
		log.Printf("init tracing: %v", err)
	}

	router.RegisterFrameworkRoutes(app)
	router.RegisterPluginRoutes(app, routes.Register)

	if err := manifest.Register(app, manifestx.Plugin()); err != nil {
		return nil, err
	}

	return app, nil
}

func waitForShutdown(app *bootstrap.App) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	if err := app.Shutdown(); err != nil && err != http.ErrServerClosed {
		log.Printf("shutdown: %v", err)
	}
}
