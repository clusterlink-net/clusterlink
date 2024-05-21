// Copyright (c) The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The clusterlink binary is used for preparing a clusterlink deployment.
// The deployment includes certificate files for establishing secure TLS connections
// with other cluster components, and configuration for spawning up the various clusterlink
// components in different environments.
package main

import (
	"os"

	"github.com/clusterlink-net/clusterlink/cmd/clusterlink/cmd"
	"github.com/clusterlink-net/clusterlink/pkg/versioninfo"
)

func main() {
	command := cmd.NewCLADMCommand()
	command.Version = versioninfo.Short()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
