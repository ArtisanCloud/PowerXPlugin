package publish

import "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"

// RegisterRoutes wires publish handler into router.
func RegisterRoutes(router bootstrap.Router) {
	handler := NewHandler(nil)
	group := router.Group("/internal/publish")
	group.Handle("POST", "/create", handler.Create)
	group.Handle("POST", "/deploy", handler.Deploy)
}
