package main

import (
	"log"

	"github.com/clagon/port-mapper/backend/internal/app"
)

func main() {
	a, err := app.New(app.AppOptions{
		OpenBrowser: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("port-mapper config: %s", a.ConfigPath())
	log.Printf("port-mapper listening on %s", a.Addr())
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
