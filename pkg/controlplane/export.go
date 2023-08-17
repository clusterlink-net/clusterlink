package controlplane

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"

	"github.ibm.com/mbg-agent/pkg/api"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/k8s/kubernetes"
)

var slog = logrus.WithField("component", "mbgControlPlane/export")

// AddExportServiceHandler - HTTP handler for add export service
func AddExportServiceHandler(w http.ResponseWriter, r *http.Request) {

	// Parse add service struct from request
	var e api.Export
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Infof("AddExportServiceHandler for service: %v", e)
	// AddService control plane logic
	err = addExportService(e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Response
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte("Add Service to MBG succeed"))
	if err != nil {
		slog.Println(err)
	}
}

// Add local service - control plane logic
func addExportService(e api.Export) error {
	store.UpdateState()
	store.AddLocalService(e.Name, e.Spec.Service.Host, e.Spec.Service.Port)
	if e.Spec.ExternalService.Host != "" && e.Spec.ExternalService.Port != 0 {
		err := createK8sExternalEndpoint(e)
		if err != nil {
			return err
		}
	}
	return nil
}

func createK8sExternalEndpoint(e api.Export) error {
	dataplanePod, err := kubernetes.Data.GetInfoApp(K8sSvcApp)
	if err != nil {
		return err
	}
	namespace := dataplanePod.Namespace
	mlog.Infof("Creating K8s endPoint at %s:%d in namespace %s that connected to external IP: %s:%d", e.Spec.Service.Host, e.Spec.Service.Port, namespace, e.Spec.ExternalService.Host, e.Spec.ExternalService.Port)
	err = kubernetes.Data.CreateEndpoint(e.Spec.Service.Host, namespace, e.Spec.ExternalService.Host, int(e.Spec.ExternalService.Port))
	if err != nil {
		return err
	}

	mlog.Infof("Creating k8s service at %s:%d in namespace %s that connected to endpoint %s", e.Name, e.Spec.Service.Port, namespace, e.Spec.Service.Host)
	err = kubernetes.Data.CreateService(e.Spec.Service.Host, int(e.Spec.Service.Port), int(e.Spec.Service.Port), namespace, "")
	if err != nil {
		mlog.Infoln("Error in creating k8s service:", err)
		mlog.Infof("Deleting K8s endPoint at %s:%d in namespace %s that connected to external IP: %s:%d", e.Spec.Service.Host, e.Spec.Service.Port, namespace, e.Spec.ExternalService.Host, e.Spec.ExternalService.Port)
		kubernetes.Data.DeleteEndpoint(e.Spec.Service.Host)
		return err
	}

	return nil
}

// GetExportServiceHandler - HTTP handler for get local service
func GetExportServiceHandler(w http.ResponseWriter, r *http.Request) {
	svcID := chi.URLParam(r, "id")

	// GetService control plane logic
	s := getExportService(svcID)
	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(s); err != nil {
		slog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get local service - control plane logic
func getExportService(svcID string) api.Export {
	store.UpdateState()
	s := store.GetLocalService(svcID)
	port, _ := strconv.Atoi(s.Port)
	return api.Export{Name: s.Id, Spec: api.ExportSpec{Service: api.Endpoint{Host: s.Ip, Port: uint16(port)}}}
}

// GetAllExportServicesHandler - HTTP handler for Get all export services
func GetAllExportServicesHandler(w http.ResponseWriter, r *http.Request) {
	sArr := getAllExportServices() // GetService control plane logic

	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sArr); err != nil {
		slog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get all local services - control plane logic
func getAllExportServices() map[string]api.Export {
	store.UpdateState()
	sArr := make(map[string]api.Export)

	for _, s := range store.GetLocalServicesArr() {
		sPort := store.GetConnectionArr()[s.Id]
		sIP := store.GetMyIp()
		port, _ := strconv.Atoi(sPort)
		sArr[s.Id] = api.Export{Name: s.Id, Spec: api.ExportSpec{Service: api.Endpoint{Host: sIP, Port: uint16(port)}}}
	}

	return sArr
}

// DelExportServiceHandler - HTTP handler for delete local service -
func DelExportServiceHandler(w http.ResponseWriter, r *http.Request) {
	// Parse del service struct from request
	svcID := chi.URLParam(r, "id")

	// AddService control plane logic
	err := delExportService(svcID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Response
	w.WriteHeader(http.StatusNoContent)
	_, err = w.Write([]byte("Service deleted successfully"))
	if err != nil {
		slog.Println(err)
	}
}

// Delete local service - control plane logic
func delExportService(svcID string) error {
	store.UpdateState()
	var svcArr []store.LocalService
	if svcID == "*" { // remove all services
		svcArr = append(svcArr, maps.Values(store.GetLocalServicesArr())...)
	} else {
		svcArr = append(svcArr, store.GetLocalService(svcID))
	}

	for _, svc := range svcArr {
		store.DelLocalService(svc.Id)
		if kubernetes.Data.CheckEndpointExist(svc.Id) {
			if err := kubernetes.Data.DeleteEndpoint(svc.Id); err != nil {
				return err
			}
		}
		if kubernetes.Data.CheckServiceExist(svc.Id) {
			if err := kubernetes.Data.DeleteService(svc.Id); err != nil {
				return err
			}
		}
	}
	return nil

}
