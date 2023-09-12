package controlplane

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/api"
	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventmanager"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	kubernetes "github.ibm.com/mbg-agent/pkg/k8s/kubernetes"
	"github.ibm.com/mbg-agent/pkg/utils/httputils"
)

var mlog = logrus.WithField("component", "ControlPlane/Import")

// K8sSvcApp represent the dataplane app name for creating k8s svc
const (
	K8sSvcApp = "dataplane"
)

// AddImportServiceHandler - HTTP handler for add remote service
func AddImportServiceHandler(w http.ResponseWriter, r *http.Request) {

	// Parse add service struct from request
	var e api.Import
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	// AddService control plane logic
	err = addImportService(e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Response
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte("Add Import Service to MBG succeed"))
	if err != nil {
		mlog.Println(err)
	}
}

// Add remote service - control logic
func addImportService(e api.Import) error {
	mlog.Infoln("addImportService ", e)
	err := createImportServiceEndpoint(e)
	if err != nil {
		mlog.Error("addImportService", err)
		return err
	}
	if MyRunTimeEnv.IsRuntimeEnvK8s() {
		err = createImportK8sService(e)
		if err != nil {
			mlog.Error("createImportK8sService", err)
			return err
		}
	}
	return nil
}

// Create remote service proxy
func createImportServiceEndpoint(e api.Import) error {
	address := store.GetAddrStart() + store.GetDataplaneEndpoint() + "/imports/serviceEndpoint/"

	j, err := json.Marshal(e)
	if err != nil {
		mlog.Error(err)
		return err
	}
	// Send Import
	resp, err := httputils.Post(address, j, store.GetHTTPClient())
	mlog.Infof("Create connection request to address %s data-plane for service(%s)- %s ", address, e.Name, string(resp))
	if err != nil {
		mlog.Error(err)
		return err
	}
	var r apiObject.ImportReply
	err = json.Unmarshal(resp, &r)
	if err != nil {
		mlog.Error(err)
		return err
	}
	store.SetConnection(r.ID, r.Port)
	return nil
}

// GetImportServiceHandler - get a import service - HTTP handler
func GetImportServiceHandler(w http.ResponseWriter, r *http.Request) {

	svcID := chi.URLParam(r, "id")

	// GetService control plane logic
	mlog.Infof("Received get local service command to service: %v", svcID)
	s := getImportService(svcID)

	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(s); err != nil {
		mlog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get remote service - control logic
func getImportService(svcID string) api.Import {
	store.UpdateState()
	return convertImportServiceToImportReq(svcID)
}

// GetAllImportServicesHandler Get All - remote service HTTP handler
func GetAllImportServicesHandler(w http.ResponseWriter, _ *http.Request) {
	// GetService control plane logic
	mlog.Infof("Received get all import services")
	sArr := getAllImportServices()

	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sArr); err != nil {
		mlog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get All remote service - control logic
func getAllImportServices() []api.Import {
	store.UpdateState()
	sArr := []api.Import{}

	for svcID := range store.GetRemoteServicesArr() {
		sArr = append(sArr, convertImportServiceToImportReq(svcID))
	}
	return sArr
}

// Convert service object to service request object
func convertImportServiceToImportReq(svcID string) api.Import {
	for _, s := range store.GetRemoteService(svcID) {
		sPort := store.GetConnectionArr()[s.ID]
		port, _ := strconv.Atoi(sPort)
		iSvc := api.Import{Name: s.ID, Spec: api.ImportSpec{Service: api.Endpoint{Host: s.ID, Port: uint16(port)}}}
		return iSvc
	}
	return api.Import{}

}

// DelImportServiceHandler - HTTP handler for Delete remote service
func DelImportServiceHandler(w http.ResponseWriter, r *http.Request) {
	// Parse del service struct from request
	svcID := chi.URLParam(r, "id")

	// AddService control plane logic
	delImportService(svcID)
	if MyRunTimeEnv.IsRuntimeEnvK8s() {
		if err := deleteImportK8sService(svcID); err != nil {

			http.Error(w, err.Error(), http.StatusInternalServerError)
			mlog.Println(err)
			return
		}
	}

	// Response
	w.WriteHeader(http.StatusNoContent)
}

// Delete remote service - control logic
func delImportService(svcID string) {
	store.UpdateState()
	store.DelRemoteService(svcID, "")
}

// RestoreImportServices restores all import services
func RestoreImportServices() {
	for svcID, svcArr := range store.GetRemoteServicesArr() {
		allow := false
		for _, svc := range svcArr {
			policyResp, err := store.GetEventManager().RaiseNewRemoteServiceEvent(eventmanager.NewRemoteServiceAttr{Service: svc.ID, Mbg: svc.MbgID})
			if err != nil {
				mlog.Error("unable to raise remote service event", store.GetMyID())
				continue
			}
			if policyResp.Action == eventmanager.Deny {
				continue
			}
			allow = true
		}
		// Create service endpoint only if the service from at least one MBG is allowed as per policy
		if allow {
			if err := createImportServiceEndpoint(api.Import{Name: svcID}); err != nil {
				mlog.Error("unable to create import endpoint", svcID)
			}
		}
	}
}

// Create k8s service endpoint after import a service
func createImportK8sService(i api.Import) error {
	sPort := store.GetConnectionArr()[i.Name]

	targetPort, err := strconv.Atoi(sPort[1:])
	if err != nil {
		return err
	}
	dataplanePod, err := kubernetes.Data.GetInfoApp(K8sSvcApp)
	if err != nil {
		return err
	}
	mlog.Infof("Creating service end point at %s:%d:%d in namespace %s for service %s", i.Name, i.Spec.Service.Port, targetPort, dataplanePod.Namespace, i.Name)
	return kubernetes.Data.CreateService(i.Spec.Service.Host, int(i.Spec.Service.Port), targetPort, dataplanePod.Namespace, K8sSvcApp)
}

// Delete local service endpoint after import a service
func deleteImportK8sService(svcID string) error {
	mlog.Infof("Deleting service end point at %s", svcID)
	return kubernetes.Data.DeleteService(svcID)
}
