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

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
)

type peerHandler struct {
	cp *controlplane.Instance
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
	return h.cp.CreatePeer(object.(*store.Peer))
}

// Update a peer.
func (h *peerHandler) Update(object any) error {
	return h.cp.UpdatePeer(object.(*store.Peer))
}

// Get a peer.
func (h *peerHandler) Get(name string) (any, error) {
	peer := peerToAPI(h.cp.GetPeer(name))
	if peer == nil {
		return nil, nil
	}
	return peer, nil
}

// Delete a peer.
func (h *peerHandler) Delete(name any) (any, error) {
	return h.cp.DeletePeer(name.(string))
}

// List all peers.
func (h *peerHandler) List() (any, error) {
	peers := h.cp.GetAllPeers()
	apiPeers := make([]*api.Peer, len(peers))
	for i, peer := range peers {
		apiPeers[i] = peerToAPI(peer)
	}
	return apiPeers, nil
}
