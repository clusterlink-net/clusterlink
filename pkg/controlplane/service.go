package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/utils/httputils"
)

var slog = logrus.WithField("component", "mbgControlPlane/AddService")

/******************* Local Service ****************************************/
// Add local service - HTTP handler
func AddLocalServiceHandler(w http.ResponseWriter, r *http.Request) {

	// Parse add service struct from request
	var s apiObject.ServiceRequest
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// AddService control plane logic
	addLocalService(s)

	// Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Service to MBG succeed"))
	if err != nil {
		slog.Println(err)
	}
}

// Add local service - control plane logic
func addLocalService(s apiObject.ServiceRequest) {
	store.UpdateState()
	store.AddLocalService(s.Id, s.Ip, s.Port, s.Description)
}

// Get local service - HTTP handler
func GetLocalServiceHandler(w http.ResponseWriter, r *http.Request) {
	svcId := chi.URLParam(r, "id")

	// GetService control plane logic
	s := getLocalService(svcId)
	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(s); err != nil {
		slog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get local service - control plane logic
func getLocalService(svcId string) apiObject.ServiceRequest {
	store.UpdateState()
	s := store.GetLocalService(svcId)
	return apiObject.ServiceRequest{Id: s.Id, Ip: s.Ip, Port: s.Port, Description: s.Description}
}

// Get all local service - HTTP handler
func GetAllLocalServicesHandler(w http.ResponseWriter, r *http.Request) {

	// GetService control plane logic
	sArr := getAllLocalServices()

	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sArr); err != nil {
		slog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get all local services - control plane logic
func getAllLocalServices() map[string]apiObject.ServiceRequest {
	store.UpdateState()
	sArr := make(map[string]apiObject.ServiceRequest)

	for _, s := range store.GetLocalServicesArr() {
		sPort := store.GetConnectionArr()[s.Id].External
		sIp := store.GetMyIp()
		sArr[s.Id] = apiObject.ServiceRequest{Id: s.Id, Ip: sIp, Port: sPort, Description: s.Description}
	}

	return sArr
}

// Delete local service - HTTP handler
func DelLocalServiceHandler(w http.ResponseWriter, r *http.Request) {

	// Parse del service struct from request
	svcId := chi.URLParam(r, "id")

	// AddService control plane logic
	delLocalService(svcId)

	// Response
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Service deleted successfully"))
	if err != nil {
		slog.Println(err)
	}
}

// Delete local service - control plane logic
func delLocalService(svcId string) {
	store.UpdateState()
	var svcArr []store.LocalService
	if svcId == "*" { //remove all services
		svcArr = append(svcArr, maps.Values(store.GetLocalServicesArr())...)
	} else {
		svcArr = append(svcArr, store.GetLocalService(svcId))
	}

	for _, svc := range svcArr {
		mbg := store.GetMyId()
		for _, peer := range svc.PeersExposed {
			peerIp := store.GetMbgTarget(peer)
			delServiceInPeerReq(svc.Id, mbg, peerIp)
		}
		store.DelLocalService(svc.Id)
	}
}

// Delete local service from specific peer- HTTP handler
func DelLocalServiceFromPeerHandler(w http.ResponseWriter, r *http.Request) {
	//Parse del service struct from request
	var s apiObject.ServiceDeleteRequest
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	//AddService control plane logic
	slog.Infof("Received delete local service : %v from peer: %v", s.Id, s.Peer)
	delLocalServiceFromPeer(s.Id, s.Peer)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Service " + s.Id + " deleted successfully from peer " + s.Peer))
	if err != nil {
		slog.Println(err)
	}
}

// Delete local service from specific peer- control plane logic
func delLocalServiceFromPeer(svcId, peer string) {
	store.UpdateState()
	svc := store.GetLocalService(svcId)
	mbg := store.GetMyId()
	if slices.Contains(svc.PeersExposed, peer) {
		peerIp := store.GetMbgTarget(peer)
		delServiceInPeerReq(svcId, mbg, peerIp)
	}
	store.DelPeerLocalService(svcId, peer)
}

// Delete local service from specific peer- http request
func delServiceInPeerReq(svcId, serviceMbg, peerIp string) {
	address := store.GetAddrStart() + peerIp + "/remoteservice/" + svcId
	j, err := json.Marshal(apiObject.ServiceRequest{Id: svcId, MbgID: serviceMbg})
	if err != nil {
		slog.Printf("Unable to marshal json: %v", err)
	}

	//send
	resp, _ := httputils.HttpDelete(address, j, store.GetHttpClient())
	slog.Printf("Response message for deleting service [%s]:%s \n", svcId, string(resp))
}

// Add remote service - HTTP handler
func AddRemoteServiceHandler(w http.ResponseWriter, r *http.Request) {

	// Parse add service struct from request
	var e apiObject.ExposeRequest
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	// AddService control plane logic
	addRemoteService(e)

	// Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Remote Service to MBG succeed"))
	if err != nil {
		slog.Println(err)
	}
}

// Add remote service - control logic
func addRemoteService(e apiObject.ExposeRequest) {
	policyResp, err := store.GetEventManager().RaiseNewRemoteServiceEvent(eventManager.NewRemoteServiceAttr{Service: e.Id, Mbg: e.MbgID})
	if err != nil {
		slog.Error("unable to raise connection request event ", store.GetMyId())
		return
	}
	if policyResp.Action == eventManager.Deny {
		slog.Errorf("unable to create service endpoint due to policy")
		return
	}
	err = createRemoteServiceEndpoint(e)
	if err != nil {
		return
	}
	store.AddRemoteService(e.Id, e.Ip, e.Description, e.MbgID)
}

// Create remote service proxy
func createRemoteServiceEndpoint(e apiObject.ExposeRequest) error {
	address := store.GetAddrStart() + store.GetDataplaneEndpoint() + "/imports/serviceEndpoint/"

	j, err := json.Marshal(e)
	if err != nil {
		mlog.Error(err)
		return err
	}
	// Send expose
	resp, err := httputils.HttpPost(address, j, store.GetHttpClient())
	mlog.Infof("Create connection request to address %s data-plane for service(%s)- %s ", address, e.Id, string(resp))
	if err != nil {
		mlog.Error(err)
		return err
	}
	var r apiObject.ServiceReply
	err = json.Unmarshal(resp, &r)
	if err != nil {
		mlog.Error(err)
		return err
	}
	store.SetConnection(r.Id, r.Port)
	return nil
}

// Get remote service - HTTP handler
func GetRemoteServiceHandler(w http.ResponseWriter, r *http.Request) {

	svcId := chi.URLParam(r, "svcId")

	//GetService control plane logic
	slog.Infof("Received get local service command to service: %v", svcId)
	s := getRemoteService(svcId)

	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(s); err != nil {
		slog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get remote service - control logic
func getRemoteService(svcId string) []apiObject.ServiceRequest {
	store.UpdateState()
	return convertRemoteServiceToRemoteReq(svcId)
}

// Get All remote service - HTTP handler
func GetAllRemoteServicesHandler(w http.ResponseWriter, r *http.Request) {
	// GetService control plane logic
	sArr := getAllRemoteServices()

	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sArr); err != nil {
		slog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get All remote service - control logic
func getAllRemoteServices() map[string][]apiObject.ServiceRequest {
	store.UpdateState()
	sArr := make(map[string][]apiObject.ServiceRequest)

	for svcId, _ := range store.GetRemoteServicesArr() {
		sArr[svcId] = convertRemoteServiceToRemoteReq(svcId)

	}

	return sArr
}

// Convert service object to service request object
func convertRemoteServiceToRemoteReq(svcId string) []apiObject.ServiceRequest {
	sArr := []apiObject.ServiceRequest{}
	for _, s := range store.GetRemoteService(svcId) {
		sPort := store.GetConnectionArr()[s.Id].Local
		sIp := sPort
		sArr = append(sArr, apiObject.ServiceRequest{Id: s.Id, Ip: sIp, Port: sPort, MbgID: s.MbgId, Description: s.Description})
	}
	return sArr
}

// Delete remote service - HTTP handler
func DelRemoteServiceHandler(w http.ResponseWriter, r *http.Request) {
	// Parse del service struct from request
	svcId := chi.URLParam(r, "svcId")
	// Parse add service struct from request
	var s apiObject.ServiceRequest
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// AddService control plane logic
	delRemoteService(svcId, s.MbgID)

	// Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Service deleted successfully"))
	if err != nil {
		slog.Println(err)
	}
}

// Delete remote service - control logic
func delRemoteService(svcId, mbgId string) {
	store.UpdateState()
	if svcId == "*" {
		for sId, _ := range store.GetRemoteServicesArr() {
			store.DelRemoteService(sId, mbgId)
		}
	} else {
		store.DelRemoteService(svcId, mbgId)
	}
}

// Restore remote services
func RestoreRemoteServices() {
	for svcId, svcArr := range store.GetRemoteServicesArr() {
		allow := false
		for _, svc := range svcArr {
			policyResp, err := store.GetEventManager().RaiseNewRemoteServiceEvent(eventManager.NewRemoteServiceAttr{Service: svc.Id, Mbg: svc.MbgId})
			if err != nil {
				slog.Error("unable to raise connection request event", store.GetMyId())
				continue
			}
			if policyResp.Action == eventManager.Deny {
				continue
			}
			allow = true
		}
		// Create service endpoint only if the service from at least one MBG is allowed as per policy
		if allow {
			createRemoteServiceEndpoint(apiObject.ExposeRequest{Id: svcId})
		}
	}
}
