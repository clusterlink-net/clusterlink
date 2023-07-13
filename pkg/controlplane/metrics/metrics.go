/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package policyEngine

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
)

var mlog = logrus.WithField("component", "Metrics")
var MyMetricsManager Metrics

type Metrics struct {
	ConnectionFlow map[string]event.ConnectionStatusAttr
}

func (m *Metrics) Routes(r *chi.Mux) chi.Router {

	r.Get("/", m.GetMetrics)
	r.Post("/"+event.ConnectionStatus, m.PostMetrics)

	// TODO : Add more end-points specific to the queries, But this could be done after a broader discussion
	// For Example, /source/<SourceApp> , should return all connections coming out of SourceApp
	// /gateway should return all connections going in/out of a gateway, etc

	return r
}

func (m *Metrics) init(router *chi.Mux) {
	m.ConnectionFlow = make(map[string]event.ConnectionStatusAttr)

	routes := m.Routes(router)

	router.Mount("/metrics", routes)
}

func (m *Metrics) GetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(m.ConnectionFlow); err != nil {
		mlog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

func (m *Metrics) PostMetrics(w http.ResponseWriter, r *http.Request) {
	var connectionStatus event.ConnectionStatusAttr
	err := json.NewDecoder(r.Body).Decode(&connectionStatus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// if _, exists := m.ConnectionFlow[connectionStatus.connectionID]; exists {
	// 	// Recompute
	// 	m.ConnectionFlow[connectionStatus.connectionID]
	// 	return
	// }
	m.ConnectionFlow[connectionStatus.ConnectionId] = connectionStatus
	return
}

func StartMetricsManager(router *chi.Mux) {
	mlog.Infof("Metrics Manager started")
	MyMetricsManager.init(router)
}
