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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Peer represents a location (or site) that can be used to import services from.
type Peer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the peer attributes.
	Spec PeerSpec `json:"spec"`
	// Status represents the peer status.
	Status PeerStatus `json:"status,omitempty"`
}

// Endpoint represents a network endpoint (i.e., host or IP and a port).
type Endpoint struct {
	// Host or IP address of the endpoint.
	Host string `json:"host"`
	// Port of the endpoint.
	Port uint16 `json:"port"`
}

// PeerSpec contains all peer attributes.
type PeerSpec struct {
	// Gateways serving the Peer.
	Gateways []Endpoint `json:"gateways"`
}

const (
	// PeerReachable is a condition type for indicating whether a peer is reachable (heartbeat responding).
	PeerReachable string = "PeerReachable"
)

// PeerStatus represents the status of a peer.
type PeerStatus struct {
	// Conditions of the peer.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Labels holds peer labels, as reported by the remote peer
	Labels map[string]string `json:"labels,omitempty"`
}

// +kubebuilder:object:root=true

// PeerList is a list of peer objects.
type PeerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of peer objects.
	Items []Peer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Peer{}, &PeerList{})
}
