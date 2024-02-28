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

package rest

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
)

type peerHandler struct {
	manager *Manager
}

func toK8SPeer(peer *store.Peer) *v1alpha1.Peer {
	k8sPeer := &v1alpha1.Peer{
		ObjectMeta: metav1.ObjectMeta{Name: peer.Name},
		Spec: v1alpha1.PeerSpec{
			Gateways: make([]v1alpha1.Endpoint, len(peer.Gateways)),
		},
	}

	for i, gw := range peer.PeerSpec.Gateways {
		k8sPeer.Spec.Gateways[i].Host = gw.Host
		k8sPeer.Spec.Gateways[i].Port = gw.Port
	}

	return k8sPeer
}

func peerToAPI(peer *store.Peer) *api.Peer {
	if peer == nil {
		return nil
	}

	return &api.Peer{
		Name:   peer.Name,
		Spec:   peer.PeerSpec,
		Status: api.PeerStatus{},
	}
}

// CreatePeer defines a new route target for egress dataplane connections.
func (m *Manager) CreatePeer(peer *store.Peer) error {
	m.logger.Infof("Creating peer '%s'.", peer.Name)

	if m.initialized {
		if err := m.peers.Create(peer); err != nil {
			return err
		}
	}

	k8sPeer := toK8SPeer(peer)
	m.authzManager.AddPeer(k8sPeer)
	return m.xdsManager.AddPeer(k8sPeer)
}

// UpdatePeer updates new route target for egress dataplane connections.
func (m *Manager) UpdatePeer(peer *store.Peer) error {
	m.logger.Infof("Updating peer '%s'.", peer.Name)

	err := m.peers.Update(peer.Name, func(old *store.Peer) *store.Peer {
		return peer
	})
	if err != nil {
		return err
	}

	k8sPeer := toK8SPeer(peer)
	m.authzManager.AddPeer(k8sPeer)
	return m.xdsManager.AddPeer(k8sPeer)
}

// GetPeer returns an existing peer.
func (m *Manager) GetPeer(name string) *store.Peer {
	m.logger.Infof("Getting peer '%s'.", name)
	return m.peers.Get(name)
}

// DeletePeer removes the possibility for egress dataplane connections to be routed to a given peer.
func (m *Manager) DeletePeer(name string) (*store.Peer, error) {
	m.logger.Infof("Deleting peer '%s'.", name)

	pr, err := m.peers.Delete(name)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, nil
	}

	m.authzManager.DeletePeer(name)

	err = m.xdsManager.DeletePeer(name)
	if err != nil {
		// practically impossible
		return nil, err
	}

	return pr, nil
}

// GetAllPeers returns the list of all peers.
func (m *Manager) GetAllPeers() []*store.Peer {
	m.logger.Info("Listing all peers.")
	return m.peers.GetAll()
}

// Decode a peer.
func (h *peerHandler) Decode(data []byte) (any, error) {
	var peer api.Peer
	if err := json.Unmarshal(data, &peer); err != nil {
		return nil, fmt.Errorf("cannot decode peer: %w", err)
	}

	if peer.Name == "" {
		return nil, fmt.Errorf("empty peer name")
	}

	for i, ep := range peer.Spec.Gateways {
		if ep.Host == "" {
			return nil, fmt.Errorf("gateway #%d missing host", i)
		}
		if ep.Port == 0 {
			return nil, fmt.Errorf("gateway #%d (host '%s') missing port", i, ep.Host)
		}
	}

	return store.NewPeer(&peer), nil
}

// Create a peer.
func (h *peerHandler) Create(object any) error {
	return h.manager.CreatePeer(object.(*store.Peer))
}

// Update a peer.
func (h *peerHandler) Update(object any) error {
	return h.manager.UpdatePeer(object.(*store.Peer))
}

// Get a peer.
func (h *peerHandler) Get(name string) (any, error) {
	peer := peerToAPI(h.manager.GetPeer(name))
	if peer == nil {
		return nil, nil
	}
	return peer, nil
}

// Delete a peer.
func (h *peerHandler) Delete(name any) (any, error) {
	return h.manager.DeletePeer(name.(string))
}

// List all peers.
func (h *peerHandler) List() (any, error) {
	peers := h.manager.GetAllPeers()
	apiPeers := make([]*api.Peer, len(peers))
	for i, peer := range peers {
		apiPeers[i] = peerToAPI(peer)
	}
	return apiPeers, nil
}
