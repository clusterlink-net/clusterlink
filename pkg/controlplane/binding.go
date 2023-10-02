package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.com/clusterlink-org/clusterlink/pkg/api"
	"github.com/clusterlink-org/clusterlink/pkg/controlplane/eventmanager"
	"github.com/clusterlink-org/clusterlink/pkg/controlplane/store"
)

var blog = logrus.WithField("component", "mbgControlPlane/binding")

// CreateBindingHandler - HTTP handler for binding an import service
func CreateBindingHandler(w http.ResponseWriter, r *http.Request) {

	// Parse add service struct from request
	var b api.Binding
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = createBinding(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Response
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte("Add Remote Service to MBG succeed"))
	if err != nil {
		blog.Println(err)
	}
}

func createBinding(b api.Binding) error {
	policyResp, err := store.GetEventManager().RaiseNewRemoteServiceEvent(eventmanager.NewRemoteServiceAttr{Service: b.Spec.Import, Mbg: b.Spec.Peer})
	if err != nil {
		blog.Error("unable to raise connection request event ", store.GetMyID())
		return err
	}
	if policyResp.Action == eventmanager.Deny {
		blog.Errorf("unable to create service endpoint due to policy")
		return err
	}
	PeerIP := store.GetMbgTarget(b.Spec.Peer)
	store.AddRemoteService(b.Spec.Import, PeerIP, "", b.Spec.Peer)
	return nil
}

// DelBindingHandler - HTTP handler for delete an import service -
func DelBindingHandler(w http.ResponseWriter, r *http.Request) {
	// Parse add service struct from request
	var s api.Binding
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// AddService control plane logic
	blog.Infof("Received delete binding command to import: %v", s.Spec.Import)
	delBinding(s.Spec.Import, s.Spec.Peer)

	// Response
	w.WriteHeader(http.StatusNoContent)
	_, err = w.Write([]byte("Service deleted successfully"))
	if err != nil {
		blog.Println(err)
	}
}

// Delete remote service - control logic
func delBinding(svcID, gwID string) {
	store.UpdateState()
	store.DelRemoteService(svcID, gwID)
}

// GetBindingHandler - HTTP handler for get binding
func GetBindingHandler(w http.ResponseWriter, r *http.Request) {

	importID := chi.URLParam(r, "id")
	// GetService control plane logic
	blog.Infof("Received get binding command to service: %v", importID)
	bArr := getBinding(importID)

	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(bArr); err != nil {
		blog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

func getBinding(svcID string) []api.Binding {
	bArr := []api.Binding{}
	for _, s := range store.GetRemoteService(svcID) {
		bArr = append(bArr, api.Binding{Spec: api.BindingSpec{Import: s.ID, Peer: s.MbgID}})
	}
	blog.Infof("getBinding bArr: %v", bArr)

	return bArr

}
