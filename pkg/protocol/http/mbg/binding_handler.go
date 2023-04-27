package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/mbgControlplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

func (m MbgHandler) bindingCreate(w http.ResponseWriter, r *http.Request) {

	//phrase expose struct from request
	var b protocol.BindingRequest
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	log.Infof("Creating binding to service: %+v", b)
	err = mbgControlplane.CreateLocalServiceEndpoint(b.Id, b.Port, b.Namespace, b.MbgApp)
	if err != nil {
		log.Errorf("Unable to create binding: %+v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//Response
	w.WriteHeader(http.StatusOK)
}

func (m MbgHandler) bindingDelete(w http.ResponseWriter, r *http.Request) {
	svcId := chi.URLParam(r, "svcId")

	log.Infof("Removing binding to service: %s", svcId)
	err := mbgControlplane.DeleteLocalServiceEndpoint(svcId)
	if err != nil {
		log.Errorf("Unable to delete binding: %+v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//Response
	w.WriteHeader(http.StatusOK)

}
