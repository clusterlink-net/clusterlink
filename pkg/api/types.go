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

// Endpoint represents a network endpoint (i.e., host or IP and a port).
type Endpoint struct {
	// Host or IP address of the endpoint.
	Host string
	// Port of the endpoint.
	Port uint16
}

// Peer represents a location (or site) that can be used to import services from.
type Peer struct {
	// Name that will be used to identify the Peer in subsequent API calls.
	Name string
	// Spec represents the attributes of a peer.
	Spec PeerSpec
	// Status field contains the peer status observed by the gateway.
	Status PeerStatus
}

// PeerSpec contains all the peer attributes
type PeerSpec struct {
	// Gateways serving the Peer.
	Gateways []Endpoint
}

// PeerStatus contains the peer status observed by the gateway.
type PeerStatus struct {
	// State contains the last peer state that observed by the gateway, indicating its reachability (Reachable/Unreachable).
	State string
	// LastSeen define the last time the state of the peer was updated.
	LastSeen string
}

// Export defines a service being exported by the local Peer for use by others.
// Only explicitly exported services can be accessed remotely.
type Export struct {
	// Name that will be used to identify the exported service in subsequent API calls.
	// Furthermore, this name will be used by remote peers to identify it as an import source.
	Name string
	// Spec represents the attributes of the export service.
	Spec ExportSpec
}

// ExportSpec contains all the export service attributes.
type ExportSpec struct {
	// Service endpoint being exported. The service is located inside the cluster.
	// In case of multi-port service, each export should
	// have a different `Name`. The endpoint Host name could be different from Name,
	// decoupling service names between Peers.
	Service Endpoint
	// ExternalService specifies a service located outside of the cluster.
	// If ExternalService is defined, a new local ExternalName Service will be created,
	// pointing to the ExternalService. Note that the port value is the same as defined
	// in the Service field.
	ExternalService string
}

// Import defines a service that is being imported to the local Peer from a remote Peer.
type Import struct {
	// Name of service imported, matches exported name by remote peers providing
	// the Service.
	Name string
	// Spec represents the attributes of the import service.
	Spec ImportSpec
	// Status field contains the import service status.
	Status ImportStatus
}

// ImportSpec contains all the import service attributes.
type ImportSpec struct {
	// Service endpoint for the import, as seen by clients in that site.
	Service Endpoint
}

// ImportStatus contains the import service status.
type ImportStatus struct {
	// Listener endpoint created for the imported service.
	Listener Endpoint
}

// Binding of an imported service to a remotely exposed service from a specific Peer.
type Binding struct {
	// Spec represents the attributes of the binding.
	Spec BindingSpec
}

// BindingSpec contains all the binding attributes.
type BindingSpec struct {
	// Import service name.
	Import string
	// Peer providing the imported service.
	Peer string
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
