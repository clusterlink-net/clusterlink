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

// The API package defines the object model used by the control plane.
// Conceptually, we envisage a set of locations or sites, called Peers.
// Peers can share services between them, where one is configured to export
// a service and others are configured to import it. Import and Export objects
// refer to two facets:
// - a peer-local service endpoint (e.g., a DNS name or IP and a port); and
// - a service name used when communicating between Peers.
// Communication between Peers is established via one or more gateways, serving
// the Peers.

// Import defines a service that is being imported to the local Peer from a remote Peer.
type Import struct {
	// Name of service imported, matches exported name by remote peers providing
	// the Service.
	Name string
	// Spec represents the attributes of the import service.
	Spec ImportSpec
}

// ImportSpec contains all the import service attributes.
type ImportSpec struct {
	// Port of the imported service.
	Port uint16
	// TargetPort of the imported service.
	// This is the internal (non user-facing) listening port used by the dataplane pods.
	TargetPort uint16
	// Peers to import from.
	Peers []string
}

// Policy is an opaque object, to be processed by the Policy Engine.
type Policy struct {
	// Name identifying the Policy instance.
	Name string
	// Spec represents the attributes of the policy.
	Spec PolicySpec
}

// PolicySpec contains all the policy attributes.
type PolicySpec struct {
	// Blob of the policy (opaque bytes).
	Blob []byte
}
