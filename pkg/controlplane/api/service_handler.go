package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	cp "github.ibm.com/mbg-agent/pkg/controlplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

/******************* Local Service ****************************************/
func (m MbgHandler) addLocalServicePost(w http.ResponseWriter, r *http.Request) {

	//phrase add service struct from request
	var s protocol.ServiceRequest
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//AddService control plane logic
	log.Debugf("Received Add local service command to service: %v", s.Id)
	cp.AddLocalService(s)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Service to MBG succeed"))
	if err != nil {
		log.Println(err)
	}
}

func (m MbgHandler) allLocalServicesGet(w http.ResponseWriter, r *http.Request) {

	//GetService control plane logic
	log.Debug("Received get local services command")
	sArr := cp.GetAllLocalServices()

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

func (m MbgHandler) localServiceGet(w http.ResponseWriter, r *http.Request) {

	svcId := chi.URLParam(r, "svcId")

	//GetService control plane logic
	log.Debugf("Received get local service command to service: %v", svcId)
	s := cp.GetLocalService(svcId)

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

func (m MbgHandler) delLocalService(w http.ResponseWriter, r *http.Request) {

	//phrase del service struct from request
	svcId := chi.URLParam(r, "svcId")

	//AddService control plane logic
	log.Debugf("Received delete local service command to service: %v", svcId)
	cp.DelLocalService(svcId)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Service deleted successfully"))
	if err != nil {
		log.Println(err)
	}
}

func (m MbgHandler) delLocalServiceFromPeer(w http.ResponseWriter, r *http.Request) {
	//phrase del service struct from request
	var s protocol.ServiceDeleteRequest
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	//AddService control plane logic
	log.Infof("Received delete local service : %v from peer: %v", s.Id, s.Peer)
	cp.DelLocalServiceFromPeer(s.Id, s.Peer)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Service " + s.Id + " deleted successfully from peer " + s.Peer))
	if err != nil {
		log.Println(err)
	}
}

/******************* Remote Service ****************************************/
func (m MbgHandler) addRemoteServicePost(w http.ResponseWriter, r *http.Request) {

	//phrase add service struct from request
	var e protocol.ExposeRequest
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	//AddService control plane logic
	log.Debugf("Received Add remote service command to service: %v", e.Id)
	cp.AddRemoteService(e)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Remote Service to MBG succeed"))
	if err != nil {
		log.Println(err)
	}
}

func (m MbgHandler) allRemoteServicesGet(w http.ResponseWriter, r *http.Request) {

	//GetService control plane logic
	log.Debug("Received get Remote services command")
	sArr := cp.GetAllRemoteServices()

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

func (m MbgHandler) remoteServiceGet(w http.ResponseWriter, r *http.Request) {

	svcId := chi.URLParam(r, "svcId")

	//GetService control plane logic
	log.Infof("Received get local service command to service: %v", svcId)
	s := cp.GetRemoteService(svcId)

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

func (m MbgHandler) delRemoteService(w http.ResponseWriter, r *http.Request) {

	//phrase del service struct from request
	svcId := chi.URLParam(r, "svcId")
	//phrase add service struct from request
	var s protocol.ServiceRequest
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//AddService control plane logic
	log.Debugf("Received delete remote service command to service: %v", svcId)
	cp.DelRemoteService(svcId, s.MbgID)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Service deleted successfully"))
	if err != nil {
		log.Println(err)
	}
}
