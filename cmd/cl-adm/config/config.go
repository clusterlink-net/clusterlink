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

package config

import (
	"path/filepath"
)

const (
	// PrivateKeyFileName is the filename used by private key files.
	PrivateKeyFileName = "key.pem"
	// CertificateFileName is the filename used by certificate files.
	CertificateFileName = "cert.pem"
	// DockerRunFile is the filename of the docker-run script.
	DockerRunFile = "docker-run.sh"
	// GWCTLInitFile is the filename of the gwctl-init script.
	GWCTLInitFile = "gwctl-init.sh"
	// K8SYamlFile is the filename of the kubernetes deployment yaml file.
	K8SYamlFile = "k8s.yaml"
	// PersistencyDirectoryName is the directory name containing container persisted files.
	PersistencyDirectoryName = "persist"

	// ControlplaneDirectoryName is the directory name containing controlplane server configuration.
	ControlplaneDirectoryName = "controlplane"
	// DataplaneDirectoryName is the directory name containing dataplane server configuration.
	DataplaneDirectoryName = "dataplane"
	// GWCTLDirectoryName is the directory name containing gwctl certificates.
	GWCTLDirectoryName = "gwctl"
)

// FabricDirectory returns the base path of the fabric.
func FabricDirectory() string {
	return "."
}

// PeerDirectory returns the base path for a specific peer.
func PeerDirectory(peer string) string {
	return filepath.Join(FabricDirectory(), peer)
}

// ControlplaneDirectory returns the path for a controlplane server.
func ControlplaneDirectory(peer string) string {
	return filepath.Join(PeerDirectory(peer), ControlplaneDirectoryName)
}

// DataplaneDirectory returns the path for a dataplane server.
func DataplaneDirectory(peer string) string {
	return filepath.Join(PeerDirectory(peer), DataplaneDirectoryName)
}

// GWCTLDirectory returns the path for a gwctl instance.
func GWCTLDirectory(peer string) string {
	return filepath.Join(PeerDirectory(peer), GWCTLDirectoryName)
}
