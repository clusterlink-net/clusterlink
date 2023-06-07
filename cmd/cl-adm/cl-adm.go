// The cl-adm binary is used for preparing a clusterlink deployment.
// The deployment includes certificate files for establishing secure TLS connections
// with other cluster components, and configuration for spawning up the various clusterlink
// components in different environments.
package main

import (
	"os"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/cmd"
)

func main() {
	command := cmd.NewCLADMCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
