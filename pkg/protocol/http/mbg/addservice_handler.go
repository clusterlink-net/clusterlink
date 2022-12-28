package handler

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/mbgControlplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

func (m MbgHandler) addServicePost(w http.ResponseWriter, r *http.Request) {

	//phrase add service struct from request
	var e protocol.AddServiceRequest
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	//Expose control plane logic
	log.Infof("Received Add service command to service: %v", e.Id)
	mbgControlplane.AddService(e)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Service to MBG succeed"))
	if err != nil {
		log.Println(err)
	}
}
