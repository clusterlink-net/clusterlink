package controlplane

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	kubernetes "github.ibm.com/mbg-agent/pkg/k8s/kubernetes"
	"github.ibm.com/mbg-agent/pkg/utils/httputils"
)

var mlog = logrus.WithField("component", "mbgControlPlane/Expose")

// Expose HTTP handler
func ExposeHandler(w http.ResponseWriter, r *http.Request) {

	// Parse expose struct from request
	var e apiObject.ExposeRequest
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	// Expose control plane logic
	mlog.Infof("Received expose to service: %v", e.Id)
	err = expose(e)

	// Response
	if err != nil {
		mlog.Error("Expose error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("Expose succeed"))
	}

}

// Expose control plane logic
func expose(e apiObject.ExposeRequest) error {
	// Update MBG state
	store.UpdateState()
	return exposeToMbg(e.Id, e.MbgID)
}

func exposeToMbg(serviceId, peerId string) error {
	exposeResp, err := store.GetEventManager().RaiseExposeRequestEvent(eventManager.ExposeRequestAttr{Service: serviceId})
	if err != nil {
		return fmt.Errorf("Unable to raise expose request event")
	}
	mlog.Infof("Response = %+v", exposeResp)
	if exposeResp.Action == eventManager.Deny {
		mlog.Errorf("Denying Expose of service %s", serviceId)
		return fmt.Errorf("Denying Expose of service %s", serviceId)
	}

	myIp := store.GetMyIp()
	svcExp := store.GetLocalService(serviceId)
	if svcExp.Ip == "" {
		return fmt.Errorf("Denying Expose of service %s - target is not set", serviceId)
	}
	svcExp.Ip = myIp
	if peerId == "" { //Expose to all
		if exposeResp.Action == eventManager.AllowAll {
			for _, mbgId := range store.GetMbgList() {
				exposeReq(svcExp, mbgId, "MBG")
			}
			return nil
		}
		for _, mbgId := range exposeResp.TargetMbgs {
			exposeReq(svcExp, mbgId, "MBG")
		}
	} else { // Expose to specific peer
		if slices.Contains(exposeResp.TargetMbgs, peerId) {
			exposeReq(svcExp, peerId, "MBG")
		}
	}
	return nil
}

func exposeReq(svcExp store.LocalService, mbgId, cType string) {
	destIp := store.GetMbgTarget(mbgId)
	mlog.Printf("Starting to expose service %v (%v)", svcExp.Id, destIp)
	address := store.GetAddrStart() + destIp + "/remoteservice"

	j, err := json.Marshal(apiObject.ExposeRequest{Id: svcExp.Id, Ip: svcExp.Ip, Description: svcExp.Description, MbgID: store.GetMyId()})
	if err != nil {
		mlog.Error(err)
		return
	}
	// Send expose
	resp, err := httputils.HttpPost(address, j, store.GetHttpClient())
	mlog.Infof("Service(%s) Expose Response message:  %s", svcExp.Id, string(resp))
	if string(resp) != httputils.RESPFAIL {
		store.AddPeerLocalService(svcExp.Id, mbgId)
	}
}

// Binding local service HTTP handler
func CreateBindingHandler(w http.ResponseWriter, r *http.Request) {
	// Parse expose struct from request
	var b apiObject.BindingRequest
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	mlog.Infof("Creating binding to service: %+v", b)
	err = createLocalServiceEndpoint(b.Id, b.Port, b.Name, b.Namespace, b.MbgApp)
	if err != nil {
		mlog.Errorf("Unable to create binding: %+v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Response
	w.WriteHeader(http.StatusOK)
}

// Create local service endpoint after import a service
func createLocalServiceEndpoint(serviceId string, port int, name, namespace, mbgAppName string) error {
	sPort := store.GetConnectionArr()[serviceId].Local

	targetPort, err := strconv.Atoi(sPort[1:])
	if err != nil {
		return err
	}
	mlog.Infof("Creating service end point at %s:%d:%d for service %s", name, port, targetPort, serviceId)
	return kubernetes.Data.CreateServiceEndpoint(name, port, targetPort, namespace, mbgAppName)
}

// Delete local service binding HTTP handler
func DeleteBindingHandler(w http.ResponseWriter, r *http.Request) {
	svcId := chi.URLParam(r, "svcId")

	mlog.Infof("Removing binding to service: %s", svcId)
	err := deleteLocalServiceEndpoint(svcId)
	if err != nil {
		mlog.Errorf("Unable to delete binding: %+v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Response
	w.WriteHeader(http.StatusOK)

}

// Delete local service endpoint after import a service
func deleteLocalServiceEndpoint(serviceId string) error {
	mlog.Infof("Deleting service end point at %s", serviceId)
	return kubernetes.Data.DeleteServiceEndpoint(serviceId)
}
