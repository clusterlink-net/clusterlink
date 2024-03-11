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

package store

import (
	"github.com/clusterlink-net/clusterlink/pkg/api"
)

const (
	peerStoreName         = "peer"
	exportStoreName       = "export"
	importStoreName       = "import"
	accessPolicyStoreName = "accessPolicy"
	lbPolicyStoreName     = "lbPolicy"

	exportStructVersion       = 1
	importStructVersion       = 1
	peerStructVersion         = 1
	accessPolicyStructVersion = 1
	lbPolicyStructVersion     = 1
)

// Peer represents a remote peer.
type Peer struct {
	api.PeerSpec
	// Name of the peer.
	Name string
	// Version of the struct when object was created.
	Version uint32
}

// NewPeer creates a new peer.
func NewPeer(peer *api.Peer) *Peer {
	return &Peer{
		PeerSpec: peer.Spec,
		Name:     peer.Name,
		Version:  peerStructVersion,
	}
}

// Export represents a local service that may be exported.
type Export struct {
	api.ExportSpec
	// Name of the export.
	Name string
	// Version of the struct when object was created.
	Version uint32
}

// NewExport creates a new export.
func NewExport(export *api.Export) *Export {
	return &Export{
		ExportSpec: export.Spec,
		Name:       export.Name,
		Version:    exportStructVersion,
	}
}

// Import represents an external service that can be bound to (multiple) exported services of remote peers.
type Import struct {
	api.ImportSpec
	// Name of import.
	Name string
	// Version of the struct when object was created.
	Version uint32
}

// NewImport creates a new import.
func NewImport(imp *api.Import) *Import {
	return &Import{
		ImportSpec: imp.Spec,
		Name:       imp.Name,
		Version:    importStructVersion,
	}
}

// AccessPolicy to allow/deny specific connections.
type AccessPolicy struct {
	api.Policy
	// Version of the struct when object was created.
	Version uint32
}

// NewAccessPolicy creates a new access policy.
func NewAccessPolicy(policy *api.Policy) *AccessPolicy {
	return &AccessPolicy{
		Policy:  *policy,
		Version: accessPolicyStructVersion,
	}
}

// LBPolicy specifies the load-balancing scheme for specific connections.
type LBPolicy struct {
	api.Policy
	// Version of the struct when object was created.
	Version uint32
}

// NewLBPolicy creates a new load-balancing policy.
func NewLBPolicy(policy *api.Policy) *LBPolicy {
	return &LBPolicy{
		Policy:  *policy,
		Version: accessPolicyStructVersion,
	}
}
