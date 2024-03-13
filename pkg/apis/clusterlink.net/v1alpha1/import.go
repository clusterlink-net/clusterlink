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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Import defines a service that is being imported to the local Peer from a remote Peer.
type Import struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the attributes of the imported service.
	Spec ImportSpec `json:"spec"`
}

// ImportSource represents an addressable exported service.
type ImportSource struct {
	// Peer name where the exported service is defined.
	Peer string `json:"peer"`
	// ExportName is the name of the exported service.
	ExportName string `json:"name"`
	// ExportNamespace is the namespace of the exported service.
	ExportNamespace string `json:"namespace"`
}

// ImportSpec contains all attributes of an imported service.
type ImportSpec struct {
	// Port of the imported service.
	Port uint16 `json:"port"`
	// TargetPort of the imported service.
	// This is the internal (non user-facing) listening port used by the dataplane pods.
	TargetPort uint16 `json:"targetPort,omitempty"`
	// Sources to import from.
	Sources []ImportSource `json:"sources"`
	// LBScheme is the load-balancing scheme to use (e.g., Random, Static, ECMP)
	LBScheme string
	// TODO: Make LBScheme a proper type (when backwards compatibility is no longer needed)
}

// ImportStatus represents the status of an imported service.
type ImportStatus struct {
	// TODO: add fields
}

// +kubebuilder:object:root=true

// ImportList is a list of import objects.
type ImportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of import objects.
	Items []Import `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Import{}, &ImportList{})
}
