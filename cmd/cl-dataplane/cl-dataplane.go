// The cl-dataplane binary runs an instance of a clink dataplane.
package main

import (
	"fmt"
	"os"

	"github.ibm.com/mbg-agent/cmd/cl-dataplane/app"
)

func main() {
	command := app.NewCLDataplaneCommand()
	if err := command.Execute(); err != nil {
		fmt.Printf("Error: %v.\n", err)
		os.Exit(1)
	}
}
