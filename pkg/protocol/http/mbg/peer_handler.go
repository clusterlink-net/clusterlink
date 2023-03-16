package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/mbgControlplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

func (m MbgHandler) peerPost(w http.ResponseWriter, r *http.Request) {

	//phrase add peer struct from request
	var p protocol.PeerRequest
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//AddPeer control plane logic
	mbgControlplane.AddPeer(p)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Peer succeed"))
	if err != nil {
		log.Println(err)
	}
}

func (m MbgHandler) peerGetAll(w http.ResponseWriter, r *http.Request) {

	//AddPeer control plane logic
	p := mbgControlplane.GetAllPeers()

	//Response
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(p)
	if err != nil {
		log.Errorf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	_, err = w.Write(jsonResp)
	if err != nil {
		log.Println(err)
	}
}

func (m MbgHandler) peerGet(w http.ResponseWriter, r *http.Request) {

	mbgID := chi.URLParam(r, "mbgID")

	//AddPeer control plane logic
	p := mbgControlplane.GetPeer(mbgID)

	//Response
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(p)
	if err != nil {
		log.Errorf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	_, err = w.Write(jsonResp)
	if err != nil {
		log.Println(err)
	}
}
func (m MbgHandler) peerRemove(w http.ResponseWriter, r *http.Request) {

	//phrase add peer struct from request
	var p protocol.PeerRemoveRequest
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//RemovePeer control plane logic
	mbgControlplane.RemovePeer(p)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Remove Peer succeed"))
	if err != nil {
		log.Println(err)
	}
}
