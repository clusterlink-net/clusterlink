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

package platform

import (
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap"
)

// Config holds a configuration to instantiate a template.
type Config struct {
	// Peer is the peer name.
	Peer string
	// Namespace is the namespace the components deployed.
	Namespace string

	// CACertificate is the CA certificate.
	CACertificate *bootstrap.Certificate
	// ControlplaneCertificate is the controlplane certificate.
	ControlplaneCertificate *bootstrap.Certificate
	// DataplaneCertificate is the dataplane certificate.
	DataplaneCertificate *bootstrap.Certificate

	// FabricCertificate is the fabric CA certificate.
	FabricCertificate *bootstrap.Certificate
	// PeerCertificate is the peer certificate.
	PeerCertificate *bootstrap.Certificate

	// Controlplanes is the number of controlplane servers to run.
	Controlplanes uint16

	// Dataplanes is the number of dataplane servers to run.
	Dataplanes uint16
	// DataplaneType is the type of dataplane to create (envoy or go-based)
	DataplaneType string

	// PeerLabels are the peer attributes to be considered by access policies
	PeerLabels map[string]string
	// LogLevel is the log level.
	LogLevel string
	// ContainerRegistry is the container registry to pull the project images.
	ContainerRegistry string
	// Tag represents the tag of the project images.
	Tag string
	// IngressType is the type of ingress to create.
	IngressType string
	// IngressPort is the ingress port number to create.
	IngressPort uint16
	// IngressAnnotations is the annotations added to the ingress service.
	IngressAnnotations map[string]string
}

const (
	// DataplaneTypeEnvoy represents an envoy-type dataplane.
	DataplaneTypeEnvoy = "envoy"
	// DataplaneTypeGo represents a go-type dataplane.
	DataplaneTypeGo = "go"
)
