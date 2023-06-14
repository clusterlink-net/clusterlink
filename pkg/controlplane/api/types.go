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

// Peer represents a location (or site) that can be used to import services from.
type Peer struct {
	// Name that will be used to identify the Peer in subsequent API calls.
	Name string
	// Gateways serving the Peer.
	Gateways []Endpoint
}

// Endpoint represents a network endpoint (i.e., host or IP and a port).
type Endpoint struct {
	// Host or IP address of the endpoint.
	Host string
	// Port of the endpoint.
	Port uint32
}

// Export defines a service being exported by the local Peer for use by others.
// Only explicitly exported services can be accessed remotely.
type Export struct {
	// Name that will be used to identify the exported service in subsequent admin API calls.
	// Furthermore, this name will be used by remote peers to identify it as an import source.
	Name string
	// Service endpoint being exported. In case of multi-port service, each export should
	// have a different `Name`. The endpoint Host name could be different from Name,
	// decoupling service names between Peers.
	Service Endpoint
}

// Import defines a service that is being imported to the local Peer from a remote Peer.
type Import struct {
	// Name of service imported, matches exported name by remote peers providing
	// the Service.
	Name string
	// Service endpoint for the import, as seen by clients in that site.
	Service Endpoint
}

// Binding of an imported service to a remotely exposed service from a specific Peer.
type Binding struct {
	// Import name.
	Import string
	// Peer providing the imported service.
	Peer string
}

// Policy is an opaque object, to be processed by the Policy Engine.
type Policy struct {
	// Name identifying the Policy instance.
	Name string
	// Spec of the policy (opaque bytes).
	Spec []byte
}
