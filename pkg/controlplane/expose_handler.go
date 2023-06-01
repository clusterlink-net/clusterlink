package controlplane

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"

	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
)

func ExposePostHandler(w http.ResponseWriter, r *http.Request) {

	//phrase expose struct from request
	var e apiObject.ExposeRequest
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	//Expose control plane logic
	log.Infof("Received expose to service: %v", e.Id)
	err = Expose(e)

	//Response
	if err != nil {
		log.Error("Expose error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("Expose succeed"))
	}

}
