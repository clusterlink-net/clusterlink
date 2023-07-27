package controlplane

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"

	"github.ibm.com/mbg-agent/pkg/api/admin"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
)

var slog = logrus.WithField("component", "mbgControlPlane/export")

// AddExportServiceHandler - HTTP handler for add export service
func AddExportServiceHandler(w http.ResponseWriter, r *http.Request) {

	// Parse add service struct from request
	var e admin.Export
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Infof("AddExportServiceHandler for service: %v", e)
	// AddService control plane logic
	addExportService(e)

	// Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Add Service to MBG succeed"))
	if err != nil {
		slog.Println(err)
	}
}

// Add local service - control plane logic
func addExportService(e admin.Export) {
	store.UpdateState()
	store.AddLocalService(e.Name, e.Spec.Service.Host, e.Spec.Service.Port)
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
func getExportService(svcID string) admin.Export {
	store.UpdateState()
	s := store.GetLocalService(svcID)
	port, _ := strconv.Atoi(s.Port)
	return admin.Export{Name: s.Id, Spec: admin.ExportSpec{Service: admin.Endpoint{Host: s.Ip, Port: uint16(port)}}}
}

// GetAllExportServicesHandler - HTTP handler for Get all export services
func GetAllExportServicesHandler(w http.ResponseWriter, r *http.Request) {

	// GetService control plane logic
	sArr := getAllExportServices()

	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(sArr); err != nil {
		slog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Get all local services - control plane logic
func getAllExportServices() map[string]admin.Export {
	store.UpdateState()
	sArr := make(map[string]admin.Export)

	for _, s := range store.GetLocalServicesArr() {
		sPort := store.GetConnectionArr()[s.Id]
		sIP := store.GetMyIp()
		port, _ := strconv.Atoi(sPort)
		sArr[s.Id] = admin.Export{Name: s.Id, Spec: admin.ExportSpec{Service: admin.Endpoint{Host: sIP, Port: uint16(port)}}}
	}

	return sArr
}

// DelExportServiceHandler - HTTP handler for delete local service -
func DelExportServiceHandler(w http.ResponseWriter, r *http.Request) {

	// Parse del service struct from request
	svcID := chi.URLParam(r, "id")

	// AddService control plane logic
	delExportService(svcID)

	// Response
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Service deleted successfully"))
	if err != nil {
		slog.Println(err)
	}
}

// Delete local service - control plane logic
func delExportService(svcID string) {
	store.UpdateState()
	var svcArr []store.LocalService
	if svcID == "*" { //remove all services
		svcArr = append(svcArr, maps.Values(store.GetLocalServicesArr())...)
	} else {
		svcArr = append(svcArr, store.GetLocalService(svcID))
	}

	for _, svc := range svcArr {
		store.DelLocalService(svc.Id)
	}
}
