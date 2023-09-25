package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/clusterlink-org/clusterlink/cmd/dataplane/app"
)

func main() {
	command := app.NewDataplaneCommand()
	if err := command.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
