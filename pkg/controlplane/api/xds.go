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

package api

const (
	// cluster names.

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

	// listener names.

	// EgressRouterListener is the listener name of the internal egress router.
	EgressRouterListener = "egress-router"
	// ImportListenerPrefix is the prefix of listeners representing imported services.
	ImportListenerPrefix = "import-"
	// IngressRouterListener is the listener name of the ingress router.
	IngressRouterListener = "ingress-router"

	// secret names.

	// ValidationSecret is the secret name of the dataplane certificate validation context
	// (which includes the CA certificate).
	ValidationSecret = "validation"
	// CertificateSecret is the secret name of the dataplane certificate.
	CertificateSecret = "certificate"
)

// ExportClusterName returns the cluster name of an exported service.
func ExportClusterName(name, namespace string) string {
	return ExportClusterPrefix + namespace + "/" + name
}

// RemotePeerClusterName returns the cluster name of a remote peer.
func RemotePeerClusterName(name string) string {
	return RemotePeerClusterPrefix + name
}

// ImportListenerName returns the listener name of an imported service.
func ImportListenerName(name, namespace string) string {
	return ImportListenerPrefix + namespace + "/" + name
}
