// The cl-dataplane binary runs an instance of a clink dataplane.
package main

import (
	"os"

	"github.ibm.com/mbg-agent/cmd/cl-dataplane/app"
)

func main() {
	command := app.NewCLDataplaneCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
