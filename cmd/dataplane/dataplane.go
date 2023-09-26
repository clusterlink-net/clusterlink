package main

import (
	"os"

	"github.com/clusterlink-org/clusterlink/cmd/dataplane/app"
)

func main() {
	command := app.NewDataplaneCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
