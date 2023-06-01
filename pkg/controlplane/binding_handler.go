package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/protocol"
)

func BindingCreateHandler(w http.ResponseWriter, r *http.Request) {

	//phrase expose struct from request
	var b protocol.BindingRequest
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	log.Infof("Creating binding to service: %+v", b)
	err = CreateLocalServiceEndpoint(b.Id, b.Port, b.Name, b.Namespace, b.MbgApp)
	if err != nil {
		log.Errorf("Unable to create binding: %+v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//Response
	w.WriteHeader(http.StatusOK)
}

func BindingDeleteHandler(w http.ResponseWriter, r *http.Request) {
	svcId := chi.URLParam(r, "svcId")

	log.Infof("Removing binding to service: %s", svcId)
	err := DeleteLocalServiceEndpoint(svcId)
	if err != nil {
		log.Errorf("Unable to delete binding: %+v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//Response
	w.WriteHeader(http.StatusOK)

}
