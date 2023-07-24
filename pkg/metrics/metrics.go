/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package metrics

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
	ConnectionFlow map[string]*event.ConnectionStatusAttr
}

func (m *Metrics) Routes(r *chi.Mux) chi.Router {
	r.Route("/"+event.ConnectionStatus, func(r chi.Router) {
		r.Get("/", m.GetConnectionMetrics)   // Get Metrics from the metrics manager
		r.Post("/", m.PostConnectionMetrics) // Post Metrics to the metrics manager
	})
	// TODO : Add more endpoints to support query
	return r
}

func (m *Metrics) init(router *chi.Mux) {
	m.ConnectionFlow = make(map[string]*event.ConnectionStatusAttr)

	routes := m.Routes(router)

	router.Mount("/metrics", routes)
}

func (m *Metrics) GetConnectionMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(m.ConnectionFlow); err != nil {
		mlog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

func (m *Metrics) PostConnectionMetrics(w http.ResponseWriter, r *http.Request) {
	var connectionStatus event.ConnectionStatusAttr
	err := json.NewDecoder(r.Body).Decode(&connectionStatus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Aggregate Metrics
	m.aggregateMetrics(connectionStatus)

	return
}

func (m *Metrics) aggregateMetrics(connectionStatus event.ConnectionStatusAttr) {
	if _, exists := m.ConnectionFlow[connectionStatus.ConnectionId]; exists {
		// Update existing metrics
		flow := m.ConnectionFlow[connectionStatus.ConnectionId]
		flow.IncomingBytes += connectionStatus.IncomingBytes
		flow.OutgoingBytes += connectionStatus.OutgoingBytes
		flow.LastTstamp = connectionStatus.LastTstamp
		flow.State = connectionStatus.State
		//m.ConnectionFlow[connectionStatus.ConnectionId] = flow
		return
	} else {
		m.ConnectionFlow[connectionStatus.ConnectionId] = &connectionStatus
	}
}

func StartMetricsManager(router *chi.Mux) {
	mlog.Infof("Metrics Manager started")
	MyMetricsManager.init(router)
}
