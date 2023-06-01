package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/healthMonitor"
)

// Send hello to specific mbg
func SendHelloHandler(w http.ResponseWriter, r *http.Request) {

	//phrase hello struct from request
	mbgID := chi.URLParam(r, "mbgID")

	//Hello control plane logic
	log.Infof("Send Hello to MBG id: %v", mbgID)
	resp := SendHello(mbgID)

	j, err := json.Marshal(apiObject.HelloResponse{Status: resp})
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
func SendHello2AllHandler(w http.ResponseWriter, r *http.Request) {

	//Hello control plane logic
	log.Infof("Send Hello to MBG peers")
	resp := SendHello2All()

	j, err := json.Marshal(apiObject.HelloResponse{Status: resp})
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
func HandleHB(w http.ResponseWriter, r *http.Request) {
	var h apiObject.HeartBeat
	err := json.NewDecoder(r.Body).Decode(&h)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	healthMonitor.RecvHeartbeat(h.Id)

	//Response
	w.WriteHeader(http.StatusOK)
}
