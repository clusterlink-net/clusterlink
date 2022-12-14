package handler

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/mbgControlplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

func (m MbgHandler) connectDelete(w http.ResponseWriter, r *http.Request) {
	//phrase Disconnect struct from request
	var d protocol.DisconnectRequest
	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Infof("Received Disconnect message for connection between %v to %v", d.Id, d.IdDest)

	//Expose control plane logic
	log.Infof("Received disconnect to service: %v", d.Id)
	mbgControlplane.Disconnect(d)
	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Disconnect succeed"))
	if err != nil {
		log.Println(err)
	}
}
