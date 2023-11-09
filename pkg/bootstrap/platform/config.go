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

package platform

import (
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap"
)

// Config holds a configuration to instantiate a template.
type Config struct {
	// Peer is the peer name.
	Peer string

	// FabricCertificate is the fabric certificate.
	FabricCertificate *bootstrap.Certificate
	// PeerCertificate is the peer certificate.
	PeerCertificate *bootstrap.Certificate
	// ControlplaneCertificate is the controlplane certificate.
	ControlplaneCertificate *bootstrap.Certificate
	// DataplaneCertificate is the dataplane certificate.
	DataplaneCertificate *bootstrap.Certificate
	// GWCTLCertificate is the gwctl certificate.
	GWCTLCertificate *bootstrap.Certificate

	// Dataplanes is the number of dataplane servers to run.
	Dataplanes uint16
	// DataplaneType is the type of dataplane to create (envoy or go-based)
	DataplaneType string

	// LogLevel is the log level.
	LogLevel string
	// ContainerRegistry is the container registry to pull the project images.
	ContainerRegistry string
}

const (
	// DataplaneTypeEnvoy represents an envoy-type dataplane.
	DataplaneTypeEnvoy = "envoy"
	// DataplaneTypeGo represents a go-type dataplane.
	DataplaneTypeGo = "go"
)
