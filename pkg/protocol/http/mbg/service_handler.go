package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/mbgControlplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

func (m MbgHandler) addServicePost(w http.ResponseWriter, r *http.Request) {

	//phrase add service struct from request
	var e protocol.ServiceRequest
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	//AddService control plane logic
	log.Infof("Received Add service command to service: %v", e.Id)
	mbgControlplane.AddService(e)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Service to MBG succeed"))
	if err != nil {
		log.Println(err)
	}
}

func (m MbgHandler) allServicesGet(w http.ResponseWriter, r *http.Request) {

	//GetService control plane logic
	log.Info("Received get service command")
	sArr := mbgControlplane.GetAllServices()

	//Response
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(sArr)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	_, err = w.Write(jsonResp)
	if err != nil {
		log.Println(err)
	}
}

func (m MbgHandler) serviceGet(w http.ResponseWriter, r *http.Request) {

	svcId := chi.URLParam(r, "svcId")

	//GetService control plane logic
	log.Infof("Received get service command to service: %v", svcId)
	s := mbgControlplane.GetService(svcId)

	//Response
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(s)
	if err != nil {
		log.Fatalf("Error happened in JSON marshal. Err: %s", err)
	}
	_, err = w.Write(jsonResp)
	if err != nil {
		log.Println(err)
	}
}
