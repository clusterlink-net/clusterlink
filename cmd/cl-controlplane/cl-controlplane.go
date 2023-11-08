// Copyright 2023 The ClusterLink Authors.
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

// The cl-controlplane binary runs a gRPC server which configure the clink dataplane.
// In addition, it runs an HTTPS server for administrative management (API),
// authorization of remote peers and dataplane connections.
package main

import (
	"os"

	"github.com/clusterlink-net/clusterlink/cmd/cl-controlplane/app"
	"github.com/clusterlink-net/clusterlink/pkg/versioninfo"
)

func main() {
	command := app.NewCLControlplaneCommand()
	command.Version = versioninfo.Short()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
