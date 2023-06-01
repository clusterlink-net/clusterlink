package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
)

func PeerPostHandler(w http.ResponseWriter, r *http.Request) {

	//phrase add peer struct from request
	var p apiObject.PeerRequest
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//AddPeer control plane logic
	AddPeer(p)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Peer succeed"))
	if err != nil {
		log.Println(err)
	}
}

func PeerGetAllHandler(w http.ResponseWriter, r *http.Request) {

	//AddPeer control plane logic
	p := GetAllPeers()

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

func PeerGetHandler(w http.ResponseWriter, r *http.Request) {

	mbgID := chi.URLParam(r, "mbgID")

	//AddPeer control plane logic
	p := GetPeer(mbgID)

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
func PeerRemoveHandler(w http.ResponseWriter, r *http.Request) {

	//phrase add peer struct from request
	var p apiObject.PeerRemoveRequest
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//RemovePeer control plane logic
	RemovePeer(p)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Remove Peer succeed"))
	if err != nil {
		log.Println(err)
	}
}
