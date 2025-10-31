package main

import (
	"log"

	"github.com/powerx-plugin/framework/backend/go/bootstrap"
	"github.com/powerx-plugin/framework/backend/go/router"
)

func main() {
	app := bootstrap.NewAppFromEnv()
	if err := router.AttachHTTPServer(app); err != nil {
		log.Fatal(err)
	}
	log.Println("external plugin skeleton compiled")
}
