package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
)

/******************* Local Service ****************************************/
func AddLocalServicePostHandler(w http.ResponseWriter, r *http.Request) {

	//phrase add service struct from request
	var s apiObject.ServiceRequest
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//AddService control plane logic
	log.Debugf("Received Add local service command to service: %v", s.Id)
	AddLocalService(s)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Service to MBG succeed"))
	if err != nil {
		log.Println(err)
	}
}

func AllLocalServicesGetHandler(w http.ResponseWriter, r *http.Request) {

	//GetService control plane logic
	log.Debug("Received get local services command")
	sArr := GetAllLocalServices()

	//Response
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(sArr)
	if err != nil {
		log.Errorf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	log.Debug("Send all services")
	_, err = w.Write(jsonResp)
	if err != nil {
		log.Println(err)
	}

}

func LocalServiceGetHandler(w http.ResponseWriter, r *http.Request) {

	svcId := chi.URLParam(r, "svcId")

	//GetService control plane logic
	log.Debugf("Received get local service command to service: %v", svcId)
	s := GetLocalService(svcId)

	//Response
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(s)
	if err != nil {
		log.Errorf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	_, err = w.Write(jsonResp)
	if err != nil {
		log.Println(err)
	}
}

func DelLocalServiceHandler(w http.ResponseWriter, r *http.Request) {

	//phrase del service struct from request
	svcId := chi.URLParam(r, "svcId")

	//AddService control plane logic
	log.Debugf("Received delete local service command to service: %v", svcId)
	DelLocalService(svcId)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Service deleted successfully"))
	if err != nil {
		log.Println(err)
	}
}

func DelLocalServiceFromPeerHandler(w http.ResponseWriter, r *http.Request) {
	//phrase del service struct from request
	var s apiObject.ServiceDeleteRequest
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	//AddService control plane logic
	log.Infof("Received delete local service : %v from peer: %v", s.Id, s.Peer)
	DelLocalServiceFromPeer(s.Id, s.Peer)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Service " + s.Id + " deleted successfully from peer " + s.Peer))
	if err != nil {
		log.Println(err)
	}
}

/******************* Remote Service ****************************************/
func AddRemoteServicePostHandler(w http.ResponseWriter, r *http.Request) {

	//phrase add service struct from request
	var e apiObject.ExposeRequest
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	//AddService control plane logic
	log.Debugf("Received Add remote service command to service: %v", e.Id)
	AddRemoteService(e)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Remote Service to MBG succeed"))
	if err != nil {
		log.Println(err)
	}
}

func AllRemoteServicesGetHandler(w http.ResponseWriter, r *http.Request) {

	//GetService control plane logic
	log.Debug("Received get Remote services command")
	sArr := GetAllRemoteServices()

	//Response
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(sArr)
	if err != nil {
		log.Errorf("Error happened in JSON marshal. Err: %s", err)
		return
	}

	_, err = w.Write(jsonResp)
	if err != nil {
		log.Println(err)
	}

}

func RemoteServiceGetHandler(w http.ResponseWriter, r *http.Request) {

	svcId := chi.URLParam(r, "svcId")

	//GetService control plane logic
	log.Infof("Received get local service command to service: %v", svcId)
	s := GetRemoteService(svcId)

	//Response
	w.Header().Set("Content-Type", "application/json")
	jsonResp, err := json.Marshal(s)
	if err != nil {
		log.Errorf("Error happened in JSON marshal. Err: %s", err)
		return
	}
	_, err = w.Write(jsonResp)
	if err != nil {
		log.Println(err)
	}
}

func DelRemoteServiceHandler(w http.ResponseWriter, r *http.Request) {

	//phrase del service struct from request
	svcId := chi.URLParam(r, "svcId")
	//phrase add service struct from request
	var s apiObject.ServiceRequest
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//AddService control plane logic
	log.Debugf("Received delete remote service command to service: %v", svcId)
	DelRemoteService(svcId, s.MbgID)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Service deleted successfully"))
	if err != nil {
		log.Println(err)
	}
}
