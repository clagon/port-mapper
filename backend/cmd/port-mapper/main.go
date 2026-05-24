package main

import (
	"log"
	"os/exec"

	"github.com/clagon/port-mapper/backend/internal/app"
)

type xdgBrowserOpener struct{}

func (xdgBrowserOpener) Open(url string) error {
	return exec.Command("xdg-open", url).Start()
}

func main() {
	a, err := app.New(app.AppOptions{
		OpenBrowser:   true,
		BrowserOpener: xdgBrowserOpener{},
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
