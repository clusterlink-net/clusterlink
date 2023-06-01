package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	cp "github.ibm.com/mbg-agent/pkg/controlplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

// Send hello to specific mbg
func (m MbgHandler) sendHello(w http.ResponseWriter, r *http.Request) {

	//phrase hello struct from request
	mbgID := chi.URLParam(r, "mbgID")

	//Hello control plane logic
	log.Infof("Send Hello to MBG id: %v", mbgID)
	resp := cp.SendHello(mbgID)

	j, err := json.Marshal(protocol.HelloResponse{Status: resp})
	if err != nil {
		log.Error(err)
		return
	}
	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(j)
	if err != nil {
		log.Println(err)
	}
}

// Send hello to all mbg peers
func (m MbgHandler) sendHello2All(w http.ResponseWriter, r *http.Request) {

	//Hello control plane logic
	log.Infof("Send Hello to MBG peers")
	resp := cp.SendHello2All()

	j, err := json.Marshal(protocol.HelloResponse{Status: resp})
	if err != nil {
		log.Error(err)
		return
	}
	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(j)
	if err != nil {
		log.Println(err)
	}
}

// Send hello to all mbg peers
func (m MbgHandler) handleHB(w http.ResponseWriter, r *http.Request) {
	var h protocol.HeartBeat
	err := json.NewDecoder(r.Body).Decode(&h)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cp.RecvHeartbeat(h.Id)

	//Response
	w.WriteHeader(http.StatusOK)
}
