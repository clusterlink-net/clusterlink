// The cl-controlplane binary runs a gRPC server which configure the clink dataplane.
// In addition, it runs an HTTPS server for administrative management (API),
// authorization of remote peers and dataplane connections.
package main

import (
	"os"

	"github.ibm.com/mbg-agent/cmd/cl-controlplane/app"
)

func main() {
	command := app.NewCLControlplaneCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
