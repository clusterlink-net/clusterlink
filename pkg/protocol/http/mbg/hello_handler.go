package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/mbgControlplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

//Send hello to specific mbg
func (m MbgHandler) sendHello(w http.ResponseWriter, r *http.Request) {

	//phrase hello struct from request
	mbgID := chi.URLParam(r, "mbgID")

	//Hello control plane logic
	log.Infof("Send Hello to MBG id: %v", mbgID)
	mbgControlplane.SendHello(mbgID)

	j, err := json.Marshal(protocol.HelloResponse{Status: "success"})
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

//Send hello to all mbg peers
func (m MbgHandler) sendHello2All(w http.ResponseWriter, r *http.Request) {

	//Hello control plane logic
	log.Infof("Send Hello to MBG peers")
	mbgControlplane.SendHello2All()

	j, err := json.Marshal(protocol.HelloResponse{Status: "success"})
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
