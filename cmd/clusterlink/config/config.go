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

package config

import (
	"path/filepath"
)

const (
	// PrivateKeyFileName is the filename used by private key files.
	PrivateKeyFileName = "key.pem"
	// CertificateFileName is the filename used by certificate files.
	CertificateFileName = "cert.pem"
	// DefaultFabric is the default fabric name.
	DefaultFabric = "default_fabric"
	// K8SYAMLFile is the filename of the kubernetes deployment yaml file.
	K8SYAMLFile = "k8s.yaml"
	// K8SClusterLinkInstanceYAMLFile is the filename of the ClusterLink instance CRD file that will use by the operator.
	K8SClusterLinkInstanceYAMLFile = "cl-instance.yaml"

	// ControlplaneDirectoryName is the directory name containing controlplane server configuration.
	ControlplaneDirectoryName = "controlplane"
	// DataplaneDirectoryName is the directory name containing dataplane server configuration.
	DataplaneDirectoryName = "dataplane"
	// CADirectoryName is the directory name containing site CA configuration.
	CADirectoryName = "ca"

	// GHCR is the path to the GitHub container registry.
	GHCR = "ghcr.io/clusterlink-net"
	// DefaultRegistry is the default container registry used.
	DefaultRegistry = GHCR
)

// FabricDirectory returns the base path of the fabric.
func FabricDirectory(name, path string) string {
	return filepath.Join(path, name)
}

// PeerDirectory returns the base path for a specific peer.
func PeerDirectory(peer, fabric, path string) string {
	return filepath.Join(FabricDirectory(fabric, path), peer)
}

// ControlplaneDirectory returns the path for a controlplane server.
func ControlplaneDirectory(peer, fabric, path string) string {
	return filepath.Join(PeerDirectory(peer, fabric, path), ControlplaneDirectoryName)
}

// DataplaneDirectory returns the path for a dataplane server.
func DataplaneDirectory(peer, fabric, path string) string {
	return filepath.Join(PeerDirectory(peer, fabric, path), DataplaneDirectoryName)
}

// CADirectory returns the path for a site CA.
func CADirectory(peer, fabric, path string) string {
	return filepath.Join(PeerDirectory(peer, fabric, path), CADirectoryName)
}

// FabricCertificate returns the fabric certificate name.
func FabricCertificate(name, path string) string {
	return filepath.Join(FabricDirectory(name, path), CertificateFileName)
}

// FabricKey returns the fabric key name.
func FabricKey(name, path string) string {
	return filepath.Join(FabricDirectory(name, path), PrivateKeyFileName)
}
