// The cl-dataplane binary runs an instance of a clink dataplane.
package main

import (
	"os"

	"github.com/clusterlink-org/clusterlink/cmd/cl-dataplane/app"
)

func main() {
	command := app.NewCLDataplaneCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
