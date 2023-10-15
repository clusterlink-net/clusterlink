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

package api

import "fmt"

const (
	// ListenPort is the port used by the dataplane to access the controlplane.
	ListenPort = 444

	// gRPCServerNamePrefix is the prefix such that <gRPCServerNamePrefix>.<peer name> is the gRPC server name.
	gRPCServerNamePrefix = "grpc"
)

// GRPCServerName returns the gRPC server name of a specific peer.
func GRPCServerName(peer string) string {
	return fmt.Sprintf("%s.%s", gRPCServerNamePrefix, peer)
}
