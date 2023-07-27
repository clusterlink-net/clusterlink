package controlplane

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/api/admin"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/controlplane/health"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
)

var plog = logrus.WithField("component", "mbgControlPlane/Peer")

// AddPeerHandler - Add peer HTTP handler
func AddPeerHandler(w http.ResponseWriter, r *http.Request) {

	// Parse add peer struct from request
	var p admin.Peer
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// AddPeer control plane logic
	addPeer(p)

	// Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Peer succeeded"))
	if err != nil {
		plog.Println(err)
	}
}

// AddPeer control plane logic
func addPeer(p admin.Peer) {
	// Update MBG state
	store.UpdateState()

	peerResp, err := store.GetEventManager().RaiseAddPeerEvent(eventManager.AddPeerAttr{PeerMbg: p.Name})
	if err != nil {
		plog.Errorf("Unable to raise connection request event")
		return
	}
	if peerResp.Action == eventManager.Deny {
		plog.Infof("Denying add peer(%s) due to policy", p.Name)
		return
	}
	plog.Infof("Peer %v Port: %v", p, strconv.Itoa(int(p.Spec.Gateways[0].Port)))
	store.AddMbgNbr(p.Name, p.Spec.Gateways[0].Host, strconv.Itoa(int(p.Spec.Gateways[0].Port)))
}

// GetAllPeersHandler -Get all peers HTTP handler
func GetAllPeersHandler(w http.ResponseWriter, r *http.Request) {

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
func getAllPeers() []admin.Peer {
	//Update MBG state
	store.UpdateState()
	pArr := []admin.Peer{}

	for _, s := range store.GetMbgList() {
		ip, sport := store.GetMbgTargetPair(s)
		port, _ := strconv.ParseUint(sport, 10, 32)
		pArr = append(pArr, admin.Peer{Name: s, Spec: admin.PeerSpec{Gateways: []admin.Endpoint{{Host: ip, Port: uint16(port)}}}})
	}
	return pArr

}

// GetPeerHandler - Get peer HTTP handler
func GetPeerHandler(w http.ResponseWriter, r *http.Request) {

	peerID := chi.URLParam(r, "id")

	//AddPeer control plane logic
	p := getPeer(peerID)

	//Response
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(p); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get peer control plane logic
func getPeer(peerID string) admin.Peer {
	peer := admin.Peer{}
	//Update MBG state
	store.UpdateState()
	ok := store.IsMbgPeer(peerID)
	if ok {
		ip, sport := store.GetMbgTargetPair(peerID)
		port, _ := strconv.ParseUint(sport, 10, 32)
		peer = admin.Peer{Name: peerID, Spec: admin.PeerSpec{Gateways: []admin.Endpoint{{Host: ip, Port: uint16(port)}}}}
	} else {
		plog.Infof("MBG %s does not exist in the peers list ", peerID)
	}
	return peer

}

// RemovePeerHandler - Remove peer HTTP handler
func RemovePeerHandler(w http.ResponseWriter, r *http.Request) {

	//Parse add peer struct from request
	var p admin.Peer
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// RemovePeer control plane logic
	removePeer(p)

	// Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Remove Peer succeed"))
	if err != nil {
		plog.Println(err)
	}
}

// Remove peer control plane logic
func removePeer(p admin.Peer) {
	//Update MBG state
	store.UpdateState()

	err := store.GetEventManager().RaiseRemovePeerEvent(eventManager.RemovePeerAttr{PeerMbg: p.Name})
	if err != nil {
		plog.Errorf("Unable to raise connection request event")
		return
	}

	// Remove peer from current GW peer
	store.RemoveMbg(p.Name)
	health.RemoveLastSeen(p.Name)
}
