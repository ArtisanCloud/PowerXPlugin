package main

import (
	"log"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/router"
)

func main() {
	app := bootstrap.NewAppFromEnv()
	if err := router.AttachHTTPServer(app); err != nil {
		log.Fatal(err)
	}
	log.Println("external plugin skeleton compiled")
}
