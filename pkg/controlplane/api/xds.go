package api

const (
	// cluster names

	// ControlplaneInternalHTTPCluster is the cluster name of the controlplane HTTP server for local dataplanes.
	ControlplaneInternalHTTPCluster = "controlplane-internal-http"
	// ControlplaneExternalHTTPCluster is the cluster name of the controlplane HTTP server for remote clients.
	ControlplaneExternalHTTPCluster = "controlplane-external-http"
	// ControlplaneGRPCCluster is the cluster name of the controlplane gRPC server.
	ControlplaneGRPCCluster = "controlplane-grpc"
	// EgressRouterCluster is the cluster name of the internal egress router.
	EgressRouterCluster = "egress-router"
	// ExportClusterPrefix is the prefix of clusters representing exported services.
	ExportClusterPrefix = "export-"
	// RemotePeerClusterPrefix is the prefix of clusters representing remote peers.
	RemotePeerClusterPrefix = "remote-peer-"

	// listener names

	// EgressRouterListener is the listener name of the internal egress router.
	EgressRouterListener = "egress-router"
	// ImportListenerPrefix is the prefix of listeners representing imported services.
	ImportListenerPrefix = "import-"
	// IngressRouterListener is the listener name of the ingress router.
	IngressRouterListener = "ingress-router"

	// secret names

	// ValidationSecret is the secret name of the dataplane certificate validation context
	// (which includes the CA certificate).
	ValidationSecret = "validation"
	// CertificateSecret is the secret name of the dataplane certificate.
	CertificateSecret = "certificate"
)

// ExportClusterName returns the cluster name of an exported service.
func ExportClusterName(name string) string {
	return ExportClusterPrefix + name
}

// RemotePeerClusterName returns the cluster name of a remote peer.
func RemotePeerClusterName(name string) string {
	return RemotePeerClusterPrefix + name
}

// ImportListenerName returns the listener name of an imported service.
func ImportListenerName(name string) string {
	return ImportListenerPrefix + name
}
