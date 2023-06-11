package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/controlplane/health"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/utils/httputils"
)

var plog = logrus.WithField("component", "mbgControlPlane/Peer")

// AddPeer HTTP handler
func AddPeerHandler(w http.ResponseWriter, r *http.Request) {

	// Parse add peer struct from request
	var p apiObject.PeerRequest
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
func addPeer(p apiObject.PeerRequest) {
	// Update MBG state
	store.UpdateState()

	peerResp, err := store.GetEventManager().RaiseAddPeerEvent(eventManager.AddPeerAttr{PeerMbg: p.Id})
	if err != nil {
		plog.Errorf("Unable to raise connection request event")
		return
	}
	if peerResp.Action == eventManager.Deny {
		plog.Infof("Denying add peer(%s) due to policy", p.Id)
		return
	}
	store.AddMbgNbr(p.Id, p.Ip, p.Cport)
}

// Get all peers HTTP handler
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
func getAllPeers() map[string]apiObject.PeerRequest {
	//Update MBG state
	store.UpdateState()
	pArr := make(map[string]apiObject.PeerRequest)

	for _, s := range store.GetMbgList() {
		ip, port := store.GetMbgTargetPair(s)
		pArr[s] = apiObject.PeerRequest{Id: s, Ip: ip, Cport: port}
	}
	return pArr

}

// Get peer HTTP handler
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
func getPeer(peerID string) apiObject.PeerRequest {
	//Update MBG state
	store.UpdateState()
	ok := store.IsMbgPeer(peerID)
	if ok {
		ip, port := store.GetMbgTargetPair(peerID)
		return apiObject.PeerRequest{Id: peerID, Ip: ip, Cport: port}
	} else {
		plog.Infof("MBG %s does not exist in the peers list ", peerID)
		return apiObject.PeerRequest{}
	}

}

// Remove peer HTTP handler
func RemovePeerHandler(w http.ResponseWriter, r *http.Request) {

	//Parse add peer struct from request
	var p apiObject.PeerRemoveRequest
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
func removePeer(p apiObject.PeerRemoveRequest) {
	//Update MBG state
	store.UpdateState()

	err := store.GetEventManager().RaiseRemovePeerEvent(eventManager.RemovePeerAttr{PeerMbg: p.Id})
	if err != nil {
		plog.Errorf("Unable to raise connection request event")
		return
	}
	if p.Propagate {
		// Remove this MBG from the remove MBG's peer list
		peerIP := store.GetMbgTarget(p.Id)
		address := store.GetAddrStart() + peerIP + "/peer/" + store.GetMyId()
		j, err := json.Marshal(apiObject.PeerRemoveRequest{Id: store.GetMyId(), Propagate: false})
		if err != nil {
			return
		}
		httputils.HttpDelete(address, j, store.GetHttpClient())
	}

	// Remove remote MBG from current MBG's peer
	store.RemoveMbg(p.Id)
	health.RemoveLastSeen(p.Id)
}
