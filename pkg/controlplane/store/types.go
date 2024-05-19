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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
)

const (
	peerStoreName         = "peer"
	exportStoreName       = "export"
	importStoreName       = "import"
	accessPolicyStoreName = "accessPolicy"

	exportStructVersion       = 1
	importStructVersion       = 1
	peerStructVersion         = 1
	accessPolicyStructVersion = 1
)

// Peer represents a remote peer.
type Peer struct {
	v1alpha1.PeerSpec
	// Name of the peer.
	Name string
	// Status of the peer.
	Status v1alpha1.PeerStatus
	// Version of the struct when object was created.
	Version uint32
}

// NewPeer creates a new peer.
func NewPeer(peer *v1alpha1.Peer) *Peer {
	return &Peer{
		PeerSpec: peer.Spec,
		Name:     peer.Name,
		Status:   peer.Status,
		Version:  peerStructVersion,
	}
}

// Export represents a local service that may be exported.
type Export struct {
	v1alpha1.ExportSpec
	// Name of the export.
	Name string
	// Status of the export.
	Status v1alpha1.ExportStatus
	// Version of the struct when object was created.
	Version uint32
}

// NewExport creates a new export.
func NewExport(export *v1alpha1.Export) *Export {
	return &Export{
		ExportSpec: export.Spec,
		Name:       export.Name,
		Status:     export.Status,
		Version:    exportStructVersion,
	}
}

// Import represents an external service that can be bound to (multiple) exported services of remote peers.
type Import struct {
	v1alpha1.ImportSpec
	// Name of import.
	Name string
	// Labels defined for the import
	Labels map[string]string
	// Version of the struct when object was created.
	Version uint32
}

// NewImport creates a new import.
func NewImport(imp *v1alpha1.Import) *Import {
	return &Import{
		ImportSpec: imp.Spec,
		Name:       imp.Name,
		Labels:     imp.Labels,
		Version:    importStructVersion,
	}
}

// AccessPolicy to allow/deny specific connections.
type AccessPolicy struct {
	v1alpha1.AccessPolicySpec
	// Name of access policy.
	Name string
	// Version of the struct when object was created.
	Version uint32
}

// NewAccessPolicy creates a new access policy.
func NewAccessPolicy(policy *v1alpha1.AccessPolicy) *AccessPolicy {
	return &AccessPolicy{
		AccessPolicySpec: policy.Spec,
		Name:             policy.Name,
		Version:          accessPolicyStructVersion,
	}
}
