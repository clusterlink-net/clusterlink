// The cl-dataplane binary runs an instance of a clink dataplane.
package main

import (
	"os"

	"github.com/clusterlink-net/clusterlink/cmd/cl-go-dataplane/app"
)

func main() {
	command := app.NewCLGoDataplaneCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
