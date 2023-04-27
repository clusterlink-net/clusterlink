package handler

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/mbgControlplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

func (m MbgHandler) exposePost(w http.ResponseWriter, r *http.Request) {

	//phrase expose struct from request
	var e protocol.ExposeRequest
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	//Expose control plane logic
	log.Infof("Received expose to service: %v", e.Id)
	err = mbgControlplane.Expose(e)

	//Response
	if err != nil {
		log.Error("Expose error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("Expose succeed"))
	}

}
