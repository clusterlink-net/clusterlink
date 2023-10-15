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

package controlplane

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/eventmanager"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/health"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
)

var plog = logrus.WithField("component", "mbgControlPlane/Peer")

// AddPeerHandler - Add peer HTTP handler
func AddPeerHandler(w http.ResponseWriter, r *http.Request) {

	// Parse add peer struct from request
	var p api.Peer
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// AddPeer control plane logic
	addPeer(p)

	// Response
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte("Add Peer succeeded"))
	if err != nil {
		plog.Println(err)
	}
}

// AddPeer control plane logic
func addPeer(p api.Peer) {
	// Update MBG state
	store.UpdateState()

	peerResp, err := store.GetEventManager().RaiseAddPeerEvent(eventmanager.AddPeerAttr{PeerMbg: p.Name})
	if err != nil {
		plog.Errorf("Unable to raise connection request event")
		return
	}
	if peerResp.Action == eventmanager.Deny {
		plog.Infof("Denying add peer(%s) due to policy", p.Name)
		return
	}
	plog.Infof("Peer %v Port: %v", p, strconv.Itoa(int(p.Spec.Gateways[0].Port)))
	store.AddMbgNbr(p.Name, p.Spec.Gateways[0].Host, strconv.Itoa(int(p.Spec.Gateways[0].Port)))
}

// GetAllPeersHandler -Get all peers HTTP handler
func GetAllPeersHandler(w http.ResponseWriter, _ *http.Request) {

	// Get Peer control plane logic
	p := getAllPeers()

	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get all peers control plane logic
func getAllPeers() []api.Peer {
	// Update MBG state
	store.UpdateState()
	pArr := []api.Peer{}

	for _, s := range store.GetMbgList() {
		ip, sport := store.GetMbgTargetPair(s)
		port, _ := strconv.ParseUint(sport, 10, 32)
		pArr = append(pArr, api.Peer{Name: s, Spec: api.PeerSpec{Gateways: []api.Endpoint{{Host: ip, Port: uint16(port)}}}})
	}
	return pArr

}

// GetPeerHandler - Get peer HTTP handler
func GetPeerHandler(w http.ResponseWriter, r *http.Request) {
	peerID := chi.URLParam(r, "id")

	// AddPeer control plane logic
	p := getPeer(peerID)

	// Response
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(p); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get peer control plane logic
func getPeer(peerID string) api.Peer {
	peer := api.Peer{}
	// Update MBG state
	store.UpdateState()
	ok := store.IsMbgPeer(peerID)
	if ok {
		ip, sport := store.GetMbgTargetPair(peerID)
		port, _ := strconv.ParseUint(sport, 10, 32)
		peer = api.Peer{Name: peerID, Spec: api.PeerSpec{Gateways: []api.Endpoint{{Host: ip, Port: uint16(port)}}}}
	} else {
		plog.Infof("MBG %s does not exist in the peers list ", peerID)
	}
	return peer

}

// RemovePeerHandler - Remove peer HTTP handler
func RemovePeerHandler(w http.ResponseWriter, r *http.Request) {
	// Parse add peer struct from request
	var p api.Peer
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// RemovePeer control plane logic
	removePeer(p)

	// Response
	w.WriteHeader(http.StatusNoContent)
	_, err = w.Write([]byte("Remove Peer succeed"))
	if err != nil {
		plog.Println(err)
	}
}

// Remove peer control plane logic
func removePeer(p api.Peer) {
	// Update MBG state
	store.UpdateState()

	err := store.GetEventManager().RaiseRemovePeerEvent(eventmanager.RemovePeerAttr{PeerMbg: p.Name})
	if err != nil {
		plog.Errorf("Unable to raise connection request event")
		return
	}

	// Remove peer from current GW peer
	store.RemoveMbg(p.Name)
	health.RemoveLastSeen(p.Name)
}
